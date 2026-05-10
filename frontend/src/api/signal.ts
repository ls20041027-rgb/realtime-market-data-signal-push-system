import { http } from './http'
import type { PageData, SignalAction, SignalSeverity, TradingSignal } from '@/types'

export interface SignalQuery {
  symbol?: string
  signal_type?: string
  action?: SignalAction
  severity?: SignalSeverity
  from?: string
  to?: string
  page?: number
  page_size?: number
}

export const fetchSignals = (query: SignalQuery = {}): Promise<PageData<TradingSignal>> =>
  http.get('/api/signals', { params: query })

export const fetchSignalById = (id: string): Promise<TradingSignal> =>
  http.get(`/api/signals/${id}`)
