export * from './envelope'
export * from './quote'
export * from './signal'

export interface StockInfo {
  symbol: string
  name: string
  exchange: 'SSE' | 'SZSE' | 'BSE'
  lot_size: number
}

export interface KLineBar {
  trade_date?: string
  trade_time?: string
  open: string
  high: string
  low: string
  close: string
  volume: string
  turnover: string
}

export interface FinanceSnapshot {
  symbol: string
  report_date: string
  total_shares?: string
  float_shares?: string
  eps?: string
  bps?: string
  net_profit?: string
}

export interface StatusWs {
  clients: number
  channels: number
  dropped_slow: number
}

export interface StatusKafkaTopic {
  topic: string
  partition: string
  lag: number
  offset: number
  messages: number
  errors: number
}

export interface StatusKafka {
  topics: StatusKafkaTopic[]
}

export interface StatusComponent {
  up: boolean
  latency_ms: number
  error?: string
}

export interface StatusRuntime {
  pid: number
  goroutines: number
  uptime_seconds: number
}

export interface StatusSnapshot {
  ws: StatusWs
  kafka: StatusKafka
  redis: StatusComponent
  postgres: StatusComponent
  runtime: StatusRuntime
}

export interface StorageStatItem {
  label: string
  count: number
  table?: string
  prefix?: string
  key?: string
  truncated?: boolean
  error?: string
}

export interface StorageStats {
  postgres: StorageStatItem[]
  redis: StorageStatItem[]
  scan_limit: number
  ts: number
}

export interface IngestStatItem {
  message_type?: string
  file_data_type?: string
  label: string
  count: number
}

export interface IngestStats {
  message_types: IngestStatItem[]
  file_data_types: IngestStatItem[]
  total_count: number
  updated_at_ms: number
  started_at_ms: number
  now_ms: number
  error?: string
}
