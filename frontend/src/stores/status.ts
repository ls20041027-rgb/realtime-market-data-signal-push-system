import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchStatus } from '@/api/status'
import { createLogger } from '@/utils/logger'
import type { StatusSnapshot } from '@/types'

const log = createLogger('store:status')

export const useStatusStore = defineStore('status', () => {
  const snapshot = ref<StatusSnapshot | null>(null)
  const loading = ref(false)
  const updatedAt = ref(0)
  const error = ref<string | null>(null)

  const refresh = async (): Promise<void> => {
    if (loading.value) return
    loading.value = true
    try {
      snapshot.value = await fetchStatus()
      updatedAt.value = Date.now()
      error.value = null
    } catch (err) {
      error.value = (err as Error)?.message ?? 'unknown'
      log.error('fetchStatus failed', err)
    } finally {
      loading.value = false
    }
  }

  return { snapshot, loading, updatedAt, error, refresh }
})
