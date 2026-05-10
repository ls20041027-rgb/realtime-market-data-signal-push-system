import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'

vi.mock('@/api/quote', () => ({
  fetchQuotes: vi.fn(async (syms: string[]) =>
    syms.map((s) => ({
      symbol: s,
      last_price: '10.00',
      prev_close: '9.80',
      open_price: '9.90',
      high_price: '10.10',
      low_price: '9.70',
      volume: '1000',
      turnover: '123456789',
      bid_levels: [],
      ask_levels: [],
      event_time: '2026-04-26T10:00:00+08:00',
    })),
  ),
  fetchQuote: vi.fn(),
  fetchFenbi: vi.fn(),
}))

const subCount = new Map<string, number>()
vi.mock('@/ws/client', () => ({
  wsClient: {
    connect: vi.fn(),
    subscribe: vi.fn((ch: string) => {
      subCount.set(ch, (subCount.get(ch) || 0) + 1)
    }),
    unsubscribe: vi.fn((ch: string) => {
      subCount.set(ch, (subCount.get(ch) || 0) - 1)
    }),
    getStatus: () => 'OPEN',
    onStatus: vi.fn(),
  },
}))

vi.mock('@/api/meta', () => ({
  fetchStockList: vi.fn(async () => ({})),
}))

import { D, fmtAmt, fmtPct, fmtPrice, trendClass } from '@/composables/useDecimal'
import { useWatchlistStore } from '@/stores/watchlist'
import { useWatchlistQuotes } from '@/composables/useWatchlistQuotes'
import { effectScope } from 'vue'

describe('useDecimal', () => {
  it('fmtPrice/fmtPct/fmtAmt/trendClass 典型路径', () => {
    expect(fmtPrice('10.1234')).toBe('10.12')
    expect(fmtPct('0.0125')).toBe('1.25%')
    expect(fmtAmt('123456789')).toBe('1.23亿')
    expect(fmtAmt('50000')).toBe('5.00万')
    expect(trendClass('0.01')).toBe('color-up')
    expect(trendClass('-0.01')).toBe('color-down')
    expect(trendClass('0')).toBe('color-neutral')
    expect(D(null).toString()).toBe('0')
  })
})

describe('useWatchlistQuotes WS 订阅差量', () => {
  beforeEach(() => {
    localStorage.clear()
    subCount.clear()
    setActivePinia(createPinia())
  })

  it('add / remove symbol 时 subscribe / unsubscribe 正确计数', async () => {
    const watchStore = useWatchlistStore()
    watchStore.add('SH600000')

    const scope = effectScope()
    scope.run(() => {
      useWatchlistQuotes()
    })
    await nextTick()
    expect(subCount.get('quote:SH600000')).toBe(1)

    watchStore.add('SZ000001')
    await nextTick()
    expect(subCount.get('quote:SH600000')).toBe(1)
    expect(subCount.get('quote:SZ000001')).toBe(1)

    watchStore.remove('SH600000')
    await nextTick()
    expect(subCount.get('quote:SH600000')).toBe(0)
    expect(subCount.get('quote:SZ000001')).toBe(1)

    scope.stop()
    await nextTick()
    expect(subCount.get('quote:SZ000001')).toBe(0)
  })
})
