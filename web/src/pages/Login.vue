<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useToastStore } from '@/stores/toast'
import { extractErrorMessage } from '@/lib/api'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'

const email = ref('')
const password = ref('')
const errors = ref<{ email?: string; password?: string }>({})
const auth = useAuthStore()
const toast = useToastStore()
const router = useRouter()
const route = useRoute()

async function onSubmit() {
  errors.value = {}
  if (!email.value) errors.value.email = 'Required'
  if (!password.value) errors.value.password = 'Required'
  if (Object.keys(errors.value).length) return

  try {
    await auth.login(email.value, password.value)
    toast.success(`Welcome back, ${auth.user?.name || auth.user?.email || ''}`)
    const redirect = (route.query.redirect as string) || '/clusters'
    router.push(redirect)
  } catch (err) {
    toast.error('Login failed', extractErrorMessage(err))
  }
}
</script>

<template>
  <div
    class="term-grid flex min-h-screen items-center justify-center bg-ink-950 px-4 py-12"
  >
    <div class="w-full max-w-md">
      <div class="mb-6 flex items-center justify-center gap-2.5 text-slate-100">
        <span
          class="flex h-9 w-9 items-center justify-center rounded-md border border-brand-500/40 bg-brand-500/10 text-brand-400 shadow-glow"
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
        <span class="text-lg font-semibold"
          ><span class="text-brand-400">$</span> kaas</span
        >
      </div>

      <div class="card-base animate-fade-in overflow-hidden">
        <!-- terminal title bar -->
        <div
          class="flex items-center gap-2 border-b border-ink-700 bg-ink-850/60 px-4 py-2.5"
        >
          <span class="h-2.5 w-2.5 rounded-full bg-red-500/70" />
          <span class="h-2.5 w-2.5 rounded-full bg-amber-500/70" />
          <span class="h-2.5 w-2.5 rounded-full bg-brand-500/70" />
          <span class="ml-2 text-xs text-slate-500">login.sh</span>
        </div>

        <div class="p-8">
          <div class="mb-6">
            <p class="text-xs text-slate-500">
              <span class="text-brand-400">➜</span> authenticate to continue
            </p>
            <h1 class="mt-1 text-lg font-semibold text-slate-100">
              sign in<span class="animate-blink text-brand-400">▊</span>
            </h1>
          </div>

          <form class="space-y-4" @submit.prevent="onSubmit">
          <Input
            id="email"
            v-model="email"
            label="Email"
            type="email"
            placeholder="you@example.com"
            autocomplete="email"
            :error="errors.email"
            required
          />
          <Input
            id="password"
            v-model="password"
            label="Password"
            type="password"
            placeholder="••••••••"
            autocomplete="current-password"
            :error="errors.password"
            required
          />
          <Button
            type="submit"
            :loading="auth.loading"
            full-width
            size="lg"
          >
            Sign in
          </Button>
        </form>

          <p class="mt-6 text-center text-sm text-slate-500">
            no account?
            <RouterLink
              to="/register"
              class="font-medium text-brand-400 hover:text-brand-300"
              >create one →</RouterLink
            >
          </p>
        </div>
      </div>
    </div>
  </div>
</template>
