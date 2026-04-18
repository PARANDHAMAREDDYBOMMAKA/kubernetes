// Package provisioner runs and manages k3s-in-Docker clusters for KaaS.
package provisioner

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	"github.com/parandhamareddybommaka/kube/pkg/models"
)

const (
	defaultK3sImage        = "rancher/k3s:v1.29.3-k3s1"
	pollInterval           = 2 * time.Second
	defaultServerReadyWait = 5 * time.Minute
)

func buildServerCmd() []string {
	cmd := []string{
		"server",
		"--disable", "traefik",
		"--tls-san=host.docker.internal",
		"--tls-san=127.0.0.1",
		"--tls-san=localhost",
	}
	if host := os.Getenv("KAAS_CLUSTER_PUBLIC_HOST"); host != "" {
		cmd = append(cmd, "--tls-san="+host)
	}
	return cmd
}

func serverReadyWait() time.Duration {
	if raw := os.Getenv("KAAS_K3S_READY_TIMEOUT"); raw != "" {
		if secs, err := strconv.Atoi(raw); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return defaultServerReadyWait
}

type StatusUpdate struct {
	Status  string
	Message string
}

type StatusCallback func(ctx context.Context, c *models.Cluster, update StatusUpdate) error

type Provisioner interface {
	CreateCluster(ctx context.Context, c *models.Cluster) error
	DeleteCluster(ctx context.Context, c *models.Cluster) error
	ScaleCluster(ctx context.Context, c *models.Cluster, newCount int) error
	Nodes(ctx context.Context, c *models.Cluster) ([]NodeInfo, error)
}

type NodeRole string

const (
	NodeRoleServer NodeRole = "server"
	NodeRoleAgent  NodeRole = "agent"
)

type NodeInfo struct {
	Name        string    `json:"name"`
	Role        NodeRole  `json:"role"`
	ContainerID string    `json:"containerId"`
	State       string    `json:"state"`
	Status      string    `json:"status"`
	ExitCode    int       `json:"exitCode,omitempty"`
	Ready       bool      `json:"ready"`
	StartedAt   time.Time `json:"startedAt"`
	FinishedAt  time.Time `json:"finishedAt"`
	Message     string    `json:"message,omitempty"`
}

type DockerK3sProvisioner struct {
	cli    *client.Client
	onStep StatusCallback
}

func NewDockerK3sProvisioner(onStep StatusCallback) (*DockerK3sProvisioner, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	return &DockerK3sProvisioner{cli: cli, onStep: onStep}, nil
}

func (p *DockerK3sProvisioner) Close() error {
	if p.cli == nil {
		return nil
	}
	return p.cli.Close()
}

func (p *DockerK3sProvisioner) emit(ctx context.Context, c *models.Cluster, status, msg string) {
	c.Status = status
	c.Message = msg
	c.UpdatedAt = time.Now().UTC()
	if p.onStep != nil {
		if err := p.onStep(ctx, c, StatusUpdate{Status: status, Message: msg}); err != nil {
			slog.Warn("status callback failed", "err", err, "cluster", c.ID)
		}
	}
}

func imageFor(c *models.Cluster) string {
	if c.K8sVersion == "" {
		return defaultK3sImage
	}
	v := c.K8sVersion
	if strings.Contains(v, "/") || strings.Contains(v, ":") {
		return v
	}
	if !strings.Contains(v, "-k3s") {
		v = v + "-k3s1"
	}
	return "rancher/k3s:" + v
}

func networkName(c *models.Cluster) string {
	return "kaas-" + sanitize(c.ID)
}

func serverName(c *models.Cluster) string {
	return "kaas-" + sanitize(c.ID) + "-server"
}

func agentName(c *models.Cluster, i int) string {
	return fmt.Sprintf("kaas-%s-agent-%d", sanitize(c.ID), i)
}

func sanitize(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return b.String()
}

func generateToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func (p *DockerK3sProvisioner) CreateCluster(ctx context.Context, c *models.Cluster) error {
	p.emit(ctx, c, models.ClusterStatusProvisioning, "preparing network")

	img := imageFor(c)
	if err := p.ensureImage(ctx, img); err != nil {
		p.emit(ctx, c, models.ClusterStatusFailed, "pull image: "+err.Error())
		return err
	}

	netName := networkName(c)
	if _, err := p.cli.NetworkCreate(ctx, netName, network.CreateOptions{
		Driver: "bridge",
		Labels: map[string]string{"kaas.cluster": c.ID},
	}); err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			p.emit(ctx, c, models.ClusterStatusFailed, "create network: "+err.Error())
			return err
		}
	}

	if c.ClusterToken == "" {
		tok, err := generateToken()
		if err != nil {
			p.emit(ctx, c, models.ClusterStatusFailed, "generate token: "+err.Error())
			return err
		}
		c.ClusterToken = tok
	}
	if c.APIPort == 0 {
		port, err := freePort()
		if err != nil {
			p.emit(ctx, c, models.ClusterStatusFailed, "allocate port: "+err.Error())
			return err
		}
		c.APIPort = port
	}

	p.emit(ctx, c, models.ClusterStatusProvisioning, "starting k3s server")

	serverID, err := p.startServer(ctx, c, img, netName)
	if err != nil {
		p.emit(ctx, c, models.ClusterStatusFailed, "start server: "+err.Error())
		return err
	}
	c.ServerContainerID = serverID

	p.emit(ctx, c, models.ClusterStatusProvisioning, "waiting for kubeconfig")
	kc, err := p.waitForKubeconfig(ctx, serverID)
	if err != nil {
		p.emit(ctx, c, models.ClusterStatusFailed, "waiting for kubeconfig: "+err.Error())
		return err
	}
	c.Kubeconfig = rewriteKubeconfig(kc, c.APIPort)

	started, agentErr := p.ensureAgents(ctx, c, img, netName, c.NodeCount)
	if agentErr != nil && started == 0 && c.NodeCount > 0 {
		p.emit(ctx, c, models.ClusterStatusFailed, "start agents: "+agentErr.Error())
		return agentErr
	}
	if agentErr != nil {
		p.emit(ctx, c, models.ClusterStatusRunning,
			fmt.Sprintf("degraded: %d/%d agents ready (%s)", started, c.NodeCount, agentErr.Error()))
		return nil
	}
	p.emit(ctx, c, models.ClusterStatusRunning, "cluster ready")
	return nil
}

func (p *DockerK3sProvisioner) startServer(ctx context.Context, c *models.Cluster, img, netName string) (string, error) {
	apiPortStr := fmt.Sprintf("%d", c.APIPort)
	exposedPort, _ := nat.NewPort("tcp", "6443")

	hostConfig := &container.HostConfig{
		Privileged:  true,
		NetworkMode: container.NetworkMode(netName),
		Tmpfs: map[string]string{
			"/run":     "",
			"/var/run": "",
		},
		Mounts: []mount.Mount{{
			Type:   mount.TypeVolume,
			Source: "kaas-" + sanitize(c.ID) + "-server-data",
			Target: "/var/lib/rancher/k3s",
		}},
		PortBindings: nat.PortMap{
			exposedPort: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: apiPortStr}},
		},
		RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
	}

	cfg := &container.Config{
		Image: img,
		Env: []string{
			"K3S_TOKEN=" + c.ClusterToken,
			"K3S_KUBECONFIG_OUTPUT=/output/kubeconfig.yaml",
			"K3S_KUBECONFIG_MODE=666",
		},
		Cmd: buildServerCmd(),
		ExposedPorts: nat.PortSet{exposedPort: struct{}{}},
		Labels: map[string]string{
			"kaas.cluster": c.ID,
			"kaas.role":    "server",
		},
	}

	netCfg := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			netName: {Aliases: []string{"server", serverName(c)}},
		},
	}

	resp, err := p.cli.ContainerCreate(ctx, cfg, hostConfig, netCfg, nil, serverName(c))
	if err != nil {
		return "", err
	}
	if err := p.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", err
	}
	return resp.ID, nil
}

func (p *DockerK3sProvisioner) ensureAgents(ctx context.Context, c *models.Cluster, img, netName string, desired int) (int, error) {
	for len(c.AgentContainerIDs) > desired {
		last := c.AgentContainerIDs[len(c.AgentContainerIDs)-1]
		if err := p.removeContainer(ctx, last); err != nil {
			slog.Warn("remove agent", "err", err, "id", last)
		}
		c.AgentContainerIDs = c.AgentContainerIDs[:len(c.AgentContainerIDs)-1]
	}
	var failures []string
	for i := len(c.AgentContainerIDs); i < desired; i++ {
		id, err := p.startAgent(ctx, c, img, netName, i)
		if err != nil {
			failures = append(failures, fmt.Sprintf("agent-%d: %s", i, err.Error()))
			continue
		}
		c.AgentContainerIDs = append(c.AgentContainerIDs, id)
	}
	started := len(c.AgentContainerIDs)
	if len(failures) > 0 {
		return started, errors.New(strings.Join(failures, "; "))
	}
	return started, nil
}

func (p *DockerK3sProvisioner) startAgent(ctx context.Context, c *models.Cluster, img, netName string, idx int) (string, error) {
	cfg := &container.Config{
		Image: img,
		Env: []string{
			"K3S_URL=https://" + serverName(c) + ":6443",
			"K3S_TOKEN=" + c.ClusterToken,
		},
		Cmd: []string{"agent"},
		Labels: map[string]string{
			"kaas.cluster": c.ID,
			"kaas.role":    "agent",
		},
	}
	hostCfg := &container.HostConfig{
		Privileged:  true,
		NetworkMode: container.NetworkMode(netName),
		Tmpfs: map[string]string{
			"/run":     "",
			"/var/run": "",
		},
		RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
	}
	netCfg := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{netName: {}},
	}
	resp, err := p.cli.ContainerCreate(ctx, cfg, hostCfg, netCfg, nil, agentName(c, idx))
	if err != nil {
		return "", err
	}
	if err := p.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", err
	}
	return resp.ID, nil
}

func (p *DockerK3sProvisioner) waitForKubeconfig(ctx context.Context, serverID string) (string, error) {
	deadline := time.Now().Add(serverReadyWait())
	var lastErr error
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		out, err := p.execCapture(ctx, serverID, []string{"cat", "/etc/rancher/k3s/k3s.yaml"})
		if err == nil && strings.Contains(out, "apiVersion") {
			return out, nil
		}
		lastErr = err
		time.Sleep(pollInterval)
	}
	if lastErr == nil {
		lastErr = errors.New("timeout waiting for kubeconfig")
	}
	return "", lastErr
}

func (p *DockerK3sProvisioner) execCapture(ctx context.Context, containerID string, cmd []string) (string, error) {
	exec, err := p.cli.ContainerExecCreate(ctx, containerID, container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return "", err
	}
	hijack, err := p.cli.ContainerExecAttach(ctx, exec.ID, container.ExecAttachOptions{})
	if err != nil {
		return "", err
	}
	defer hijack.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, hijack.Reader); err != nil {
		return "", err
	}
	return stripDockerStream(buf.Bytes()), nil
}

func stripDockerStream(in []byte) string {
	var out bytes.Buffer
	for len(in) >= 8 {
		if in[0] > 2 {
			out.Write(in)
			break
		}
		size := int(in[4])<<24 | int(in[5])<<16 | int(in[6])<<8 | int(in[7])
		if size < 0 || 8+size > len(in) {
			out.Write(in)
			break
		}
		out.Write(in[8 : 8+size])
		in = in[8+size:]
	}
	return out.String()
}

func rewriteKubeconfig(raw string, port int) string {
	replacements := []string{
		"https://127.0.0.1:6443",
		"https://0.0.0.0:6443",
		"https://localhost:6443",
		"https://server:6443",
	}
	host := os.Getenv("KAAS_CLUSTER_PUBLIC_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	target := fmt.Sprintf("https://%s:%d", host, port)
	out := raw
	for _, r := range replacements {
		out = strings.ReplaceAll(out, r, target)
	}
	return out
}

func (p *DockerK3sProvisioner) ensureImage(ctx context.Context, ref string) error {
	rc, err := p.cli.ImagePull(ctx, ref, image.PullOptions{})
	if err != nil {
		return err
	}
	defer rc.Close()
	_, _ = io.Copy(io.Discard, rc)
	return nil
}

func (p *DockerK3sProvisioner) ScaleCluster(ctx context.Context, c *models.Cluster, newCount int) error {
	if newCount < 0 {
		return errors.New("node count must be >= 0")
	}
	p.emit(ctx, c, models.ClusterStatusProvisioning, fmt.Sprintf("scaling to %d agents", newCount))
	img := imageFor(c)
	netName := networkName(c)
	started, err := p.ensureAgents(ctx, c, img, netName, newCount)
	if err != nil && started == 0 && newCount > 0 {
		p.emit(ctx, c, models.ClusterStatusFailed, "scale: "+err.Error())
		return err
	}
	c.NodeCount = newCount
	if err != nil {
		p.emit(ctx, c, models.ClusterStatusRunning,
			fmt.Sprintf("degraded: %d/%d agents ready (%s)", started, newCount, err.Error()))
		return nil
	}
	p.emit(ctx, c, models.ClusterStatusRunning, fmt.Sprintf("scaled to %d agents", newCount))
	return nil
}

func (p *DockerK3sProvisioner) DeleteCluster(ctx context.Context, c *models.Cluster) error {
	p.emit(ctx, c, models.ClusterStatusDeleting, "removing containers")
	for _, id := range c.AgentContainerIDs {
		if err := p.removeContainer(ctx, id); err != nil {
			slog.Warn("remove agent", "err", err, "id", id)
		}
	}
	c.AgentContainerIDs = nil

	if c.ServerContainerID != "" {
		if err := p.removeContainer(ctx, c.ServerContainerID); err != nil {
			slog.Warn("remove server", "err", err, "id", c.ServerContainerID)
		}
		c.ServerContainerID = ""
	}

	if err := p.cli.NetworkRemove(ctx, networkName(c)); err != nil {
		if !strings.Contains(err.Error(), "No such network") && !strings.Contains(err.Error(), "not found") {
			slog.Warn("network remove", "err", err)
		}
	}

	p.emit(ctx, c, models.ClusterStatusDeleted, "cluster deleted")
	return nil
}

func (p *DockerK3sProvisioner) removeContainer(ctx context.Context, id string) error {
	timeout := 10
	_ = p.cli.ContainerStop(ctx, id, container.StopOptions{Timeout: &timeout})
	return p.cli.ContainerRemove(ctx, id, container.RemoveOptions{Force: true, RemoveVolumes: true})
}

func (p *DockerK3sProvisioner) Nodes(ctx context.Context, c *models.Cluster) ([]NodeInfo, error) {
	var out []NodeInfo
	if c.ServerContainerID != "" {
		out = append(out, p.inspectNode(ctx, serverName(c), NodeRoleServer, c.ServerContainerID))
	}
	for i, id := range c.AgentContainerIDs {
		out = append(out, p.inspectNode(ctx, agentName(c, i), NodeRoleAgent, id))
	}
	return out, nil
}

func (p *DockerK3sProvisioner) inspectNode(ctx context.Context, name string, role NodeRole, id string) NodeInfo {
	info := NodeInfo{Name: name, Role: role, ContainerID: id, State: "unknown", Status: "unknown"}
	if id == "" {
		info.State = "missing"
		info.Status = "failed"
		info.Message = "no container id"
		return info
	}
	insp, err := p.cli.ContainerInspect(ctx, id)
	if err != nil {
		info.State = "missing"
		info.Status = "failed"
		info.Message = err.Error()
		return info
	}
	if insp.State != nil {
		info.State = insp.State.Status
		info.Status = insp.State.Status
		info.ExitCode = insp.State.ExitCode
		if t, err := time.Parse(time.RFC3339Nano, insp.State.StartedAt); err == nil {
			info.StartedAt = t
		}
		if t, err := time.Parse(time.RFC3339Nano, insp.State.FinishedAt); err == nil && !t.IsZero() {
			info.FinishedAt = t
		}
		info.Ready = insp.State.Running && !insp.State.Restarting && !insp.State.OOMKilled && !insp.State.Dead
		if insp.State.Error != "" {
			info.Message = insp.State.Error
		}
	}
	if !info.Ready && info.Message == "" {
		if tail := p.tailLogs(ctx, id); tail != "" {
			info.Message = tail
		}
	}
	return info
}

func (p *DockerK3sProvisioner) tailLogs(ctx context.Context, id string) string {
	rc, err := p.cli.ContainerLogs(ctx, id, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       "40",
	})
	if err != nil {
		return ""
	}
	defer rc.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, rc)
	out := stripDockerStream(buf.Bytes())
	if len(out) > 800 {
		out = out[len(out)-800:]
	}
	return strings.TrimSpace(out)
}
