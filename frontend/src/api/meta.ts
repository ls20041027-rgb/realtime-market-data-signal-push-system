import { http } from './http'
import type { StockInfo } from '@/types'

interface StockListItem {
  symbol: string
  name: string
}
interface ItemsEnvelope<T> {
  items: T[]
  total: number
}

export const fetchStockList = async (): Promise<Record<string, string>> => {
  const resp = await http.get<unknown, ItemsEnvelope<StockListItem>>('/api/stock-list')
  const items = Array.isArray(resp?.items) ? resp.items : []
  const out: Record<string, string> = {}
  for (const it of items) {
    if (it && it.symbol) out[it.symbol] = it.name ?? ''
  }
  return out
}

export const fetchStockInfo = (symbol: string): Promise<StockInfo> =>
  http.get(`/api/stock/${symbol}`)
