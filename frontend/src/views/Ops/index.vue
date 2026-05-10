<script setup lang="ts">
// 运维大屏：/api/status 轮询（3s，可见性感知暂停）+ system:events 事件流
// - status 全局仅在本页 bind 轮询（其它页不跑）
// - system:events 由 AppShell 全局 bind 一次，这里只消费
import { computed, onMounted } from 'vue'
import { Button, Space, Tag } from 'ant-design-vue'
import StatusGrid from './StatusGrid.vue'
import KafkaLagTable from './KafkaLagTable.vue'
import EventsStream from './EventsStream.vue'
import StorageStatsPanel from './StorageStatsPanel.vue'
import IngestStatsPanel from './IngestStatsPanel.vue'
import { useStatusStore } from '@/stores/status'
import { useStorageStatsStore } from '@/stores/storage-stats'
import { useIngestStatsStore } from '@/stores/ingest-stats'
import { useAutoRefresh } from '@/composables/useAutoRefresh'
import { fmtTime } from '@/utils/format'

const statusStore = useStatusStore()
const storageStore = useStorageStatsStore()
const ingestStore = useIngestStatsStore()

const { start } = useAutoRefresh(() => statusStore.refresh(), {
  intervalMs: 3000,
  immediate: true,
})

// 存储统计相对更重（SCAN / COUNT(*)），单独 15s 轮询
const { start: startStorage } = useAutoRefresh(() => storageStore.refresh(), {
  intervalMs: 15000,
  immediate: true,
})

// 入口接收统计：Redis HGETALL 轻量，2s 一次便于算 QPS
const { start: startIngest } = useAutoRefresh(() => ingestStore.refresh(), {
  intervalMs: 2000,
  immediate: true,
})

onMounted(() => {
  start()
  startStorage()
  startIngest()
})

const updatedLabel = computed(() =>
  statusStore.updatedAt > 0
    ? fmtTime(new Date(statusStore.updatedAt).toISOString())
    : '-',
)

const topics = computed(() => statusStore.snapshot?.kafka.topics ?? [])
</script>

<template>
  <div class="page">
    <header class="bar">
      <div class="title">
        <h2>运维大屏</h2>
        <Tag color="default">/api/status · 3s 轮询</Tag>
        <Tag color="default" class="mono">更新 {{ updatedLabel }}</Tag>
        <Tag v-if="statusStore.error" color="red">加载失败：{{ statusStore.error }}</Tag>
      </div>
      <Space>
        <Button :loading="statusStore.loading || storageStore.loading" @click="statusStore.refresh(); storageStore.refresh()">刷新</Button>
      </Space>
    </header>

    <StatusGrid :snapshot="statusStore.snapshot" />

    <IngestStatsPanel
      :snapshot="ingestStore.snapshot"
      :qps="ingestStore.qps"
      :total-qps="ingestStore.totalQps"
      :loading="ingestStore.loading"
    />

    <StorageStatsPanel :snapshot="storageStore.snapshot" :loading="storageStore.loading" />

    <KafkaLagTable :topics="topics" />

    <EventsStream />
  </div>
</template>

<style scoped>
.page { padding: 16px; color: var(--text); display: flex; flex-direction: column; gap: 14px; }
.bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
.title { display: flex; align-items: center; gap: 10px; }
.title h2 { margin: 0; font-size: 18px; font-weight: 600; }
</style>
