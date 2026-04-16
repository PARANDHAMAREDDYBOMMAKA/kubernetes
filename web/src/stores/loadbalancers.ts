import { defineStore } from 'pinia'
import { api } from '@/lib/api'
import { downloadTextFile } from '@/lib/utils'
import type { CreateLoadBalancerPayload, LoadBalancer } from '@/types'

interface LBState {
  list: LoadBalancer[]
  current: LoadBalancer | null
  loadingList: boolean
  loadingCurrent: boolean
}

export const useLoadBalancersStore = defineStore('loadbalancers', {
  state: (): LBState => ({
    list: [],
    current: null,
    loadingList: false,
    loadingCurrent: false
  }),
  getters: {
    hasPendingOps(state): boolean {
      return state.list.some(
        (l) => l.status === 'Provisioning' || l.status === 'Deleting'
      )
    }
  },
  actions: {
    async fetchAll(silent = false) {
      if (!silent) this.loadingList = true
      try {
        const { data } = await api.get<LoadBalancer[]>('/api/v1/loadbalancers')
        this.list = data ?? []
      } finally {
        if (!silent) this.loadingList = false
      }
    },
    async fetchOne(id: string, silent = false) {
      if (!silent) this.loadingCurrent = true
      try {
        const { data } = await api.get<LoadBalancer>(
          `/api/v1/loadbalancers/${id}`
        )
        this.current = data
        const idx = this.list.findIndex((l) => l.id === id)
        if (idx >= 0) this.list[idx] = data
        return data
      } finally {
        if (!silent) this.loadingCurrent = false
      }
    },
    async create(payload: CreateLoadBalancerPayload) {
      const { data } = await api.post<LoadBalancer>(
        '/api/v1/loadbalancers',
        payload
      )
      this.list = [data, ...this.list]
      return data
    },
    async remove(id: string) {
      await api.delete(`/api/v1/loadbalancers/${id}`)
      const idx = this.list.findIndex((l) => l.id === id)
      if (idx >= 0) {
        this.list[idx] = { ...this.list[idx], status: 'Deleting' }
      }
      if (this.current?.id === id) {
        this.current = { ...this.current, status: 'Deleting' }
      }
    },
    async fetchYaml(id: string): Promise<string> {
      const { data } = await api.get<string>(
        `/api/v1/loadbalancers/${id}/yaml`,
        { responseType: 'text', transformResponse: [(d) => d] }
      )
      return typeof data === 'string' ? data : String(data)
    },
    async downloadYaml(lb: LoadBalancer) {
      const text = await this.fetchYaml(lb.id)
      downloadTextFile(text, `${lb.name}.yaml`)
      return text
    }
  }
})
