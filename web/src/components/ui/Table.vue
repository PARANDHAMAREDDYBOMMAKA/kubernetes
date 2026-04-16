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
  @apply border-b border-slate-800 bg-slate-900/60;
}
:deep(thead th) {
  @apply px-4 py-3 text-xs font-medium uppercase tracking-wider text-slate-400;
}
:deep(tbody tr) {
  @apply border-b border-slate-800/70 last:border-b-0 transition-colors hover:bg-slate-800/40;
}
:deep(tbody td) {
  @apply px-4 py-3 align-middle text-slate-200;
}
</style>
