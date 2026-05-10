
export interface OrderLevel {
  price: string
  volume: string
}

export interface QuoteSnapshot {
  symbol: string
  last_price: string
  prev_close: string
  open_price: string
  high_price: string
  low_price: string
  volume: string
  turnover: string
  bid_levels: OrderLevel[]
  ask_levels: OrderLevel[]
  event_time: string
}

export interface IndicatorSnapshot {
  symbol: string
  change_pct?: string
  change_amt?: string
  volume_ratio?: string
  turnover_rate?: string
  ma5?: string
  ma10?: string
  ma20?: string
  ma60?: string
  rsi14?: string
  kdj_k?: string
  kdj_d?: string
  kdj_j?: string
  boll_mid?: string
  boll_up?: string
  boll_low?: string
  macd_dif?: string
  macd_dea?: string
  macd_hist?: string
  updated_at?: string
}

export interface CapitalSnapshot {
  symbol: string
  big_buy: string
  big_sell: string
  net_inflow: string
  buy_tick_count: number
  sell_tick_count: number
  last_reset_date: string
}

export interface FenbiTick {
  price: string
  volume: string
  amount: string
  direction: 'BUY' | 'SELL' | 'NEUTRAL'
  trade_time: string
  bid1: string
  ask1: string
}
