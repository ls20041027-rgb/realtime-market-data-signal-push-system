import { computed, onScopeDispose, ref, shallowRef, watch, type Ref } from 'vue'
import { fetchQuote } from '@/api/quote'
import { fetchIndicators, fetchCapital } from '@/api/indicator'
import { useQuoteStore } from '@/stores/quote'
import { wsClient } from '@/ws/client'
import { CH } from '@/ws/channels'
import { useAutoRefresh } from '@/composables/useAutoRefresh'
import { createLogger } from '@/utils/logger'
import type {
  CapitalSnapshot,
  IndicatorSnapshot,
  QuoteSnapshot,
  TradingSignal,
  WSFrame,
} from '@/types'

const log = createLogger('stock:detail')

const LIVE_SIGNAL_MAX = 20

export interface UseStockDetailReturn {
  quote: Ref<QuoteSnapshot | null>
  indicators: Ref<IndicatorSnapshot | null>
  capital: Ref<CapitalSnapshot | null>
  liveSignals: Ref<TradingSignal[]>
  loadingQuote: Ref<boolean>
  loadingIndicators: Ref<boolean>
  loadingCapital: Ref<boolean>
  refreshAll: () => Promise<void>
}

export function useStockDetail(symbolRef: Ref<string>): UseStockDetailReturn {
  const quoteStore = useQuoteStore()

  const indicators = ref<IndicatorSnapshot | null>(null)
  const capital = ref<CapitalSnapshot | null>(null)
  const liveSignals = ref<TradingSignal[]>([])

  const loadingQuote = ref(false)
  const loadingIndicators = ref(false)
  const loadingCapital = ref(false)

  const subscribed = shallowRef(new Map<string, (f: WSFrame) => void>())

  const quote = computed(() => quoteStore.get(symbolRef.value) ?? null)

  const refreshQuote = async () => {
    const sym = symbolRef.value
    if (!sym) return
    loadingQuote.value = true
    try {
      const q = await fetchQuote(sym)
      quoteStore.set(q)
    } catch (err) {
      log.error('fetchQuote failed', err)
    } finally {
      loadingQuote.value = false
    }
  }

  const refreshIndicators = async () => {
    const sym = symbolRef.value
    if (!sym) return
    loadingIndicators.value = true
    try {
      indicators.value = await fetchIndicators(sym)
    } catch (err) {
      log.error('fetchIndicators failed', err)
    } finally {
      loadingIndicators.value = false
    }
  }

  const refreshCapital = async () => {
    const sym = symbolRef.value
    if (!sym) return
    loadingCapital.value = true
    try {
      capital.value = await fetchCapital(sym)
    } catch (err) {
      log.error('fetchCapital failed', err)
    } finally {
      loadingCapital.value = false
    }
  }

  const refreshAll = async () => {
    await Promise.all([refreshQuote(), refreshIndicators(), refreshCapital()])
  }

  const unsubscribeAll = () => {
    for (const [ch, handler] of subscribed.value) {
      wsClient.unsubscribe(ch, handler)
    }
    subscribed.value = new Map()
  }

  const subscribeSymbol = (sym: string) => {
    if (!sym) return
    const quoteCh = CH.quote(sym)
    const signalCh = CH.signal(sym)
    const map = subscribed.value

    if (!map.has(quoteCh)) {
      const h = (frame: WSFrame) => {
        const snap = frame.data as QuoteSnapshot | undefined
        if (snap && snap.symbol) quoteStore.set(snap)
      }
      wsClient.subscribe(quoteCh, h)
      map.set(quoteCh, h)
    }
    if (!map.has(signalCh)) {
      const h = (frame: WSFrame) => {
        const sig = frame.data as TradingSignal | undefined
        if (!sig) return
        liveSignals.value = [sig, ...liveSignals.value].slice(0, LIVE_SIGNAL_MAX)
      }
      wsClient.subscribe(signalCh, h)
      map.set(signalCh, h)
    }
  }

  const poll = useAutoRefresh(
    async () => {
      await Promise.all([refreshIndicators(), refreshCapital()])
    },
    { intervalMs: 3000, immediate: false },
  )

  wsClient.connect()
  watch(
    symbolRef,
    (next, prev) => {
      if (prev) {
        unsubscribeAll()
        liveSignals.value = []
        indicators.value = null
        capital.value = null
      }
      if (!next) return
      subscribeSymbol(next)
      void refreshAll()
      poll.start()
    },
    { immediate: true },
  )

  onScopeDispose(() => {
    unsubscribeAll()
    poll.stop()
  })

  return {
    quote,
    indicators,
    capital,
    liveSignals,
    loadingQuote,
    loadingIndicators,
    loadingCapital,
    refreshAll,
  }
}
