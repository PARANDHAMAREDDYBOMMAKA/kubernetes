<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Plus, Eye, Trash2, LayoutGrid, RefreshCw } from 'lucide-vue-next'
import { useClustersStore } from '@/stores/clusters'
import { useToastStore } from '@/stores/toast'
import { extractErrorMessage } from '@/lib/api'
import Button from '@/components/ui/Button.vue'
import Table from '@/components/ui/Table.vue'
import Card from '@/components/ui/Card.vue'
import SkeletonRow from '@/components/ui/SkeletonRow.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import CreateClusterModal from '@/components/CreateClusterModal.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import { formatRelative } from '@/lib/utils'

const clusters = useClustersStore()
const toast = useToastStore()
const router = useRouter()

const createOpen = ref(false)
const confirmDeleteId = ref<string | null>(null)
const confirmDeleteName = ref('')
const deleting = ref(false)

const pollTimer = ref<number | null>(null)

async function refresh(silent = false) {
  try {
    await clusters.fetchAll(silent)
  } catch (err) {
    if (!silent) toast.error('Failed to load clusters', extractErrorMessage(err))
  }
}

function schedulePolling() {
  if (pollTimer.value) return
  pollTimer.value = window.setInterval(() => {
    if (clusters.hasPendingOps) refresh(true)
  }, 5000)
}

watch(
  () => clusters.hasPendingOps,
  (pending) => {
    if (pending) schedulePolling()
  },
  { immediate: true }
)

onMounted(() => {
  refresh()
  schedulePolling()
})

onBeforeUnmount(() => {
  if (pollTimer.value) {
    clearInterval(pollTimer.value)
    pollTimer.value = null
  }
})

const hasClusters = computed(() => clusters.list.length > 0)

function openDetail(id: string) {
  router.push({ name: 'cluster-detail', params: { id } })
}

function askDelete(id: string, name: string) {
  confirmDeleteId.value = id
  confirmDeleteName.value = name
}

async function doDelete() {
  if (!confirmDeleteId.value) return
  deleting.value = true
  try {
    await clusters.remove(confirmDeleteId.value)
    toast.success('Cluster deletion started')
    confirmDeleteId.value = null
    schedulePolling()
  } catch (err) {
    toast.error('Failed to delete', extractErrorMessage(err))
  } finally {
    deleting.value = false
  }
}
</script>

<template>
  <div class="space-y-6">
    <header class="flex flex-wrap items-center justify-between gap-4">
      <div>
        <h1 class="text-xl font-semibold text-slate-100">
          <span class="text-brand-400">#</span> clusters
        </h1>
        <p class="mt-1 text-sm text-slate-500">
          managed k3s clusters in your account
        </p>
      </div>
      <div class="flex items-center gap-2">
        <Button
          variant="outline"
          size="md"
          :disabled="clusters.loadingList"
          @click="refresh(false)"
        >
          <RefreshCw
            class="h-4 w-4"
            :class="clusters.loadingList ? 'animate-spin' : ''"
          />
          Refresh
        </Button>
        <Button @click="createOpen = true">
          <Plus class="h-4 w-4" />
          Create cluster
        </Button>
      </div>
    </header>

    <Card padded>
      <Table v-if="clusters.loadingList || hasClusters">
        <thead>
          <tr>
            <th>Name</th>
            <th>K8s version</th>
            <th>Region</th>
            <th>Nodes</th>
            <th>Status</th>
            <th>Created</th>
            <th class="text-right">Actions</th>
          </tr>
        </thead>
        <tbody>
          <SkeletonRow v-if="clusters.loadingList" :columns="7" :rows="4" />
          <tr
            v-for="c in clusters.list"
            v-else
            :key="c.id"
            class="cursor-pointer"
            @click="openDetail(c.id)"
          >
            <td>
              <div class="font-medium text-slate-100">{{ c.name }}</div>
              <div class="text-xs text-slate-600">{{ c.id }}</div>
            </td>
            <td class="text-xs text-brand-300/80">{{ c.k8sVersion }}</td>
            <td class="text-slate-300">{{ c.region }}</td>
            <td class="text-slate-300">{{ c.nodeCount }}</td>
            <td>
              <StatusBadge :status="c.status" />
            </td>
            <td class="text-slate-400">{{ formatRelative(c.createdAt) }}</td>
            <td class="text-right" @click.stop>
              <div class="inline-flex items-center gap-1">
                <button
                  type="button"
                  class="rounded-sm p-1.5 text-slate-400 hover:bg-brand-500/10 hover:text-brand-300"
                  title="View"
                  @click="openDetail(c.id)"
                >
                  <Eye class="h-4 w-4" />
                </button>
                <button
                  type="button"
                  class="rounded-sm p-1.5 text-slate-400 hover:bg-red-500/10 hover:text-red-300 disabled:opacity-40"
                  title="Delete"
                  :disabled="c.status === 'Deleting' || c.status === 'Deleted'"
                  @click="askDelete(c.id, c.name)"
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
        title="No clusters yet"
        description="Create your first managed Kubernetes cluster to get started."
        :icon="LayoutGrid"
      >
        <Button @click="createOpen = true">
          <Plus class="h-4 w-4" />
          Create your first cluster
        </Button>
      </EmptyState>
    </Card>

    <CreateClusterModal
      :open="createOpen"
      @close="createOpen = false"
      @created="refresh(true)"
    />

    <ConfirmDialog
      :open="!!confirmDeleteId"
      title="Delete cluster?"
      :description="`This will permanently delete '${confirmDeleteName}' and all its workloads. This action cannot be undone.`"
      confirm-text="Delete cluster"
      variant="danger"
      :loading="deleting"
      @close="confirmDeleteId = null"
      @confirm="doDelete"
    />
  </div>
</template>
