<script setup lang="ts">
import Modal from './ui/Modal.vue'
import Button from './ui/Button.vue'

defineProps<{
  open: boolean
  title: string
  description?: string
  confirmText?: string
  cancelText?: string
  variant?: 'danger' | 'primary'
  loading?: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'confirm'): void
}>()
</script>

<template>
  <Modal
    :open="open"
    :title="title"
    :description="description"
    size="sm"
    @close="emit('close')"
  >
    <template #footer>
      <Button variant="ghost" :disabled="loading" @click="emit('close')">
        {{ cancelText || 'Cancel' }}
      </Button>
      <Button
        :variant="variant === 'primary' ? 'primary' : 'danger'"
        :loading="loading"
        @click="emit('confirm')"
      >
        {{ confirmText || 'Confirm' }}
      </Button>
    </template>
  </Modal>
</template>
