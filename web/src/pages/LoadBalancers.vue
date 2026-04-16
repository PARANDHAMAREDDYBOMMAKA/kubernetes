<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Plus, Eye, Trash2, Network, RefreshCw } from 'lucide-vue-next'
import { useLoadBalancersStore } from '@/stores/loadbalancers'
import { useClustersStore } from '@/stores/clusters'
import { useToastStore } from '@/stores/toast'
import { extractErrorMessage } from '@/lib/api'
import Button from '@/components/ui/Button.vue'
import Card from '@/components/ui/Card.vue'
import Table from '@/components/ui/Table.vue'
import SkeletonRow from '@/components/ui/SkeletonRow.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import CreateLoadBalancerModal from '@/components/CreateLoadBalancerModal.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import { formatRelative } from '@/lib/utils'

const lbStore = useLoadBalancersStore()
const clusterStore = useClustersStore()
const toast = useToastStore()
const router = useRouter()

const createOpen = ref(false)
const confirmDeleteId = ref<string | null>(null)
const confirmDeleteName = ref('')
const deleting = ref(false)
const pollTimer = ref<number | null>(null)

async function refresh(silent = false) {
  try {
    await lbStore.fetchAll(silent)
  } catch (err) {
    if (!silent)
      toast.error('Failed to load load balancers', extractErrorMessage(err))
  }
}

function schedulePolling() {
  if (pollTimer.value) return
  pollTimer.value = window.setInterval(() => {
    if (lbStore.hasPendingOps) refresh(true)
  }, 5000)
}

watch(
  () => lbStore.hasPendingOps,
  (p) => {
    if (p) schedulePolling()
  }
)

onMounted(() => {
  refresh()
  if (clusterStore.list.length === 0) clusterStore.fetchAll(true).catch(() => {})
  schedulePolling()
})

onBeforeUnmount(() => {
  if (pollTimer.value) {
    clearInterval(pollTimer.value)
    pollTimer.value = null
  }
})

const hasItems = computed(() => lbStore.list.length > 0)

function clusterName(id: string) {
  const c = clusterStore.list.find((x) => x.id === id)
  return c?.name || id.slice(0, 8)
}

function openDetail(id: string) {
  router.push({ name: 'loadbalancer-detail', params: { id } })
}

function askDelete(id: string, name: string) {
  confirmDeleteId.value = id
  confirmDeleteName.value = name
}

async function doDelete() {
  if (!confirmDeleteId.value) return
  deleting.value = true
  try {
    await lbStore.remove(confirmDeleteId.value)
    toast.success('Load balancer deletion started')
    confirmDeleteId.value = null
    schedulePolling()
  } catch (err) {
    toast.error('Delete failed', extractErrorMessage(err))
  } finally {
    deleting.value = false
  }
}
</script>

<template>
  <div class="space-y-6">
    <header class="flex flex-wrap items-center justify-between gap-4">
      <div>
        <h1 class="text-2xl font-semibold text-white">Load Balancers</h1>
        <p class="mt-1 text-sm text-slate-400">
          HTTP/TCP load balancers routing traffic to your cluster workloads.
        </p>
      </div>
      <div class="flex items-center gap-2">
        <Button
          variant="outline"
          :disabled="lbStore.loadingList"
          @click="refresh(false)"
        >
          <RefreshCw
            class="h-4 w-4"
            :class="lbStore.loadingList ? 'animate-spin' : ''"
          />
          Refresh
        </Button>
        <Button @click="createOpen = true">
          <Plus class="h-4 w-4" />
          Create load balancer
        </Button>
      </div>
    </header>

    <Card padded>
      <Table v-if="lbStore.loadingList || hasItems">
        <thead>
          <tr>
            <th>Name</th>
            <th>Cluster</th>
            <th>Port</th>
            <th>Algorithm</th>
            <th>Backends</th>
            <th>Status</th>
            <th>Created</th>
            <th class="text-right">Actions</th>
          </tr>
        </thead>
        <tbody>
          <SkeletonRow v-if="lbStore.loadingList" :columns="8" :rows="4" />
          <tr
            v-for="lb in lbStore.list"
            v-else
            :key="lb.id"
            class="cursor-pointer"
            @click="openDetail(lb.id)"
          >
            <td>
              <div class="font-medium text-white">{{ lb.name }}</div>
              <div class="text-xs text-slate-500">{{ lb.id }}</div>
            </td>
            <td class="text-slate-300">{{ clusterName(lb.clusterId) }}</td>
            <td class="font-mono text-xs text-slate-300">
              {{ lb.port }} → {{ lb.targetPort }}
            </td>
            <td class="text-slate-300">{{ lb.algorithm }}</td>
            <td class="text-slate-300">{{ lb.backends?.length ?? 0 }}</td>
            <td>
              <StatusBadge :status="lb.status" />
            </td>
            <td class="text-slate-400">{{ formatRelative(lb.createdAt) }}</td>
            <td class="text-right" @click.stop>
              <div class="inline-flex items-center gap-1">
                <button
                  type="button"
                  class="rounded p-1.5 text-slate-400 hover:bg-slate-800 hover:text-slate-100"
                  title="View"
                  @click="openDetail(lb.id)"
                >
                  <Eye class="h-4 w-4" />
                </button>
                <button
                  type="button"
                  class="rounded p-1.5 text-slate-400 hover:bg-red-500/10 hover:text-red-300 disabled:opacity-40"
                  title="Delete"
                  :disabled="lb.status === 'Deleting'"
                  @click="askDelete(lb.id, lb.name)"
                >
                  <Trash2 class="h-4 w-4" />
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </Table>

      <EmptyState
        v-else
        title="No load balancers yet"
        description="Create a load balancer to expose services running in your clusters."
        :icon="Network"
      >
        <Button @click="createOpen = true">
          <Plus class="h-4 w-4" />
          Create load balancer
        </Button>
      </EmptyState>
    </Card>

    <CreateLoadBalancerModal
      :open="createOpen"
      @close="createOpen = false"
      @created="refresh(true)"
    />

    <ConfirmDialog
      :open="!!confirmDeleteId"
      title="Delete load balancer?"
      :description="`This will remove '${confirmDeleteName}' and stop routing traffic. This action cannot be undone.`"
      confirm-text="Delete"
      variant="danger"
      :loading="deleting"
      @close="confirmDeleteId = null"
      @confirm="doDelete"
    />
  </div>
</template>
