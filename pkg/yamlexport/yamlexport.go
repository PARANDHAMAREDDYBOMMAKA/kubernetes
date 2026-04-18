package yamlexport

import (
	"bytes"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/parandhamareddybommaka/kube/pkg/models"
)

func ClusterToYAML(c *models.Cluster) (string, error) {
	doc := map[string]any{
		"apiVersion": "kaas.local/v1",
		"kind":       "Cluster",
		"metadata": map[string]any{
			"name":              c.Name,
			"uid":               c.ID,
			"creationTimestamp": formatTime(c.CreatedAt),
		},
		"spec": map[string]any{
			"k8sVersion": c.K8sVersion,
			"region":     c.Region,
			"nodeCount":  c.NodeCount,
		},
		"status": map[string]any{
			"phase":   c.Status,
			"message": c.Message,
			"apiPort": c.APIPort,
		},
	}
	return marshal(doc)
}

func LoadBalancerToYAML(lb *models.LoadBalancer) (string, error) {
	svc := map[string]any{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata": map[string]any{
			"name":              lb.Name,
			"uid":               lb.ID,
			"creationTimestamp": formatTime(lb.CreatedAt),
			"annotations": map[string]any{
				"kaas.local/cluster-id":  lb.ClusterID,
				"kaas.local/algorithm":   nonEmpty(lb.Algorithm, models.AlgoRoundRobin),
				"kaas.local/status":      lb.Status,
				"kaas.local/container":   lb.ContainerID,
			},
		},
		"spec": map[string]any{
			"type": "LoadBalancer",
			"ports": []map[string]any{{
				"port":       lb.Port,
				"targetPort": lb.TargetPort,
				"protocol":   "TCP",
			}},
			"selector": map[string]any{
				"kaas.local/lb": lb.ID,
			},
		},
	}

	addrs := make([]map[string]any, 0, len(lb.Backends))
	for _, b := range lb.Backends {
		addrs = append(addrs, map[string]any{"ip": b.Host})
	}
	port := lb.TargetPort
	if port == 0 {
		port = lb.Port
	}
	subsets := []map[string]any{{
		"addresses": addrs,
		"ports": []map[string]any{{
			"port":     port,
			"protocol": "TCP",
		}},
	}}
	endpoints := map[string]any{
		"apiVersion": "v1",
		"kind":       "Endpoints",
		"metadata": map[string]any{
			"name": lb.Name,
		},
		"subsets": subsets,
	}

	svcYAML, err := marshal(svc)
	if err != nil {
		return "", err
	}
	epYAML, err := marshal(endpoints)
	if err != nil {
		return "", err
	}
	return svcYAML + "---\n" + epYAML, nil
}

func Kubeconfig(c *models.Cluster) string {
	return c.Kubeconfig
}

func marshal(v any) (string, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	if err := enc.Close(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func nonEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

