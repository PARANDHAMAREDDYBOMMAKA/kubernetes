import axios, { AxiosError, type AxiosInstance } from 'axios'
import { useAuthStore } from '@/stores/auth'
import { useToastStore } from '@/stores/toast'
import router from '@/router'

const baseURL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'

export const api: AxiosInstance = axios.create({
  baseURL,
  timeout: 30000
})

api.interceptors.request.use((config) => {
  const auth = useAuthStore()
  if (auth.token) {
    config.headers.Authorization = `Bearer ${auth.token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => response,
  (error: AxiosError<{ error?: string; message?: string }>) => {
    const status = error.response?.status
    const auth = useAuthStore()
    const toast = useToastStore()

    if (status === 401) {
      // only logout + redirect when we actually had a token
      if (auth.token) {
        auth.logout(false)
        const current = router.currentRoute.value
        if (current.name !== 'login' && current.name !== 'register') {
          router.push({
            name: 'login',
            query: { redirect: current.fullPath }
          })
          toast.error('Session expired. Please log in again.')
        }
      }
    } else if (status && status >= 500) {
      toast.error('Server error. Please try again shortly.')
    }

    return Promise.reject(error)
  }
)

export function extractErrorMessage(err: unknown, fallback = 'Something went wrong'): string {
  if (axios.isAxiosError(err)) {
    const data = err.response?.data as
      | { error?: string; message?: string }
      | undefined
    return data?.error || data?.message || err.message || fallback
  }
  if (err instanceof Error) return err.message
  return fallback
}
