import { http } from './http'
import type { FinanceSnapshot } from '@/types'

interface FinanceEnvelope {
  symbol: string
  items: FinanceSnapshot[]
  total: number
}

export const fetchFinance = async (symbol: string, limit = 8): Promise<FinanceSnapshot[]> => {
  const resp = await http.get<unknown, FinanceEnvelope>(`/api/finance/${symbol}`, {
    params: { limit },
  })
  return Array.isArray(resp?.items) ? resp.items : []
}
