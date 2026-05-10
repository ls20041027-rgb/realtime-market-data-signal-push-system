export const CH = {
  quote: (symbol: string) => `quote:${symbol}` as const,
  signal: (symbol: string) => `signal:${symbol}` as const,
  signalAll: 'signal:ALL' as const,
  systemEvents: 'system:events' as const,
} as const

const CHANNEL_RE = /^(quote:[A-Z]{2}\d{6}|signal:[A-Z]{2}\d{6}|signal:ALL|system:events)$/

export const isValidChannel = (channel: string): boolean => CHANNEL_RE.test(channel)

export type ChannelPrefix = 'quote' | 'signal' | 'system'

export const getChannelPrefix = (channel: string): ChannelPrefix | null => {
  if (channel.startsWith('quote:')) return 'quote'
  if (channel.startsWith('signal:')) return 'signal'
  if (channel.startsWith('system:')) return 'system'
  return null
}
