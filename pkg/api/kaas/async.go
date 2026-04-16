package kaas

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/parandhamareddybommaka/kube/pkg/db"
	"github.com/parandhamareddybommaka/kube/pkg/models"
)

// asyncProvision runs the provisioner in a background goroutine.
func (s *Server) asyncProvision(clusterID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	cluster, err := s.loadClusterByID(ctx, clusterID)
	if err != nil {
		s.Logger.Error("async provision: load cluster", "err", err, "cluster", clusterID)
		return
	}
	if s.Prov == nil {
		s.markClusterFailed(ctx, cluster, "provisioner not configured")
		return
	}
	if err := s.Prov.CreateCluster(ctx, cluster); err != nil {
		s.Logger.Error("provision cluster", "err", err, "cluster", clusterID)
	}
	// Persist final state regardless.
	if _, pErr := db.Clusters().UpdateOne(ctx, bson.M{"_id": cluster.ID}, bson.M{"$set": cluster}); pErr != nil {
		s.Logger.Error("persist cluster after provision", "err", pErr)
	}
}

// asyncDelete tears down the cluster and removes its document.
func (s *Server) asyncDelete(clusterID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cluster, err := s.loadClusterByID(ctx, clusterID)
	if err != nil {
		s.Logger.Error("async delete: load cluster", "err", err, "cluster", clusterID)
		return
	}
	if s.Prov != nil {
		if err := s.Prov.DeleteCluster(ctx, cluster); err != nil {
			s.Logger.Error("delete cluster", "err", err, "cluster", clusterID)
		}
	}
	// Remove the Mongo document after teardown.
	if _, err := db.Clusters().DeleteOne(ctx, bson.M{"_id": clusterID}); err != nil {
		s.Logger.Error("delete cluster doc", "err", err)
	}
}

// asyncScale scales the agent count.
func (s *Server) asyncScale(clusterID string, desired int) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cluster, err := s.loadClusterByID(ctx, clusterID)
	if err != nil {
		s.Logger.Error("async scale: load cluster", "err", err, "cluster", clusterID)
		return
	}
	if s.Prov == nil {
		s.markClusterFailed(ctx, cluster, "provisioner not configured")
		return
	}
	if err := s.Prov.ScaleCluster(ctx, cluster, desired); err != nil {
		s.Logger.Error("scale cluster", "err", err, "cluster", clusterID)
	}
	if _, pErr := db.Clusters().UpdateOne(ctx, bson.M{"_id": cluster.ID}, bson.M{"$set": cluster}); pErr != nil {
		s.Logger.Error("persist cluster after scale", "err", pErr)
	}
}

// loadClusterByID fetches the cluster doc without ownership check. Used by workers.
func (s *Server) loadClusterByID(ctx context.Context, id string) (*models.Cluster, error) {
	if !db.IsConnected() {
		return nil, errors.New("db not connected")
	}
	var c models.Cluster
	if err := db.Clusters().FindOne(ctx, bson.M{"_id": id}).Decode(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Server) markClusterFailed(ctx context.Context, c *models.Cluster, msg string) {
	c.Status = models.ClusterStatusFailed
	c.Message = msg
	c.UpdatedAt = nowUTC()
	if _, err := db.Clusters().UpdateOne(ctx, bson.M{"_id": c.ID}, bson.M{"$set": c}); err != nil {
		s.Logger.Error("mark failed", "err", err)
	}
}

// Ensure mongo sentinel is referenced (avoids unused import if later changes remove usage).
var _ = mongo.ErrNoDocuments
