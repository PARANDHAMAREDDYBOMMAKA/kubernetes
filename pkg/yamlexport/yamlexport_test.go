package yamlexport

import (
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/parandhamareddybommaka/kube/pkg/models"
)

func TestClusterToYAML(t *testing.T) {
	c := &models.Cluster{
		ID:         "cl-1",
		Name:       "prod",
		K8sVersion: "v1.29.3-k3s1",
		Region:     "local",
		NodeCount:  3,
		Status:     models.ClusterStatusRunning,
		Message:    "cluster ready",
		APIPort:    6443,
		CreatedAt:  time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}

	out, err := ClusterToYAML(c)
	if err != nil {
		t.Fatalf("ClusterToYAML: %v", err)
	}

	var doc map[string]any
	if err := yaml.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("output is not valid YAML: %v", err)
	}

	if doc["kind"] != "Cluster" {
		t.Errorf("kind = %v, want Cluster", doc["kind"])
	}
	meta := doc["metadata"].(map[string]any)
	if meta["name"] != "prod" || meta["uid"] != "cl-1" {
		t.Errorf("metadata mismatch: %v", meta)
	}
	if meta["creationTimestamp"] != "2026-01-02T03:04:05Z" {
		t.Errorf("creationTimestamp = %v", meta["creationTimestamp"])
	}
	spec := doc["spec"].(map[string]any)
	if spec["nodeCount"] != 3 {
		t.Errorf("nodeCount = %v, want 3", spec["nodeCount"])
	}
}

func TestClusterToYAMLZeroTime(t *testing.T) {
	out, err := ClusterToYAML(&models.Cluster{ID: "x", Name: "n"})
	if err != nil {
		t.Fatalf("ClusterToYAML: %v", err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("invalid YAML: %v", err)
	}
	meta := doc["metadata"].(map[string]any)
	if meta["creationTimestamp"] != "" {
		t.Errorf("zero time should render empty, got %v", meta["creationTimestamp"])
	}
}

func TestLoadBalancerToYAML(t *testing.T) {
	lb := &models.LoadBalancer{
		ID:         "lb-1",
		Name:       "web-lb",
		ClusterID:  "cl-1",
		Status:     models.LBStatusRunning,
		Algorithm:  models.AlgoLeastConn,
		Port:       30080,
		TargetPort: 8080,
		Backends: []models.Backend{
			{Host: "10.0.0.1", Port: 8080},
			{Host: "10.0.0.2", Port: 8080},
		},
		CreatedAt: time.Date(2026, 6, 27, 0, 0, 0, 0, time.UTC),
	}

	out, err := LoadBalancerToYAML(lb)
	if err != nil {
		t.Fatalf("LoadBalancerToYAML: %v", err)
	}

	if !strings.Contains(out, "\n---\n") {
		t.Fatal("expected Service and Endpoints separated by ---")
	}

	docs := splitYAML(t, out)
	if len(docs) != 2 {
		t.Fatalf("expected 2 documents, got %d", len(docs))
	}

	svc, ep := docs[0], docs[1]
	if svc["kind"] != "Service" {
		t.Errorf("first doc kind = %v, want Service", svc["kind"])
	}
	if ep["kind"] != "Endpoints" {
		t.Errorf("second doc kind = %v, want Endpoints", ep["kind"])
	}

	meta := svc["metadata"].(map[string]any)
	ann := meta["annotations"].(map[string]any)
	if ann["kaas.local/algorithm"] != models.AlgoLeastConn {
		t.Errorf("algorithm annotation = %v", ann["kaas.local/algorithm"])
	}

	subsets := ep["subsets"].([]any)
	addrs := subsets[0].(map[string]any)["addresses"].([]any)
	if len(addrs) != 2 {
		t.Fatalf("expected 2 endpoint addresses, got %d", len(addrs))
	}
}

// When Algorithm is empty, the annotation should fall back to round_robin.
func TestLoadBalancerToYAMLDefaultAlgo(t *testing.T) {
	out, err := LoadBalancerToYAML(&models.LoadBalancer{ID: "lb", Name: "n", Port: 80})
	if err != nil {
		t.Fatalf("LoadBalancerToYAML: %v", err)
	}
	docs := splitYAML(t, out)
	ann := docs[0]["metadata"].(map[string]any)["annotations"].(map[string]any)
	if ann["kaas.local/algorithm"] != models.AlgoRoundRobin {
		t.Errorf("empty algorithm should default to round_robin, got %v", ann["kaas.local/algorithm"])
	}
}

// TargetPort 0 should fall back to the listen port for endpoint ports.
func TestLoadBalancerToYAMLTargetPortFallback(t *testing.T) {
	out, err := LoadBalancerToYAML(&models.LoadBalancer{
		ID: "lb", Name: "n", Port: 9000, TargetPort: 0,
		Backends: []models.Backend{{Host: "1.1.1.1"}},
	})
	if err != nil {
		t.Fatalf("LoadBalancerToYAML: %v", err)
	}
	docs := splitYAML(t, out)
	ports := docs[1]["subsets"].([]any)[0].(map[string]any)["ports"].([]any)
	if ports[0].(map[string]any)["port"] != 9000 {
		t.Errorf("endpoint port = %v, want fallback to 9000", ports[0].(map[string]any)["port"])
	}
}

func TestKubeconfig(t *testing.T) {
	c := &models.Cluster{Kubeconfig: "apiVersion: v1\nkind: Config\n"}
	if got := Kubeconfig(c); got != c.Kubeconfig {
		t.Errorf("Kubeconfig returned %q", got)
	}
}

func splitYAML(t *testing.T, s string) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, part := range strings.Split(s, "---\n") {
		if strings.TrimSpace(part) == "" {
			continue
		}
		var m map[string]any
		if err := yaml.Unmarshal([]byte(part), &m); err != nil {
			t.Fatalf("invalid YAML document: %v\n%s", err, part)
		}
		out = append(out, m)
	}
	return out
}
