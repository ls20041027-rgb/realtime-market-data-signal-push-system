import { http } from './http'
import type { FenbiTick, QuoteSnapshot } from '@/types'

interface ItemsEnvelope<T> {
  items: T[]
  total: number
}

export const fetchQuote = (symbol: string): Promise<QuoteSnapshot> =>
  http.get(`/api/quote/${symbol}`)

export const fetchQuotes = async (symbols: string[]): Promise<QuoteSnapshot[]> => {
  if (symbols.length === 0) return []
  const resp = await http.get<unknown, ItemsEnvelope<QuoteSnapshot>>('/api/quotes', {
    params: { symbols: symbols.join(',') },
  })
  return Array.isArray(resp?.items) ? resp.items : []
}

export const fetchFenbi = async (symbol: string, limit = 100): Promise<FenbiTick[]> => {
  const resp = await http.get<unknown, ItemsEnvelope<FenbiTick>>(`/api/fenbi/${symbol}`, {
    params: { limit },
  })
  return Array.isArray(resp?.items) ? resp.items : []
}
