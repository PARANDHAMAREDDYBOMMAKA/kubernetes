package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/parandhamareddybommaka/kube/pkg/api/types"
	"github.com/parandhamareddybommaka/kube/pkg/store"
)

type Manager struct {
	store       store.Store
	controllers []Controller
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

type Controller interface {
	Name() string
	Run(ctx context.Context) error
	Stop()
}

func NewManager(s store.Store) *Manager {
	m := &Manager{
		store:  s,
		stopCh: make(chan struct{}),
	}

	m.controllers = []Controller{
		NewDeploymentController(s),
		NewReplicaController(s),
		NewServiceController(s),
		NewNodeController(s),
	}

	return m
}

func (m *Manager) Run(ctx context.Context) error {
	for _, c := range m.controllers {
		m.wg.Add(1)
		go func(ctrl Controller) {
			defer m.wg.Done()
			ctrl.Run(ctx)
		}(c)
	}
	return nil
}

func (m *Manager) Stop() {
	close(m.stopCh)
	for _, c := range m.controllers {
		c.Stop()
	}
	m.wg.Wait()
}

type DeploymentController struct {
	store  store.Store
	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewDeploymentController(s store.Store) *DeploymentController {
	return &DeploymentController{
		store:  s,
		stopCh: make(chan struct{}),
	}
}

func (c *DeploymentController) Name() string { return "deployment" }

func (c *DeploymentController) Run(ctx context.Context) error {
	c.wg.Add(1)
	go c.syncLoop(ctx)
	return nil
}

func (c *DeploymentController) Stop() {
	close(c.stopCh)
	c.wg.Wait()
}

func (c *DeploymentController) syncLoop(ctx context.Context) {
	defer c.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.syncDeployments(ctx)
		}
	}
}

func (c *DeploymentController) syncDeployments(ctx context.Context) {
	data, err := c.store.List(ctx, "/deployments/")
	if err != nil {
		return
	}

	for _, d := range data {
		var dep types.Deployment
		if err := json.Unmarshal(d, &dep); err != nil {
			continue
		}
		c.reconcileDeployment(ctx, &dep)
	}
}

func (c *DeploymentController) reconcileDeployment(ctx context.Context, dep *types.Deployment) {
	pods, _ := c.getPodsForDeployment(ctx, dep)
	currentReplicas := len(pods)
	desiredReplicas := dep.Spec.Replicas

	if currentReplicas < desiredReplicas {
		for i := currentReplicas; i < desiredReplicas; i++ {
			c.createPod(ctx, dep, i)
		}
	} else if currentReplicas > desiredReplicas {
		for i := desiredReplicas; i < currentReplicas; i++ {
			c.deletePod(ctx, pods[i])
		}
	}

	c.updateDeploymentStatus(ctx, dep, currentReplicas)
}

func (c *DeploymentController) getPodsForDeployment(ctx context.Context, dep *types.Deployment) ([]*types.Pod, error) {
	data, err := c.store.List(ctx, fmt.Sprintf("/pods/%s/", dep.Namespace))
	if err != nil {
		return nil, err
	}

	var pods []*types.Pod
	for _, d := range data {
		var pod types.Pod
		if err := json.Unmarshal(d, &pod); err != nil {
			continue
		}

		match := true
		for k, v := range dep.Spec.Selector {
			if pod.Labels[k] != v {
				match = false
				break
			}
		}

		if match {
			pods = append(pods, &pod)
		}
	}

	return pods, nil
}

func (c *DeploymentController) createPod(ctx context.Context, dep *types.Deployment, index int) {
	pod := &types.Pod{
		ObjectMeta: types.ObjectMeta{
			Name:      fmt.Sprintf("%s-%d", dep.Name, index),
			Namespace: dep.Namespace,
			Labels:    dep.Spec.Selector,
			Created:   time.Now(),
		},
		Spec: dep.Spec.Template,
		Status: types.PodStatus{
			Phase: types.PodPending,
		},
	}

	pod.UID = fmt.Sprintf("%016x", time.Now().UnixNano())

	data, _ := json.Marshal(pod)
	key := fmt.Sprintf("/pods/%s/%s", pod.Namespace, pod.Name)
	c.store.Create(ctx, key, data)
}

func (c *DeploymentController) deletePod(ctx context.Context, pod *types.Pod) {
	key := fmt.Sprintf("/pods/%s/%s", pod.Namespace, pod.Name)
	c.store.Delete(ctx, key)
}

func (c *DeploymentController) updateDeploymentStatus(ctx context.Context, dep *types.Deployment, replicas int) {
	dep.Status.Replicas = replicas
	dep.Status.ReadyReplicas = replicas
	dep.Status.AvailableReplicas = replicas
	dep.Updated = time.Now()

	data, _ := json.Marshal(dep)
	key := fmt.Sprintf("/deployments/%s/%s", dep.Namespace, dep.Name)
	c.store.Put(ctx, key, data)
}

type ReplicaController struct {
	store  store.Store
	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewReplicaController(s store.Store) *ReplicaController {
	return &ReplicaController{
		store:  s,
		stopCh: make(chan struct{}),
	}
}

func (c *ReplicaController) Name() string { return "replica" }

func (c *ReplicaController) Run(ctx context.Context) error {
	c.wg.Add(1)
	go c.watchPods(ctx)
	return nil
}

func (c *ReplicaController) Stop() {
	close(c.stopCh)
	c.wg.Wait()
}

func (c *ReplicaController) watchPods(ctx context.Context) {
	defer c.wg.Done()

	watchCh := c.store.Watch(ctx, "/pods/")
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case event, ok := <-watchCh:
			if !ok {
				return
			}
			if event.Type == store.EventDelete {
				c.handlePodDeletion(ctx, event.Key)
			}
		}
	}
}

func (c *ReplicaController) handlePodDeletion(ctx context.Context, key string) {
}

type ServiceController struct {
	store     store.Store
	endpoints map[string][]string
	mu        sync.RWMutex
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

func NewServiceController(s store.Store) *ServiceController {
	return &ServiceController{
		store:     s,
		endpoints: make(map[string][]string),
		stopCh:    make(chan struct{}),
	}
}

func (c *ServiceController) Name() string { return "service" }

func (c *ServiceController) Run(ctx context.Context) error {
	c.wg.Add(1)
	go c.syncLoop(ctx)
	return nil
}

func (c *ServiceController) Stop() {
	close(c.stopCh)
	c.wg.Wait()
}

func (c *ServiceController) syncLoop(ctx context.Context) {
	defer c.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.syncServices(ctx)
		}
	}
}

func (c *ServiceController) syncServices(ctx context.Context) {
	svcData, err := c.store.List(ctx, "/services/")
	if err != nil {
		return
	}

	for _, d := range svcData {
		var svc types.Service
		if err := json.Unmarshal(d, &svc); err != nil {
			continue
		}
		c.updateEndpoints(ctx, &svc)
	}
}

func (c *ServiceController) updateEndpoints(ctx context.Context, svc *types.Service) {
	podData, err := c.store.List(ctx, fmt.Sprintf("/pods/%s/", svc.Namespace))
	if err != nil {
		return
	}

	var endpoints []string
	for _, d := range podData {
		var pod types.Pod
		if err := json.Unmarshal(d, &pod); err != nil {
			continue
		}

		if pod.Status.Phase != types.PodRunning {
			continue
		}

		match := true
		for k, v := range svc.Spec.Selector {
			if pod.Labels[k] != v {
				match = false
				break
			}
		}

		if match && pod.Status.PodIP != "" {
			endpoints = append(endpoints, pod.Status.PodIP)
		}
	}

	c.mu.Lock()
	c.endpoints[fmt.Sprintf("%s/%s", svc.Namespace, svc.Name)] = endpoints
	c.mu.Unlock()
}

func (c *ServiceController) GetEndpoints(namespace, name string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.endpoints[fmt.Sprintf("%s/%s", namespace, name)]
}

type NodeController struct {
	store  store.Store
	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewNodeController(s store.Store) *NodeController {
	return &NodeController{
		store:  s,
		stopCh: make(chan struct{}),
	}
}

func (c *NodeController) Name() string { return "node" }

func (c *NodeController) Run(ctx context.Context) error {
	c.wg.Add(1)
	go c.healthCheckLoop(ctx)
	return nil
}

func (c *NodeController) Stop() {
	close(c.stopCh)
	c.wg.Wait()
}

func (c *NodeController) healthCheckLoop(ctx context.Context) {
	defer c.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.checkNodeHealth(ctx)
		}
	}
}

func (c *NodeController) checkNodeHealth(ctx context.Context) {
	data, err := c.store.List(ctx, "/nodes/")
	if err != nil {
		return
	}

	now := time.Now()
	for _, d := range data {
		var node types.Node
		if err := json.Unmarshal(d, &node); err != nil {
			continue
		}

		for i := range node.Status.Conditions {
			if node.Status.Conditions[i].Type == types.NodeReady {
				if now.Sub(node.Status.Conditions[i].Updated) > 5*time.Minute {
					node.Status.Conditions[i].Status = false
					node.Status.Conditions[i].Message = "node not responding"

					data, _ := json.Marshal(node)
					c.store.Put(ctx, "/nodes/"+node.Name, data)
				}
				break
			}
		}
	}
}

type WorkQueue struct {
	items []any
	mu    sync.Mutex
	cond  *sync.Cond
}

func NewWorkQueue() *WorkQueue {
	wq := &WorkQueue{
		items: make([]any, 0),
	}
	wq.cond = sync.NewCond(&wq.mu)
	return wq
}

func (q *WorkQueue) Add(item any) {
	q.mu.Lock()
	q.items = append(q.items, item)
	q.mu.Unlock()
	q.cond.Signal()
}

func (q *WorkQueue) Get() any {
	q.mu.Lock()
	defer q.mu.Unlock()

	for len(q.items) == 0 {
		q.cond.Wait()
	}

	item := q.items[0]
	q.items = q.items[1:]
	return item
}

func (q *WorkQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}
