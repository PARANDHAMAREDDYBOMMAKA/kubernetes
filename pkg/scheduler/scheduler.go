package scheduler

import (
	"container/heap"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/parandhamareddybommaka/kube/pkg/api/types"
	"github.com/parandhamareddybommaka/kube/pkg/store"
)

type Scheduler struct {
	store       store.Store
	queue       *PriorityQueue
	nodes       map[string]*NodeInfo
	nodesMu     sync.RWMutex
	stopCh      chan struct{}
	plugins     []FilterPlugin
	scorePlugin []ScorePlugin
	wg          sync.WaitGroup
}

type NodeInfo struct {
	Node        *types.Node
	Pods        []*types.Pod
	UsedCPU     float64
	UsedMemory  int64
	UsedStorage int64
	PodCount    int
}

type FilterPlugin interface {
	Name() string
	Filter(ctx context.Context, pod *types.Pod, node *NodeInfo) bool
}

type ScorePlugin interface {
	Name() string
	Score(ctx context.Context, pod *types.Pod, node *NodeInfo) int
}

type SchedulerConfig struct {
	Store           store.Store
	ScheduleTimeout time.Duration
}

func New(cfg SchedulerConfig) *Scheduler {
	s := &Scheduler{
		store:  cfg.Store,
		queue:  NewPriorityQueue(),
		nodes:  make(map[string]*NodeInfo),
		stopCh: make(chan struct{}),
	}

	s.plugins = []FilterPlugin{
		&ResourceFilter{},
		&NodeSelectorFilter{},
		&TaintTolerationFilter{},
	}

	s.scorePlugin = []ScorePlugin{
		&LeastRequestedScore{},
		&BalancedAllocationScore{},
	}

	return s
}

func (s *Scheduler) Run(ctx context.Context) error {
	s.wg.Add(2)
	go s.watchNodes(ctx)
	go s.scheduleLoop(ctx)
	return nil
}

func (s *Scheduler) Stop() {
	close(s.stopCh)
	s.wg.Wait()
}

func (s *Scheduler) watchNodes(ctx context.Context) {
	defer s.wg.Done()

	nodes, _ := s.store.List(ctx, "/nodes/")
	for _, data := range nodes {
		var node types.Node
		if err := store.UnmarshalJSON(data, &node); err == nil {
			s.updateNode(&node)
		}
	}

	watchCh := s.store.Watch(ctx, "/nodes/")
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case event, ok := <-watchCh:
			if !ok {
				return
			}
			if event.Type == store.EventDelete {
				s.nodesMu.Lock()
				delete(s.nodes, event.Key)
				s.nodesMu.Unlock()
			} else {
				var node types.Node
				if err := store.UnmarshalJSON(event.Value, &node); err == nil {
					s.updateNode(&node)
				}
			}
		}
	}
}

func (s *Scheduler) updateNode(node *types.Node) {
	s.nodesMu.Lock()
	defer s.nodesMu.Unlock()

	info, ok := s.nodes[node.Name]
	if !ok {
		info = &NodeInfo{Pods: make([]*types.Pod, 0)}
		s.nodes[node.Name] = info
	}
	info.Node = node
}

func (s *Scheduler) scheduleLoop(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		default:
			pod := s.queue.PopPod()
			if pod == nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			nodeName, err := s.schedulePod(ctx, pod)
			if err != nil {
				s.queue.AddPod(pod)
				continue
			}

			pod.Spec.NodeName = nodeName
			pod.Status.Phase = types.PodPending
			data, _ := store.MarshalJSON(pod)
			key := fmt.Sprintf("/pods/%s/%s", pod.Namespace, pod.Name)
			_ = s.store.Put(ctx, key, data)
		}
	}
}

func (s *Scheduler) AddPod(pod *types.Pod) {
	s.queue.Push(pod)
}

func (s *Scheduler) schedulePod(ctx context.Context, pod *types.Pod) (string, error) {
	s.nodesMu.RLock()
	defer s.nodesMu.RUnlock()

	if len(s.nodes) == 0 {
		return "", fmt.Errorf("no nodes available")
	}

	feasibleNodes := s.filterNodes(ctx, pod)
	if len(feasibleNodes) == 0 {
		return "", fmt.Errorf("no feasible nodes")
	}

	scores := s.scoreNodes(ctx, pod, feasibleNodes)

	var bestNode string
	var bestScore int = -1
	for node, score := range scores {
		if score > bestScore {
			bestScore = score
			bestNode = node
		}
	}

	if bestNode == "" {
		return "", fmt.Errorf("no suitable node found")
	}

	return bestNode, nil
}

func (s *Scheduler) filterNodes(ctx context.Context, pod *types.Pod) []*NodeInfo {
	var feasible []*NodeInfo

	for _, info := range s.nodes {
		if info.Node.Spec.Unschedulable {
			continue
		}

		passed := true
		for _, plugin := range s.plugins {
			if !plugin.Filter(ctx, pod, info) {
				passed = false
				break
			}
		}

		if passed {
			feasible = append(feasible, info)
		}
	}

	return feasible
}

func (s *Scheduler) scoreNodes(ctx context.Context, pod *types.Pod, nodes []*NodeInfo) map[string]int {
	scores := make(map[string]int)

	for _, info := range nodes {
		var totalScore int
		for _, plugin := range s.scorePlugin {
			totalScore += plugin.Score(ctx, pod, info)
		}
		scores[info.Node.Name] = totalScore
	}

	return scores
}

type ResourceFilter struct{}

func (f *ResourceFilter) Name() string { return "ResourceFilter" }

func (f *ResourceFilter) Filter(ctx context.Context, pod *types.Pod, node *NodeInfo) bool {
	var reqCPU float64
	var reqMem int64

	for _, c := range pod.Spec.Containers {
		reqCPU += c.Resources.CPUCores
		reqMem += c.Resources.MemoryMB
	}

	availCPU := node.Node.Status.Allocatable.CPUCores - node.UsedCPU
	availMem := node.Node.Status.Allocatable.MemoryMB - node.UsedMemory

	return reqCPU <= availCPU && reqMem <= availMem
}

type NodeSelectorFilter struct{}

func (f *NodeSelectorFilter) Name() string { return "NodeSelectorFilter" }

func (f *NodeSelectorFilter) Filter(ctx context.Context, pod *types.Pod, node *NodeInfo) bool {
	if len(pod.Spec.NodeSelector) == 0 {
		return true
	}

	for k, v := range pod.Spec.NodeSelector {
		if node.Node.Labels[k] != v {
			return false
		}
	}
	return true
}

type TaintTolerationFilter struct{}

func (f *TaintTolerationFilter) Name() string { return "TaintTolerationFilter" }

func (f *TaintTolerationFilter) Filter(ctx context.Context, pod *types.Pod, node *NodeInfo) bool {
	return true
}

type LeastRequestedScore struct{}

func (s *LeastRequestedScore) Name() string { return "LeastRequestedScore" }

func (s *LeastRequestedScore) Score(ctx context.Context, pod *types.Pod, node *NodeInfo) int {
	capacity := node.Node.Status.Allocatable
	cpuScore := int((capacity.CPUCores - node.UsedCPU) / capacity.CPUCores * 100)
	memScore := int(float64(capacity.MemoryMB-node.UsedMemory) / float64(capacity.MemoryMB) * 100)
	return (cpuScore + memScore) / 2
}

type BalancedAllocationScore struct{}

func (s *BalancedAllocationScore) Name() string { return "BalancedAllocationScore" }

func (s *BalancedAllocationScore) Score(ctx context.Context, pod *types.Pod, node *NodeInfo) int {
	capacity := node.Node.Status.Allocatable
	cpuFraction := node.UsedCPU / capacity.CPUCores
	memFraction := float64(node.UsedMemory) / float64(capacity.MemoryMB)

	diff := cpuFraction - memFraction
	if diff < 0 {
		diff = -diff
	}

	return int((1 - diff) * 100)
}

type PriorityQueue struct {
	items []*queueItem
	mu    sync.Mutex
}

type queueItem struct {
	pod      *types.Pod
	priority int
	index    int
}

func NewPriorityQueue() *PriorityQueue {
	pq := &PriorityQueue{
		items: make([]*queueItem, 0),
	}
	heap.Init(pq)
	return pq
}

func (pq *PriorityQueue) Len() int { return len(pq.items) }

func (pq *PriorityQueue) Less(i, j int) bool {
	return pq.items[i].priority > pq.items[j].priority
}

func (pq *PriorityQueue) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
	pq.items[i].index = i
	pq.items[j].index = j
}

func (pq *PriorityQueue) Push(x any) {
	n := len(pq.items)
	item := x.(*queueItem)
	item.index = n
	pq.items = append(pq.items, item)
}

func (pq *PriorityQueue) Pop() any {
	old := pq.items
	n := len(old)
	if n == 0 {
		return nil
	}
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	pq.items = old[0 : n-1]
	return item
}

func (pq *PriorityQueue) AddPod(pod *types.Pod) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	heap.Push(pq, &queueItem{
		pod:      pod,
		priority: getPodPriority(pod),
	})
}

func (pq *PriorityQueue) PopPod() *types.Pod {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if pq.Len() == 0 {
		return nil
	}

	item := heap.Pop(pq).(*queueItem)
	return item.pod
}

func getPodPriority(pod *types.Pod) int {
	if v, ok := pod.Annotations["scheduler.kube/priority"]; ok {
		var p int
		fmt.Sscanf(v, "%d", &p)
		return p
	}
	return 0
}
