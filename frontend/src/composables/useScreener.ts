import { computed, ref } from 'vue'
import type { ComputedRef, Ref } from 'vue'
import Decimal from 'decimal.js'
import { fetchQuotes } from '@/api/quote'
import { D } from '@/composables/useDecimal'
import { createLogger } from '@/utils/logger'
import type { QuoteSnapshot } from '@/types'

const log = createLogger('screener')

const BATCH_SIZE = 200
const BATCH_CONCURRENCY = 3

export interface BoardRow {
  symbol: string
  last_price: string
  prev_close: string
  open_price: string
  high_price: string
  low_price: string
  volume: string
  turnover: string
  change_pct: Decimal
  amplitude: Decimal
  turnover_d: Decimal
}

const toRow = (q: QuoteSnapshot): BoardRow => {
  const prev = D(q.prev_close)
  const last = D(q.last_price)
  const high = D(q.high_price)
  const low = D(q.low_price)
  const changePct = prev.lte(0) ? new Decimal(0) : last.sub(prev).div(prev)
  const amplitude = prev.lte(0) ? new Decimal(0) : high.sub(low).div(prev)
  return {
    symbol: q.symbol,
    last_price: q.last_price,
    prev_close: q.prev_close,
    open_price: q.open_price,
    high_price: q.high_price,
    low_price: q.low_price,
    volume: q.volume,
    turnover: q.turnover,
    change_pct: changePct,
    amplitude,
    turnover_d: D(q.turnover),
  }
}

const loadAll = async (symbols: string[]): Promise<QuoteSnapshot[]> => {
  const batches: string[][] = []
  for (let i = 0; i < symbols.length; i += BATCH_SIZE) {
    batches.push(symbols.slice(i, i + BATCH_SIZE))
  }
  const out: QuoteSnapshot[] = []
  for (let i = 0; i < batches.length; i += BATCH_CONCURRENCY) {
    const group = batches.slice(i, i + BATCH_CONCURRENCY)
    const results = await Promise.allSettled(group.map((b) => fetchQuotes(b)))
    for (const r of results) {
      if (r.status === 'fulfilled') {
        if (Array.isArray(r.value)) {
          out.push(...r.value)
        } else {
          log.warn('fetchQuotes returned non-array, skip', { value: r.value })
        }
      } else {
        log.warn('batch fetchQuotes failed', r.reason)
      }
    }
  }
  return out
}

export interface UseScreenerReturn {
  rows: ComputedRef<BoardRow[]>
  loading: Ref<boolean>
  updatedAt: Ref<number>
  error: Ref<string | null>
  load: (symbols: string[]) => Promise<void>
}

export function useScreener(): UseScreenerReturn {
  const raw = ref<BoardRow[]>([])
  const loading = ref(false)
  const updatedAt = ref(0)
  const error = ref<string | null>(null)

  const load = async (symbols: string[]) => {
    if (loading.value) return
    error.value = null
    if (symbols.length === 0) {
      raw.value = []
      return
    }
    loading.value = true
    try {
      const quotes = await loadAll(symbols)
      raw.value = quotes
        .filter((q) => q && q.symbol && q.prev_close)
        .map(toRow)
      updatedAt.value = Date.now()
    } catch (err) {
      log.error('screener load failed', err)
      error.value = (err as Error)?.message ?? 'unknown'
    } finally {
      loading.value = false
    }
  }

  const rows = computed(() => raw.value)
  return { rows, loading, updatedAt, error, load }
}
