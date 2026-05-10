export type Exchange = 'SSE' | 'SZSE' | 'BSE'

const SH_RE = /^SH\d{6}$/
const SZ_RE = /^SZ\d{6}$/
const BJ_RE = /^BJ\d{6}$/

export const normalizeSymbol = (raw: string): string => {
  return (raw || '').trim().toUpperCase().replace(/\s+/g, '')
}

export const detectExchange = (symbol: string): Exchange | null => {
  const s = normalizeSymbol(symbol)
  if (SH_RE.test(s)) return 'SSE'
  if (SZ_RE.test(s)) return 'SZSE'
  if (BJ_RE.test(s)) return 'BSE'
  return null
}

export const isValidSymbol = (symbol: string): boolean => detectExchange(symbol) !== null
