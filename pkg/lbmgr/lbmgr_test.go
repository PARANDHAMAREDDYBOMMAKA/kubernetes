package lbmgr

import (
	"strings"
	"testing"

	"github.com/parandhamareddybommaka/kube/pkg/models"
)

func TestBuildNginxConfigRoundRobin(t *testing.T) {
	lb := &models.LoadBalancer{
		Port:       30080,
		TargetPort: 8080,
		Algorithm:  models.AlgoRoundRobin,
		Backends: []models.Backend{
			{Host: "10.0.0.1", Port: 8080},
			{Host: "10.0.0.2"}, // port omitted -> uses TargetPort
		},
	}
	conf := BuildNginxConfig(lb)

	if strings.Contains(conf, "least_conn;") || strings.Contains(conf, "ip_hash;") {
		t.Error("round_robin config should not contain an algorithm directive")
	}
	if !strings.Contains(conf, "server 10.0.0.1:8080 max_fails=3 fail_timeout=10s;") {
		t.Errorf("missing first backend:\n%s", conf)
	}
	if !strings.Contains(conf, "server 10.0.0.2:8080 max_fails=3 fail_timeout=10s;") {
		t.Errorf("backend without port should inherit TargetPort:\n%s", conf)
	}
	if !strings.Contains(conf, "listen 30080;") {
		t.Errorf("missing listen directive:\n%s", conf)
	}
}

func TestBuildNginxConfigLeastConn(t *testing.T) {
	conf := BuildNginxConfig(&models.LoadBalancer{
		Port: 80, Algorithm: models.AlgoLeastConn,
		Backends: []models.Backend{{Host: "1.1.1.1", Port: 80}},
	})
	if !strings.Contains(conf, "least_conn;") {
		t.Errorf("expected least_conn directive:\n%s", conf)
	}
}

func TestBuildNginxConfigIPHash(t *testing.T) {
	conf := BuildNginxConfig(&models.LoadBalancer{
		Port: 80, Algorithm: models.AlgoIPHash,
		Backends: []models.Backend{{Host: "1.1.1.1", Port: 80}},
	})
	if !strings.Contains(conf, "ip_hash;") {
		t.Errorf("expected ip_hash directive:\n%s", conf)
	}
}

// No backends must still produce a valid upstream (nginx errors on empty upstream).
func TestBuildNginxConfigNoBackends(t *testing.T) {
	conf := BuildNginxConfig(&models.LoadBalancer{Port: 8080})
	if !strings.Contains(conf, "server 127.0.0.1:65535 down;") {
		t.Errorf("empty backend list should emit a placeholder down server:\n%s", conf)
	}
}

// Backends with an empty host are skipped entirely.
func TestBuildNginxConfigSkipsEmptyHost(t *testing.T) {
	conf := BuildNginxConfig(&models.LoadBalancer{
		Port: 80, TargetPort: 80,
		Backends: []models.Backend{{Host: ""}, {Host: "9.9.9.9", Port: 80}},
	})
	if strings.Contains(conf, "server :") {
		t.Errorf("empty host should be skipped, not emitted:\n%s", conf)
	}
	if !strings.Contains(conf, "server 9.9.9.9:80") {
		t.Errorf("valid backend missing:\n%s", conf)
	}
}

func TestSanitize(t *testing.T) {
	cases := map[string]string{
		"abc-123_XYZ":  "abc-123_XYZ",
		"a b c":        "a-b-c",
		"with/slash":   "with-slash",
		"dots.and:col": "dots-and-col",
	}
	for in, want := range cases {
		if got := sanitize(in); got != want {
			t.Errorf("sanitize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestClusterNetworkName(t *testing.T) {
	if got := clusterNetworkName("cl/1"); got != "kaas-cl-1" {
		t.Errorf("clusterNetworkName = %q, want kaas-cl-1", got)
	}
}

func TestContainerName(t *testing.T) {
	if got := containerName(&models.LoadBalancer{ID: "lb-9"}); got != "kaas-lb-lb-9" {
		t.Errorf("containerName = %q, want kaas-lb-lb-9", got)
	}
}
