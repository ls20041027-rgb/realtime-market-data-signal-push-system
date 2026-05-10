<script setup lang="ts">
// 信号 tab：
// - 顶部 LIVE 区：useStockDetail 传入的 liveSignals（订阅 signal:{symbol}）
// - 底部 HISTORY 区：/api/signals?symbol=... 分页
import { watch } from 'vue'
import { Table as ATable, Tag } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import { usePagination } from '@/composables/usePagination'
import { fetchSignals } from '@/api/signal'
import { fmtPrice } from '@/composables/useDecimal'
import { fmtTime } from '@/utils/format'
import type { TradingSignal, SignalAction, SignalSeverity } from '@/types'

const props = defineProps<{
  symbol: string
  liveSignals: TradingSignal[]
}>()

const pager = usePagination<TradingSignal>(
  (page, pageSize) => fetchSignals({ symbol: props.symbol, page, page_size: pageSize }),
  { pageSize: 20 },
)
const { items, total, page, pageSize, loading } = pager

watch(
  () => props.symbol,
  () => {
    if (props.symbol) void pager.load(1)
  },
  { immediate: true },
)

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
  { title: '时间', key: 'signal_time', width: 140 },
  { title: '动作', key: 'action', width: 80 },
  { title: '类型', dataIndex: 'signal_type', key: 'signal_type', width: 120 },
  { title: '策略', dataIndex: 'strategy_name', key: 'strategy_name', width: 140 },
  { title: '触发价', key: 'trigger_price', width: 100, align: 'right' },
  { title: '置信度', key: 'confidence', width: 90, align: 'right' },
  { title: '等级', key: 'severity', width: 80 },
  { title: '原因', dataIndex: 'reason', key: 'reason', ellipsis: true },
]

const onPageChange = (p: number, ps: number) => {
  pageSize.value = ps
  void pager.load(p)
}
</script>

<template>
  <div class="wrap">
    <section class="live">
      <div class="head">
        <span class="title">实时信号</span>
        <span class="dim">（WS signal:{{ symbol }}）</span>
      </div>
      <div v-if="liveSignals.length === 0" class="empty">等待实时信号...</div>
      <ul v-else class="live-list">
        <li v-for="s in liveSignals" :key="s.signal_id" class="live-item">
          <span class="mono dim">{{ fmtTime(s.signal_time) }}</span>
          <Tag :color="actionColor[s.action]">{{ s.action }}</Tag>
          <span class="strat dim">{{ s.strategy_name }}</span>
          <span class="mono">触发 {{ fmtPrice(s.trigger_price) }}</span>
          <span class="mono dim">置信 {{ s.confidence }}</span>
          <span class="reason">{{ s.reason }}</span>
        </li>
      </ul>
    </section>

    <section class="history">
      <div class="head">
        <span class="title">历史信号</span>
        <span class="dim">共 {{ total }} 条</span>
      </div>
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
          onChange: onPageChange,
          onShowSizeChange: onPageChange,
        }"
        row-key="signal_id"
        size="small"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'signal_time'">
            <span class="mono dim">{{ record.signal_time }}</span>
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
    </section>
  </div>
</template>

<style scoped>
.wrap { display: flex; flex-direction: column; gap: 14px; }
.head { display: flex; align-items: baseline; gap: 8px; margin-bottom: 6px; }
.title { font-size: 14px; font-weight: 600; }
.dim { color: var(--text-dim); font-size: 12px; }

.live {
  background: var(--panel);
  border: 1px solid var(--divider);
  border-radius: 6px;
  padding: 10px 14px;
}
.empty { color: var(--text-dim); font-size: 12px; padding: 8px 0; }
.live-list { list-style: none; padding: 0; margin: 0; display: flex; flex-direction: column; gap: 6px; max-height: 200px; overflow: auto; }
.live-item { display: flex; align-items: center; gap: 10px; font-size: 12px; }
.strat { min-width: 100px; }
.reason { color: var(--text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.history {
  background: var(--panel);
  border: 1px solid var(--divider);
  border-radius: 6px;
  padding: 10px 14px;
}
</style>
