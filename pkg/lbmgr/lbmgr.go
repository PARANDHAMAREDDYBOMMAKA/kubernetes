package lbmgr

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	"github.com/parandhamareddybommaka/kube/pkg/models"
)

const (
	defaultImage = "nginx:alpine"
)

type Manager struct {
	cli *client.Client
}

func New() (*Manager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	return &Manager{cli: cli}, nil
}

func (m *Manager) Close() error {
	if m.cli == nil {
		return nil
	}
	return m.cli.Close()
}

func BuildNginxConfig(lb *models.LoadBalancer) string {
	algo := ""
	switch lb.Algorithm {
	case models.AlgoLeastConn:
		algo = "    least_conn;\n"
	case models.AlgoIPHash:
		algo = "    ip_hash;\n"
	case models.AlgoRoundRobin, "":
	}
	var servers strings.Builder
	for _, b := range lb.Backends {
		host := b.Host
		if host == "" {
			continue
		}
		port := b.Port
		if port == 0 {
			port = lb.TargetPort
		}
		fmt.Fprintf(&servers, "    server %s:%d max_fails=3 fail_timeout=10s;\n", host, port)
	}
	if servers.Len() == 0 {
		servers.WriteString("    server 127.0.0.1:65535 down;\n")
	}
	return fmt.Sprintf(`worker_processes  1;
events { worker_connections  1024; }

http {
  upstream kaas_backend {
%s%s  }
  server {
    listen %d;
    location / {
      proxy_pass http://kaas_backend;
      proxy_set_header Host $host;
      proxy_set_header X-Real-IP $remote_addr;
      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
  }
}
`, algo, servers.String(), lb.Port)
}

func containerName(lb *models.LoadBalancer) string {
	return "kaas-lb-" + lb.ID
}

func sanitize(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return b.String()
}

func clusterNetworkName(clusterID string) string {
	return "kaas-" + sanitize(clusterID)
}

func (m *Manager) Create(ctx context.Context, lb *models.LoadBalancer) error {
	lb.Status = models.LBStatusProvisioning
	lb.UpdatedAt = time.Now().UTC()

	if err := m.ensureImage(ctx); err != nil {
		lb.Status = models.LBStatusFailed
		return fmt.Errorf("pull nginx image: %w", err)
	}

	port := fmt.Sprintf("%d", lb.Port)
	natPort, _ := nat.NewPort("tcp", port)

	cfg := &container.Config{
		Image:        defaultImage,
		ExposedPorts: nat.PortSet{natPort: struct{}{}},
		Labels: map[string]string{
			"kaas.lb":      lb.ID,
			"kaas.cluster": lb.ClusterID,
		},
	}
	hostCfg := &container.HostConfig{
		PortBindings: nat.PortMap{
			natPort: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: port}},
		},
		RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
	}

	var netCfg *network.NetworkingConfig
	if lb.ClusterID != "" {
		netName := clusterNetworkName(lb.ClusterID)
		hostCfg.NetworkMode = container.NetworkMode(netName)
		netCfg = &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				netName: {Aliases: []string{containerName(lb)}},
			},
		}
	}

	resp, err := m.cli.ContainerCreate(ctx, cfg, hostCfg, netCfg, nil, containerName(lb))
	if err != nil {
		lb.Status = models.LBStatusFailed
		return fmt.Errorf("container create: %w", err)
	}
	lb.ContainerID = resp.ID

	if err := m.copyConfig(ctx, resp.ID, BuildNginxConfig(lb)); err != nil {
		_ = m.cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		lb.Status = models.LBStatusFailed
		return fmt.Errorf("copy config: %w", err)
	}

	if err := m.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		_ = m.cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		lb.Status = models.LBStatusFailed
		return fmt.Errorf("container start: %w", err)
	}

	lb.Status = models.LBStatusRunning
	lb.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *Manager) Delete(ctx context.Context, lb *models.LoadBalancer) error {
	lb.Status = models.LBStatusDeleting
	if lb.ContainerID != "" {
		timeout := 5
		_ = m.cli.ContainerStop(ctx, lb.ContainerID, container.StopOptions{Timeout: &timeout})
		if err := m.cli.ContainerRemove(ctx, lb.ContainerID, container.RemoveOptions{Force: true}); err != nil {
			if !strings.Contains(err.Error(), "No such container") && !strings.Contains(err.Error(), "not found") {
				return err
			}
		}
	}
	lb.Status = models.LBStatusDeleted
	lb.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *Manager) ensureImage(ctx context.Context) error {
	rc, err := m.cli.ImagePull(ctx, defaultImage, image.PullOptions{})
	if err != nil {
		return err
	}
	defer rc.Close()
	_, _ = io.Copy(io.Discard, rc)
	return nil
}

func (m *Manager) copyConfig(ctx context.Context, containerID, conf string) error {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	hdr := &tar.Header{
		Name: "nginx.conf",
		Mode: 0o644,
		Size: int64(len(conf)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := tw.Write([]byte(conf)); err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return m.cli.CopyToContainer(ctx, containerID, "/etc/nginx/", &buf, container.CopyToContainerOptions{})
}
