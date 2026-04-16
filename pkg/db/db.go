// Package db wraps the MongoDB client used for KaaS persistence.
package db

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	CollectionUsers         = "users"
	CollectionClusters      = "clusters"
	CollectionLoadBalancers = "loadbalancers"
)

var (
	mu       sync.RWMutex
	client   *mongo.Client
	database *mongo.Database
)

// Connect establishes a Mongo connection using MONGO_URI and MONGO_DB env vars.
// If already connected, returns the existing client.
func Connect(ctx context.Context) (*mongo.Client, error) {
	mu.Lock()
	defer mu.Unlock()

	if client != nil {
		return client, nil
	}

	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	dbName := os.Getenv("MONGO_DB")
	if dbName == "" {
		dbName = "kaas"
	}

	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	c, err := mongo.Connect(connectCtx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}

	pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pingCancel()
	if err := c.Ping(pingCtx, nil); err != nil {
		_ = c.Disconnect(context.Background())
		return nil, fmt.Errorf("mongo ping: %w", err)
	}

	client = c
	database = c.Database(dbName)
	return client, nil
}

// Disconnect closes the underlying Mongo client, if any.
func Disconnect(ctx context.Context) error {
	mu.Lock()
	defer mu.Unlock()
	if client == nil {
		return nil
	}
	err := client.Disconnect(ctx)
	client = nil
	database = nil
	return err
}

// DB returns the active *mongo.Database. Nil if not connected.
func DB() *mongo.Database {
	mu.RLock()
	defer mu.RUnlock()
	return database
}

// IsConnected returns true once Connect has succeeded.
func IsConnected() bool {
	mu.RLock()
	defer mu.RUnlock()
	return database != nil
}

// Users returns the users collection.
func Users() *mongo.Collection {
	mu.RLock()
	defer mu.RUnlock()
	if database == nil {
		return nil
	}
	return database.Collection(CollectionUsers)
}

// Clusters returns the clusters collection.
func Clusters() *mongo.Collection {
	mu.RLock()
	defer mu.RUnlock()
	if database == nil {
		return nil
	}
	return database.Collection(CollectionClusters)
}

// LoadBalancers returns the loadbalancers collection.
func LoadBalancers() *mongo.Collection {
	mu.RLock()
	defer mu.RUnlock()
	if database == nil {
		return nil
	}
	return database.Collection(CollectionLoadBalancers)
}
