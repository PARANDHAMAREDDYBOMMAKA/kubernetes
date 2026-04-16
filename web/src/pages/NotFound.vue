<script setup lang="ts">
import { RouterLink } from 'vue-router'
import Button from '@/components/ui/Button.vue'
</script>

<template>
  <div
    class="flex min-h-screen flex-col items-center justify-center bg-slate-950 px-4 text-center"
  >
    <p class="font-mono text-sm text-brand-300">404</p>
    <h1 class="mt-2 text-3xl font-semibold text-white">Page not found</h1>
    <p class="mt-2 max-w-sm text-sm text-slate-400">
      The page you're looking for doesn't exist or has been moved.
    </p>
    <RouterLink to="/clusters" class="mt-6">
      <Button variant="primary">Back to dashboard</Button>
    </RouterLink>
  </div>
</template>
