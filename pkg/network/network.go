package network

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"github.com/parandhamareddybommaka/kube/pkg/api/types"
	"github.com/parandhamareddybommaka/kube/pkg/store"
)

type NetworkManager struct {
	store           store.Store
	clusterCIDR     string
	serviceCIDR     string
	dnsIP           string
	endpoints       map[string]*Endpoints
	services        map[string]*types.Service
	mu              sync.RWMutex
	loadBalancers   map[string]*LoadBalancer
}

type Endpoints struct {
	Service   string
	Namespace string
	Addresses []EndpointAddress
}

type EndpointAddress struct {
	IP       string
	Port     int
	Protocol string
	Ready    bool
}

type NetworkConfig struct {
	Store       store.Store
	ClusterCIDR string
	ServiceCIDR string
	DNSIP       string
}

func NewNetworkManager(cfg NetworkConfig) *NetworkManager {
	if cfg.ClusterCIDR == "" {
		cfg.ClusterCIDR = "10.244.0.0/16"
	}
	if cfg.ServiceCIDR == "" {
		cfg.ServiceCIDR = "10.96.0.0/12"
	}
	if cfg.DNSIP == "" {
		cfg.DNSIP = "10.96.0.10"
	}

	return &NetworkManager{
		store:         cfg.Store,
		clusterCIDR:   cfg.ClusterCIDR,
		serviceCIDR:   cfg.ServiceCIDR,
		dnsIP:         cfg.DNSIP,
		endpoints:     make(map[string]*Endpoints),
		services:      make(map[string]*types.Service),
		loadBalancers: make(map[string]*LoadBalancer),
	}
}

func (m *NetworkManager) Run(ctx context.Context) error {
	go m.watchServices(ctx)
	go m.watchPods(ctx)
	return nil
}

func (m *NetworkManager) watchServices(ctx context.Context) {
	watchCh := m.store.Watch(ctx, "/services/")
	for event := range watchCh {
		if event.Type == store.EventDelete {
			m.mu.Lock()
			delete(m.services, event.Key)
			delete(m.endpoints, event.Key)
			delete(m.loadBalancers, event.Key)
			m.mu.Unlock()
			continue
		}

		var svc types.Service
		if err := json.Unmarshal(event.Value, &svc); err != nil {
			continue
		}

		key := fmt.Sprintf("%s/%s", svc.Namespace, svc.Name)
		m.mu.Lock()
		m.services[key] = &svc
		if _, ok := m.loadBalancers[key]; !ok {
			m.loadBalancers[key] = NewLoadBalancer()
		}
		m.mu.Unlock()

		m.updateEndpoints(ctx, &svc)
	}
}

func (m *NetworkManager) watchPods(ctx context.Context) {
	watchCh := m.store.Watch(ctx, "/pods/")
	for event := range watchCh {
		if event.Type == store.EventDelete {
			continue
		}

		var pod types.Pod
		if err := json.Unmarshal(event.Value, &pod); err != nil {
			continue
		}

		m.mu.RLock()
		for _, svc := range m.services {
			if svc.Namespace != pod.Namespace {
				continue
			}
			if m.podMatchesSelector(&pod, svc.Spec.Selector) {
				m.mu.RUnlock()
				m.updateEndpoints(ctx, svc)
				m.mu.RLock()
			}
		}
		m.mu.RUnlock()
	}
}

func (m *NetworkManager) podMatchesSelector(pod *types.Pod, selector map[string]string) bool {
	for k, v := range selector {
		if pod.Labels[k] != v {
			return false
		}
	}
	return true
}

func (m *NetworkManager) updateEndpoints(ctx context.Context, svc *types.Service) {
	data, err := m.store.List(ctx, fmt.Sprintf("/pods/%s/", svc.Namespace))
	if err != nil {
		return
	}

	key := fmt.Sprintf("%s/%s", svc.Namespace, svc.Name)
	endpoints := &Endpoints{
		Service:   svc.Name,
		Namespace: svc.Namespace,
	}

	for _, d := range data {
		var pod types.Pod
		if err := json.Unmarshal(d, &pod); err != nil {
			continue
		}

		if pod.Status.Phase != types.PodRunning || pod.Status.PodIP == "" {
			continue
		}

		if !m.podMatchesSelector(&pod, svc.Spec.Selector) {
			continue
		}

		for _, port := range svc.Spec.Ports {
			endpoints.Addresses = append(endpoints.Addresses, EndpointAddress{
				IP:       pod.Status.PodIP,
				Port:     port.TargetPort,
				Protocol: port.Protocol,
				Ready:    true,
			})
		}
	}

	m.mu.Lock()
	m.endpoints[key] = endpoints
	if lb, ok := m.loadBalancers[key]; ok {
		var backends []string
		for _, addr := range endpoints.Addresses {
			backends = append(backends, fmt.Sprintf("%s:%d", addr.IP, addr.Port))
		}
		lb.UpdateBackends(backends)
	}
	m.mu.Unlock()
}

func (m *NetworkManager) GetEndpoints(namespace, name string) *Endpoints {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.endpoints[fmt.Sprintf("%s/%s", namespace, name)]
}

func (m *NetworkManager) ResolveService(namespace, name string) (string, int, error) {
	m.mu.RLock()
	svc, ok := m.services[fmt.Sprintf("%s/%s", namespace, name)]
	if !ok {
		m.mu.RUnlock()
		return "", 0, fmt.Errorf("service not found")
	}
	m.mu.RUnlock()

	if len(svc.Spec.Ports) == 0 {
		return "", 0, fmt.Errorf("no ports defined")
	}

	return svc.Spec.ClusterIP, svc.Spec.Ports[0].Port, nil
}

func (m *NetworkManager) GetBackend(namespace, name string) (string, error) {
	key := fmt.Sprintf("%s/%s", namespace, name)

	m.mu.RLock()
	lb, ok := m.loadBalancers[key]
	m.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("service not found")
	}

	return lb.Next()
}

type LoadBalancer struct {
	backends []string
	current  int
	mu       sync.Mutex
}

func NewLoadBalancer() *LoadBalancer {
	return &LoadBalancer{
		backends: make([]string, 0),
	}
}

func (lb *LoadBalancer) UpdateBackends(backends []string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.backends = backends
	if lb.current >= len(backends) {
		lb.current = 0
	}
}

func (lb *LoadBalancer) Next() (string, error) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if len(lb.backends) == 0 {
		return "", fmt.Errorf("no backends available")
	}

	backend := lb.backends[lb.current]
	lb.current = (lb.current + 1) % len(lb.backends)
	return backend, nil
}

type DNSServer struct {
	store    store.Store
	addr     string
	services map[string]string
	mu       sync.RWMutex
}

func NewDNSServer(s store.Store, addr string) *DNSServer {
	return &DNSServer{
		store:    s,
		addr:     addr,
		services: make(map[string]string),
	}
}

func (d *DNSServer) Run(ctx context.Context) error {
	go d.watchServices(ctx)
	go d.serve(ctx)
	return nil
}

func (d *DNSServer) watchServices(ctx context.Context) {
	watchCh := d.store.Watch(ctx, "/services/")
	for event := range watchCh {
		if event.Type == store.EventDelete {
			continue
		}

		var svc types.Service
		if err := json.Unmarshal(event.Value, &svc); err != nil {
			continue
		}

		d.mu.Lock()
		fqdn := fmt.Sprintf("%s.%s.svc.cluster.local", svc.Name, svc.Namespace)
		d.services[fqdn] = svc.Spec.ClusterIP
		d.mu.Unlock()
	}
}

func (d *DNSServer) serve(ctx context.Context) {
	addr, err := net.ResolveUDPAddr("udp", d.addr)
	if err != nil {
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return
	}
	defer conn.Close()

	buf := make([]byte, 512)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, remoteAddr, err := conn.ReadFromUDP(buf)
			if err != nil {
				continue
			}
			go d.handleQuery(conn, remoteAddr, buf[:n])
		}
	}
}

func (d *DNSServer) handleQuery(conn *net.UDPConn, addr *net.UDPAddr, query []byte) {
	if len(query) < 12 {
		return
	}

	response := make([]byte, 512)
	copy(response, query)

	response[2] = 0x81
	response[3] = 0x80
	response[6] = 0x00
	response[7] = 0x01

	conn.WriteToUDP(response[:len(query)+16], addr)
}

func (d *DNSServer) Resolve(name string) (string, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	ip, ok := d.services[name]
	return ip, ok
}

type ProxyServer struct {
	network *NetworkManager
	addr    string
}

func NewProxyServer(nm *NetworkManager, addr string) *ProxyServer {
	return &ProxyServer{
		network: nm,
		addr:    addr,
	}
}

func (p *ProxyServer) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", p.addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				continue
			}
		}
		go p.handleConnection(conn)
	}
}

func (p *ProxyServer) handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	localAddr := clientConn.LocalAddr().String()
	host, portStr, _ := net.SplitHostPort(localAddr)
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	p.network.mu.RLock()
	var targetSvc *types.Service
	for _, svc := range p.network.services {
		if svc.Spec.ClusterIP == host {
			for _, sp := range svc.Spec.Ports {
				if sp.Port == port {
					targetSvc = svc
					break
				}
			}
		}
	}
	p.network.mu.RUnlock()

	if targetSvc == nil {
		return
	}

	backend, err := p.network.GetBackend(targetSvc.Namespace, targetSvc.Name)
	if err != nil {
		return
	}

	backendConn, err := net.Dial("tcp", backend)
	if err != nil {
		return
	}
	defer backendConn.Close()

	done := make(chan struct{})
	go func() {
		copyBuffer(backendConn, clientConn)
		done <- struct{}{}
	}()
	copyBuffer(clientConn, backendConn)
	<-done
}

func copyBuffer(dst net.Conn, src net.Conn) {
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}
