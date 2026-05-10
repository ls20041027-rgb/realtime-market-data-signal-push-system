<script setup lang="ts">
// SignalList：历史信号分页表格
// - 数据/分页由父组件传入（便于筛选变化时重置）
// - 点击行 emit('detail', id)
import { Table as ATable, Tag } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import { fmtPrice } from '@/composables/useDecimal'
import type { SignalAction, SignalSeverity, TradingSignal } from '@/types'

defineProps<{
  items: TradingSignal[]
  total: number
  page: number
  pageSize: number
  loading: boolean
}>()

const emit = defineEmits<{
  (e: 'page-change', page: number, pageSize: number): void
  (e: 'detail', id: string): void
}>()

const actionColor: Record<SignalAction, string> = {
  BUY: 'green',
  SELL: 'red',
  WATCH: 'blue',
  RISK: 'orange',
}
const severityColor: Record<SignalSeverity, string> = {
  INFO: 'default',
  WARN: 'orange',
  CRITICAL: 'red',
}

const columns: TableColumnsType<TradingSignal> = [
  { title: '时间', key: 'signal_time', width: 160 },
  { title: '代码', dataIndex: 'symbol', key: 'symbol', width: 100 },
  { title: '动作', key: 'action', width: 80 },
  { title: '类型', dataIndex: 'signal_type', key: 'signal_type', width: 130 },
  { title: '策略', dataIndex: 'strategy_name', key: 'strategy_name', width: 140 },
  { title: '触发价', key: 'trigger_price', width: 100, align: 'right' },
  { title: '置信度', key: 'confidence', width: 90, align: 'right' },
  { title: '等级', key: 'severity', width: 90 },
  { title: '原因', dataIndex: 'reason', key: 'reason', ellipsis: true },
]
</script>

<template>
  <ATable
    :columns="columns"
    :data-source="items"
    :loading="loading"
    :pagination="{
      current: page,
      pageSize: pageSize,
      total: total,
      showSizeChanger: true,
      pageSizeOptions: ['10', '20', '50', '100'],
      onChange: (p: number, ps: number) => emit('page-change', p, ps),
      onShowSizeChange: (p: number, ps: number) => emit('page-change', p, ps),
    }"
    row-key="signal_id"
    size="small"
    :custom-row="(r: TradingSignal) => ({
      onClick: () => emit('detail', r.signal_id),
      style: { cursor: 'pointer' },
    })"
  >
    <template #bodyCell="{ column, record }">
      <template v-if="column.key === 'signal_time'">
        <span class="mono dim">{{ record.signal_time }}</span>
      </template>
      <template v-else-if="column.key === 'symbol'">
        <router-link :to="`/stock/${record.symbol}`" class="mono link" @click.stop>
          {{ record.symbol }}
        </router-link>
      </template>
      <template v-else-if="column.key === 'action'">
        <Tag :color="actionColor[record.action as SignalAction]">{{ record.action }}</Tag>
      </template>
      <template v-else-if="column.key === 'trigger_price'">
        <span class="mono">{{ fmtPrice(record.trigger_price) }}</span>
      </template>
      <template v-else-if="column.key === 'confidence'">
        <span class="mono">{{ record.confidence }}</span>
      </template>
      <template v-else-if="column.key === 'severity'">
        <Tag v-if="record.severity" :color="severityColor[record.severity as SignalSeverity]">
          {{ record.severity }}
        </Tag>
      </template>
    </template>
  </ATable>
</template>

<style scoped>
.dim { color: var(--text-dim); }
.link { color: var(--info); text-decoration: none; }
.link:hover { text-decoration: underline; }
</style>
