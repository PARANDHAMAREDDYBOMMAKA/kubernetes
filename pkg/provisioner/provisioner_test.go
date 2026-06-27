package provisioner

import (
	"testing"

	"github.com/parandhamareddybommaka/kube/pkg/models"
)

func TestSanitize(t *testing.T) {
	cases := map[string]string{
		"abc":        "abc",
		"a/b":        "a-b",
		"a:b c":      "a-b-c",
		"keep-_09AZ": "keep-_09AZ",
	}
	for in, want := range cases {
		if got := sanitize(in); got != want {
			t.Errorf("sanitize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestImageFor(t *testing.T) {
	cases := []struct {
		name string
		ver  string
		want string
	}{
		{"empty uses default", "", defaultK3sImage},
		{"plain version gets suffix+repo", "v1.30.0", "rancher/k3s:v1.30.0-k3s1"},
		{"already has k3s suffix", "v1.29.3-k3s1", "rancher/k3s:v1.29.3-k3s1"},
		{"full ref with colon passes through", "myrepo/k3s:custom", "myrepo/k3s:custom"},
		{"ref with slash passes through", "ghcr.io/foo", "ghcr.io/foo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := imageFor(&models.Cluster{K8sVersion: tc.ver})
			if got != tc.want {
				t.Errorf("imageFor(%q) = %q, want %q", tc.ver, got, tc.want)
			}
		})
	}
}

func TestNames(t *testing.T) {
	c := &models.Cluster{ID: "cl/1"}
	if got := networkName(c); got != "kaas-cl-1" {
		t.Errorf("networkName = %q", got)
	}
	if got := serverName(c); got != "kaas-cl-1-server" {
		t.Errorf("serverName = %q", got)
	}
	if got := agentName(c, 2); got != "kaas-cl-1-agent-2" {
		t.Errorf("agentName = %q", got)
	}
}

func TestBuildServerCmd(t *testing.T) {
	t.Setenv("KAAS_CLUSTER_PUBLIC_HOST", "")
	cmd := buildServerCmd()
	joined := join(cmd)
	for _, want := range []string{"server", "--disable", "traefik", "--tls-san=127.0.0.1", "--tls-san=localhost"} {
		if !contains(cmd, want) {
			t.Errorf("buildServerCmd missing %q in %v", want, cmd)
		}
	}
	if contains(cmd, "--tls-san=example.com") {
		t.Errorf("unexpected public host san: %s", joined)
	}

	t.Setenv("KAAS_CLUSTER_PUBLIC_HOST", "example.com")
	if !contains(buildServerCmd(), "--tls-san=example.com") {
		t.Error("public host should add a tls-san entry")
	}
}

func TestServerReadyWait(t *testing.T) {
	t.Setenv("KAAS_K3S_READY_TIMEOUT", "")
	if got := serverReadyWait(); got != defaultServerReadyWait {
		t.Errorf("default = %v, want %v", got, defaultServerReadyWait)
	}

	t.Setenv("KAAS_K3S_READY_TIMEOUT", "30")
	if got := serverReadyWait().Seconds(); got != 30 {
		t.Errorf("30s env = %v", got)
	}

	t.Setenv("KAAS_K3S_READY_TIMEOUT", "bogus")
	if got := serverReadyWait(); got != defaultServerReadyWait {
		t.Errorf("invalid value should fall back to default, got %v", got)
	}
}

func TestRewriteKubeconfig(t *testing.T) {
	t.Setenv("KAAS_CLUSTER_PUBLIC_HOST", "")
	raw := "server: https://127.0.0.1:6443\nother: https://server:6443\n"
	out := rewriteKubeconfig(raw, 12345)
	want := "server: https://127.0.0.1:12345\nother: https://127.0.0.1:12345\n"
	if out != want {
		t.Errorf("rewriteKubeconfig =\n%q\nwant\n%q", out, want)
	}
}

func TestRewriteKubeconfigPublicHost(t *testing.T) {
	t.Setenv("KAAS_CLUSTER_PUBLIC_HOST", "kaas.example.com")
	out := rewriteKubeconfig("x: https://localhost:6443\n", 8443)
	if out != "x: https://kaas.example.com:8443\n" {
		t.Errorf("got %q", out)
	}
}

func TestStripDockerStream(t *testing.T) {
	// Docker multiplexed frame: [stream(1)][000][size(4 BE)] + payload.
	payload := []byte("hello")
	frame := []byte{1, 0, 0, 0, 0, 0, 0, byte(len(payload))}
	frame = append(frame, payload...)
	if got := stripDockerStream(frame); got != "hello" {
		t.Errorf("stripDockerStream = %q, want hello", got)
	}
}

func TestStripDockerStreamRaw(t *testing.T) {
	// Input shorter than a header is returned/handled without panicking.
	if got := stripDockerStream([]byte("hi")); got != "" {
		t.Errorf("short input = %q, want empty", got)
	}
}

func TestGenerateTokenUnique(t *testing.T) {
	a, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken: %v", err)
	}
	b, _ := generateToken()
	if a == b {
		t.Error("tokens should be unique")
	}
	if len(a) != 48 { // 24 bytes hex-encoded
		t.Errorf("token length = %d, want 48", len(a))
	}
}

func TestFreePort(t *testing.T) {
	p, err := freePort()
	if err != nil {
		t.Fatalf("freePort: %v", err)
	}
	if p <= 0 || p > 65535 {
		t.Errorf("freePort returned %d", p)
	}
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

func join(xs []string) string {
	out := ""
	for i, x := range xs {
		if i > 0 {
			out += " "
		}
		out += x
	}
	return out
}
