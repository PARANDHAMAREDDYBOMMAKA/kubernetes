<script setup lang="ts">
import { RouterLink } from 'vue-router'
import Button from '@/components/ui/Button.vue'
</script>

<template>
  <div
    class="term-grid flex min-h-screen flex-col items-center justify-center bg-ink-950 px-4 text-center"
  >
    <p class="text-sm text-red-400">
      <span class="text-slate-600">$</span> cat /page → exit 404
    </p>
    <h1 class="mt-3 text-2xl font-semibold text-slate-100">
      page not found<span class="animate-blink text-brand-400">▊</span>
    </h1>
    <p class="mt-2 max-w-sm text-sm text-slate-500">
      the route you requested doesn't exist or has been moved.
    </p>
    <RouterLink to="/clusters" class="mt-6">
      <Button variant="primary">← back to dashboard</Button>
    </RouterLink>
  </div>
</template>
