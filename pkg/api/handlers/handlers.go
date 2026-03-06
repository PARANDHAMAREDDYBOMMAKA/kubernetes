package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/parandhamareddybommaka/kube/pkg/api/types"
	"github.com/parandhamareddybommaka/kube/pkg/store"
)

type APIHandler struct {
	store     store.Store
	scheduler PodScheduler
	mu        sync.RWMutex
}

type PodScheduler interface {
	AddPod(pod *types.Pod)
}

func New(s store.Store, scheduler PodScheduler) *APIHandler {
	return &APIHandler{
		store:     s,
		scheduler: scheduler,
	}
}

func (h *APIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	switch {
	case len(parts) >= 1 && parts[0] == "namespaces":
		h.handleNamespaces(w, r, parts[1:])
	case len(parts) >= 1 && parts[0] == "nodes":
		h.handleNodes(w, r, parts[1:])
	case len(parts) >= 1 && parts[0] == "pods":
		h.handleAllPods(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *APIHandler) handleNamespaces(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		switch r.Method {
		case http.MethodGet:
			h.listNamespaces(w, r)
		case http.MethodPost:
			h.createNamespace(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	ns := parts[0]
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			h.getNamespace(w, r, ns)
		case http.MethodDelete:
			h.deleteNamespace(w, r, ns)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	resource := parts[1]
	name := ""
	if len(parts) > 2 {
		name = parts[2]
	}

	switch resource {
	case "pods":
		h.handlePods(w, r, ns, name)
	case "services":
		h.handleServices(w, r, ns, name)
	case "deployments":
		h.handleDeployments(w, r, ns, name)
	case "configmaps":
		h.handleConfigMaps(w, r, ns, name)
	case "secrets":
		h.handleSecrets(w, r, ns, name)
	default:
		http.NotFound(w, r)
	}
}

func (h *APIHandler) listNamespaces(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	data, err := h.store.List(ctx, "/namespaces/")
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	var items []types.Namespace
	for _, d := range data {
		var ns types.Namespace
		if err := json.Unmarshal(d, &ns); err == nil {
			items = append(items, ns)
		}
	}

	writeJSON(w, map[string]any{"items": items})
}

func (h *APIHandler) createNamespace(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var ns types.Namespace
	if err := readJSON(r, &ns); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}

	ns.UID = generateUID()
	ns.Created = time.Now()
	ns.Status.Phase = types.NamespaceActive

	data, _ := json.Marshal(ns)
	key := "/namespaces/" + ns.Name
	if err := h.store.Create(ctx, key, data); err != nil {
		writeError(w, err, http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, ns)
}

func (h *APIHandler) getNamespace(w http.ResponseWriter, r *http.Request, name string) {
	ctx := r.Context()
	data, _, err := h.store.Get(ctx, "/namespaces/"+name)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, err, http.StatusNotFound)
			return
		}
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	var ns types.Namespace
	json.Unmarshal(data, &ns)
	writeJSON(w, ns)
}

func (h *APIHandler) deleteNamespace(w http.ResponseWriter, r *http.Request, name string) {
	ctx := r.Context()
	if err := h.store.Delete(ctx, "/namespaces/"+name); err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *APIHandler) handlePods(w http.ResponseWriter, r *http.Request, ns, name string) {
	ctx := r.Context()

	if name == "" {
		switch r.Method {
		case http.MethodGet:
			h.listPods(w, ctx, ns)
		case http.MethodPost:
			h.createPod(w, r, ns)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getPod(w, ctx, ns, name)
	case http.MethodPut:
		h.updatePod(w, r, ns, name)
	case http.MethodDelete:
		h.deletePod(w, ctx, ns, name)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *APIHandler) listPods(w http.ResponseWriter, ctx context.Context, ns string) {
	data, err := h.store.List(ctx, fmt.Sprintf("/pods/%s/", ns))
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	var items []types.Pod
	for _, d := range data {
		var pod types.Pod
		if err := json.Unmarshal(d, &pod); err == nil {
			items = append(items, pod)
		}
	}

	writeJSON(w, map[string]any{"items": items})
}

func (h *APIHandler) handleAllPods(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	data, err := h.store.List(ctx, "/pods/")
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	var items []types.Pod
	for _, d := range data {
		var pod types.Pod
		if err := json.Unmarshal(d, &pod); err == nil {
			items = append(items, pod)
		}
	}

	writeJSON(w, map[string]any{"items": items})
}

func (h *APIHandler) createPod(w http.ResponseWriter, r *http.Request, ns string) {
	ctx := r.Context()
	var pod types.Pod
	if err := readJSON(r, &pod); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}

	pod.UID = generateUID()
	pod.Namespace = ns
	pod.Created = time.Now()
	pod.Status.Phase = types.PodPending

	if pod.Spec.RestartPolicy == "" {
		pod.Spec.RestartPolicy = types.RestartAlways
	}

	data, _ := json.Marshal(pod)
	key := fmt.Sprintf("/pods/%s/%s", ns, pod.Name)
	if err := h.store.Create(ctx, key, data); err != nil {
		writeError(w, err, http.StatusConflict)
		return
	}

	if h.scheduler != nil && pod.Spec.NodeName == "" {
		h.scheduler.AddPod(&pod)
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, pod)
}

func (h *APIHandler) getPod(w http.ResponseWriter, ctx context.Context, ns, name string) {
	key := fmt.Sprintf("/pods/%s/%s", ns, name)
	data, _, err := h.store.Get(ctx, key)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, err, http.StatusNotFound)
			return
		}
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	var pod types.Pod
	json.Unmarshal(data, &pod)
	writeJSON(w, pod)
}

func (h *APIHandler) updatePod(w http.ResponseWriter, r *http.Request, ns, name string) {
	ctx := r.Context()
	key := fmt.Sprintf("/pods/%s/%s", ns, name)

	_, version, err := h.store.Get(ctx, key)
	if err != nil {
		writeError(w, err, http.StatusNotFound)
		return
	}

	var pod types.Pod
	if err := readJSON(r, &pod); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}

	pod.Updated = time.Now()
	data, _ := json.Marshal(pod)

	if err := h.store.Update(ctx, key, data, version); err != nil {
		writeError(w, err, http.StatusConflict)
		return
	}

	writeJSON(w, pod)
}

func (h *APIHandler) deletePod(w http.ResponseWriter, ctx context.Context, ns, name string) {
	key := fmt.Sprintf("/pods/%s/%s", ns, name)
	if err := h.store.Delete(ctx, key); err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *APIHandler) handleNodes(w http.ResponseWriter, r *http.Request, parts []string) {
	ctx := r.Context()

	if len(parts) == 0 {
		switch r.Method {
		case http.MethodGet:
			h.listNodes(w, ctx)
		case http.MethodPost:
			h.createNode(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	name := parts[0]
	switch r.Method {
	case http.MethodGet:
		h.getNode(w, ctx, name)
	case http.MethodPut:
		h.updateNode(w, r, name)
	case http.MethodDelete:
		h.deleteNode(w, ctx, name)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *APIHandler) listNodes(w http.ResponseWriter, ctx context.Context) {
	data, err := h.store.List(ctx, "/nodes/")
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	var items []types.Node
	for _, d := range data {
		var node types.Node
		if err := json.Unmarshal(d, &node); err == nil {
			items = append(items, node)
		}
	}

	writeJSON(w, map[string]any{"items": items})
}

func (h *APIHandler) createNode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var node types.Node
	if err := readJSON(r, &node); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}

	node.UID = generateUID()
	node.Created = time.Now()

	data, _ := json.Marshal(node)
	key := "/nodes/" + node.Name
	if err := h.store.Create(ctx, key, data); err != nil {
		writeError(w, err, http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, node)
}

func (h *APIHandler) getNode(w http.ResponseWriter, ctx context.Context, name string) {
	data, _, err := h.store.Get(ctx, "/nodes/"+name)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, err, http.StatusNotFound)
			return
		}
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	var node types.Node
	json.Unmarshal(data, &node)
	writeJSON(w, node)
}

func (h *APIHandler) updateNode(w http.ResponseWriter, r *http.Request, name string) {
	ctx := r.Context()
	key := "/nodes/" + name

	_, version, err := h.store.Get(ctx, key)
	if err != nil {
		writeError(w, err, http.StatusNotFound)
		return
	}

	var node types.Node
	if err := readJSON(r, &node); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}

	node.Updated = time.Now()
	data, _ := json.Marshal(node)

	if err := h.store.Update(ctx, key, data, version); err != nil {
		writeError(w, err, http.StatusConflict)
		return
	}

	writeJSON(w, node)
}

func (h *APIHandler) deleteNode(w http.ResponseWriter, ctx context.Context, name string) {
	if err := h.store.Delete(ctx, "/nodes/"+name); err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *APIHandler) handleServices(w http.ResponseWriter, r *http.Request, ns, name string) {
	ctx := r.Context()

	if name == "" {
		switch r.Method {
		case http.MethodGet:
			h.listResources(w, ctx, fmt.Sprintf("/services/%s/", ns), func() any { return &types.Service{} })
		case http.MethodPost:
			h.createService(w, r, ns)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	key := fmt.Sprintf("/services/%s/%s", ns, name)
	switch r.Method {
	case http.MethodGet:
		h.getResource(w, ctx, key, &types.Service{})
	case http.MethodDelete:
		h.deleteResource(w, ctx, key)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *APIHandler) createService(w http.ResponseWriter, r *http.Request, ns string) {
	ctx := r.Context()
	var svc types.Service
	if err := readJSON(r, &svc); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}

	svc.UID = generateUID()
	svc.Namespace = ns
	svc.Created = time.Now()

	if svc.Spec.Type == "" {
		svc.Spec.Type = types.ServiceTypeClusterIP
	}

	if svc.Spec.ClusterIP == "" {
		svc.Spec.ClusterIP = generateClusterIP()
	}

	data, _ := json.Marshal(svc)
	key := fmt.Sprintf("/services/%s/%s", ns, svc.Name)
	if err := h.store.Create(ctx, key, data); err != nil {
		writeError(w, err, http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, svc)
}

func (h *APIHandler) handleDeployments(w http.ResponseWriter, r *http.Request, ns, name string) {
	ctx := r.Context()

	if name == "" {
		switch r.Method {
		case http.MethodGet:
			h.listResources(w, ctx, fmt.Sprintf("/deployments/%s/", ns), func() any { return &types.Deployment{} })
		case http.MethodPost:
			h.createDeployment(w, r, ns)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	key := fmt.Sprintf("/deployments/%s/%s", ns, name)
	switch r.Method {
	case http.MethodGet:
		h.getResource(w, ctx, key, &types.Deployment{})
	case http.MethodPut:
		h.updateDeployment(w, r, ns, name)
	case http.MethodDelete:
		h.deleteResource(w, ctx, key)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *APIHandler) createDeployment(w http.ResponseWriter, r *http.Request, ns string) {
	ctx := r.Context()
	var dep types.Deployment
	if err := readJSON(r, &dep); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}

	dep.UID = generateUID()
	dep.Namespace = ns
	dep.Created = time.Now()

	data, _ := json.Marshal(dep)
	key := fmt.Sprintf("/deployments/%s/%s", ns, dep.Name)
	if err := h.store.Create(ctx, key, data); err != nil {
		writeError(w, err, http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, dep)
}

func (h *APIHandler) updateDeployment(w http.ResponseWriter, r *http.Request, ns, name string) {
	ctx := r.Context()
	key := fmt.Sprintf("/deployments/%s/%s", ns, name)

	_, version, err := h.store.Get(ctx, key)
	if err != nil {
		writeError(w, err, http.StatusNotFound)
		return
	}

	var dep types.Deployment
	if err := readJSON(r, &dep); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}

	dep.Updated = time.Now()
	data, _ := json.Marshal(dep)

	if err := h.store.Update(ctx, key, data, version); err != nil {
		writeError(w, err, http.StatusConflict)
		return
	}

	writeJSON(w, dep)
}

func (h *APIHandler) handleConfigMaps(w http.ResponseWriter, r *http.Request, ns, name string) {
	ctx := r.Context()

	if name == "" {
		switch r.Method {
		case http.MethodGet:
			h.listResources(w, ctx, fmt.Sprintf("/configmaps/%s/", ns), func() any { return &types.ConfigMap{} })
		case http.MethodPost:
			h.createConfigMap(w, r, ns)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	key := fmt.Sprintf("/configmaps/%s/%s", ns, name)
	switch r.Method {
	case http.MethodGet:
		h.getResource(w, ctx, key, &types.ConfigMap{})
	case http.MethodDelete:
		h.deleteResource(w, ctx, key)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *APIHandler) createConfigMap(w http.ResponseWriter, r *http.Request, ns string) {
	ctx := r.Context()
	var cm types.ConfigMap
	if err := readJSON(r, &cm); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}

	cm.UID = generateUID()
	cm.Namespace = ns
	cm.Created = time.Now()

	data, _ := json.Marshal(cm)
	key := fmt.Sprintf("/configmaps/%s/%s", ns, cm.Name)
	if err := h.store.Create(ctx, key, data); err != nil {
		writeError(w, err, http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, cm)
}

func (h *APIHandler) handleSecrets(w http.ResponseWriter, r *http.Request, ns, name string) {
	ctx := r.Context()

	if name == "" {
		switch r.Method {
		case http.MethodGet:
			h.listResources(w, ctx, fmt.Sprintf("/secrets/%s/", ns), func() any { return &types.Secret{} })
		case http.MethodPost:
			h.createSecret(w, r, ns)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	key := fmt.Sprintf("/secrets/%s/%s", ns, name)
	switch r.Method {
	case http.MethodGet:
		h.getResource(w, ctx, key, &types.Secret{})
	case http.MethodDelete:
		h.deleteResource(w, ctx, key)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *APIHandler) createSecret(w http.ResponseWriter, r *http.Request, ns string) {
	ctx := r.Context()
	var secret types.Secret
	if err := readJSON(r, &secret); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}

	secret.UID = generateUID()
	secret.Namespace = ns
	secret.Created = time.Now()

	if secret.Type == "" {
		secret.Type = "Opaque"
	}

	data, _ := json.Marshal(secret)
	key := fmt.Sprintf("/secrets/%s/%s", ns, secret.Name)
	if err := h.store.Create(ctx, key, data); err != nil {
		writeError(w, err, http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, secret)
}

func (h *APIHandler) listResources(w http.ResponseWriter, ctx context.Context, prefix string, newFunc func() any) {
	data, err := h.store.List(ctx, prefix)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	items := make([]any, 0, len(data))
	for _, d := range data {
		obj := newFunc()
		if err := json.Unmarshal(d, obj); err == nil {
			items = append(items, obj)
		}
	}

	writeJSON(w, map[string]any{"items": items})
}

func (h *APIHandler) getResource(w http.ResponseWriter, ctx context.Context, key string, obj any) {
	data, _, err := h.store.Get(ctx, key)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, err, http.StatusNotFound)
			return
		}
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	json.Unmarshal(data, obj)
	writeJSON(w, obj)
}

func (h *APIHandler) deleteResource(w http.ResponseWriter, ctx context.Context, key string) {
	if err := h.store.Delete(ctx, key); err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func readJSON(r *http.Request, v any) error {
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

var (
	uidCounter uint64
	uidMu      sync.Mutex
	clusterIPCounter uint32 = 1
	clusterIPMu sync.Mutex
)

func generateUID() string {
	uidMu.Lock()
	uidCounter++
	uid := uidCounter
	uidMu.Unlock()
	return fmt.Sprintf("%016x", uid)
}

func generateClusterIP() string {
	clusterIPMu.Lock()
	clusterIPCounter++
	ip := clusterIPCounter
	clusterIPMu.Unlock()
	return fmt.Sprintf("10.96.%d.%d", (ip>>8)&0xFF, ip&0xFF)
}
