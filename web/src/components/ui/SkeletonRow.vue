<script setup lang="ts">
withDefaults(
  defineProps<{
    columns?: number
    rows?: number
  }>(),
  { columns: 5, rows: 4 }
)
</script>

<template>
  <tr v-for="r in rows" :key="r">
    <td v-for="c in columns" :key="c">
      <div class="skeleton h-4 w-full max-w-[12rem]" />
    </td>
  </tr>
</template>
