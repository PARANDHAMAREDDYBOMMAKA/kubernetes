package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/parandhamareddybommaka/kube/pkg/api/types"
	"github.com/parandhamareddybommaka/kube/pkg/runtime"
	"github.com/parandhamareddybommaka/kube/pkg/store"
)

type Agent struct {
	nodeName    string
	store       store.Store
	runtime     runtime.ContainerRuntime
	pods        map[string]*types.Pod
	podsMu      sync.RWMutex
	stopCh      chan struct{}
	wg          sync.WaitGroup
	nodeIP      string
	podCIDR     string
	ipAllocator *IPAllocator
}

type AgentConfig struct {
	NodeName string
	Store    store.Store
	Runtime  runtime.ContainerRuntime
	PodCIDR  string
}

func New(cfg AgentConfig) (*Agent, error) {
	nodeIP, err := getNodeIP()
	if err != nil {
		nodeIP = "127.0.0.1"
	}

	ipAlloc, err := NewIPAllocator(cfg.PodCIDR)
	if err != nil {
		return nil, err
	}

	return &Agent{
		nodeName:    cfg.NodeName,
		store:       cfg.Store,
		runtime:     cfg.Runtime,
		pods:        make(map[string]*types.Pod),
		stopCh:      make(chan struct{}),
		nodeIP:      nodeIP,
		podCIDR:     cfg.PodCIDR,
		ipAllocator: ipAlloc,
	}, nil
}

func (a *Agent) Run(ctx context.Context) error {
	if err := a.registerNode(ctx); err != nil {
		return err
	}

	a.wg.Add(3)
	go a.syncLoop(ctx)
	go a.statusLoop(ctx)
	go a.heartbeatLoop(ctx)

	return nil
}

func (a *Agent) Stop() {
	close(a.stopCh)
	a.wg.Wait()
}

func (a *Agent) registerNode(ctx context.Context) error {
	node := &types.Node{
		ObjectMeta: types.ObjectMeta{
			Name:    a.nodeName,
			UID:     fmt.Sprintf("%016x", time.Now().UnixNano()),
			Created: time.Now(),
			Labels:  make(map[string]string),
		},
		Spec: types.NodeSpec{
			PodCIDR: a.podCIDR,
		},
		Status: types.NodeStatus{
			Conditions: []types.NodeCondition{
				{
					Type:    types.NodeReady,
					Status:  true,
					Updated: time.Now(),
					Message: "node is ready",
				},
			},
			Capacity:    a.getCapacity(),
			Allocatable: a.getAllocatable(),
			Addresses: []types.NodeAddress{
				{Type: "InternalIP", Address: a.nodeIP},
				{Type: "Hostname", Address: a.nodeName},
			},
		},
	}

	data, _ := json.Marshal(node)
	key := "/nodes/" + a.nodeName

	err := a.store.Create(ctx, key, data)
	if err == store.ErrKeyExists {
		return a.store.Put(ctx, key, data)
	}
	return err
}

func (a *Agent) getCapacity() types.NodeResources {
	return types.NodeResources{
		CPUCores:  4,
		MemoryMB:  8192,
		StorageMB: 102400,
		PodCount:  110,
	}
}

func (a *Agent) getAllocatable() types.NodeResources {
	cap := a.getCapacity()
	return types.NodeResources{
		CPUCores:  cap.CPUCores - 0.5,
		MemoryMB:  cap.MemoryMB - 512,
		StorageMB: cap.StorageMB - 10240,
		PodCount:  cap.PodCount - 10,
	}
}

func (a *Agent) syncLoop(ctx context.Context) {
	defer a.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	a.syncPods(ctx)

	watchCh := a.store.Watch(ctx, "/pods/")

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.syncPods(ctx)
		case event, ok := <-watchCh:
			if !ok {
				return
			}
			a.handlePodEvent(ctx, event)
		}
	}
}

func (a *Agent) syncPods(ctx context.Context) {
	data, err := a.store.List(ctx, "/pods/")
	if err != nil {
		return
	}

	desired := make(map[string]*types.Pod)
	for _, d := range data {
		var pod types.Pod
		if err := json.Unmarshal(d, &pod); err != nil {
			continue
		}
		if pod.Spec.NodeName == a.nodeName {
			desired[pod.UID] = &pod
		}
	}

	a.podsMu.Lock()
	for uid, pod := range desired {
		if _, ok := a.pods[uid]; !ok {
			a.pods[uid] = pod
			go a.runPod(ctx, pod)
		}
	}

	for uid, pod := range a.pods {
		if _, ok := desired[uid]; !ok {
			go a.stopPod(ctx, pod)
			delete(a.pods, uid)
		}
	}
	a.podsMu.Unlock()
}

func (a *Agent) handlePodEvent(ctx context.Context, event store.WatchEvent) {
	if event.Type == store.EventDelete {
		return
	}

	var pod types.Pod
	if err := json.Unmarshal(event.Value, &pod); err != nil {
		return
	}

	if pod.Spec.NodeName != a.nodeName {
		return
	}

	a.podsMu.Lock()
	if _, ok := a.pods[pod.UID]; !ok {
		a.pods[pod.UID] = &pod
		go a.runPod(ctx, &pod)
	}
	a.podsMu.Unlock()
}

func (a *Agent) runPod(ctx context.Context, pod *types.Pod) {
	podIP, err := a.ipAllocator.Allocate()
	if err != nil {
		a.updatePodStatus(ctx, pod, types.PodFailed, err.Error())
		return
	}

	pod.Status.PodIP = podIP
	pod.Status.HostIP = a.nodeIP
	pod.Status.StartTime = time.Now()

	for _, initContainer := range pod.Spec.InitContainers {
		if err := a.runContainer(ctx, pod, &initContainer, true); err != nil {
			a.updatePodStatus(ctx, pod, types.PodFailed, err.Error())
			return
		}
	}

	var containerStatuses []types.ContainerStatus
	for _, container := range pod.Spec.Containers {
		if err := a.runContainer(ctx, pod, &container, false); err != nil {
			a.updatePodStatus(ctx, pod, types.PodFailed, err.Error())
			return
		}

		containerStatuses = append(containerStatuses, types.ContainerStatus{
			Name:        container.Name,
			ContainerID: fmt.Sprintf("%s-%s", pod.Name, container.Name),
			State:       types.ContainerRunning,
			Ready:       true,
			StartedAt:   time.Now(),
		})
	}

	pod.Status.ContainerStatuses = containerStatuses
	a.updatePodStatus(ctx, pod, types.PodRunning, "")
}

func (a *Agent) runContainer(ctx context.Context, pod *types.Pod, container *types.Container, wait bool) error {
	cfg := runtime.ContainerConfig{
		Name:       fmt.Sprintf("%s-%s", pod.Name, container.Name),
		Image:      container.Image,
		Command:    container.Command,
		Args:       container.Args,
		WorkingDir: container.WorkingDir,
		Labels: map[string]string{
			"pod":       pod.Name,
			"namespace": pod.Namespace,
		},
	}

	for _, env := range container.Env {
		cfg.Env = append(cfg.Env, fmt.Sprintf("%s=%s", env.Name, env.Value))
	}

	for _, vm := range container.VolumeMounts {
		for _, v := range pod.Spec.Volumes {
			if v.Name == vm.Name {
				cfg.Mounts = append(cfg.Mounts, runtime.Mount{
					Source:      v.Source.HostPath,
					Destination: vm.MountPath,
					ReadOnly:    vm.ReadOnly,
				})
			}
		}
	}

	if container.Resources.MemoryMB > 0 {
		cfg.MemoryMB = container.Resources.MemoryMB
	}

	id, err := a.runtime.CreateContainer(ctx, cfg)
	if err != nil {
		return err
	}

	if err := a.runtime.StartContainer(ctx, id); err != nil {
		return err
	}

	if wait {
		for {
			status, err := a.runtime.GetContainerStatus(ctx, id)
			if err != nil {
				return err
			}
			if status.State == "stopped" || status.State == "exited" {
				if status.ExitCode != 0 {
					return fmt.Errorf("container exited with code %d", status.ExitCode)
				}
				break
			}
			time.Sleep(time.Second)
		}
	}

	return nil
}

func (a *Agent) stopPod(ctx context.Context, pod *types.Pod) {
	for _, container := range pod.Spec.Containers {
		containerName := fmt.Sprintf("%s-%s", pod.Name, container.Name)
		a.runtime.StopContainer(ctx, containerName, 30*time.Second)
		a.runtime.RemoveContainer(ctx, containerName)
	}

	if pod.Status.PodIP != "" {
		a.ipAllocator.Release(pod.Status.PodIP)
	}
}

func (a *Agent) updatePodStatus(ctx context.Context, pod *types.Pod, phase types.PodPhase, message string) {
	pod.Status.Phase = phase
	pod.Status.Message = message
	pod.Updated = time.Now()

	data, _ := json.Marshal(pod)
	key := fmt.Sprintf("/pods/%s/%s", pod.Namespace, pod.Name)
	a.store.Put(ctx, key, data)
}

func (a *Agent) statusLoop(ctx context.Context) {
	defer a.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.updateContainerStatuses(ctx)
		}
	}
}

func (a *Agent) updateContainerStatuses(ctx context.Context) {
	a.podsMu.RLock()
	pods := make([]*types.Pod, 0, len(a.pods))
	for _, pod := range a.pods {
		pods = append(pods, pod)
	}
	a.podsMu.RUnlock()

	for _, pod := range pods {
		if pod.Status.Phase != types.PodRunning {
			continue
		}

		var allRunning = true
		for i, container := range pod.Spec.Containers {
			containerName := fmt.Sprintf("%s-%s", pod.Name, container.Name)
			status, err := a.runtime.GetContainerStatus(ctx, containerName)
			if err != nil || status.State != "running" {
				allRunning = false
				if i < len(pod.Status.ContainerStatuses) {
					pod.Status.ContainerStatuses[i].Ready = false
					pod.Status.ContainerStatuses[i].State = types.ContainerTerminated
				}
			}
		}

		if !allRunning && pod.Spec.RestartPolicy == types.RestartAlways {
			go a.runPod(ctx, pod)
		}
	}
}

func (a *Agent) heartbeatLoop(ctx context.Context) {
	defer a.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.sendHeartbeat(ctx)
		}
	}
}

func (a *Agent) sendHeartbeat(ctx context.Context) {
	data, _, err := a.store.Get(ctx, "/nodes/"+a.nodeName)
	if err != nil {
		return
	}

	var node types.Node
	if err := json.Unmarshal(data, &node); err != nil {
		return
	}

	for i := range node.Status.Conditions {
		if node.Status.Conditions[i].Type == types.NodeReady {
			node.Status.Conditions[i].Updated = time.Now()
			node.Status.Conditions[i].Status = true
			break
		}
	}

	a.podsMu.RLock()
	node.Status.Allocatable.PodCount = 100 - len(a.pods)
	a.podsMu.RUnlock()

	data, _ = json.Marshal(node)
	a.store.Put(ctx, "/nodes/"+a.nodeName, data)
}

type IPAllocator struct {
	network *net.IPNet
	used    map[string]bool
	mu      sync.Mutex
	current net.IP
}

func NewIPAllocator(cidr string) (*IPAllocator, error) {
	if cidr == "" {
		cidr = "10.244.0.0/24"
	}

	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	ip := make(net.IP, len(network.IP))
	copy(ip, network.IP)
	ip[3]++

	return &IPAllocator{
		network: network,
		used:    make(map[string]bool),
		current: ip,
	}, nil
}

func (a *IPAllocator) Allocate() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i := 0; i < 254; i++ {
		ip := a.current.String()
		if !a.used[ip] && a.network.Contains(a.current) {
			a.used[ip] = true
			a.increment()
			return ip, nil
		}
		a.increment()
	}

	return "", fmt.Errorf("no available IPs")
}

func (a *IPAllocator) Release(ip string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.used, ip)
}

func (a *IPAllocator) increment() {
	for i := len(a.current) - 1; i >= 0; i-- {
		a.current[i]++
		if a.current[i] != 0 {
			break
		}
	}
}

func getNodeIP() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}

	addrs, err := net.LookupHost(hostname)
	if err != nil {
		ifaces, err := net.Interfaces()
		if err != nil {
			return "", err
		}

		for _, iface := range ifaces {
			if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
				continue
			}

			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}

			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
					return ipnet.IP.String(), nil
				}
			}
		}
		return "", fmt.Errorf("no suitable IP found")
	}

	for _, addr := range addrs {
		if ip := net.ParseIP(addr); ip != nil && ip.To4() != nil {
			return addr, nil
		}
	}

	return "", fmt.Errorf("no IPv4 address found")
}
