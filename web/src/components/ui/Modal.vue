<script setup lang="ts">
import { watch } from 'vue'
import { X } from 'lucide-vue-next'

const props = defineProps<{
  open: boolean
  title?: string
  description?: string
  size?: 'sm' | 'md' | 'lg' | 'xl'
}>()

const emit = defineEmits<{ (e: 'close'): void }>()

watch(
  () => props.open,
  (open) => {
    if (open) {
      document.documentElement.style.overflow = 'hidden'
    } else {
      document.documentElement.style.overflow = ''
    }
  }
)

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape' && props.open) emit('close')
}
</script>

<template>
  <Teleport to="body">
    <div
      v-if="open"
      class="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-ink-950/80 px-4 py-8 backdrop-blur-sm animate-fade-in"
      @click.self="emit('close')"
      @keydown="onKey"
      tabindex="-1"
    >
      <div
        class="relative w-full rounded-md border border-ink-700 bg-ink-900 shadow-2xl shadow-black/60"
        :class="{
          'max-w-sm': size === 'sm',
          'max-w-lg': !size || size === 'md',
          'max-w-2xl': size === 'lg',
          'max-w-4xl': size === 'xl'
        }"
      >
        <header
          v-if="title || $slots.header"
          class="flex items-start justify-between gap-4 border-b border-ink-700 px-6 py-4"
        >
          <div class="flex-1">
            <slot name="header">
              <h2 class="text-sm font-semibold uppercase tracking-wider text-slate-100">
                <span class="text-brand-400">&gt;</span> {{ title }}
              </h2>
              <p v-if="description" class="mt-1 text-sm normal-case text-slate-400">
                {{ description }}
              </p>
            </slot>
          </div>
          <button
            type="button"
            class="rounded-sm p-1 text-slate-500 hover:bg-ink-800 hover:text-brand-300"
            @click="emit('close')"
          >
            <X class="h-4 w-4" />
          </button>
        </header>
        <div class="px-6 py-5">
          <slot />
        </div>
        <footer
          v-if="$slots.footer"
          class="flex items-center justify-end gap-2 border-t border-ink-700 px-6 py-4"
        >
          <slot name="footer" />
        </footer>
      </div>
    </div>
  </Teleport>
</template>
