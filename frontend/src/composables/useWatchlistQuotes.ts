import { computed, onScopeDispose, ref, shallowRef, watch } from 'vue'
import type { ComputedRef } from 'vue'
import { useWatchlistStore } from '@/stores/watchlist'
import { useQuoteStore } from '@/stores/quote'
import { useMetaStore } from '@/stores/meta'
import { fetchQuotes } from '@/api/quote'
import { wsClient } from '@/ws/client'
import { CH } from '@/ws/channels'
import { D } from '@/composables/useDecimal'
import { createLogger } from '@/utils/logger'
import type { QuoteSnapshot, WSFrame } from '@/types'

const log = createLogger('watchlist:quotes')

export interface WatchRow {
  symbol: string
  name: string
  last_price: string
  prev_close: string
  change_pct: string
  change_amt: string
  volume: string
  turnover: string
  event_time: string
}

export interface UseWatchlistQuotesReturn {
  rows: ComputedRef<WatchRow[]>
  refreshing: ComputedRef<boolean>
  refresh: () => Promise<void>
}

export function useWatchlistQuotes(): UseWatchlistQuotesReturn {
  const watchStore = useWatchlistStore()
  const quoteStore = useQuoteStore()
  const metaStore = useMetaStore()

  const refreshing = ref(false)
  const subscribed = shallowRef(new Map<string, (f: WSFrame) => void>())

  const refresh = async () => {
    const syms = [...watchStore.symbols]
    if (syms.length === 0) return
    refreshing.value = true
    try {
      const list = await fetchQuotes(syms)
      quoteStore.setMany(list)
    } catch (err) {
      log.error('fetchQuotes failed', err)
    } finally {
      refreshing.value = false
    }
  }

  const reconcileSubscriptions = (next: string[]) => {
    const map = subscribed.value
    const nextSet = new Set<string>(next.map((s) => CH.quote(s)))

    for (const [ch, handler] of map) {
      if (!nextSet.has(ch as ReturnType<typeof CH.quote>)) {
        wsClient.unsubscribe(ch, handler)
        map.delete(ch)
      }
    }
    for (const sym of next) {
      const ch = CH.quote(sym)
      if (map.has(ch)) continue
      const handler = (frame: WSFrame) => {
        const snap = frame.data as QuoteSnapshot | undefined
        if (snap && snap.symbol) quoteStore.set(snap)
      }
      wsClient.subscribe(ch, handler)
      map.set(ch, handler)
    }
  }

  wsClient.connect()
  void metaStore.load()

  watch(
    () => [...watchStore.symbols],
    (next, prev) => {
      reconcileSubscriptions(next)
      const prevSet = new Set(prev || [])
      const changed =
        !prev || next.length !== prev.length || next.some((s) => !prevSet.has(s))
      if (changed) void refresh()
    },
    { immediate: true },
  )

  onScopeDispose(() => {
    for (const [ch, handler] of subscribed.value) {
      wsClient.unsubscribe(ch, handler)
    }
    subscribed.value = new Map()
  })

  const rows = computed<WatchRow[]>(() => {
    return watchStore.symbols.map((sym) => {
      const q = quoteStore.get(sym)
      if (!q) {
        return {
          symbol: sym,
          name: metaStore.nameOf(sym),
          last_price: '-',
          prev_close: '-',
          change_pct: '0',
          change_amt: '0',
          volume: '-',
          turnover: '-',
          event_time: '',
        }
      }
      const prev = D(q.prev_close)
      const last = D(q.last_price)
      const changeAmt = last.minus(prev)
      const changePct = prev.gt(0) ? changeAmt.div(prev).mul(100) : D(0)
      return {
        symbol: q.symbol,
        name: metaStore.nameOf(q.symbol),
        last_price: q.last_price,
        prev_close: q.prev_close,
        change_pct: changePct.toFixed(2),
        change_amt: changeAmt.toFixed(2),
        volume: q.volume,
        turnover: q.turnover,
        event_time: q.event_time,
      }
    })
  })

  return {
    rows,
    refreshing: computed(() => refreshing.value),
    refresh,
  }
}
