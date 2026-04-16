<script setup lang="ts">
import { computed } from 'vue'
import Badge from './ui/Badge.vue'
import type { ClusterStatus, LoadBalancerStatus } from '@/types'

type AnyStatus = ClusterStatus | LoadBalancerStatus

const props = defineProps<{ status: AnyStatus }>()

const tone = computed<
  'success' | 'warning' | 'danger' | 'info' | 'muted' | 'neutral'
>(() => {
  switch (props.status) {
    case 'Running':
      return 'success'
    case 'Provisioning':
      return 'info'
    case 'Deleting':
      return 'warning'
    case 'Failed':
      return 'danger'
    case 'Deleted':
      return 'muted'
    default:
      return 'neutral'
  }
})

const pulse = computed(
  () => props.status === 'Provisioning' || props.status === 'Deleting'
)
</script>

<template>
  <Badge :tone="tone" dot :pulse="pulse">{{ status }}</Badge>
</template>
