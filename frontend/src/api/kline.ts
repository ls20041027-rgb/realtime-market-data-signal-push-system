import { http } from './http'
import type { KLineBar } from '@/types'

export interface KLineParams {
  start?: string
  end?: string
  limit?: number
}

interface KlineEnvelope {
  symbol: string
  items: KLineBar[] | null
  total: number
}

export const fetchDailyKline = async (
  symbol: string,
  params: KLineParams = {},
): Promise<KLineBar[]> => {
  const resp = await http.get<unknown, KlineEnvelope>(`/api/kline/${symbol}`, { params })
  return Array.isArray(resp?.items) ? resp.items : []
}

export const fetch5MinKline = async (
  symbol: string,
  params: KLineParams = {},
): Promise<KLineBar[]> => {
  const resp = await http.get<unknown, KlineEnvelope>(`/api/kline5m/${symbol}`, { params })
  return Array.isArray(resp?.items) ? resp.items : []
}
