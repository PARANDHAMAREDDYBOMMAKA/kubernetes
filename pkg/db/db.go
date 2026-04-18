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

	connectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	opts := options.Client().
		ApplyURI(uri).
		SetServerSelectionTimeout(3 * time.Second).
		SetConnectTimeout(3 * time.Second)
	c, err := mongo.Connect(connectCtx, opts)
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}

	pingCtx, pingCancel := context.WithTimeout(ctx, 3*time.Second)
	defer pingCancel()
	if err := c.Ping(pingCtx, nil); err != nil {
		_ = c.Disconnect(context.Background())
		return nil, fmt.Errorf("mongo ping: %w", err)
	}

	client = c
	database = c.Database(dbName)
	return client, nil
}

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

func DB() *mongo.Database {
	mu.RLock()
	defer mu.RUnlock()
	return database
}

func IsConnected() bool {
	mu.RLock()
	defer mu.RUnlock()
	return database != nil
}

func Users() *mongo.Collection {
	mu.RLock()
	defer mu.RUnlock()
	if database == nil {
		return nil
	}
	return database.Collection(CollectionUsers)
}

func Clusters() *mongo.Collection {
	mu.RLock()
	defer mu.RUnlock()
	if database == nil {
		return nil
	}
	return database.Collection(CollectionClusters)
}

func LoadBalancers() *mongo.Collection {
	mu.RLock()
	defer mu.RUnlock()
	if database == nil {
		return nil
	}
	return database.Collection(CollectionLoadBalancers)
}
