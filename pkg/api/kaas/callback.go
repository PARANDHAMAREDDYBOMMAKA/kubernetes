package kaas

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/parandhamareddybommaka/kube/pkg/db"
	"github.com/parandhamareddybommaka/kube/pkg/models"
	"github.com/parandhamareddybommaka/kube/pkg/provisioner"
)

// DBStatusCallback returns a provisioner.StatusCallback that persists cluster
// status updates to MongoDB. Safe to pass nil db (no-op).
func DBStatusCallback() provisioner.StatusCallback {
	return func(ctx context.Context, c *models.Cluster, _ provisioner.StatusUpdate) error {
		if !db.IsConnected() {
			return nil
		}
		_, err := db.Clusters().UpdateOne(ctx, bson.M{"_id": c.ID}, bson.M{"$set": c})
		return err
	}
}
