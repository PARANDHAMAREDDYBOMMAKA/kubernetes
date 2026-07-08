<script setup lang="ts">
defineProps<{
  title?: string
  description?: string
  padded?: boolean
}>()
</script>

<template>
  <section class="card-base">
    <header
      v-if="title || $slots.header || $slots.actions"
      class="flex items-start justify-between gap-4 border-b border-ink-700 px-5 py-4"
    >
      <div>
        <slot name="header">
          <h3 v-if="title" class="text-xs font-semibold uppercase tracking-wider text-slate-200">
            {{ title }}
          </h3>
          <p v-if="description" class="mt-0.5 text-xs text-slate-400">
            {{ description }}
          </p>
        </slot>
      </div>
      <div v-if="$slots.actions" class="flex items-center gap-2">
        <slot name="actions" />
      </div>
    </header>
    <div :class="padded === false ? '' : 'p-5'">
      <slot />
    </div>
  </section>
</template>
