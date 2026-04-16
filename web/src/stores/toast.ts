import { defineStore } from 'pinia'

export type ToastKind = 'success' | 'error' | 'info' | 'warning'

export interface Toast {
  id: number
  kind: ToastKind
  title: string
  message?: string
  timeout: number
}

let nextId = 1

export const useToastStore = defineStore('toast', {
  state: () => ({
    toasts: [] as Toast[]
  }),
  actions: {
    push(toast: Omit<Toast, 'id'>): number {
      const id = nextId++
      const t: Toast = { id, ...toast }
      this.toasts.push(t)
      if (toast.timeout > 0) {
        window.setTimeout(() => this.dismiss(id), toast.timeout)
      }
      return id
    },
    dismiss(id: number) {
      this.toasts = this.toasts.filter((t) => t.id !== id)
    },
    success(title: string, message?: string) {
      return this.push({ kind: 'success', title, message, timeout: 3500 })
    },
    error(title: string, message?: string) {
      return this.push({ kind: 'error', title, message, timeout: 5000 })
    },
    info(title: string, message?: string) {
      return this.push({ kind: 'info', title, message, timeout: 3500 })
    },
    warning(title: string, message?: string) {
      return this.push({ kind: 'warning', title, message, timeout: 4000 })
    }
  }
})
