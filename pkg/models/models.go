package models

import "time"

// Cluster status constants.
const (
	ClusterStatusProvisioning = "Provisioning"
	ClusterStatusRunning      = "Running"
	ClusterStatusFailed       = "Failed"
	ClusterStatusDeleting     = "Deleting"
	ClusterStatusDeleted      = "Deleted"
)

// LoadBalancer status constants.
const (
	LBStatusProvisioning = "Provisioning"
	LBStatusRunning      = "Running"
	LBStatusFailed       = "Failed"
	LBStatusDeleting     = "Deleting"
	LBStatusDeleted      = "Deleted"
)

// Load balancer algorithms.
const (
	AlgoRoundRobin = "round_robin"
	AlgoLeastConn  = "least_conn"
	AlgoIPHash     = "ip_hash"
)

// User represents an authenticated user of the KaaS platform.
type User struct {
	ID           string    `bson:"_id,omitempty" json:"id"`
	Email        string    `bson:"email" json:"email"`
	PasswordHash string    `bson:"password_hash" json:"-"`
	Name         string    `bson:"name" json:"name"`
	CreatedAt    time.Time `bson:"created_at" json:"createdAt"`
}

// Cluster is a provisioned k3s-in-Docker cluster.
type Cluster struct {
	ID                string    `bson:"_id,omitempty" json:"id"`
	UserID            string    `bson:"user_id" json:"userId"`
	Name              string    `bson:"name" json:"name"`
	K8sVersion        string    `bson:"k8s_version" json:"k8sVersion"`
	Region            string    `bson:"region" json:"region"`
	Status            string    `bson:"status" json:"status"`
	NodeCount         int       `bson:"node_count" json:"nodeCount"`
	ServerContainerID string    `bson:"server_container_id" json:"serverContainerId"`
	AgentContainerIDs []string  `bson:"agent_container_ids" json:"agentContainerIds"`
	APIPort           int       `bson:"api_port" json:"apiPort"`
	Kubeconfig        string    `bson:"kubeconfig" json:"-"`
	ClusterToken      string    `bson:"cluster_token" json:"-"`
	CreatedAt         time.Time `bson:"created_at" json:"createdAt"`
	UpdatedAt         time.Time `bson:"updated_at" json:"updatedAt"`
	Message           string    `bson:"message" json:"message"`
}

// Backend is a downstream endpoint behind a LoadBalancer.
type Backend struct {
	Host string `bson:"host" json:"host"`
	Port int    `bson:"port" json:"port"`
}

// LoadBalancer is an nginx-based L4/L7 load balancer.
type LoadBalancer struct {
	ID          string    `bson:"_id,omitempty" json:"id"`
	UserID      string    `bson:"user_id" json:"userId"`
	ClusterID   string    `bson:"cluster_id" json:"clusterId"`
	Name        string    `bson:"name" json:"name"`
	Status      string    `bson:"status" json:"status"`
	Algorithm   string    `bson:"algorithm" json:"algorithm"`
	Port        int       `bson:"port" json:"port"`
	TargetPort  int       `bson:"target_port" json:"targetPort"`
	Backends    []Backend `bson:"backends" json:"backends"`
	ContainerID string    `bson:"container_id" json:"containerId"`
	CreatedAt   time.Time `bson:"created_at" json:"createdAt"`
	UpdatedAt   time.Time `bson:"updated_at" json:"updatedAt"`
}
