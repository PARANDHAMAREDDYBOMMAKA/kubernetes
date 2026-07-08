<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router'
import {
  LayoutGrid,
  Network,
  Settings,
  LogOut,
  Menu,
  X,
  Sun,
  Moon
} from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const sidebarOpen = ref(false)

const nav = [
  { name: 'Clusters', to: '/clusters', icon: LayoutGrid, match: 'clusters' },
  {
    name: 'Load Balancers',
    to: '/loadbalancers',
    icon: Network,
    match: 'loadbalancers'
  },
  { name: 'Settings', to: '/settings', icon: Settings, match: 'settings' }
]

const breadcrumbs = computed(() => {
  const parts: { label: string; to?: string }[] = []
  const name = (route.name as string) || ''
  if (name === 'clusters') parts.push({ label: 'Clusters' })
  else if (name === 'cluster-detail')
    parts.push(
      { label: 'Clusters', to: '/clusters' },
      { label: (route.params.id as string) || 'Detail' }
    )
  else if (name === 'loadbalancers') parts.push({ label: 'Load Balancers' })
  else if (name === 'loadbalancer-detail')
    parts.push(
      { label: 'Load Balancers', to: '/loadbalancers' },
      { label: (route.params.id as string) || 'Detail' }
    )
  else if (name === 'settings') parts.push({ label: 'Settings' })
  return parts
})

function isActive(match: string) {
  return (route.name as string)?.startsWith(match)
}

function logout() {
  auth.logout(false)
  router.push({ name: 'login' })
}

const isDark = ref(document.documentElement.classList.contains('dark'))
function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
}
</script>

<template>
  <div class="term-grid relative flex min-h-screen bg-ink-950">
    <!-- mobile overlay -->
    <div
      v-if="sidebarOpen"
      class="fixed inset-0 z-30 bg-ink-950/70 backdrop-blur-sm md:hidden"
      @click="sidebarOpen = false"
    />

    <!-- Sidebar -->
    <aside
      class="fixed inset-y-0 left-0 z-40 flex w-64 flex-col border-r border-ink-700 bg-ink-900/90 backdrop-blur transition-transform md:translate-x-0"
      :class="sidebarOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0'"
    >
      <div class="flex h-16 items-center justify-between border-b border-ink-700 px-5">
        <RouterLink
          to="/clusters"
          class="group flex items-center gap-2.5 font-semibold text-slate-100"
        >
          <span
            class="flex h-8 w-8 items-center justify-center rounded-md border border-brand-500/40 bg-brand-500/10 text-brand-400 shadow-glow"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 32 32"
              class="h-5 w-5"
              fill="none"
              stroke="currentColor"
              stroke-width="1.8"
              stroke-linejoin="round"
            >
              <path d="M16 5l9 5v7c0 5.5-3.8 9.5-9 10-5.2-.5-9-4.5-9-10V10l9-5z" />
              <circle cx="16" cy="16" r="2.2" fill="currentColor" stroke="none" />
            </svg>
          </span>
          <span class="flex flex-col leading-none">
            <span class="text-sm tracking-tight"
              ><span class="text-brand-400">$</span> kaas</span
            >
            <span class="mt-0.5 text-[10px] font-normal text-slate-500">v1 · k3s</span>
          </span>
        </RouterLink>
        <button
          type="button"
          class="rounded-sm p-1 text-slate-500 hover:bg-ink-800 hover:text-brand-300 md:hidden"
          @click="sidebarOpen = false"
        >
          <X class="h-4 w-4" />
        </button>
      </div>

      <div class="px-4 pb-1 pt-4 text-[10px] uppercase tracking-widest text-slate-600">
        navigation
      </div>
      <nav class="flex-1 space-y-0.5 px-3 py-1">
        <RouterLink
          v-for="item in nav"
          :key="item.name"
          :to="item.to"
          class="group relative flex items-center gap-3 rounded-md px-3 py-2 text-sm transition"
          :class="
            isActive(item.match)
              ? 'bg-brand-500/10 text-brand-300'
              : 'text-slate-400 hover:bg-ink-800 hover:text-slate-100'
          "
          @click="sidebarOpen = false"
        >
          <span
            v-if="isActive(item.match)"
            class="absolute inset-y-1.5 left-0 w-0.5 rounded-full bg-brand-400"
          />
          <component :is="item.icon" class="h-4 w-4" />
          <span>{{ item.name }}</span>
        </RouterLink>
      </nav>

      <div class="border-t border-ink-700 p-3">
        <div
          v-if="auth.user"
          class="mb-2 rounded-md border border-ink-700 bg-ink-850/60 px-3 py-2"
        >
          <p class="flex items-center gap-1.5 truncate text-sm font-medium text-slate-100">
            <span class="h-1.5 w-1.5 shrink-0 rounded-full bg-brand-400" />
            {{ auth.user.name || auth.user.email }}
          </p>
          <p class="mt-0.5 truncate text-xs text-slate-500">{{ auth.user.email }}</p>
        </div>
        <button
          type="button"
          class="flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm text-slate-400 transition hover:bg-red-500/10 hover:text-red-300"
          @click="logout"
        >
          <LogOut class="h-4 w-4" />
          logout
        </button>
      </div>
    </aside>

    <!-- Main -->
    <div class="flex min-h-screen w-full flex-col md:pl-64">
      <!-- Topbar -->
      <header
        class="sticky top-0 z-20 flex h-16 items-center gap-3 border-b border-ink-700 bg-ink-950/85 px-4 backdrop-blur sm:px-6"
      >
        <button
          type="button"
          class="rounded-sm p-1.5 text-slate-400 hover:bg-ink-800 hover:text-brand-300 md:hidden"
          @click="sidebarOpen = true"
        >
          <Menu class="h-5 w-5" />
        </button>
        <nav class="flex items-center gap-1.5 text-sm text-slate-500">
          <span class="text-brand-400/80">kaas</span>
          <span class="text-slate-600">:</span>
          <template v-for="(bc, i) in breadcrumbs" :key="i">
            <span v-if="i > 0" class="text-slate-600">/</span>
            <RouterLink
              v-if="bc.to"
              :to="bc.to"
              class="hover:text-brand-300"
              >{{ bc.label }}</RouterLink
            >
            <span v-else class="text-slate-200">{{ bc.label }}</span>
          </template>
          <span class="animate-blink text-brand-400">▊</span>
        </nav>
        <div class="ml-auto flex items-center gap-3">
          <span
            class="hidden items-center gap-1.5 text-xs text-slate-500 sm:flex"
            title="API connected"
          >
            <span class="h-1.5 w-1.5 rounded-full bg-brand-400" />
            online
          </span>
          <button
            type="button"
            class="rounded-md p-2 text-slate-400 hover:bg-ink-800 hover:text-brand-300"
            :title="isDark ? 'Switch to light' : 'Switch to dark'"
            @click="toggleTheme"
          >
            <component :is="isDark ? Sun : Moon" class="h-4 w-4" />
          </button>
        </div>
      </header>

      <main class="flex-1 px-4 py-6 sm:px-6 lg:px-8">
        <RouterView />
      </main>
    </div>
  </div>
</template>
