package types

import (
	"sync"
	"time"
)

type ObjectMeta struct {
	UID         string            `json:"uid"`
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Created     time.Time         `json:"created"`
	Updated     time.Time         `json:"updated"`
}

type PodPhase string

const (
	PodPending   PodPhase = "Pending"
	PodRunning   PodPhase = "Running"
	PodSucceeded PodPhase = "Succeeded"
	PodFailed    PodPhase = "Failed"
	PodUnknown   PodPhase = "Unknown"
)

type ContainerState string

const (
	ContainerCreated    ContainerState = "Created"
	ContainerRunning    ContainerState = "Running"
	ContainerTerminated ContainerState = "Terminated"
)

type ResourceRequirements struct {
	CPUCores   float64 `json:"cpuCores"`
	MemoryMB   int64   `json:"memoryMB"`
	StorageMB  int64   `json:"storageMB,omitempty"`
	GPUCount   int     `json:"gpuCount,omitempty"`
}

type ContainerPort struct {
	Name          string `json:"name,omitempty"`
	ContainerPort int    `json:"containerPort"`
	HostPort      int    `json:"hostPort,omitempty"`
	Protocol      string `json:"protocol,omitempty"`
}

type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type VolumeMount struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	ReadOnly  bool   `json:"readOnly,omitempty"`
}

type Container struct {
	Name         string               `json:"name"`
	Image        string               `json:"image"`
	Command      []string             `json:"command,omitempty"`
	Args         []string             `json:"args,omitempty"`
	Env          []EnvVar             `json:"env,omitempty"`
	Ports        []ContainerPort      `json:"ports,omitempty"`
	VolumeMounts []VolumeMount        `json:"volumeMounts,omitempty"`
	Resources    ResourceRequirements `json:"resources,omitempty"`
	WorkingDir   string               `json:"workingDir,omitempty"`
}

type ContainerStatus struct {
	Name        string         `json:"name"`
	ContainerID string         `json:"containerID"`
	State       ContainerState `json:"state"`
	Ready       bool           `json:"ready"`
	RestartCnt  int            `json:"restartCount"`
	StartedAt   time.Time      `json:"startedAt,omitempty"`
	FinishedAt  time.Time      `json:"finishedAt,omitempty"`
	ExitCode    int            `json:"exitCode,omitempty"`
}

type VolumeSource struct {
	HostPath  string `json:"hostPath,omitempty"`
	EmptyDir  bool   `json:"emptyDir,omitempty"`
	ConfigMap string `json:"configMap,omitempty"`
	Secret    string `json:"secret,omitempty"`
}

type Volume struct {
	Name   string       `json:"name"`
	Source VolumeSource `json:"source"`
}

type RestartPolicy string

const (
	RestartAlways    RestartPolicy = "Always"
	RestartOnFailure RestartPolicy = "OnFailure"
	RestartNever     RestartPolicy = "Never"
)

type PodSpec struct {
	Containers     []Container   `json:"containers"`
	InitContainers []Container   `json:"initContainers,omitempty"`
	Volumes        []Volume      `json:"volumes,omitempty"`
	NodeSelector   map[string]string `json:"nodeSelector,omitempty"`
	NodeName       string        `json:"nodeName,omitempty"`
	RestartPolicy  RestartPolicy `json:"restartPolicy,omitempty"`
	HostNetwork    bool          `json:"hostNetwork,omitempty"`
	DNSPolicy      string        `json:"dnsPolicy,omitempty"`
}

type PodStatus struct {
	Phase             PodPhase          `json:"phase"`
	HostIP            string            `json:"hostIP,omitempty"`
	PodIP             string            `json:"podIP,omitempty"`
	ContainerStatuses []ContainerStatus `json:"containerStatuses,omitempty"`
	StartTime         time.Time         `json:"startTime,omitempty"`
	Message           string            `json:"message,omitempty"`
	Reason            string            `json:"reason,omitempty"`
}

type Pod struct {
	ObjectMeta `json:"metadata"`
	Spec       PodSpec   `json:"spec"`
	Status     PodStatus `json:"status,omitempty"`
}

type NodeConditionType string

const (
	NodeReady              NodeConditionType = "Ready"
	NodeMemoryPressure     NodeConditionType = "MemoryPressure"
	NodeDiskPressure       NodeConditionType = "DiskPressure"
	NodeNetworkUnavailable NodeConditionType = "NetworkUnavailable"
)

type NodeCondition struct {
	Type    NodeConditionType `json:"type"`
	Status  bool              `json:"status"`
	Updated time.Time         `json:"updated"`
	Message string            `json:"message,omitempty"`
}

type NodeResources struct {
	CPUCores  float64 `json:"cpuCores"`
	MemoryMB  int64   `json:"memoryMB"`
	StorageMB int64   `json:"storageMB"`
	PodCount  int     `json:"podCount"`
}

type NodeSpec struct {
	PodCIDR       string `json:"podCIDR,omitempty"`
	Unschedulable bool   `json:"unschedulable,omitempty"`
}

type NodeStatus struct {
	Conditions  []NodeCondition `json:"conditions,omitempty"`
	Capacity    NodeResources   `json:"capacity"`
	Allocatable NodeResources   `json:"allocatable"`
	Addresses   []NodeAddress   `json:"addresses,omitempty"`
}

type NodeAddress struct {
	Type    string `json:"type"`
	Address string `json:"address"`
}

type Node struct {
	ObjectMeta `json:"metadata"`
	Spec       NodeSpec   `json:"spec"`
	Status     NodeStatus `json:"status,omitempty"`
}

type ServiceType string

const (
	ServiceTypeClusterIP ServiceType = "ClusterIP"
	ServiceTypeNodePort  ServiceType = "NodePort"
)

type ServicePort struct {
	Name       string `json:"name,omitempty"`
	Protocol   string `json:"protocol,omitempty"`
	Port       int    `json:"port"`
	TargetPort int    `json:"targetPort"`
	NodePort   int    `json:"nodePort,omitempty"`
}

type ServiceSpec struct {
	Type       ServiceType       `json:"type,omitempty"`
	ClusterIP  string            `json:"clusterIP,omitempty"`
	Ports      []ServicePort     `json:"ports"`
	Selector   map[string]string `json:"selector"`
	ExternalIP []string          `json:"externalIPs,omitempty"`
}

type ServiceStatus struct {
	LoadBalancer string `json:"loadBalancer,omitempty"`
}

type Service struct {
	ObjectMeta `json:"metadata"`
	Spec       ServiceSpec   `json:"spec"`
	Status     ServiceStatus `json:"status,omitempty"`
}

type DeploymentSpec struct {
	Replicas int               `json:"replicas"`
	Selector map[string]string `json:"selector"`
	Template PodSpec           `json:"template"`
	Strategy DeploymentStrategy `json:"strategy,omitempty"`
}

type DeploymentStrategyType string

const (
	RollingUpdateStrategy DeploymentStrategyType = "RollingUpdate"
	RecreateStrategy      DeploymentStrategyType = "Recreate"
)

type DeploymentStrategy struct {
	Type           DeploymentStrategyType `json:"type,omitempty"`
	MaxUnavailable int                    `json:"maxUnavailable,omitempty"`
	MaxSurge       int                    `json:"maxSurge,omitempty"`
}

type DeploymentStatus struct {
	Replicas          int `json:"replicas"`
	ReadyReplicas     int `json:"readyReplicas"`
	AvailableReplicas int `json:"availableReplicas"`
	UpdatedReplicas   int `json:"updatedReplicas"`
}

type Deployment struct {
	ObjectMeta `json:"metadata"`
	Spec       DeploymentSpec   `json:"spec"`
	Status     DeploymentStatus `json:"status,omitempty"`
}

type ConfigMap struct {
	ObjectMeta `json:"metadata"`
	Data       map[string]string `json:"data,omitempty"`
	BinaryData map[string][]byte `json:"binaryData,omitempty"`
}

type Secret struct {
	ObjectMeta `json:"metadata"`
	Type       string            `json:"type,omitempty"`
	Data       map[string][]byte `json:"data,omitempty"`
	StringData map[string]string `json:"stringData,omitempty"`
}

type Namespace struct {
	ObjectMeta `json:"metadata"`
	Status     NamespaceStatus `json:"status,omitempty"`
}

type NamespacePhase string

const (
	NamespaceActive      NamespacePhase = "Active"
	NamespaceTerminating NamespacePhase = "Terminating"
)

type NamespaceStatus struct {
	Phase NamespacePhase `json:"phase,omitempty"`
}

type Event struct {
	ObjectMeta     `json:"metadata"`
	InvolvedObject ObjectRef `json:"involvedObject"`
	Reason         string    `json:"reason"`
	Message        string    `json:"message"`
	Type           string    `json:"type"`
	Count          int       `json:"count"`
	FirstTimestamp time.Time `json:"firstTimestamp"`
	LastTimestamp  time.Time `json:"lastTimestamp"`
}

type ObjectRef struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	UID       string `json:"uid"`
}

type WatchEvent struct {
	Type   string      `json:"type"`
	Object interface{} `json:"object"`
}

type ObjectPool[T any] struct {
	pool sync.Pool
}

func NewObjectPool[T any](newFunc func() T) *ObjectPool[T] {
	return &ObjectPool[T]{
		pool: sync.Pool{
			New: func() interface{} {
				return newFunc()
			},
		},
	}
}

func (p *ObjectPool[T]) Get() T {
	return p.pool.Get().(T)
}

func (p *ObjectPool[T]) Put(obj T) {
	p.pool.Put(obj)
}
