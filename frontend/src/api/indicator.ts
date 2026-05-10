import { http } from './http'
import type { CapitalSnapshot, IndicatorSnapshot } from '@/types'

export const fetchIndicators = (symbol: string): Promise<IndicatorSnapshot> =>
  http.get(`/api/indicators/${symbol}`)

export const fetchCapital = (
  symbol: string,
  history = false,
): Promise<CapitalSnapshot> =>
  http.get(`/api/capital/${symbol}`, { params: history ? { history: 1 } : {} })
