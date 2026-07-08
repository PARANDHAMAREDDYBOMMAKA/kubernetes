<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { Plus, X } from 'lucide-vue-next'
import Modal from './ui/Modal.vue'
import Input from './ui/Input.vue'
import Select from './ui/Select.vue'
import Button from './ui/Button.vue'
import { useLoadBalancersStore } from '@/stores/loadbalancers'
import { useClustersStore } from '@/stores/clusters'
import { useToastStore } from '@/stores/toast'
import { extractErrorMessage } from '@/lib/api'
import type { Backend, LoadBalancerAlgorithm } from '@/types'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'created'): void }>()

const lbStore = useLoadBalancersStore()
const clusterStore = useClustersStore()
const toast = useToastStore()

interface FormState {
  name: string
  clusterId: string
  port: number
  targetPort: number
  algorithm: LoadBalancerAlgorithm
  backends: Backend[]
}

const form = reactive<FormState>({
  name: '',
  clusterId: '',
  port: 30080,
  targetPort: 30080,
  algorithm: 'round_robin',
  backends: [{ host: '', port: 30080 }]
})
const errors = reactive<Record<string, string>>({})
const submitting = ref(false)

const algorithms: { value: LoadBalancerAlgorithm; label: string }[] = [
  { value: 'round_robin', label: 'Round Robin' },
  { value: 'least_conn', label: 'Least Connections' },
  { value: 'ip_hash', label: 'IP Hash' }
]

const clusterOptions = computed(() =>
  clusterStore.list
    .filter((c) => c.status === 'Running' || c.status === 'Provisioning')
    .map((c) => ({ value: c.id, label: `${c.name} (${c.region})` }))
)

const selectedCluster = computed(() =>
  clusterStore.list.find((c) => c.id === form.clusterId)
)

const backendHint = computed(() => {
  const c = selectedCluster.value
  if (!c) return ''
  const safe = c.id.replace(/[^A-Za-z0-9_-]/g, '-')
  return `kaas-${safe}-agent-0`
})

onMounted(() => {
  if (clusterStore.list.length === 0) clusterStore.fetchAll(true).catch(() => {})
})

watch(
  () => props.open,
  async (o) => {
    if (o) {
      form.name = ''
      form.clusterId = ''
      form.port = 30080
      form.targetPort = 30080
      form.algorithm = 'round_robin'
      form.backends = [{ host: '', port: 30080 }]
      Object.keys(errors).forEach((k) => delete errors[k])
      if (clusterStore.list.length === 0) {
        await clusterStore.fetchAll(true).catch(() => {})
      }
      if (!form.clusterId && clusterOptions.value[0]) {
        form.clusterId = clusterOptions.value[0].value
      }
    }
  }
)

function addBackend() {
  form.backends.push({ host: '', port: form.targetPort || 8080 })
}
function removeBackend(i: number) {
  form.backends.splice(i, 1)
  if (form.backends.length === 0) addBackend()
}

async function submit() {
  Object.keys(errors).forEach((k) => delete errors[k])
  if (!form.name.trim()) errors.name = 'Required'
  if (!form.clusterId) errors.clusterId = 'Select a cluster'
  if (!form.port || form.port < 30000 || form.port > 40000)
    errors.port = 'Must be in 30000–40000 (reserved/firewall range)'
  if (form.port === 80 || form.port === 443)
    errors.port = 'Ports 80 and 443 are reserved by the platform'
  if (!form.targetPort || form.targetPort < 1 || form.targetPort > 65535)
    errors.targetPort = 'Invalid port'

  const cleanBackends = form.backends
    .map((b) => ({ host: b.host.trim(), port: Number(b.port) }))
    .filter((b) => b.host)
  if (cleanBackends.length === 0) {
    errors.backends = 'At least one backend is required'
  } else if (cleanBackends.some((b) => !b.port || b.port < 1 || b.port > 65535)) {
    errors.backends = 'All backend ports must be between 1 and 65535'
  }

  if (Object.keys(errors).length) return

  submitting.value = true
  try {
    await lbStore.create({
      name: form.name.trim(),
      clusterId: form.clusterId,
      port: Number(form.port),
      targetPort: Number(form.targetPort),
      algorithm: form.algorithm,
      backends: cleanBackends
    })
    toast.success('Load balancer creation started')
    emit('created')
    emit('close')
  } catch (err) {
    toast.error('Failed to create load balancer', extractErrorMessage(err))
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <Modal
    :open="open"
    title="Create load balancer"
    description="Route external traffic to a service running in your cluster."
    size="lg"
    @close="emit('close')"
  >
    <form class="space-y-5" @submit.prevent="submit">
      <Input
        id="lb-name"
        v-model="form.name"
        label="Name"
        placeholder="e.g. web-lb"
        :error="errors.name"
        required
      />

      <div v-if="clusterOptions.length === 0" class="rounded-lg border border-amber-500/30 bg-amber-500/5 p-3 text-sm text-amber-200">
        You need at least one cluster before creating a load balancer.
      </div>
      <Select
        v-else
        id="lb-cluster"
        v-model="form.clusterId"
        label="Cluster"
        :options="clusterOptions"
        required
      />
      <p v-if="errors.clusterId" class="-mt-3 text-xs text-red-400">
        {{ errors.clusterId }}
      </p>

      <div class="grid gap-4 sm:grid-cols-3">
        <Input
          id="lb-port"
          v-model.number="form.port"
          label="Listen port"
          type="number"
          :min="1"
          :max="65535"
          :error="errors.port"
          required
        />
        <Input
          id="lb-target-port"
          v-model.number="form.targetPort"
          label="Target port"
          type="number"
          :min="1"
          :max="65535"
          :error="errors.targetPort"
          required
        />
        <Select
          id="lb-algo"
          v-model="form.algorithm"
          label="Algorithm"
          :options="algorithms"
          required
        />
      </div>

      <div>
        <div class="mb-2 flex items-center justify-between">
          <label class="text-xs font-medium text-slate-300">Backends</label>
          <button
            type="button"
            class="inline-flex items-center gap-1 rounded-sm px-2 py-1 text-xs text-brand-300 hover:bg-ink-800"
            @click="addBackend"
          >
            <Plus class="h-3.5 w-3.5" />
            Add backend
          </button>
        </div>
        <p v-if="backendHint" class="mb-2 text-xs text-slate-500">
          Tip: for k8s NodePort services, use
          <code class="rounded-sm bg-ink-800 px-1 py-0.5 text-brand-300">{{ backendHint }}</code>
          as host and the NodePort (30000–32767) as port.
        </p>
        <div class="space-y-2">
          <div
            v-for="(b, i) in form.backends"
            :key="i"
            class="flex items-center gap-2"
          >
            <input
              v-model="b.host"
              type="text"
              placeholder="host or IP (e.g. 10.0.0.5)"
              class="input-base flex-1"
            />
            <input
              v-model.number="b.port"
              type="number"
              placeholder="Port"
              :min="1"
              :max="65535"
              class="input-base w-28"
            />
            <button
              type="button"
              class="rounded-sm p-2 text-slate-400 hover:bg-red-500/10 hover:text-red-300 disabled:opacity-40"
              :disabled="form.backends.length === 1"
              @click="removeBackend(i)"
            >
              <X class="h-4 w-4" />
            </button>
          </div>
        </div>
        <p v-if="errors.backends" class="mt-1 text-xs text-red-400">
          {{ errors.backends }}
        </p>
      </div>
    </form>

    <template #footer>
      <Button variant="ghost" :disabled="submitting" @click="emit('close')"
        >Cancel</Button
      >
      <Button
        :loading="submitting"
        :disabled="clusterOptions.length === 0"
        @click="submit"
      >
        Create load balancer
      </Button>
    </template>
  </Modal>
</template>
