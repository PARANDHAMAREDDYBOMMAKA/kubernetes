<script setup lang="ts">
import { computed } from 'vue'
import Spinner from './Spinner.vue'

type Variant = 'primary' | 'secondary' | 'ghost' | 'danger' | 'outline'
type Size = 'sm' | 'md' | 'lg'

const props = withDefaults(
  defineProps<{
    variant?: Variant
    size?: Size
    type?: 'button' | 'submit' | 'reset'
    loading?: boolean
    disabled?: boolean
    fullWidth?: boolean
  }>(),
  {
    variant: 'primary',
    size: 'md',
    type: 'button',
    loading: false,
    disabled: false,
    fullWidth: false
  }
)

const variantClass = computed(() => {
  switch (props.variant) {
    case 'primary':
      return 'bg-brand-500 hover:bg-brand-400 text-ink-950 font-semibold focus:ring-brand-500/50 shadow-glow'
    case 'secondary':
      return 'bg-ink-800 hover:bg-ink-700 text-slate-100 focus:ring-brand-500/40 border border-ink-700'
    case 'ghost':
      return 'bg-transparent hover:bg-ink-800 text-slate-300 focus:ring-brand-500/40'
    case 'outline':
      return 'bg-transparent hover:bg-brand-500/10 hover:text-brand-300 text-slate-200 border border-ink-700 hover:border-brand-500/40 focus:ring-brand-500/40'
    case 'danger':
      return 'bg-red-500/90 hover:bg-red-500 text-white focus:ring-red-500/50'
    default:
      return ''
  }
})

const sizeClass = computed(() => {
  switch (props.size) {
    case 'sm':
      return 'px-3 py-1.5 text-xs'
    case 'md':
      return 'px-3.5 py-2 text-sm'
    case 'lg':
      return 'px-4 py-2.5 text-sm'
    default:
      return ''
  }
})
</script>

<template>
  <button
    :type="type"
    :disabled="disabled || loading"
    class="btn-base"
    :class="[variantClass, sizeClass, fullWidth ? 'w-full' : '']"
  >
    <Spinner v-if="loading" class="h-4 w-4" />
    <slot />
  </button>
</template>
