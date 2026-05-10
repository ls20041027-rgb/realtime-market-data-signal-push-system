import Decimal from 'decimal.js'

Decimal.set({ precision: 20, rounding: Decimal.ROUND_HALF_UP })

export type DecimalInput = string | number | Decimal | null | undefined

export const D = (x: DecimalInput): Decimal => {
  if (x === null || x === undefined || x === '') return new Decimal(0)
  return new Decimal(x as string | number | Decimal)
}

export const fmtPrice = (x: DecimalInput, digits = 2): string => D(x).toFixed(digits)

export const fmtPct = (x: DecimalInput, digits = 2): string =>
  `${D(x).mul(100).toFixed(digits)}%`

export const fmtAmt = (x: DecimalInput): string => {
  const v = D(x)
  const abs = v.abs()
  if (abs.gte(1e8)) return `${v.div(1e8).toFixed(2)}亿`
  if (abs.gte(1e4)) return `${v.div(1e4).toFixed(2)}万`
  return v.toFixed(0)
}

export const trendClass = (change: DecimalInput): 'color-up' | 'color-down' | 'color-neutral' => {
  const v = D(change)
  if (v.gt(0)) return 'color-up'
  if (v.lt(0)) return 'color-down'
  return 'color-neutral'
}
