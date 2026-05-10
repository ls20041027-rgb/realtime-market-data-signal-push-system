import axios, { type AxiosInstance } from 'axios'
import type { Envelope } from '@/types'
import {
  ApiError,
  NotFoundError,
  ServiceDownError,
  UnknownError,
  ValidationError,
} from '@/utils/errors'
import { createLogger } from '@/utils/logger'

const log = createLogger('api')

const baseURL = import.meta.env.VITE_API_BASE || ''

export const http: AxiosInstance = axios.create({
  baseURL,
  timeout: 15000,
  headers: { 'Content-Type': 'application/json' },
})

http.interceptors.response.use(
  (resp) => {
    const body = resp.data as Envelope<unknown>
    if (!body || typeof body.code !== 'number') {
      return resp.data
    }
    if (body.code === 0) return body.data
    switch (body.code) {
      case 40001:
        throw new NotFoundError(body.message)
      case 40002:
        throw new ValidationError(body.message)
      case 50001:
      case 50002:
        throw new ServiceDownError(body.code, body.message)
      default:
        throw new UnknownError(body.message, body.code)
    }
  },
  (err) => {
    log.error('http error', { url: err?.config?.url, status: err?.response?.status })
    if (err?.response?.data?.code) {
      const { code, message } = err.response.data
      throw new ApiError(code, message)
    }
    throw new UnknownError(err?.message || 'network error')
  },
)
