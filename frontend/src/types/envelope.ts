export interface Envelope<T> {
  code: number
  message: string
  data: T
}

export interface PageData<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

export interface WSFrame<T = unknown> {
  channel?: string
  type: string
  data?: T
  ts?: number
  code?: number
  message?: string
}
