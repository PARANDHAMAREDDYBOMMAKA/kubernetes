<script setup lang="ts">
import { onMounted } from 'vue'
import { RouterView } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import Toaster from '@/components/ui/Toaster.vue'

const auth = useAuthStore()

onMounted(async () => {
  if (auth.token && !auth.user) {
    await auth.fetchMe()
  }
})
</script>

<template>
  <RouterView />
  <Toaster />
</template>
