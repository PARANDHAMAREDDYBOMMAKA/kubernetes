package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/parandhamareddybommaka/kube/pkg/api/types"
)

var (
	apiServer = "http://localhost:8080"
	namespace = "default"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	if env := os.Getenv("KUBE_API_SERVER"); env != "" {
		apiServer = env
	}

	if env := os.Getenv("KUBE_NAMESPACE"); env != "" {
		namespace = env
	}

	args := os.Args[1:]
	for i, arg := range args {
		if arg == "-n" || arg == "--namespace" {
			if i+1 < len(args) {
				namespace = args[i+1]
				args = append(args[:i], args[i+2:]...)
				break
			}
		}
	}

	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	cmd := args[0]
	cmdArgs := args[1:]

	var err error
	switch cmd {
	case "get":
		err = cmdGet(cmdArgs)
	case "create":
		err = cmdCreate(cmdArgs)
	case "delete":
		err = cmdDelete(cmdArgs)
	case "apply":
		err = cmdApply(cmdArgs)
	case "describe":
		err = cmdDescribe(cmdArgs)
	case "logs":
		err = cmdLogs(cmdArgs)
	case "exec":
		err = cmdExec(cmdArgs)
	case "version":
		fmt.Println("kube CLI v1.0.0")
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Usage: kube <command> [options]

Commands:
  get       Get resources
  create    Create a resource
  delete    Delete a resource
  apply     Apply a configuration
  describe  Show details of a resource
  logs      Print container logs
  exec      Execute a command in a container
  version   Print version

Options:
  -n, --namespace    Specify the namespace

Examples:
  kube get pods
  kube get nodes
  kube create -f pod.yaml
  kube delete pod nginx
  kube logs nginx`)
}

func cmdGet(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("resource type required")
	}

	resource := args[0]
	name := ""
	if len(args) > 1 {
		name = args[1]
	}

	switch resource {
	case "pods", "pod", "po":
		return getPods(name)
	case "nodes", "node", "no":
		return getNodes(name)
	case "services", "service", "svc":
		return getServices(name)
	case "deployments", "deployment", "deploy":
		return getDeployments(name)
	case "namespaces", "namespace", "ns":
		return getNamespaces(name)
	case "configmaps", "configmap", "cm":
		return getConfigMaps(name)
	case "secrets", "secret":
		return getSecrets(name)
	default:
		return fmt.Errorf("unknown resource type: %s", resource)
	}
}

func getPods(name string) error {
	var url string
	if name != "" {
		url = fmt.Sprintf("%s/api/v1/namespaces/%s/pods/%s", apiServer, namespace, name)
	} else {
		url = fmt.Sprintf("%s/api/v1/namespaces/%s/pods", apiServer, namespace)
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return handleError(resp)
	}

	if name != "" {
		var pod types.Pod
		if err := json.NewDecoder(resp.Body).Decode(&pod); err != nil {
			return err
		}
		printPods([]types.Pod{pod})
	} else {
		var result struct {
			Items []types.Pod `json:"items"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return err
		}
		printPods(result.Items)
	}

	return nil
}

func printPods(pods []types.Pod) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tREADY\tSTATUS\tRESTARTS\tAGE")

	for _, pod := range pods {
		ready := 0
		total := len(pod.Spec.Containers)
		restarts := 0

		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Ready {
				ready++
			}
			restarts += cs.RestartCnt
		}

		age := formatAge(pod.Created)
		fmt.Fprintf(w, "%s\t%d/%d\t%s\t%d\t%s\n", pod.Name, ready, total, pod.Status.Phase, restarts, age)
	}
	w.Flush()
}

func getNodes(name string) error {
	var url string
	if name != "" {
		url = fmt.Sprintf("%s/api/v1/nodes/%s", apiServer, name)
	} else {
		url = fmt.Sprintf("%s/api/v1/nodes", apiServer)
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return handleError(resp)
	}

	if name != "" {
		var node types.Node
		if err := json.NewDecoder(resp.Body).Decode(&node); err != nil {
			return err
		}
		printNodes([]types.Node{node})
	} else {
		var result struct {
			Items []types.Node `json:"items"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return err
		}
		printNodes(result.Items)
	}

	return nil
}

func printNodes(nodes []types.Node) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tROLES\tAGE\tVERSION")

	for _, node := range nodes {
		status := "NotReady"
		for _, cond := range node.Status.Conditions {
			if cond.Type == types.NodeReady && cond.Status {
				status = "Ready"
				break
			}
		}

		roles := "<none>"
		if v, ok := node.Labels["node-role.kubernetes.io/master"]; ok && v == "true" {
			roles = "master"
		}
		if v, ok := node.Labels["node-role.kubernetes.io/worker"]; ok && v == "true" {
			if roles == "<none>" {
				roles = "worker"
			} else {
				roles += ",worker"
			}
		}

		age := formatAge(node.Created)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\tv1.0.0\n", node.Name, status, roles, age)
	}
	w.Flush()
}

func getServices(name string) error {
	var url string
	if name != "" {
		url = fmt.Sprintf("%s/api/v1/namespaces/%s/services/%s", apiServer, namespace, name)
	} else {
		url = fmt.Sprintf("%s/api/v1/namespaces/%s/services", apiServer, namespace)
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return handleError(resp)
	}

	var services []types.Service
	if name != "" {
		var svc types.Service
		if err := json.NewDecoder(resp.Body).Decode(&svc); err != nil {
			return err
		}
		services = []types.Service{svc}
	} else {
		var result struct {
			Items []types.Service `json:"items"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return err
		}
		services = result.Items
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tTYPE\tCLUSTER-IP\tPORT(S)\tAGE")

	for _, svc := range services {
		var ports []string
		for _, p := range svc.Spec.Ports {
			if svc.Spec.Type == types.ServiceTypeNodePort && p.NodePort > 0 {
				ports = append(ports, fmt.Sprintf("%d:%d/%s", p.Port, p.NodePort, p.Protocol))
			} else {
				ports = append(ports, fmt.Sprintf("%d/%s", p.Port, p.Protocol))
			}
		}

		age := formatAge(svc.Created)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", svc.Name, svc.Spec.Type, svc.Spec.ClusterIP, strings.Join(ports, ","), age)
	}
	w.Flush()

	return nil
}

func getDeployments(name string) error {
	var url string
	if name != "" {
		url = fmt.Sprintf("%s/api/v1/namespaces/%s/deployments/%s", apiServer, namespace, name)
	} else {
		url = fmt.Sprintf("%s/api/v1/namespaces/%s/deployments", apiServer, namespace)
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return handleError(resp)
	}

	var deployments []types.Deployment
	if name != "" {
		var dep types.Deployment
		if err := json.NewDecoder(resp.Body).Decode(&dep); err != nil {
			return err
		}
		deployments = []types.Deployment{dep}
	} else {
		var result struct {
			Items []types.Deployment `json:"items"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return err
		}
		deployments = result.Items
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tREADY\tUP-TO-DATE\tAVAILABLE\tAGE")

	for _, dep := range deployments {
		age := formatAge(dep.Created)
		fmt.Fprintf(w, "%s\t%d/%d\t%d\t%d\t%s\n", dep.Name, dep.Status.ReadyReplicas, dep.Spec.Replicas, dep.Status.UpdatedReplicas, dep.Status.AvailableReplicas, age)
	}
	w.Flush()

	return nil
}

func getNamespaces(name string) error {
	var url string
	if name != "" {
		url = fmt.Sprintf("%s/api/v1/namespaces/%s", apiServer, name)
	} else {
		url = fmt.Sprintf("%s/api/v1/namespaces", apiServer)
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return handleError(resp)
	}

	var namespaces []types.Namespace
	if name != "" {
		var ns types.Namespace
		if err := json.NewDecoder(resp.Body).Decode(&ns); err != nil {
			return err
		}
		namespaces = []types.Namespace{ns}
	} else {
		var result struct {
			Items []types.Namespace `json:"items"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return err
		}
		namespaces = result.Items
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tAGE")

	for _, ns := range namespaces {
		age := formatAge(ns.Created)
		fmt.Fprintf(w, "%s\t%s\t%s\n", ns.Name, ns.Status.Phase, age)
	}
	w.Flush()

	return nil
}

func getConfigMaps(name string) error {
	var url string
	if name != "" {
		url = fmt.Sprintf("%s/api/v1/namespaces/%s/configmaps/%s", apiServer, namespace, name)
	} else {
		url = fmt.Sprintf("%s/api/v1/namespaces/%s/configmaps", apiServer, namespace)
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return handleError(resp)
	}

	var configmaps []types.ConfigMap
	if name != "" {
		var cm types.ConfigMap
		if err := json.NewDecoder(resp.Body).Decode(&cm); err != nil {
			return err
		}
		configmaps = []types.ConfigMap{cm}
	} else {
		var result struct {
			Items []types.ConfigMap `json:"items"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return err
		}
		configmaps = result.Items
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tDATA\tAGE")

	for _, cm := range configmaps {
		age := formatAge(cm.Created)
		fmt.Fprintf(w, "%s\t%d\t%s\n", cm.Name, len(cm.Data), age)
	}
	w.Flush()

	return nil
}

func getSecrets(name string) error {
	var url string
	if name != "" {
		url = fmt.Sprintf("%s/api/v1/namespaces/%s/secrets/%s", apiServer, namespace, name)
	} else {
		url = fmt.Sprintf("%s/api/v1/namespaces/%s/secrets", apiServer, namespace)
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return handleError(resp)
	}

	var secrets []types.Secret
	if name != "" {
		var s types.Secret
		if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
			return err
		}
		secrets = []types.Secret{s}
	} else {
		var result struct {
			Items []types.Secret `json:"items"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return err
		}
		secrets = result.Items
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tTYPE\tDATA\tAGE")

	for _, s := range secrets {
		age := formatAge(s.Created)
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", s.Name, s.Type, len(s.Data), age)
	}
	w.Flush()

	return nil
}

func cmdCreate(args []string) error {
	if len(args) < 2 || args[0] != "-f" {
		return fmt.Errorf("usage: kube create -f <file>")
	}

	return applyFile(args[1])
}

func cmdApply(args []string) error {
	if len(args) < 2 || args[0] != "-f" {
		return fmt.Errorf("usage: kube apply -f <file>")
	}

	return applyFile(args[1])
}

func applyFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}

	kind, _ := obj["kind"].(string)
	metadata, _ := obj["metadata"].(map[string]any)
	name, _ := metadata["name"].(string)
	ns := namespace
	if v, ok := metadata["namespace"].(string); ok {
		ns = v
	}

	var url string
	switch strings.ToLower(kind) {
	case "pod":
		url = fmt.Sprintf("%s/api/v1/namespaces/%s/pods", apiServer, ns)
	case "service":
		url = fmt.Sprintf("%s/api/v1/namespaces/%s/services", apiServer, ns)
	case "deployment":
		url = fmt.Sprintf("%s/api/v1/namespaces/%s/deployments", apiServer, ns)
	case "configmap":
		url = fmt.Sprintf("%s/api/v1/namespaces/%s/configmaps", apiServer, ns)
	case "secret":
		url = fmt.Sprintf("%s/api/v1/namespaces/%s/secrets", apiServer, ns)
	case "namespace":
		url = fmt.Sprintf("%s/api/v1/namespaces", apiServer)
	default:
		return fmt.Errorf("unknown kind: %s", kind)
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return handleError(resp)
	}

	fmt.Printf("%s/%s created\n", strings.ToLower(kind), name)
	return nil
}

func cmdDelete(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: kube delete <resource> <name>")
	}

	resource := args[0]
	name := args[1]

	var url string
	switch resource {
	case "pod", "pods", "po":
		url = fmt.Sprintf("%s/api/v1/namespaces/%s/pods/%s", apiServer, namespace, name)
	case "service", "services", "svc":
		url = fmt.Sprintf("%s/api/v1/namespaces/%s/services/%s", apiServer, namespace, name)
	case "deployment", "deployments", "deploy":
		url = fmt.Sprintf("%s/api/v1/namespaces/%s/deployments/%s", apiServer, namespace, name)
	case "configmap", "configmaps", "cm":
		url = fmt.Sprintf("%s/api/v1/namespaces/%s/configmaps/%s", apiServer, namespace, name)
	case "secret", "secrets":
		url = fmt.Sprintf("%s/api/v1/namespaces/%s/secrets/%s", apiServer, namespace, name)
	case "namespace", "namespaces", "ns":
		url = fmt.Sprintf("%s/api/v1/namespaces/%s", apiServer, name)
	case "node", "nodes", "no":
		url = fmt.Sprintf("%s/api/v1/nodes/%s", apiServer, name)
	default:
		return fmt.Errorf("unknown resource type: %s", resource)
	}

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return handleError(resp)
	}

	fmt.Printf("%s \"%s\" deleted\n", resource, name)
	return nil
}

func cmdDescribe(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: kube describe <resource> <name>")
	}

	resource := args[0]
	name := args[1]

	var url string
	switch resource {
	case "pod", "pods", "po":
		url = fmt.Sprintf("%s/api/v1/namespaces/%s/pods/%s", apiServer, namespace, name)
	case "node", "nodes", "no":
		url = fmt.Sprintf("%s/api/v1/nodes/%s", apiServer, name)
	case "service", "services", "svc":
		url = fmt.Sprintf("%s/api/v1/namespaces/%s/services/%s", apiServer, namespace, name)
	case "deployment", "deployments", "deploy":
		url = fmt.Sprintf("%s/api/v1/namespaces/%s/deployments/%s", apiServer, namespace, name)
	default:
		return fmt.Errorf("unknown resource type: %s", resource)
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return handleError(resp)
	}

	var obj map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&obj); err != nil {
		return err
	}

	data, _ := json.MarshalIndent(obj, "", "  ")
	fmt.Println(string(data))

	return nil
}

func cmdLogs(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: kube logs <pod>")
	}

	fmt.Println("logs command not yet implemented")
	return nil
}

func cmdExec(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: kube exec <pod> -- <command>")
	}

	fmt.Println("exec command not yet implemented")
	return nil
}

func formatAge(t time.Time) string {
	d := time.Since(t)

	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

func handleError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var errResp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
		return fmt.Errorf("%s", errResp.Error)
	}
	return fmt.Errorf("request failed: %s", resp.Status)
}
