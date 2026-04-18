import { defineStore } from 'pinia'
import { api } from '@/lib/api'
import { downloadTextFile } from '@/lib/utils'
import type { Cluster, ClusterNode, CreateClusterPayload } from '@/types'

interface ClustersState {
  list: Cluster[]
  current: Cluster | null
  loadingList: boolean
  loadingCurrent: boolean
}

export const useClustersStore = defineStore('clusters', {
  state: (): ClustersState => ({
    list: [],
    current: null,
    loadingList: false,
    loadingCurrent: false
  }),
  getters: {
    hasPendingOps(state): boolean {
      return state.list.some(
        (c) => c.status === 'Provisioning' || c.status === 'Deleting'
      )
    }
  },
  actions: {
    async fetchAll(silent = false) {
      if (!silent) this.loadingList = true
      try {
        const { data } = await api.get<Cluster[]>('/api/v1/clusters')
        this.list = Array.isArray(data) ? data : []
      } finally {
        if (!silent) this.loadingList = false
      }
    },
    async fetchOne(id: string, silent = false) {
      if (!silent) this.loadingCurrent = true
      try {
        const { data } = await api.get<Cluster>(`/api/v1/clusters/${id}`)
        this.current = data
        if (!Array.isArray(this.list)) this.list = []
        const idx = this.list.findIndex((c) => c.id === id)
        if (idx >= 0) this.list[idx] = data
        return data
      } finally {
        if (!silent) this.loadingCurrent = false
      }
    },
    async create(payload: CreateClusterPayload) {
      const { data } = await api.post<Cluster>('/api/v1/clusters', payload)
      const current = Array.isArray(this.list) ? this.list : []
      this.list = data ? [data, ...current] : current
      return data
    },
    async remove(id: string) {
      await api.delete(`/api/v1/clusters/${id}`)
      if (!Array.isArray(this.list)) this.list = []
      const idx = this.list.findIndex((c) => c.id === id)
      if (idx >= 0) {
        this.list[idx] = { ...this.list[idx], status: 'Deleting' }
      }
      if (this.current?.id === id) {
        this.current = { ...this.current, status: 'Deleting' }
      }
    },
    async scale(id: string, nodeCount: number) {
      const { data } = await api.post<Cluster>(
        `/api/v1/clusters/${id}/scale`,
        { nodeCount }
      )
      if (!Array.isArray(this.list)) this.list = []
      const idx = this.list.findIndex((c) => c.id === id)
      if (idx >= 0) this.list[idx] = data
      if (this.current?.id === id) this.current = data
      return data
    },
    async fetchKubeconfig(id: string): Promise<string> {
      const { data } = await api.get<string>(
        `/api/v1/clusters/${id}/kubeconfig`,
        { responseType: 'text', transformResponse: [(d) => d] }
      )
      return typeof data === 'string' ? data : String(data)
    },
    async downloadKubeconfig(cluster: Cluster) {
      const text = await this.fetchKubeconfig(cluster.id)
      downloadTextFile(text, `${cluster.name}-kubeconfig.yaml`)
      return text
    },
    async fetchYaml(id: string): Promise<string> {
      const { data } = await api.get<string>(`/api/v1/clusters/${id}/yaml`, {
        responseType: 'text',
        transformResponse: [(d) => d]
      })
      return typeof data === 'string' ? data : String(data)
    },
    async downloadYaml(cluster: Cluster) {
      const text = await this.fetchYaml(cluster.id)
      downloadTextFile(text, `${cluster.name}.yaml`)
      return text
    },
    async fetchNodes(id: string): Promise<ClusterNode[]> {
      const { data } = await api.get<ClusterNode[]>(`/api/v1/clusters/${id}/nodes`)
      return Array.isArray(data) ? data : []
    }
  }
})
