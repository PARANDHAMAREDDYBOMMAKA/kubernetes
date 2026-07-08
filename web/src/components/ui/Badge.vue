<script setup lang="ts">
import { computed } from 'vue'

type Tone =
  | 'neutral'
  | 'success'
  | 'warning'
  | 'danger'
  | 'info'
  | 'brand'
  | 'muted'

const props = withDefaults(
  defineProps<{
    tone?: Tone
    dot?: boolean
    pulse?: boolean
  }>(),
  { tone: 'neutral', dot: false, pulse: false }
)

const tones: Record<Tone, string> = {
  neutral: 'bg-ink-800 text-slate-300 ring-1 ring-inset ring-ink-700',
  success:
    'bg-brand-500/10 text-brand-300 ring-1 ring-inset ring-brand-500/30',
  warning:
    'bg-amber-500/10 text-amber-300 ring-1 ring-inset ring-amber-500/30',
  danger: 'bg-red-500/10 text-red-300 ring-1 ring-inset ring-red-500/30',
  info: 'bg-sky-500/10 text-sky-300 ring-1 ring-inset ring-sky-500/30',
  brand:
    'bg-brand-500/10 text-brand-300 ring-1 ring-inset ring-brand-500/30',
  muted:
    'bg-ink-800/60 text-slate-500 ring-1 ring-inset ring-ink-700/70'
}

const dotColor: Record<Tone, string> = {
  neutral: 'bg-slate-400',
  success: 'bg-brand-400',
  warning: 'bg-amber-400',
  danger: 'bg-red-400',
  info: 'bg-sky-400',
  brand: 'bg-brand-400',
  muted: 'bg-slate-500'
}

const cls = computed(() => tones[props.tone])
const dc = computed(() => dotColor[props.tone])
</script>

<template>
  <span
    class="inline-flex items-center gap-1.5 rounded-sm px-1.5 py-0.5 text-[11px] font-medium uppercase tracking-wide"
    :class="cls"
  >
    <span
      v-if="dot"
      class="h-1.5 w-1.5 rounded-full"
      :class="[dc, pulse ? 'animate-pulse' : '']"
    />
    <slot />
  </span>
</template>
