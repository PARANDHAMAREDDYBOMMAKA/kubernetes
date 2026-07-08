<script setup lang="ts">
import { ref } from 'vue'
import { Copy, Check } from 'lucide-vue-next'
import { copyToClipboard } from '@/lib/utils'
import { useToastStore } from '@/stores/toast'

defineProps<{ code: string; language?: string; maxHeight?: string }>()
const copied = ref(false)
const toast = useToastStore()

async function doCopy(code: string) {
  try {
    await copyToClipboard(code)
    copied.value = true
    toast.success('Copied to clipboard')
    setTimeout(() => (copied.value = false), 1500)
  } catch {
    toast.error('Copy failed')
  }
}
</script>

<template>
  <div class="relative rounded-md border border-ink-700 bg-ink-950/80">
    <div
      class="flex items-center justify-between border-b border-ink-700 px-3 py-2 text-xs text-slate-400"
    >
      <span class="uppercase tracking-wider text-brand-400/80">◆ {{ language || 'yaml' }}</span>
      <button
        type="button"
        class="inline-flex items-center gap-1.5 rounded-sm px-2 py-1 hover:bg-ink-800 hover:text-brand-300"
        @click="doCopy(code)"
      >
        <component :is="copied ? Check : Copy" class="h-3.5 w-3.5" />
        {{ copied ? 'Copied' : 'Copy' }}
      </button>
    </div>
    <pre
      class="overflow-auto px-4 py-3 font-mono text-xs leading-relaxed text-slate-200"
      :style="{ maxHeight: maxHeight || '480px' }"
    ><code>{{ code }}</code></pre>
  </div>
</template>
