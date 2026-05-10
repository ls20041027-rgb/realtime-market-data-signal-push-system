import { defineStore } from 'pinia'
import { ref } from 'vue'
import { wsClient, type ConnStatus } from '@/ws/client'
import { CH } from '@/ws/channels'
import type { SystemEvent, WSFrame } from '@/types'

const MAX_EVENTS = 100

export const useSystemStore = defineStore('system', () => {
  const events = ref<SystemEvent[]>([])
  const connStatus = ref<ConnStatus>(wsClient.getStatus())

  wsClient.onStatus((s) => (connStatus.value = s))

  const highestLevel = ref<'INFO' | 'WARN' | 'ERROR' | 'CRITICAL'>('INFO')

  const apply = (frame: WSFrame<SystemEvent>) => {
    if (!frame.data) return
    const ev = frame.data
    events.value = [ev, ...events.value].slice(0, MAX_EVENTS)
    const rank = { INFO: 0, WARN: 1, ERROR: 2, CRITICAL: 3 } as const
    if (rank[ev.level] >= rank[highestLevel.value]) {
      highestLevel.value = ev.level
    }
  }

  let bound = false
  const bind = () => {
    if (bound) return
    bound = true
    wsClient.connect()
    wsClient.subscribe(CH.systemEvents, apply as (f: WSFrame) => void)
  }

  const clear = () => {
    events.value = []
    highestLevel.value = 'INFO'
  }

  return { events, connStatus, highestLevel, bind, clear }
})
