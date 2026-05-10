<script setup lang="ts">
// Kafka 消费 lag 表格：按 lag 降序展示
// - 同 topic 可能多 partition，直接平铺
// - lag > 1000 标红，> 100 标黄
import { computed } from 'vue'
import { Table as ATable, Tag } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import type { StatusKafkaTopic } from '@/types'

const props = defineProps<{ topics: StatusKafkaTopic[] }>()

const rows = computed<StatusKafkaTopic[]>(() =>
  [...props.topics].sort((a, b) => b.lag - a.lag),
)

const lagClass = (lag: number): string => {
  if (lag >= 1000) return 'lag-critical'
  if (lag >= 100) return 'lag-warn'
  return 'lag-ok'
}

const columns: TableColumnsType<StatusKafkaTopic> = [
  { title: 'Topic', dataIndex: 'topic', key: 'topic' },
  { title: '分区', dataIndex: 'partition', key: 'partition', width: 80 },
  { title: 'Lag', key: 'lag', width: 120, align: 'right' },
  { title: 'Offset', dataIndex: 'offset', key: 'offset', width: 120, align: 'right' },
  { title: 'Messages', dataIndex: 'messages', key: 'messages', width: 120, align: 'right' },
  { title: 'Errors', key: 'errors', width: 100, align: 'right' },
]
</script>

<template>
  <section class="card">
    <header class="hd">
      <span class="title">Kafka 消费进度</span>
      <Tag color="default">{{ rows.length }} 分区</Tag>
    </header>
    <ATable
      :columns="columns"
      :data-source="rows"
      :pagination="false"
      row-key="partition"
      size="small"
      :locale="{ emptyText: '暂无数据' }"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'lag'">
          <span :class="['mono', lagClass(record.lag)]">{{ record.lag.toLocaleString() }}</span>
        </template>
        <template v-else-if="column.key === 'errors'">
          <span :class="['mono', record.errors > 0 ? 'lag-critical' : '']">
            {{ record.errors }}
          </span>
        </template>
      </template>
    </ATable>
  </section>
</template>

<style scoped>
.card {
  background: var(--panel);
  border: 1px solid var(--divider);
  border-radius: 6px;
  padding: 12px 14px;
}
.hd { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; }
.title { font-size: 13px; font-weight: 600; }
.lag-ok { color: var(--text); }
.lag-warn { color: var(--warn); }
.lag-critical { color: var(--down); font-weight: 600; }
</style>
