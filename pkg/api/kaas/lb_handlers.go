package kaas

import (
	"context"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/parandhamareddybommaka/kube/pkg/auth"
	"github.com/parandhamareddybommaka/kube/pkg/db"
	"github.com/parandhamareddybommaka/kube/pkg/models"
	"github.com/parandhamareddybommaka/kube/pkg/yamlexport"
)

type createLBRequest struct {
	Name       string           `json:"name"`
	ClusterID  string           `json:"clusterId"`
	Port       int              `json:"port"`
	TargetPort int              `json:"targetPort"`
	Algorithm  string           `json:"algorithm"`
	Backends   []models.Backend `json:"backends"`
}

func (s *Server) ListLBs(w http.ResponseWriter, r *http.Request) {
	if !requireDB(w) {
		return
	}
	userID := auth.UserIDFromCtx(r.Context())
	cur, err := db.LoadBalancers().Find(r.Context(), bson.M{"user_id": userID})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "database error")
		return
	}
	defer cur.Close(r.Context())
	lbs := []*models.LoadBalancer{}
	if err := cur.All(r.Context(), &lbs); err != nil {
		writeErr(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, lbs)
}

func (s *Server) CreateLB(w http.ResponseWriter, r *http.Request) {
	if !requireDB(w) {
		return
	}
	if s.LB == nil {
		writeErr(w, http.StatusServiceUnavailable, "load balancer manager not configured")
		return
	}
	var req createLBRequest
	if !readJSON(w, r, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || req.Port == 0 {
		writeErr(w, http.StatusBadRequest, "name and port are required")
		return
	}
	if req.TargetPort == 0 {
		req.TargetPort = req.Port
	}
	if req.Algorithm == "" {
		req.Algorithm = models.AlgoRoundRobin
	}
	switch req.Algorithm {
	case models.AlgoRoundRobin, models.AlgoLeastConn, models.AlgoIPHash:
	default:
		writeErr(w, http.StatusBadRequest, "unknown algorithm")
		return
	}

	userID := auth.UserIDFromCtx(r.Context())

	if req.ClusterID != "" {
		if _, err := s.loadOwnedCluster(r.Context(), req.ClusterID); err != nil {
			writeNotFound(w, err)
			return
		}
	}

	now := time.Now().UTC()
	lb := &models.LoadBalancer{
		ID:         genID(),
		UserID:     userID,
		ClusterID:  req.ClusterID,
		Name:       req.Name,
		Algorithm:  req.Algorithm,
		Port:       req.Port,
		TargetPort: req.TargetPort,
		Backends:   req.Backends,
		Status:     models.LBStatusProvisioning,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if _, err := db.LoadBalancers().InsertOne(r.Context(), lb); err != nil {
		s.Logger.Error("insert lb", "err", err)
		writeErr(w, http.StatusInternalServerError, "failed to create load balancer")
		return
	}

	go s.asyncCreateLB(lb.ID)

	writeJSON(w, http.StatusAccepted, lb)
}

func (s *Server) GetLB(w http.ResponseWriter, r *http.Request) {
	if !requireDB(w) {
		return
	}
	lb, err := s.loadOwnedLB(r.Context(), r.PathValue("id"))
	if err != nil {
		writeNotFound(w, err)
		return
	}
	writeJSON(w, http.StatusOK, lb)
}

func (s *Server) DeleteLB(w http.ResponseWriter, r *http.Request) {
	if !requireDB(w) {
		return
	}
	lb, err := s.loadOwnedLB(r.Context(), r.PathValue("id"))
	if err != nil {
		writeNotFound(w, err)
		return
	}
	lb.Status = models.LBStatusDeleting
	lb.UpdatedAt = time.Now().UTC()
	_, _ = db.LoadBalancers().UpdateOne(r.Context(), bson.M{"_id": lb.ID}, bson.M{"$set": lb})

	go s.asyncDeleteLB(lb.ID)

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) LBYAML(w http.ResponseWriter, r *http.Request) {
	if !requireDB(w) {
		return
	}
	lb, err := s.loadOwnedLB(r.Context(), r.PathValue("id"))
	if err != nil {
		writeNotFound(w, err)
		return
	}
	yamlStr, err := yamlexport.LoadBalancerToYAML(lb)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "render yaml: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/yaml")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+lb.Name+".yaml\"")
	_, _ = w.Write([]byte(yamlStr))
}

func (s *Server) loadOwnedLB(ctx context.Context, id string) (*models.LoadBalancer, error) {
	if id == "" {
		return nil, mongo.ErrNoDocuments
	}
	userID := auth.UserIDFromCtx(ctx)
	var lb models.LoadBalancer
	if err := db.LoadBalancers().FindOne(ctx, bson.M{"_id": id}).Decode(&lb); err != nil {
		return nil, err
	}
	if lb.UserID != userID {
		return nil, mongo.ErrNoDocuments
	}
	return &lb, nil
}

func (s *Server) asyncCreateLB(id string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	var lb models.LoadBalancer
	if err := db.LoadBalancers().FindOne(ctx, bson.M{"_id": id}).Decode(&lb); err != nil {
		s.Logger.Error("async lb load", "err", err, "lb", id)
		return
	}
	if err := s.LB.Create(ctx, &lb); err != nil {
		s.Logger.Error("create lb", "err", err, "lb", id)
	}
	if _, err := db.LoadBalancers().UpdateOne(ctx, bson.M{"_id": lb.ID}, bson.M{"$set": lb}); err != nil {
		s.Logger.Error("persist lb", "err", err)
	}
}

func (s *Server) asyncDeleteLB(id string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	var lb models.LoadBalancer
	if err := db.LoadBalancers().FindOne(ctx, bson.M{"_id": id}).Decode(&lb); err != nil {
		s.Logger.Error("async lb delete load", "err", err, "lb", id)
		return
	}
	if s.LB != nil {
		if err := s.LB.Delete(ctx, &lb); err != nil {
			s.Logger.Error("delete lb", "err", err, "lb", id)
		}
	}
	if _, err := db.LoadBalancers().DeleteOne(ctx, bson.M{"_id": id}); err != nil {
		s.Logger.Error("delete lb doc", "err", err)
	}
}
