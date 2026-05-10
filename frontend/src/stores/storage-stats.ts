import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchStorageStats } from '@/api/status'
import { createLogger } from '@/utils/logger'
import type { StorageStats } from '@/types'

const log = createLogger('store:storage-stats')

export const useStorageStatsStore = defineStore('storage-stats', () => {
  const snapshot = ref<StorageStats | null>(null)
  const loading = ref(false)
  const updatedAt = ref(0)
  const error = ref<string | null>(null)

  const refresh = async (): Promise<void> => {
    if (loading.value) return
    loading.value = true
    try {
      snapshot.value = await fetchStorageStats()
      updatedAt.value = Date.now()
      error.value = null
    } catch (err) {
      error.value = (err as Error)?.message ?? 'unknown'
      log.error('fetchStorageStats failed', err)
    } finally {
      loading.value = false
    }
  }

  return { snapshot, loading, updatedAt, error, refresh }
})
