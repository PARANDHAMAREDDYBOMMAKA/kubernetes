package runtime

import (
	"context"
	"fmt"
	"io"
	"time"
)

type ContainerdRuntime struct {
	address   string
	namespace string
}

func NewContainerdRuntime(address, namespace string) (*ContainerdRuntime, error) {
	return &ContainerdRuntime{
		address:   address,
		namespace: namespace,
	}, nil
}

func (r *ContainerdRuntime) PullImage(ctx context.Context, ref string) error {
	return fmt.Errorf("containerd runtime not available, use docker or mock runtime")
}

func (r *ContainerdRuntime) CreateContainer(ctx context.Context, cfg ContainerConfig) (string, error) {
	return "", fmt.Errorf("containerd runtime not available, use docker or mock runtime")
}

func (r *ContainerdRuntime) StartContainer(ctx context.Context, id string) error {
	return fmt.Errorf("containerd runtime not available, use docker or mock runtime")
}

func (r *ContainerdRuntime) StopContainer(ctx context.Context, id string, timeout time.Duration) error {
	return fmt.Errorf("containerd runtime not available, use docker or mock runtime")
}

func (r *ContainerdRuntime) RemoveContainer(ctx context.Context, id string) error {
	return fmt.Errorf("containerd runtime not available, use docker or mock runtime")
}

func (r *ContainerdRuntime) GetContainerStatus(ctx context.Context, id string) (ContainerStatus, error) {
	return ContainerStatus{}, fmt.Errorf("containerd runtime not available, use docker or mock runtime")
}

func (r *ContainerdRuntime) ListContainers(ctx context.Context) ([]ContainerInfo, error) {
	return nil, fmt.Errorf("containerd runtime not available, use docker or mock runtime")
}

func (r *ContainerdRuntime) ExecInContainer(ctx context.Context, id string, cmd []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	return -1, fmt.Errorf("containerd runtime not available, use docker or mock runtime")
}

func (r *ContainerdRuntime) GetContainerLogs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	return nil, fmt.Errorf("containerd runtime not available, use docker or mock runtime")
}

func (r *ContainerdRuntime) Close() error {
	return nil
}
