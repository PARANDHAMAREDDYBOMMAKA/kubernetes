export interface User {
  id: string
  email: string
  name: string
  createdAt: string
}

export type ClusterStatus =
  | 'Provisioning'
  | 'Running'
  | 'Failed'
  | 'Deleting'
  | 'Deleted'

export interface Cluster {
  id: string
  userId: string
  name: string
  k8sVersion: string
  region: string
  nodeCount: number
  status: ClusterStatus
  apiPort?: number
  message?: string
  createdAt: string
  updatedAt: string
}

export interface Backend {
  host: string
  port: number
}

export type LoadBalancerStatus =
  | 'Provisioning'
  | 'Running'
  | 'Failed'
  | 'Deleting'

export type LoadBalancerAlgorithm =
  | 'round_robin'
  | 'least_conn'
  | 'ip_hash'

export interface LoadBalancer {
  id: string
  userId: string
  clusterId: string
  name: string
  status: LoadBalancerStatus
  port: number
  targetPort: number
  algorithm: LoadBalancerAlgorithm
  backends: Backend[]
  createdAt: string
  updatedAt: string
}

export interface AuthResponse {
  token: string
  user: User
}

export interface CreateClusterPayload {
  name: string
  k8sVersion: string
  nodeCount: number
  region: string
}

export interface CreateLoadBalancerPayload {
  name: string
  clusterId: string
  port: number
  targetPort: number
  algorithm: LoadBalancerAlgorithm
  backends: Backend[]
}
