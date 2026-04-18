package kaas

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/parandhamareddybommaka/kube/pkg/db"
	"github.com/parandhamareddybommaka/kube/pkg/models"
	"github.com/parandhamareddybommaka/kube/pkg/provisioner"
)

func DBStatusCallback() provisioner.StatusCallback {
	return func(ctx context.Context, c *models.Cluster, _ provisioner.StatusUpdate) error {
		if !db.IsConnected() {
			return nil
		}
		_, err := db.Clusters().UpdateOne(ctx, bson.M{"_id": c.ID}, bson.M{"$set": c})
		return err
	}
}
