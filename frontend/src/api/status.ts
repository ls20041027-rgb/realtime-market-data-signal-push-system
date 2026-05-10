import { http } from './http'
import type { IngestStats, StatusSnapshot, StorageStats } from '@/types'

export const fetchHealthz = (): Promise<unknown> => http.get('/healthz')

export const fetchStatus = (): Promise<StatusSnapshot> => http.get('/api/status')

export const fetchStorageStats = (): Promise<StorageStats> => http.get('/api/storage-stats')

export const fetchIngestStats = (): Promise<IngestStats> => http.get('/api/ingest-stats')
