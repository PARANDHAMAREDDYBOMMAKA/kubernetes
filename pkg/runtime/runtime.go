package runtime

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

type ContainerRuntime interface {
	CreateContainer(ctx context.Context, cfg ContainerConfig) (string, error)
	StartContainer(ctx context.Context, id string) error
	StopContainer(ctx context.Context, id string, timeout time.Duration) error
	RemoveContainer(ctx context.Context, id string) error
	GetContainerStatus(ctx context.Context, id string) (ContainerStatus, error)
	ListContainers(ctx context.Context) ([]ContainerInfo, error)
	PullImage(ctx context.Context, ref string) error
	ExecInContainer(ctx context.Context, id string, cmd []string, stdin io.Reader, stdout, stderr io.Writer) (int, error)
	GetContainerLogs(ctx context.Context, id string, follow bool) (io.ReadCloser, error)
}

type ContainerConfig struct {
	Name       string
	Image      string
	Command    []string
	Args       []string
	Env        []string
	Mounts     []Mount
	Labels     map[string]string
	CPUShares  int64
	MemoryMB   int64
	WorkingDir string
	Hostname   string
	User       string
}

type Mount struct {
	Source      string
	Destination string
	Type        string
	ReadOnly    bool
}

type ContainerStatus struct {
	ID         string
	State      string
	Pid        int
	ExitCode   int
	StartedAt  time.Time
	FinishedAt time.Time
}

type ContainerInfo struct {
	ID     string
	Name   string
	Image  string
	Status ContainerStatus
	Labels map[string]string
}

type MockRuntime struct {
	mu         sync.RWMutex
	containers map[string]*mockContainer
	images     map[string]bool
}

type mockContainer struct {
	cfg       ContainerConfig
	status    ContainerStatus
	running   bool
	startTime time.Time
}

func NewMockRuntime() *MockRuntime {
	return &MockRuntime{
		containers: make(map[string]*mockContainer),
		images:     make(map[string]bool),
	}
}

func (r *MockRuntime) PullImage(ctx context.Context, ref string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.images[ref] = true
	return nil
}

func (r *MockRuntime) CreateContainer(ctx context.Context, cfg ContainerConfig) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := cfg.Name
	r.containers[id] = &mockContainer{
		cfg: cfg,
		status: ContainerStatus{
			ID:    id,
			State: "created",
		},
	}
	return id, nil
}

func (r *MockRuntime) StartContainer(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	c, ok := r.containers[id]
	if !ok {
		return fmt.Errorf("container not found")
	}

	c.running = true
	c.startTime = time.Now()
	c.status.State = "running"
	c.status.StartedAt = c.startTime
	c.status.Pid = 1000 + len(r.containers)
	return nil
}

func (r *MockRuntime) StopContainer(ctx context.Context, id string, timeout time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	c, ok := r.containers[id]
	if !ok {
		return nil
	}

	c.running = false
	c.status.State = "stopped"
	c.status.FinishedAt = time.Now()
	return nil
}

func (r *MockRuntime) RemoveContainer(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.containers, id)
	return nil
}

func (r *MockRuntime) GetContainerStatus(ctx context.Context, id string) (ContainerStatus, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	c, ok := r.containers[id]
	if !ok {
		return ContainerStatus{}, fmt.Errorf("container not found")
	}
	return c.status, nil
}

func (r *MockRuntime) ListContainers(ctx context.Context) ([]ContainerInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]ContainerInfo, 0, len(r.containers))
	for id, c := range r.containers {
		infos = append(infos, ContainerInfo{
			ID:     id,
			Name:   c.cfg.Name,
			Image:  c.cfg.Image,
			Status: c.status,
			Labels: c.cfg.Labels,
		})
	}
	return infos, nil
}

func (r *MockRuntime) ExecInContainer(ctx context.Context, id string, cmd []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	return 0, nil
}

func (r *MockRuntime) GetContainerLogs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	return nil, nil
}
