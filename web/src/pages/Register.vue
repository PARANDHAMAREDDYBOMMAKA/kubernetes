<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useToastStore } from '@/stores/toast'
import { extractErrorMessage } from '@/lib/api'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'

const name = ref('')
const email = ref('')
const password = ref('')
const errors = ref<{ name?: string; email?: string; password?: string }>({})
const auth = useAuthStore()
const toast = useToastStore()
const router = useRouter()

async function onSubmit() {
  errors.value = {}
  if (!name.value) errors.value.name = 'Required'
  if (!email.value) errors.value.email = 'Required'
  if (!password.value) errors.value.password = 'Required'
  else if (password.value.length < 8)
    errors.value.password = 'Minimum 8 characters'
  if (Object.keys(errors.value).length) return

  try {
    await auth.register(email.value, password.value, name.value)
    toast.success('Account created. Welcome to KaaS!')
    router.push('/clusters')
  } catch (err) {
    toast.error('Registration failed', extractErrorMessage(err))
  }
}
</script>

<template>
  <div
    class="flex min-h-screen items-center justify-center bg-slate-950 px-4 py-12"
  >
    <div class="w-full max-w-md">
      <div class="mb-8 flex items-center justify-center gap-2 text-white">
        <span
          class="flex h-9 w-9 items-center justify-center rounded-lg bg-gradient-to-br from-brand-500 to-brand-700 shadow-glow"
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
        <span class="text-lg font-semibold">KaaS</span>
      </div>

      <div class="card-base p-8 animate-fade-in">
        <div class="mb-6">
          <h1 class="text-xl font-semibold text-white">Create account</h1>
          <p class="mt-1 text-sm text-slate-400">
            Spin up production-ready Kubernetes clusters in minutes.
          </p>
        </div>

        <form class="space-y-4" @submit.prevent="onSubmit">
          <Input
            id="name"
            v-model="name"
            label="Name"
            placeholder="Jane Doe"
            autocomplete="name"
            :error="errors.name"
            required
          />
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
            placeholder="At least 8 characters"
            autocomplete="new-password"
            :error="errors.password"
            required
          />
          <Button
            type="submit"
            :loading="auth.loading"
            full-width
            size="lg"
          >
            Create account
          </Button>
        </form>

        <p class="mt-6 text-center text-sm text-slate-400">
          Already have an account?
          <RouterLink
            to="/login"
            class="font-medium text-brand-300 hover:text-brand-200"
            >Sign in</RouterLink
          >
        </p>
      </div>
    </div>
  </div>
</template>
