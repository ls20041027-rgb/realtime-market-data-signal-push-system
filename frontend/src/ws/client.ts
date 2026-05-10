import { isValidChannel } from './channels'
import { createLogger } from '@/utils/logger'
import type { WSFrame } from '@/types'

const log = createLogger('ws')

export type ConnStatus = 'IDLE' | 'CONNECTING' | 'OPEN' | 'RECONNECTING' | 'CLOSED'

type FrameHandler = (frame: WSFrame) => void
type StatusListener = (status: ConnStatus) => void

const HEARTBEAT_MS = 25_000
const DEAD_MS = 40_000
const BACKOFF_BASE = 1_000
const BACKOFF_MAX = 30_000
const RESET_AFTER_FRAMES = 10

class WSClient {
  private url: string
  private ws: WebSocket | null = null
  private status: ConnStatus = 'IDLE'
  private statusListeners = new Set<StatusListener>()

  private refCount = new Map<string, number>()
  private handlers = new Map<string, Set<FrameHandler>>()
  private pending: { action: 'subscribe' | 'unsubscribe'; channels: string[] }[] = []

  private heartbeatTimer: number | null = null
  private deadTimer: number | null = null
  private reconnectTimer: number | null = null
  private retryAttempt = 0
  private framesSinceOpen = 0

  constructor(url: string) {
    this.url = url
  }

  connect(): void {
    if (this.ws && (this.status === 'CONNECTING' || this.status === 'OPEN')) return
    this.setStatus('CONNECTING')
    try {
      this.ws = new WebSocket(this.url)
    } catch (err) {
      log.error('ws construct failed', err)
      this.scheduleReconnect()
      return
    }
    this.ws.onopen = this.handleOpen
    this.ws.onmessage = this.handleMessage
    this.ws.onclose = this.handleClose
    this.ws.onerror = this.handleError
  }

  subscribe(channel: string, handler: FrameHandler): void {
    if (!isValidChannel(channel)) {
      log.warn('reject invalid channel', channel)
      return
    }
    const hs = this.handlers.get(channel) ?? new Set()
    hs.add(handler)
    this.handlers.set(channel, hs)

    const next = (this.refCount.get(channel) ?? 0) + 1
    this.refCount.set(channel, next)
    if (next === 1) this.sendAction('subscribe', [channel])
  }

  unsubscribe(channel: string, handler: FrameHandler): void {
    const hs = this.handlers.get(channel)
    if (hs) {
      hs.delete(handler)
      if (hs.size === 0) this.handlers.delete(channel)
    }
    const cur = this.refCount.get(channel) ?? 0
    const next = Math.max(0, cur - 1)
    if (next === 0) {
      this.refCount.delete(channel)
      this.sendAction('unsubscribe', [channel])
    } else {
      this.refCount.set(channel, next)
    }
  }

  onStatus(listener: StatusListener): () => void {
    this.statusListeners.add(listener)
    listener(this.status)
    return () => this.statusListeners.delete(listener)
  }

  getStatus(): ConnStatus {
    return this.status
  }


  private setStatus(s: ConnStatus): void {
    if (this.status === s) return
    this.status = s
    this.statusListeners.forEach((fn) => fn(s))
  }

  private handleOpen = (): void => {
    log.info('ws open')
    this.setStatus('OPEN')
    this.retryAttempt = 0
    this.framesSinceOpen = 0
    const live = Array.from(this.refCount.keys())
    if (live.length > 0) this.sendRaw({ action: 'subscribe', channels: live })
    this.pending.forEach((p) => this.sendRaw(p))
    this.pending = []
    this.startHeartbeat()
    this.armDeadTimer()
  }

  private handleMessage = (ev: MessageEvent): void => {
    this.armDeadTimer()
    this.framesSinceOpen += 1
    if (this.framesSinceOpen >= RESET_AFTER_FRAMES) this.retryAttempt = 0

    let frame: WSFrame
    try {
      frame = JSON.parse(ev.data as string) as WSFrame
    } catch {
      log.warn('bad frame', ev.data)
      return
    }
    if (frame.type === 'pong') return
    if (frame.type === 'error') {
      log.warn('server error frame', { code: frame.code, message: frame.message })
      return
    }
    const channel = frame.channel
    if (!channel) return
    const hs = this.handlers.get(channel)
    if (!hs || hs.size === 0) return
    hs.forEach((fn) => {
      try {
        fn(frame)
      } catch (err) {
        log.error('handler error', err)
      }
    })
  }

  private handleClose = (ev: CloseEvent): void => {
    log.warn('ws close', { code: ev.code, reason: ev.reason })
    this.cleanupTimers()
    this.ws = null
    this.scheduleReconnect()
  }

  private handleError = (ev: Event): void => {
    log.error('ws error', ev)
  }

  private sendAction(action: 'subscribe' | 'unsubscribe', channels: string[]): void {
    if (channels.length === 0) return
    if (this.status === 'OPEN' && this.ws) {
      this.sendRaw({ action, channels })
    } else {
      this.pending.push({ action, channels })
      if (this.status === 'IDLE' || this.status === 'CLOSED') this.connect()
    }
  }

  private sendRaw(obj: unknown): void {
    try {
      this.ws?.send(JSON.stringify(obj))
    } catch (err) {
      log.error('ws send failed', err)
    }
  }

  private startHeartbeat(): void {
    this.stopHeartbeat()
    this.heartbeatTimer = window.setInterval(() => {
      this.sendRaw({ action: 'ping' })
    }, HEARTBEAT_MS)
  }

  private stopHeartbeat(): void {
    if (this.heartbeatTimer !== null) {
      clearInterval(this.heartbeatTimer)
      this.heartbeatTimer = null
    }
  }

  private armDeadTimer(): void {
    if (this.deadTimer !== null) clearTimeout(this.deadTimer)
    this.deadTimer = window.setTimeout(() => {
      log.warn('ws dead, force close')
      try {
        this.ws?.close(4000, 'dead')
      } catch {
      }
    }, DEAD_MS)
  }

  private cleanupTimers(): void {
    this.stopHeartbeat()
    if (this.deadTimer !== null) {
      clearTimeout(this.deadTimer)
      this.deadTimer = null
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimer !== null) return
    this.setStatus('RECONNECTING')
    const delay = Math.min(BACKOFF_BASE * 2 ** this.retryAttempt, BACKOFF_MAX)
    this.retryAttempt += 1
    log.info('ws reconnect scheduled', { delay_ms: delay, attempt: this.retryAttempt })
    this.reconnectTimer = window.setTimeout(() => {
      this.reconnectTimer = null
      this.connect()
    }, delay)
  }
}

function resolveWsUrl(): string {
  const fromEnv = import.meta.env.VITE_WS_URL
  if (fromEnv && /^wss?:\/\//.test(fromEnv)) return fromEnv
  if (typeof window !== 'undefined' && window.location) {
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${proto}//${window.location.host}/ws`
  }
  return 'ws://localhost:8080/ws'
}

const wsUrl = resolveWsUrl()
export const wsClient = new WSClient(wsUrl)
