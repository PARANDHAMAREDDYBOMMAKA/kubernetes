<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ArrowLeft,
  Trash2,
  Download,
  RefreshCw,
  Server,
  Globe,
  Shuffle
} from 'lucide-vue-next'
import { useLoadBalancersStore } from '@/stores/loadbalancers'
import { useClustersStore } from '@/stores/clusters'
import { useToastStore } from '@/stores/toast'
import { extractErrorMessage } from '@/lib/api'
import Button from '@/components/ui/Button.vue'
import Card from '@/components/ui/Card.vue'
import CodeBlock from '@/components/ui/CodeBlock.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import { formatDate, uptime } from '@/lib/utils'

const props = defineProps<{ id: string }>()
const route = useRoute()
const router = useRouter()
const lbStore = useLoadBalancersStore()
const clusterStore = useClustersStore()
const toast = useToastStore()

const lbId = computed(() => (props.id || (route.params.id as string)))

const yaml = ref('')
const loadingYaml = ref(false)
const confirmDelete = ref(false)
const deleting = ref(false)
const pollTimer = ref<number | null>(null)

const lb = computed(() => lbStore.current)

async function fetch(silent = false) {
  try {
    await lbStore.fetchOne(lbId.value, silent)
  } catch (err) {
    if (!silent)
      toast.error('Failed to load load balancer', extractErrorMessage(err))
  }
}

function schedulePolling() {
  if (pollTimer.value) return
  pollTimer.value = window.setInterval(() => {
    const s = lb.value?.status
    if (s === 'Provisioning' || s === 'Deleting') fetch(true)
  }, 3000)
}

watch(
  () => lb.value?.status,
  (s) => {
    if (s === 'Provisioning' || s === 'Deleting') schedulePolling()
  }
)

onMounted(() => {
  fetch()
  if (clusterStore.list.length === 0) clusterStore.fetchAll(true).catch(() => {})
  schedulePolling()
})

onBeforeUnmount(() => {
  if (pollTimer.value) {
    clearInterval(pollTimer.value)
    pollTimer.value = null
  }
  lbStore.current = null
})

async function downloadYaml() {
  if (!lb.value) return
  loadingYaml.value = true
  try {
    const text = await lbStore.downloadYaml(lb.value)
    yaml.value = text
    toast.success('YAML downloaded')
  } catch (err) {
    toast.error('YAML download failed', extractErrorMessage(err))
  } finally {
    loadingYaml.value = false
  }
}

async function fetchYaml() {
  if (!lb.value || yaml.value) return
  loadingYaml.value = true
  try {
    yaml.value = await lbStore.fetchYaml(lb.value.id)
  } catch (err) {
    toast.error('YAML fetch failed', extractErrorMessage(err))
  } finally {
    loadingYaml.value = false
  }
}

async function doDelete() {
  if (!lb.value) return
  deleting.value = true
  try {
    await lbStore.remove(lb.value.id)
    toast.success('Load balancer deletion started')
    confirmDelete.value = false
    router.push({ name: 'loadbalancers' })
  } catch (err) {
    toast.error('Delete failed', extractErrorMessage(err))
  } finally {
    deleting.value = false
  }
}

const clusterName = computed(() => {
  if (!lb.value) return ''
  const c = clusterStore.list.find((x) => x.id === lb.value!.clusterId)
  return c?.name || lb.value.clusterId
})
</script>

<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div class="flex items-start gap-3">
        <button
          class="mt-1 rounded p-1.5 text-slate-400 hover:bg-slate-800 hover:text-slate-100"
          @click="router.push({ name: 'loadbalancers' })"
        >
          <ArrowLeft class="h-4 w-4" />
        </button>
        <div>
          <div class="flex items-center gap-3">
            <h1 class="text-2xl font-semibold text-white">
              {{ lb?.name || '...' }}
            </h1>
            <StatusBadge v-if="lb" :status="lb.status" />
          </div>
          <p class="mt-1 font-mono text-xs text-slate-500">
            {{ lb?.id || '—' }}
          </p>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <Button
          variant="outline"
          :disabled="lbStore.loadingCurrent"
          @click="fetch()"
        >
          <RefreshCw
            class="h-4 w-4"
            :class="lbStore.loadingCurrent ? 'animate-spin' : ''"
          />
          Refresh
        </Button>
        <Button
          variant="outline"
          :loading="loadingYaml"
          :disabled="!lb"
          @click="downloadYaml"
        >
          <Download class="h-4 w-4" />
          YAML
        </Button>
        <Button
          variant="danger"
          :disabled="!lb || lb.status === 'Deleting'"
          @click="confirmDelete = true"
        >
          <Trash2 class="h-4 w-4" />
          Delete
        </Button>
      </div>
    </div>

    <!-- Stats -->
    <div v-if="lb" class="grid grid-cols-2 gap-4 md:grid-cols-4">
      <div class="card-base p-4">
        <div
          class="flex items-center gap-2 text-xs font-medium uppercase tracking-wider text-slate-400"
        >
          <Globe class="h-3.5 w-3.5" />
          Listen port
        </div>
        <div class="mt-2 font-mono text-xl font-semibold text-white">
          {{ lb.port }}
        </div>
      </div>
      <div class="card-base p-4">
        <div
          class="flex items-center gap-2 text-xs font-medium uppercase tracking-wider text-slate-400"
        >
          <Server class="h-3.5 w-3.5" />
          Target port
        </div>
        <div class="mt-2 font-mono text-xl font-semibold text-white">
          {{ lb.targetPort }}
        </div>
      </div>
      <div class="card-base p-4">
        <div
          class="flex items-center gap-2 text-xs font-medium uppercase tracking-wider text-slate-400"
        >
          <Shuffle class="h-3.5 w-3.5" />
          Algorithm
        </div>
        <div class="mt-2 text-xl font-semibold capitalize text-white">
          {{ lb.algorithm.replace('_', ' ') }}
        </div>
      </div>
      <div class="card-base p-4">
        <div
          class="flex items-center gap-2 text-xs font-medium uppercase tracking-wider text-slate-400"
        >
          Uptime
        </div>
        <div class="mt-2 text-xl font-semibold text-white">
          {{ uptime(lb.createdAt) }}
        </div>
      </div>
    </div>

    <div v-if="lb" class="grid gap-6 lg:grid-cols-3">
      <Card title="Details" class="lg:col-span-2">
        <dl class="grid grid-cols-1 gap-y-4 sm:grid-cols-2">
          <div>
            <dt
              class="text-xs font-medium uppercase tracking-wider text-slate-500"
            >
              Cluster
            </dt>
            <dd class="mt-1 text-sm text-slate-100">{{ clusterName }}</dd>
          </div>
          <div>
            <dt
              class="text-xs font-medium uppercase tracking-wider text-slate-500"
            >
              Algorithm
            </dt>
            <dd class="mt-1 text-sm text-slate-100 capitalize">
              {{ lb.algorithm.replace('_', ' ') }}
            </dd>
          </div>
          <div>
            <dt
              class="text-xs font-medium uppercase tracking-wider text-slate-500"
            >
              Created
            </dt>
            <dd class="mt-1 text-sm text-slate-100">
              {{ formatDate(lb.createdAt) }}
            </dd>
          </div>
          <div>
            <dt
              class="text-xs font-medium uppercase tracking-wider text-slate-500"
            >
              Updated
            </dt>
            <dd class="mt-1 text-sm text-slate-100">
              {{ formatDate(lb.updatedAt) }}
            </dd>
          </div>
        </dl>
      </Card>

      <Card title="Backends" :description="`${lb.backends?.length ?? 0} total`">
        <ul v-if="lb.backends?.length" class="divide-y divide-slate-800">
          <li
            v-for="(b, i) in lb.backends"
            :key="`${b.host}-${b.port}-${i}`"
            class="flex items-center justify-between py-2.5 text-sm"
          >
            <span class="font-mono text-slate-200">{{ b.host }}</span>
            <span class="font-mono text-slate-400">:{{ b.port }}</span>
          </li>
        </ul>
        <p v-else class="text-sm text-slate-400">No backends configured.</p>
      </Card>
    </div>

    <Card
      v-if="lb"
      title="YAML"
      description="Infrastructure-as-code definition for this load balancer."
    >
      <template #actions>
        <Button
          variant="outline"
          size="sm"
          :loading="loadingYaml"
          @click="fetchYaml"
        >
          <Download class="h-4 w-4" />
          {{ yaml ? 'Refresh' : 'Load' }}
        </Button>
        <Button
          variant="outline"
          size="sm"
          :loading="loadingYaml"
          @click="downloadYaml"
        >
          <Download class="h-4 w-4" />
          Download
        </Button>
      </template>
      <CodeBlock v-if="yaml" :code="yaml" language="yaml" max-height="500px" />
      <p v-else-if="!loadingYaml" class="text-sm text-slate-400">
        Click "Load" to fetch the YAML definition.
      </p>
      <div v-else class="space-y-2">
        <div class="skeleton h-4 w-full" />
        <div class="skeleton h-4 w-3/4" />
        <div class="skeleton h-4 w-1/2" />
      </div>
    </Card>

    <ConfirmDialog
      :open="confirmDelete"
      title="Delete load balancer?"
      :description="`This will remove '${lb?.name}' and stop routing traffic. This action cannot be undone.`"
      confirm-text="Delete"
      variant="danger"
      :loading="deleting"
      @close="confirmDelete = false"
      @confirm="doDelete"
    />
  </div>
</template>
