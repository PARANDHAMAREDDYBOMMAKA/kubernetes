<script setup lang="ts">
withDefaults(
  defineProps<{
    modelValue: string | number
    type?: string
    placeholder?: string
    label?: string
    id?: string
    error?: string
    hint?: string
    disabled?: boolean
    required?: boolean
    min?: number | string
    max?: number | string
    step?: number | string
    autocomplete?: string
  }>(),
  {
    type: 'text',
    disabled: false,
    required: false
  }
)

defineEmits<{
  (e: 'update:modelValue', value: string | number): void
}>()
</script>

<template>
  <div class="w-full">
    <label v-if="label" :for="id" class="mb-1.5 block text-xs font-medium text-slate-300">
      {{ label }}
      <span v-if="required" class="text-red-400">*</span>
    </label>
    <input
      :id="id"
      :type="type"
      :value="modelValue"
      :placeholder="placeholder"
      :disabled="disabled"
      :required="required"
      :min="min"
      :max="max"
      :step="step"
      :autocomplete="autocomplete"
      class="input-base"
      :class="error ? 'border-red-500/70 focus:border-red-500 focus:ring-red-500/40' : ''"
      @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)"
    />
    <p v-if="error" class="mt-1 text-xs text-red-400">{{ error }}</p>
    <p v-else-if="hint" class="mt-1 text-xs text-slate-500">{{ hint }}</p>
  </div>
</template>
