import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { TradingSignal, WSFrame } from '@/types'
import { wsClient } from '@/ws/client'
import { CH } from '@/ws/channels'

const LIVE_MAX = 30

export const useSignalStore = defineStore('signal', () => {
  const live = ref<TradingSignal[]>([])
  const unread = ref(0)

  const apply = (frame: WSFrame<TradingSignal>) => {
    if (!frame.data) return
    live.value = [frame.data, ...live.value].slice(0, LIVE_MAX)
    unread.value += 1
  }

  let bound = false
  const bind = () => {
    if (bound) return
    bound = true
    wsClient.connect()
    wsClient.subscribe(CH.signalAll, apply as (f: WSFrame) => void)
  }

  const markRead = () => (unread.value = 0)

  return { live, unread, bind, markRead }
})
