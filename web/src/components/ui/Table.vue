<script setup lang="ts">
defineProps<{ dense?: boolean }>()
</script>

<template>
  <div class="overflow-x-auto">
    <table class="w-full text-left text-sm">
      <slot />
    </table>
  </div>
</template>

<style scoped>
:deep(thead tr) {
  @apply border-b border-ink-700 bg-ink-850/60;
}
:deep(thead th) {
  @apply px-4 py-2.5 text-[11px] font-medium uppercase tracking-wider text-brand-400/70;
}
:deep(tbody tr) {
  @apply border-b border-ink-700/60 last:border-b-0 transition-colors hover:bg-brand-500/[0.04];
}
:deep(tbody td) {
  @apply px-4 py-3 align-middle text-slate-200;
}
</style>
