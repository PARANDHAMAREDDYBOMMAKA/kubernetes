<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Trash2,
  Download,
  ArrowLeft,
  RefreshCw,
  Server,
  Cpu,
  Network,
  Clock
} from 'lucide-vue-next'
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
const clusters = useClustersStore()
const toast = useToastStore()

const clusterId = computed(() => (props.id || (route.params.id as string)))
const loadingKubeconfig = ref(false)
const loadingYaml = ref(false)
const kubeconfig = ref<string>('')
const yaml = ref<string>('')
const activeTab = ref<'overview' | 'kubeconfig' | 'yaml' | 'nodes'>('overview')

const scaleValue = ref(1)
const scaling = ref(false)
const confirmDelete = ref(false)
const deleting = ref(false)

const pollTimer = ref<number | null>(null)

const cluster = computed(() => clusters.current)

async function fetch(silent = false) {
  try {
    const c = await clusters.fetchOne(clusterId.value, silent)
    scaleValue.value = c.nodeCount
  } catch (err) {
    if (!silent) toast.error('Failed to load cluster', extractErrorMessage(err))
  }
}

function schedulePolling() {
  if (pollTimer.value) return
  pollTimer.value = window.setInterval(() => {
    const s = cluster.value?.status
    if (s === 'Provisioning' || s === 'Deleting') fetch(true)
  }, 3000)
}

watch(
  () => cluster.value?.status,
  (s) => {
    if (s === 'Provisioning' || s === 'Deleting') schedulePolling()
  }
)

onMounted(() => {
  fetch()
  schedulePolling()
})

onBeforeUnmount(() => {
  if (pollTimer.value) {
    clearInterval(pollTimer.value)
    pollTimer.value = null
  }
  clusters.current = null
})

async function downloadKube() {
  if (!cluster.value) return
  loadingKubeconfig.value = true
  try {
    const text = await clusters.downloadKubeconfig(cluster.value)
    kubeconfig.value = text
    toast.success('Kubeconfig downloaded')
  } catch (err) {
    toast.error('Kubeconfig download failed', extractErrorMessage(err))
  } finally {
    loadingKubeconfig.value = false
  }
}

async function fetchKubeconfigOnly() {
  if (!cluster.value || kubeconfig.value) return
  loadingKubeconfig.value = true
  try {
    kubeconfig.value = await clusters.fetchKubeconfig(cluster.value.id)
  } catch (err) {
    toast.error('Kubeconfig fetch failed', extractErrorMessage(err))
  } finally {
    loadingKubeconfig.value = false
  }
}

async function downloadYaml() {
  if (!cluster.value) return
  loadingYaml.value = true
  try {
    const text = await clusters.downloadYaml(cluster.value)
    yaml.value = text
    toast.success('YAML downloaded')
  } catch (err) {
    toast.error('YAML download failed', extractErrorMessage(err))
  } finally {
    loadingYaml.value = false
  }
}

async function fetchYamlOnly() {
  if (!cluster.value || yaml.value) return
  loadingYaml.value = true
  try {
    yaml.value = await clusters.fetchYaml(cluster.value.id)
  } catch (err) {
    toast.error('YAML fetch failed', extractErrorMessage(err))
  } finally {
    loadingYaml.value = false
  }
}

watch(activeTab, (t) => {
  if (t === 'kubeconfig') fetchKubeconfigOnly()
  if (t === 'yaml') fetchYamlOnly()
})

async function applyScale() {
  if (!cluster.value) return
  const target = Number(scaleValue.value)
  if (!Number.isFinite(target) || target < 1 || target > 5) {
    toast.warning('Node count must be between 1 and 5')
    return
  }
  if (target === cluster.value.nodeCount) {
    toast.info('No change', 'Node count is already at this value.')
    return
  }
  scaling.value = true
  try {
    await clusters.scale(cluster.value.id, target)
    toast.success(`Scaling to ${target} node${target === 1 ? '' : 's'}`)
    schedulePolling()
  } catch (err) {
    toast.error('Scale failed', extractErrorMessage(err))
  } finally {
    scaling.value = false
  }
}

async function doDelete() {
  if (!cluster.value) return
  deleting.value = true
  try {
    await clusters.remove(cluster.value.id)
    toast.success('Cluster deletion started')
    confirmDelete.value = false
    router.push({ name: 'clusters' })
  } catch (err) {
    toast.error('Delete failed', extractErrorMessage(err))
  } finally {
    deleting.value = false
  }
}

const stats = computed(() => {
  const c = cluster.value
  if (!c) return []
  return [
    { label: 'Nodes', value: c.nodeCount, icon: Server },
    { label: 'K8s version', value: c.k8sVersion, icon: Cpu },
    {
      label: 'API port',
      value: c.apiPort ? c.apiPort.toString() : '—',
      icon: Network
    },
    { label: 'Uptime', value: uptime(c.createdAt), icon: Clock }
  ]
})

const tabs = [
  { id: 'overview', label: 'Overview' },
  { id: 'kubeconfig', label: 'Kubeconfig' },
  { id: 'yaml', label: 'YAML' },
  { id: 'nodes', label: 'Nodes' }
] as const
</script>

<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div class="flex items-start gap-3">
        <button
          class="mt-1 rounded p-1.5 text-slate-400 hover:bg-slate-800 hover:text-slate-100"
          @click="router.push({ name: 'clusters' })"
        >
          <ArrowLeft class="h-4 w-4" />
        </button>
        <div>
          <div class="flex items-center gap-3">
            <h1 class="text-2xl font-semibold text-white">
              {{ cluster?.name || '...' }}
            </h1>
            <StatusBadge v-if="cluster" :status="cluster.status" />
          </div>
          <p class="mt-1 font-mono text-xs text-slate-500">
            {{ cluster?.id || '—' }}
          </p>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <Button
          variant="outline"
          :disabled="clusters.loadingCurrent"
          @click="fetch()"
        >
          <RefreshCw
            class="h-4 w-4"
            :class="clusters.loadingCurrent ? 'animate-spin' : ''"
          />
          Refresh
        </Button>
        <Button
          variant="outline"
          :loading="loadingKubeconfig"
          :disabled="!cluster || cluster.status !== 'Running'"
          @click="downloadKube"
        >
          <Download class="h-4 w-4" />
          Kubeconfig
        </Button>
        <Button
          variant="outline"
          :loading="loadingYaml"
          :disabled="!cluster"
          @click="downloadYaml"
        >
          <Download class="h-4 w-4" />
          YAML
        </Button>
        <Button
          variant="danger"
          :disabled="
            !cluster || cluster.status === 'Deleting' || cluster.status === 'Deleted'
          "
          @click="confirmDelete = true"
        >
          <Trash2 class="h-4 w-4" />
          Delete
        </Button>
      </div>
    </div>

    <!-- Message banner -->
    <div
      v-if="cluster?.message"
      class="rounded-lg border border-amber-500/30 bg-amber-500/5 px-4 py-3 text-sm text-amber-200"
    >
      {{ cluster.message }}
    </div>

    <!-- Stats -->
    <div
      v-if="cluster"
      class="grid grid-cols-2 gap-4 md:grid-cols-4"
    >
      <div
        v-for="s in stats"
        :key="s.label"
        class="card-base p-4"
      >
        <div class="flex items-center gap-2 text-xs font-medium uppercase tracking-wider text-slate-400">
          <component :is="s.icon" class="h-3.5 w-3.5" />
          {{ s.label }}
        </div>
        <div class="mt-2 text-xl font-semibold text-white">{{ s.value }}</div>
      </div>
    </div>

    <!-- Loading skeleton for stats -->
    <div
      v-else-if="clusters.loadingCurrent"
      class="grid grid-cols-2 gap-4 md:grid-cols-4"
    >
      <div v-for="i in 4" :key="i" class="card-base p-4">
        <div class="skeleton h-3 w-20" />
        <div class="skeleton mt-3 h-5 w-16" />
      </div>
    </div>

    <!-- Tabs -->
    <div
      v-if="cluster"
      class="flex items-center gap-1 border-b border-slate-800"
    >
      <button
        v-for="t in tabs"
        :key="t.id"
        class="relative px-3 py-2 text-sm font-medium transition"
        :class="
          activeTab === t.id
            ? 'text-white'
            : 'text-slate-400 hover:text-slate-200'
        "
        @click="activeTab = t.id"
      >
        {{ t.label }}
        <span
          v-if="activeTab === t.id"
          class="absolute inset-x-0 -bottom-px h-0.5 rounded bg-brand-500"
        />
      </button>
    </div>

    <!-- Overview tab -->
    <div v-if="cluster && activeTab === 'overview'" class="grid gap-6 lg:grid-cols-3">
      <Card title="Details" class="lg:col-span-2">
        <dl class="grid grid-cols-1 gap-y-4 sm:grid-cols-2">
          <div>
            <dt class="text-xs font-medium uppercase tracking-wider text-slate-500">
              Name
            </dt>
            <dd class="mt-1 text-sm text-slate-100">{{ cluster.name }}</dd>
          </div>
          <div>
            <dt class="text-xs font-medium uppercase tracking-wider text-slate-500">
              Region
            </dt>
            <dd class="mt-1 text-sm text-slate-100">{{ cluster.region }}</dd>
          </div>
          <div>
            <dt class="text-xs font-medium uppercase tracking-wider text-slate-500">
              Kubernetes
            </dt>
            <dd class="mt-1 font-mono text-sm text-slate-100">
              {{ cluster.k8sVersion }}
            </dd>
          </div>
          <div>
            <dt class="text-xs font-medium uppercase tracking-wider text-slate-500">
              API port
            </dt>
            <dd class="mt-1 font-mono text-sm text-slate-100">
              {{ cluster.apiPort ?? '—' }}
            </dd>
          </div>
          <div>
            <dt class="text-xs font-medium uppercase tracking-wider text-slate-500">
              Created
            </dt>
            <dd class="mt-1 text-sm text-slate-100">
              {{ formatDate(cluster.createdAt) }}
            </dd>
          </div>
          <div>
            <dt class="text-xs font-medium uppercase tracking-wider text-slate-500">
              Updated
            </dt>
            <dd class="mt-1 text-sm text-slate-100">
              {{ formatDate(cluster.updatedAt) }}
            </dd>
          </div>
        </dl>
      </Card>

      <Card title="Scale" description="Adjust node count for this cluster.">
        <div class="space-y-4">
          <div>
            <label class="mb-1.5 block text-xs font-medium text-slate-300">
              Nodes: <span class="text-slate-100">{{ scaleValue }}</span>
            </label>
            <input
              v-model.number="scaleValue"
              type="range"
              min="1"
              max="5"
              step="1"
              class="h-2 w-full cursor-pointer appearance-none rounded-lg bg-slate-800 accent-brand-500"
            />
            <div class="mt-1 flex justify-between text-xs text-slate-500">
              <span>1</span><span>2</span><span>3</span><span>4</span><span>5</span>
            </div>
          </div>
          <Button
            :loading="scaling"
            :disabled="cluster.status !== 'Running'"
            full-width
            @click="applyScale"
          >
            Apply scale
          </Button>
          <p
            v-if="cluster.status !== 'Running'"
            class="text-xs text-slate-500"
          >
            Scaling is only available when cluster is Running.
          </p>
        </div>
      </Card>
    </div>

    <!-- Kubeconfig tab -->
    <div v-if="cluster && activeTab === 'kubeconfig'">
      <Card
        title="kubeconfig"
        description="Use this file with kubectl to access your cluster."
      >
        <template #actions>
          <Button
            variant="outline"
            size="sm"
            :loading="loadingKubeconfig"
            @click="downloadKube"
          >
            <Download class="h-4 w-4" />
            Download
          </Button>
        </template>
        <div v-if="loadingKubeconfig && !kubeconfig" class="space-y-2">
          <div class="skeleton h-4 w-full" />
          <div class="skeleton h-4 w-4/5" />
          <div class="skeleton h-4 w-3/5" />
        </div>
        <CodeBlock
          v-else-if="kubeconfig"
          :code="kubeconfig"
          language="yaml"
          max-height="500px"
        />
        <p v-else class="text-sm text-slate-400">
          Kubeconfig is only available once the cluster is Running.
        </p>
      </Card>
    </div>

    <!-- YAML tab -->
    <div v-if="cluster && activeTab === 'yaml'">
      <Card
        title="Cluster YAML"
        description="Infrastructure-as-code definition for this cluster."
      >
        <template #actions>
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
        <div v-if="loadingYaml && !yaml" class="space-y-2">
          <div class="skeleton h-4 w-full" />
          <div class="skeleton h-4 w-4/5" />
          <div class="skeleton h-4 w-3/5" />
        </div>
        <CodeBlock
          v-else-if="yaml"
          :code="yaml"
          language="yaml"
          max-height="500px"
        />
        <p v-else class="text-sm text-slate-400">No YAML available.</p>
      </Card>
    </div>

    <!-- Nodes tab (placeholder) -->
    <div v-if="cluster && activeTab === 'nodes'">
      <Card title="Nodes" description="Worker nodes powering this cluster.">
        <div class="rounded-lg border border-dashed border-slate-800 p-10 text-center">
          <Server class="mx-auto h-8 w-8 text-slate-600" />
          <p class="mt-3 text-sm text-slate-300">
            Per-node metrics will be available soon.
          </p>
          <p class="mt-1 text-xs text-slate-500">
            This cluster currently runs {{ cluster.nodeCount }} node{{
              cluster.nodeCount === 1 ? '' : 's'
            }}.
          </p>
        </div>
      </Card>
    </div>

    <ConfirmDialog
      :open="confirmDelete"
      title="Delete cluster?"
      :description="`This will permanently delete '${cluster?.name}' and all its workloads. This action cannot be undone.`"
      confirm-text="Delete cluster"
      variant="danger"
      :loading="deleting"
      @close="confirmDelete = false"
      @confirm="doDelete"
    />
  </div>
</template>
