import { defineStore } from 'pinia'
import { ref, watch } from 'vue'
import { isValidSymbol, normalizeSymbol } from '@/utils/symbol'

const STORAGE_KEY = 'tornado:watchlist'

const loadFromStorage = (): string[] => {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const arr = JSON.parse(raw)
    return Array.isArray(arr) ? arr.filter(isValidSymbol) : []
  } catch {
    return []
  }
}

export const useWatchlistStore = defineStore('watchlist', () => {
  const symbols = ref<string[]>(loadFromStorage())

  watch(
    symbols,
    (v) => {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(v))
    },
    { deep: true },
  )

  const has = (s: string) => symbols.value.includes(normalizeSymbol(s))

  const add = (s: string) => {
    const sym = normalizeSymbol(s)
    if (!isValidSymbol(sym) || has(sym)) return
    symbols.value = [...symbols.value, sym]
  }

  const remove = (s: string) => {
    const sym = normalizeSymbol(s)
    symbols.value = symbols.value.filter((x) => x !== sym)
  }

  const toggle = (s: string) => (has(s) ? remove(s) : add(s))

  return { symbols, has, add, remove, toggle }
})
