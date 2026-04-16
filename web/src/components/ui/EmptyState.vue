<script setup lang="ts">
import type { Component } from 'vue'

defineProps<{
  title: string
  description?: string
  icon?: Component
}>()
</script>

<template>
  <div
    class="flex flex-col items-center justify-center rounded-xl border border-dashed border-slate-800 bg-slate-900/40 px-6 py-14 text-center"
  >
    <div
      v-if="icon"
      class="mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-brand-500/10 text-brand-300 ring-1 ring-inset ring-brand-500/20"
    >
      <component :is="icon" class="h-5 w-5" />
    </div>
    <h3 class="text-base font-semibold text-slate-100">{{ title }}</h3>
    <p v-if="description" class="mt-1 max-w-sm text-sm text-slate-400">
      {{ description }}
    </p>
    <div class="mt-5">
      <slot />
    </div>
  </div>
</template>
