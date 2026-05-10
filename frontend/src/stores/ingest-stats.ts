import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchIngestStats } from '@/api/status'
import { createLogger } from '@/utils/logger'
import type { IngestStats } from '@/types'

const log = createLogger('store:ingest-stats')

export const useIngestStatsStore = defineStore('ingest-stats', () => {
  const snapshot = ref<IngestStats | null>(null)
  const prevCounts = ref<Record<string, number>>({})
  const prevNowMs = ref<number>(0)
  const qps = ref<Record<string, number>>({})
  const totalQps = ref<number>(0)

  const loading = ref(false)
  const updatedAt = ref(0)
  const error = ref<string | null>(null)

  const flatten = (snap: IngestStats): Record<string, number> => {
    const out: Record<string, number> = {}
    for (const it of snap.message_types) {
      out[`mt:${it.message_type ?? ''}`] = it.count
    }
    for (const it of snap.file_data_types) {
      out[`ft:${it.file_data_type ?? ''}`] = it.count
    }
    return out
  }

  const refresh = async (): Promise<void> => {
    if (loading.value) return
    loading.value = true
    try {
      const snap = await fetchIngestStats()
      const curr = flatten(snap)
      const currNow = snap.now_ms || Date.now()
      const prevNow = prevNowMs.value

      if (prevNow > 0 && currNow > prevNow) {
        const dt = (currNow - prevNow) / 1000
        const q: Record<string, number> = {}
        let total = 0
        for (const [k, v] of Object.entries(curr)) {
          const delta = Math.max(0, v - (prevCounts.value[k] ?? 0))
          const rate = delta / dt
          q[k] = rate
          if (k.startsWith('mt:')) total += rate
        }
        qps.value = q
        totalQps.value = total
      }

      prevCounts.value = curr
      prevNowMs.value = currNow
      snapshot.value = snap
      updatedAt.value = Date.now()
      error.value = snap.error ?? null
    } catch (err) {
      error.value = (err as Error)?.message ?? 'unknown'
      log.error('fetchIngestStats failed', err)
    } finally {
      loading.value = false
    }
  }

  return { snapshot, qps, totalQps, loading, updatedAt, error, refresh }
})
