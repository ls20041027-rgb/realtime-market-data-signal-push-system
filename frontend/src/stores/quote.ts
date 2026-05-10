import { defineStore } from 'pinia'
import { markRaw, ref } from 'vue'
import type { QuoteSnapshot } from '@/types'

const MAX_SIZE = 500

export const useQuoteStore = defineStore('quote', () => {
  const map = ref(new Map<string, QuoteSnapshot>())

  const get = (symbol: string): QuoteSnapshot | undefined => map.value.get(symbol)

  const set = (snap: QuoteSnapshot) => {
    const m = map.value
    if (!m.has(snap.symbol) && m.size >= MAX_SIZE) {
      const firstKey = m.keys().next().value
      if (firstKey) m.delete(firstKey)
    }
    m.set(snap.symbol, markRaw(snap))
    map.value = new Map(m)
  }

  const setMany = (list: QuoteSnapshot[]) => {
    list.forEach((s) => set(s))
  }

  return { map, get, set, setMany }
})
