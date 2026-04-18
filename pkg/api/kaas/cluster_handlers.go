package kaas

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/parandhamareddybommaka/kube/pkg/auth"
	"github.com/parandhamareddybommaka/kube/pkg/db"
	"github.com/parandhamareddybommaka/kube/pkg/models"
	"github.com/parandhamareddybommaka/kube/pkg/provisioner"
	"github.com/parandhamareddybommaka/kube/pkg/yamlexport"
)

type createClusterRequest struct {
	Name       string `json:"name"`
	K8sVersion string `json:"k8sVersion"`
	NodeCount  int    `json:"nodeCount"`
	Region     string `json:"region"`
}

type scaleClusterRequest struct {
	NodeCount int `json:"nodeCount"`
}

func (s *Server) ListClusters(w http.ResponseWriter, r *http.Request) {
	if !requireDB(w) {
		return
	}
	userID := auth.UserIDFromCtx(r.Context())
	cur, err := db.Clusters().Find(r.Context(), bson.M{"user_id": userID})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "database error")
		return
	}
	defer cur.Close(r.Context())
	clusters := []*models.Cluster{}
	if err := cur.All(r.Context(), &clusters); err != nil {
		writeErr(w, http.StatusInternalServerError, "database error")
		return
	}
	for _, c := range clusters {
		c.Kubeconfig = ""
		c.ClusterToken = ""
	}
	writeJSON(w, http.StatusOK, clusters)
}

func (s *Server) CreateCluster(w http.ResponseWriter, r *http.Request) {
	if !requireDB(w) {
		return
	}
	var req createClusterRequest
	if !readJSON(w, r, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeErr(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.NodeCount < 0 {
		writeErr(w, http.StatusBadRequest, "nodeCount must be >= 0")
		return
	}
	if req.K8sVersion == "" {
		req.K8sVersion = "v1.29.3-k3s1"
	}
	if req.Region == "" {
		req.Region = "local"
	}

	userID := auth.UserIDFromCtx(r.Context())
	now := nowUTC()
	cluster := &models.Cluster{
		ID:         genID(),
		UserID:     userID,
		Name:       req.Name,
		K8sVersion: req.K8sVersion,
		Region:     req.Region,
		NodeCount:  req.NodeCount,
		Status:     models.ClusterStatusProvisioning,
		CreatedAt:  now,
		UpdatedAt:  now,
		Message:    "queued",
	}
	if _, err := db.Clusters().InsertOne(r.Context(), cluster); err != nil {
		s.Logger.Error("insert cluster", "err", err)
		writeErr(w, http.StatusInternalServerError, "failed to create cluster")
		return
	}

	go s.asyncProvision(cluster.ID)

	writeJSON(w, http.StatusAccepted, cluster)
}

func (s *Server) GetCluster(w http.ResponseWriter, r *http.Request) {
	if !requireDB(w) {
		return
	}
	cluster, err := s.loadOwnedCluster(r.Context(), r.PathValue("id"))
	if err != nil {
		writeNotFound(w, err)
		return
	}
	cluster.Kubeconfig = ""
	cluster.ClusterToken = ""
	writeJSON(w, http.StatusOK, cluster)
}

func (s *Server) DeleteCluster(w http.ResponseWriter, r *http.Request) {
	if !requireDB(w) {
		return
	}
	cluster, err := s.loadOwnedCluster(r.Context(), r.PathValue("id"))
	if err != nil {
		writeNotFound(w, err)
		return
	}
	cluster.Status = models.ClusterStatusDeleting
	cluster.UpdatedAt = nowUTC()
	_ = s.persistCluster(r.Context(), cluster)

	go s.asyncDelete(cluster.ID)

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) ScaleCluster(w http.ResponseWriter, r *http.Request) {
	if !requireDB(w) {
		return
	}
	cluster, err := s.loadOwnedCluster(r.Context(), r.PathValue("id"))
	if err != nil {
		writeNotFound(w, err)
		return
	}
	var req scaleClusterRequest
	if !readJSON(w, r, &req) {
		return
	}
	if req.NodeCount < 0 {
		writeErr(w, http.StatusBadRequest, "nodeCount must be >= 0")
		return
	}
	cluster.NodeCount = req.NodeCount
	cluster.Status = models.ClusterStatusProvisioning
	cluster.Message = "scaling"
	cluster.UpdatedAt = nowUTC()
	if err := s.persistCluster(r.Context(), cluster); err != nil {
		writeErr(w, http.StatusInternalServerError, "database error")
		return
	}
	go s.asyncScale(cluster.ID, req.NodeCount)

	cluster.Kubeconfig = ""
	cluster.ClusterToken = ""
	writeJSON(w, http.StatusAccepted, cluster)
}

func (s *Server) Kubeconfig(w http.ResponseWriter, r *http.Request) {
	if !requireDB(w) {
		return
	}
	cluster, err := s.loadOwnedCluster(r.Context(), r.PathValue("id"))
	if err != nil {
		writeNotFound(w, err)
		return
	}
	if cluster.Kubeconfig == "" {
		writeErr(w, http.StatusConflict, "kubeconfig not yet available")
		return
	}
	w.Header().Set("Content-Type", "application/yaml")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+cluster.Name+"-kubeconfig.yaml\"")
	_, _ = w.Write([]byte(cluster.Kubeconfig))
}

func (s *Server) ListNodes(w http.ResponseWriter, r *http.Request) {
	if !requireDB(w) {
		return
	}
	cluster, err := s.loadOwnedCluster(r.Context(), r.PathValue("id"))
	if err != nil {
		writeNotFound(w, err)
		return
	}
	if s.Prov == nil {
		writeJSON(w, http.StatusOK, []provisioner.NodeInfo{})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	nodes, err := s.Prov.Nodes(ctx, cluster)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "list nodes: "+err.Error())
		return
	}
	if nodes == nil {
		nodes = []provisioner.NodeInfo{}
	}
	writeJSON(w, http.StatusOK, nodes)
}

func (s *Server) ClusterYAML(w http.ResponseWriter, r *http.Request) {
	if !requireDB(w) {
		return
	}
	cluster, err := s.loadOwnedCluster(r.Context(), r.PathValue("id"))
	if err != nil {
		writeNotFound(w, err)
		return
	}
	yamlStr, err := yamlexport.ClusterToYAML(cluster)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "render yaml: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/yaml")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+cluster.Name+".yaml\"")
	_, _ = w.Write([]byte(yamlStr))
}

func (s *Server) loadOwnedCluster(ctx context.Context, id string) (*models.Cluster, error) {
	if id == "" {
		return nil, mongo.ErrNoDocuments
	}
	userID := auth.UserIDFromCtx(ctx)
	var c models.Cluster
	err := db.Clusters().FindOne(ctx, bson.M{"_id": id}).Decode(&c)
	if err != nil {
		return nil, err
	}
	if c.UserID != userID {
		return nil, mongo.ErrNoDocuments
	}
	return &c, nil
}

func (s *Server) persistCluster(ctx context.Context, c *models.Cluster) error {
	c.UpdatedAt = nowUTC()
	_, err := db.Clusters().UpdateOne(ctx, bson.M{"_id": c.ID}, bson.M{"$set": c})
	return err
}

func writeNotFound(w http.ResponseWriter, err error) {
	if errors.Is(err, mongo.ErrNoDocuments) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	writeErr(w, http.StatusInternalServerError, "database error")
}
