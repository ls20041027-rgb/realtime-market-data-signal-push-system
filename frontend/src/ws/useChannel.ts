import { onScopeDispose, readonly, ref, shallowRef, watch, type Ref } from 'vue'
import { wsClient, type ConnStatus } from './client'
import type { WSFrame } from '@/types'

export interface UseChannelOptions<T> {
  onMessage?: (frame: WSFrame<T>) => void
  immediate?: boolean
}

export interface UseChannelReturn<T> {
  latest: Ref<T | null>
  status: Ref<ConnStatus>
  subscribe: () => void
  unsubscribe: () => void
}

const _status = ref<ConnStatus>(wsClient.getStatus())
wsClient.onStatus((s) => (_status.value = s))

export function useChannel<T = unknown>(
  channel: string | Ref<string>,
  options: UseChannelOptions<T> = {},
): UseChannelReturn<T> {
  const latest = shallowRef<T | null>(null)
  let current = ''

  const handler = (frame: WSFrame) => {
    latest.value = (frame.data as T) ?? null
    options.onMessage?.(frame as WSFrame<T>)
  }

  const subscribe = () => {
    const ch = typeof channel === 'string' ? channel : channel.value
    if (!ch || ch === current) return
    if (current) wsClient.unsubscribe(current, handler)
    current = ch
    wsClient.subscribe(ch, handler)
  }

  const unsubscribe = () => {
    if (!current) return
    wsClient.unsubscribe(current, handler)
    current = ''
  }

  if (options.immediate !== false) {
    wsClient.connect()
    subscribe()
  }

  if (typeof channel !== 'string') {
    watch(channel, () => subscribe())
  }

  onScopeDispose(() => unsubscribe())

  return { latest, status: readonly(_status) as Ref<ConnStatus>, subscribe, unsubscribe }
}
