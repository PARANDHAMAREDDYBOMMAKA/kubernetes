<script setup lang="ts">
import { useToastStore } from '@/stores/toast'
import { CheckCircle2, AlertTriangle, Info, XCircle, X } from 'lucide-vue-next'
import { computed } from 'vue'

const store = useToastStore()
const toasts = computed(() => store.toasts)

function iconFor(kind: string) {
  switch (kind) {
    case 'success':
      return CheckCircle2
    case 'error':
      return XCircle
    case 'warning':
      return AlertTriangle
    default:
      return Info
  }
}

function toneClass(kind: string) {
  switch (kind) {
    case 'success':
      return 'border-emerald-600/40 bg-emerald-500/10 text-emerald-100'
    case 'error':
      return 'border-red-600/40 bg-red-500/10 text-red-100'
    case 'warning':
      return 'border-amber-600/40 bg-amber-500/10 text-amber-100'
    default:
      return 'border-sky-600/40 bg-sky-500/10 text-sky-100'
  }
}

function iconColor(kind: string) {
  switch (kind) {
    case 'success':
      return 'text-emerald-400'
    case 'error':
      return 'text-red-400'
    case 'warning':
      return 'text-amber-400'
    default:
      return 'text-sky-400'
  }
}
</script>

<template>
  <Teleport to="body">
    <div
      class="pointer-events-none fixed inset-x-0 bottom-4 z-[60] flex flex-col items-center gap-2 px-4 sm:inset-x-auto sm:bottom-4 sm:right-4 sm:items-end"
    >
      <transition-group name="toast">
        <div
          v-for="t in toasts"
          :key="t.id"
          class="pointer-events-auto w-full max-w-sm rounded-lg border bg-slate-900/95 px-4 py-3 shadow-lg backdrop-blur animate-fade-in"
          :class="toneClass(t.kind)"
        >
          <div class="flex items-start gap-3">
            <component
              :is="iconFor(t.kind)"
              class="mt-0.5 h-4 w-4 shrink-0"
              :class="iconColor(t.kind)"
            />
            <div class="flex-1 text-sm">
              <p class="font-medium">{{ t.title }}</p>
              <p v-if="t.message" class="mt-0.5 text-xs opacity-80">
                {{ t.message }}
              </p>
            </div>
            <button
              type="button"
              class="rounded p-1 text-slate-400 hover:bg-slate-800 hover:text-slate-200"
              @click="store.dismiss(t.id)"
            >
              <X class="h-3.5 w-3.5" />
            </button>
          </div>
        </div>
      </transition-group>
    </div>
  </Teleport>
</template>

<style scoped>
.toast-enter-active,
.toast-leave-active {
  transition: all 180ms ease;
}
.toast-enter-from {
  opacity: 0;
  transform: translateY(8px);
}
.toast-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
