import { defineStore } from 'pinia'
import { api } from '@/lib/api'
import type { AuthResponse, User } from '@/types'

const TOKEN_KEY = 'kaas_token'
const USER_KEY = 'kaas_user'

interface AuthState {
  token: string | null
  user: User | null
  loading: boolean
  initialized: boolean
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    token: localStorage.getItem(TOKEN_KEY),
    user: (() => {
      const raw = localStorage.getItem(USER_KEY)
      if (!raw) return null
      try {
        return JSON.parse(raw) as User
      } catch {
        return null
      }
    })(),
    loading: false,
    initialized: false
  }),
  getters: {
    isAuthenticated: (s) => !!s.token
  },
  actions: {
    persist() {
      if (this.token) localStorage.setItem(TOKEN_KEY, this.token)
      else localStorage.removeItem(TOKEN_KEY)
      if (this.user) localStorage.setItem(USER_KEY, JSON.stringify(this.user))
      else localStorage.removeItem(USER_KEY)
    },
    async login(email: string, password: string) {
      this.loading = true
      try {
        const { data } = await api.post<AuthResponse>('/api/v1/auth/login', {
          email,
          password
        })
        this.token = data.token
        this.user = data.user
        this.persist()
      } finally {
        this.loading = false
      }
    },
    async register(email: string, password: string, name: string) {
      this.loading = true
      try {
        const { data } = await api.post<AuthResponse>(
          '/api/v1/auth/register',
          { email, password, name }
        )
        this.token = data.token
        this.user = data.user
        this.persist()
      } finally {
        this.loading = false
      }
    },
    async fetchMe() {
      if (!this.token) return
      try {
        const { data } = await api.get<{ user: User }>('/api/v1/auth/me')
        this.user = data.user
        this.persist()
      } catch {
        // interceptor handles 401
      } finally {
        this.initialized = true
      }
    },
    logout(redirect = true) {
      this.token = null
      this.user = null
      this.persist()
      if (redirect) {
        // avoid circular import; navigate via location if router already past init
        import('@/router').then(({ default: router }) => {
          router.push({ name: 'login' })
        })
      }
    }
  }
})
