<script setup lang="ts">
// 分笔 tab：首屏 + 轮询 /api/fenbi/:symbol，按时间倒序展示
import { ref, watch } from 'vue'
import { Table as ATable } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import { fetchFenbi } from '@/api/quote'
import { useAutoRefresh } from '@/composables/useAutoRefresh'
import { fmtPrice, fmtAmt } from '@/composables/useDecimal'
import { fmtTime } from '@/utils/format'
import { createLogger } from '@/utils/logger'
import type { FenbiTick } from '@/types'

const log = createLogger('stock:fenbi')
const props = defineProps<{ symbol: string }>()

const ticks = ref<FenbiTick[]>([])
const loading = ref(false)

const load = async () => {
  if (!props.symbol) return
  loading.value = true
  try {
    const list = await fetchFenbi(props.symbol, 100)
    // 后端返回按时间升序（RPUSH），这里倒序显示最新在上
    ticks.value = [...list].reverse()
  } catch (err) {
    log.error('fetchFenbi failed', err)
  } finally {
    loading.value = false
  }
}

const poll = useAutoRefresh(load, { intervalMs: 3000, immediate: false })

watch(
  () => props.symbol,
  () => {
    poll.stop()
    ticks.value = []
    void load()
    poll.start()
  },
  { immediate: true },
)

const columns: TableColumnsType<FenbiTick> = [
  { title: '时间', key: 'trade_time', width: 100 },
  { title: '价格', key: 'price', width: 100, align: 'right' },
  { title: '量', key: 'volume', width: 100, align: 'right' },
  { title: '金额', key: 'amount', width: 120, align: 'right' },
  { title: '方向', key: 'direction', width: 80, align: 'center' },
]

const dirCls = (d: FenbiTick['direction']) =>
  d === 'BUY' ? 'color-up' : d === 'SELL' ? 'color-down' : 'color-neutral'
const dirText = (d: FenbiTick['direction']) =>
  d === 'BUY' ? '买' : d === 'SELL' ? '卖' : '中'
</script>

<template>
  <div class="wrap">
    <ATable
      :columns="columns"
      :data-source="ticks"
      :pagination="false"
      :loading="loading"
      row-key="trade_time"
      size="small"
      :scroll="{ y: 420 }"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'trade_time'">
          <span class="mono dim">{{ fmtTime(record.trade_time) }}</span>
        </template>
        <template v-else-if="column.key === 'price'">
          <span class="mono" :class="dirCls(record.direction)">{{ fmtPrice(record.price) }}</span>
        </template>
        <template v-else-if="column.key === 'volume'">
          <span class="mono">{{ record.volume }}</span>
        </template>
        <template v-else-if="column.key === 'amount'">
          <span class="mono">{{ fmtAmt(record.amount) }}</span>
        </template>
        <template v-else-if="column.key === 'direction'">
          <span class="badge" :class="dirCls(record.direction)">
            {{ dirText(record.direction) }}
          </span>
        </template>
      </template>
    </ATable>
  </div>
</template>

<style scoped>
.wrap { background: var(--panel); border: 1px solid var(--divider); border-radius: 6px; overflow: hidden; }
.dim { color: var(--text-dim); }
.badge {
  display: inline-block;
  min-width: 22px;
  padding: 1px 6px;
  border-radius: 3px;
  font-size: 12px;
  background: rgba(255, 255, 255, 0.04);
}
.badge.color-up { color: var(--up); }
.badge.color-down { color: var(--down); }
.badge.color-neutral { color: var(--neutral); }
</style>
