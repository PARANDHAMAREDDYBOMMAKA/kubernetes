<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import { LogOut, UserCircle } from 'lucide-vue-next'
import { formatDate } from '@/lib/utils'

const auth = useAuthStore()
const router = useRouter()

function logout() {
  auth.logout(false)
  router.push({ name: 'login' })
}
</script>

<template>
  <div class="space-y-6">
    <header>
      <h1 class="text-xl font-semibold text-slate-100">
        <span class="text-brand-400">#</span> settings
      </h1>
      <p class="mt-1 text-sm text-slate-500">manage your account and session</p>
    </header>

    <Card title="Account">
      <div v-if="auth.user" class="flex items-start gap-4">
        <div
          class="flex h-12 w-12 items-center justify-center rounded-md bg-brand-500/10 text-brand-300 ring-1 ring-inset ring-brand-500/20"
        >
          <UserCircle class="h-7 w-7" />
        </div>
        <dl class="grid flex-1 grid-cols-1 gap-y-4 sm:grid-cols-2">
          <div>
            <dt
              class="text-xs font-medium uppercase tracking-wider text-slate-500"
            >
              Name
            </dt>
            <dd class="mt-1 text-sm text-slate-100">
              {{ auth.user.name || '—' }}
            </dd>
          </div>
          <div>
            <dt
              class="text-xs font-medium uppercase tracking-wider text-slate-500"
            >
              Email
            </dt>
            <dd class="mt-1 text-sm text-slate-100">{{ auth.user.email }}</dd>
          </div>
          <div>
            <dt
              class="text-xs font-medium uppercase tracking-wider text-slate-500"
            >
              User ID
            </dt>
            <dd class="mt-1 font-mono text-xs text-slate-300">
              {{ auth.user.id }}
            </dd>
          </div>
          <div>
            <dt
              class="text-xs font-medium uppercase tracking-wider text-slate-500"
            >
              Member since
            </dt>
            <dd class="mt-1 text-sm text-slate-100">
              {{ formatDate(auth.user.createdAt) }}
            </dd>
          </div>
        </dl>
      </div>
      <div v-else class="text-sm text-slate-400">Not signed in.</div>
    </Card>

    <Card title="Session">
      <p class="mb-4 text-sm text-slate-400">
        Sign out of your KaaS account on this device.
      </p>
      <Button variant="danger" @click="logout">
        <LogOut class="h-4 w-4" />
        Sign out
      </Button>
    </Card>
  </div>
</template>
