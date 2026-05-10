import { describe, expect, it, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

vi.mock('@/api/quote', () => ({
  fetchQuotes: vi.fn(async (symbols: string[]) =>
    symbols.map((s, i) => ({
      symbol: s,
      last_price: '11',
      prev_close: '10',
      open_price: '10',
      high_price: '12',
      low_price: '9',
      volume: '100',
      turnover: String((i + 1) * 1000),
      bid_levels: [],
      ask_levels: [],
      event_time: '2026-04-26T10:00:00Z',
    })),
  ),
}))

import { useScreener } from '@/composables/useScreener'

describe('useScreener', () => {
  it('derives change_pct/amplitude and merges batches', async () => {
    setActivePinia(createPinia())
    const { rows, load } = useScreener()
    const symbols = Array.from({ length: 250 }, (_, i) => `SH${600000 + i}`)
    await load(symbols)

    expect(rows.value.length).toBe(250)
    const r = rows.value[0]
    expect(r.change_pct.toFixed(2)).toBe('0.10')
    expect(r.amplitude.toFixed(2)).toBe('0.30')
  })

  it('handles empty input without fetch', async () => {
    setActivePinia(createPinia())
    const { rows, load } = useScreener()
    await load([])
    expect(rows.value.length).toBe(0)
  })
})
