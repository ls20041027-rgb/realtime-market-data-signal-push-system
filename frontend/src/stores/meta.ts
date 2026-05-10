import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { fetchStockList } from '@/api/meta'
import { createLogger } from '@/utils/logger'

const log = createLogger('store:meta')

export const useMetaStore = defineStore('meta', () => {
  const nameMap = ref<Record<string, string>>({})
  const loaded = ref(false)
  const loading = ref(false)

  const nameOf = (symbol: string) => nameMap.value[symbol] || symbol

  const load = async () => {
    if (loaded.value || loading.value) return
    loading.value = true
    try {
      nameMap.value = await fetchStockList()
      loaded.value = true
    } catch (err) {
      log.error('stock-list load failed', err)
    } finally {
      loading.value = false
    }
  }

  const search = (keyword: string, limit = 20) => {
    const kw = keyword.trim().toUpperCase()
    if (!kw) return [] as Array<{ symbol: string; name: string }>
    const out: Array<{ symbol: string; name: string }> = []
    for (const [symbol, name] of Object.entries(nameMap.value)) {
      if (symbol.includes(kw) || name.includes(keyword)) {
        out.push({ symbol, name })
        if (out.length >= limit) break
      }
    }
    return out
  }

  const count = computed(() => Object.keys(nameMap.value).length)

  return { nameMap, loaded, loading, count, load, nameOf, search }
})
