<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import Modal from './ui/Modal.vue'
import Input from './ui/Input.vue'
import Select from './ui/Select.vue'
import Button from './ui/Button.vue'
import { useClustersStore } from '@/stores/clusters'
import { useToastStore } from '@/stores/toast'
import { extractErrorMessage } from '@/lib/api'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'created'): void }>()

const clusters = useClustersStore()
const toast = useToastStore()

const form = reactive({
  name: '',
  k8sVersion: 'v1.29.3',
  region: 'local-docker',
  nodeCount: 3
})
const errors = reactive<Record<string, string>>({})
const submitting = ref(false)

watch(
  () => props.open,
  (o) => {
    if (o) {
      form.name = ''
      form.k8sVersion = 'v1.29.3'
      form.region = 'local-docker'
      form.nodeCount = 3
      Object.keys(errors).forEach((k) => delete errors[k])
    }
  }
)

const versions = [
  { value: 'v1.29.3', label: 'v1.29.3 (latest)' },
  { value: 'v1.28.8', label: 'v1.28.8' },
  { value: 'v1.27.12', label: 'v1.27.12' }
]
const regions = [{ value: 'local-docker', label: 'Local (Docker)' }]

async function submit() {
  Object.keys(errors).forEach((k) => delete errors[k])
  if (!form.name.trim()) errors.name = 'Required'
  else if (!/^[a-z0-9][a-z0-9-]{1,30}[a-z0-9]$/.test(form.name))
    errors.name = 'Lowercase letters, digits, dashes (3–32 chars)'
  if (form.nodeCount < 1 || form.nodeCount > 5)
    errors.nodeCount = 'Must be between 1 and 5'

  if (Object.keys(errors).length) return

  submitting.value = true
  try {
    await clusters.create({
      name: form.name.trim(),
      k8sVersion: form.k8sVersion,
      nodeCount: form.nodeCount,
      region: form.region
    })
    toast.success('Cluster creation started', 'Provisioning in background.')
    emit('created')
    emit('close')
  } catch (err) {
    toast.error('Failed to create cluster', extractErrorMessage(err))
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <Modal
    :open="open"
    title="Create cluster"
    description="Provision a new Kubernetes cluster. This usually takes a minute or two."
    size="lg"
    @close="emit('close')"
  >
    <form class="space-y-5" @submit.prevent="submit">
      <Input
        id="cluster-name"
        v-model="form.name"
        label="Name"
        placeholder="e.g. prod-us-east"
        :error="errors.name"
        required
        hint="Lowercase letters, numbers and dashes. Must be unique."
      />

      <div class="grid gap-4 sm:grid-cols-2">
        <Select
          id="k8s-version"
          v-model="form.k8sVersion"
          label="Kubernetes version"
          :options="versions"
          required
        />
        <Select
          id="region"
          v-model="form.region"
          label="Region"
          :options="regions"
          required
        />
      </div>

      <div>
        <label class="mb-1.5 block text-xs font-medium text-slate-300">
          Nodes: <span class="text-slate-100">{{ form.nodeCount }}</span>
        </label>
        <input
          v-model.number="form.nodeCount"
          type="range"
          min="1"
          max="5"
          step="1"
          class="h-2 w-full cursor-pointer appearance-none rounded-lg bg-slate-800 accent-brand-500"
        />
        <div class="mt-1 flex justify-between text-xs text-slate-500">
          <span>1</span><span>2</span><span>3</span><span>4</span><span>5</span>
        </div>
        <p v-if="errors.nodeCount" class="mt-1 text-xs text-red-400">
          {{ errors.nodeCount }}
        </p>
      </div>
    </form>

    <template #footer>
      <Button variant="ghost" :disabled="submitting" @click="emit('close')"
        >Cancel</Button
      >
      <Button :loading="submitting" @click="submit">Create cluster</Button>
    </template>
  </Modal>
</template>
