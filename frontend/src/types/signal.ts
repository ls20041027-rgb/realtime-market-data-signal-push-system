export type SignalAction = 'BUY' | 'SELL' | 'WATCH' | 'RISK'
export type SignalSeverity = 'INFO' | 'WARN' | 'CRITICAL'

export interface TradingSignal {
  signal_id: string
  symbol: string
  signal_type: string
  action: SignalAction
  strategy_name: string
  confidence: string
  signal_time: string
  trigger_price: string
  reason: string
  severity?: SignalSeverity
  summary?: string
  indicators?: Record<string, string>
}

export type SystemLevel = 'INFO' | 'WARN' | 'ERROR' | 'CRITICAL'

export interface SystemEvent {
  event_id: string
  service: string
  level: SystemLevel
  event_type: string
  message: string
  details?: Record<string, unknown>
  retry_count?: number
  related_topic?: string
}
