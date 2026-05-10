<script setup lang="ts">
// 财务 tab：/api/finance 最近 N 期
// 表格展示 + EPS / NetProfit 迷你趋势（复用 ECharts，无需新封装）
import { onMounted, onBeforeUnmount, ref, shallowRef, watch } from 'vue'
import { Table as ATable } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import * as echarts from 'echarts/core'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import { fetchFinance } from '@/api/finance'
import { fmtPrice } from '@/composables/useDecimal'
import { createLogger } from '@/utils/logger'
import type { FinanceSnapshot } from '@/types'

echarts.use([LineChart, GridComponent, TooltipComponent, CanvasRenderer])

const log = createLogger('stock:finance')
const props = defineProps<{ symbol: string }>()

const list = ref<FinanceSnapshot[]>([])
const loading = ref(false)

const load = async () => {
  if (!props.symbol) return
  loading.value = true
  try {
    list.value = await fetchFinance(props.symbol, 8)
  } catch (err) {
    log.error('fetchFinance failed', err)
  } finally {
    loading.value = false
  }
}

watch(() => props.symbol, () => void load(), { immediate: true })

// 迷你趋势图
const epsRoot = shallowRef<HTMLElement | null>(null)
const profitRoot = shallowRef<HTMLElement | null>(null)
let epsChart: echarts.ECharts | null = null
let profitChart: echarts.ECharts | null = null

const buildLineOption = (title: string, xs: string[], ys: Array<number | null>) => ({
  animation: false,
  title: { text: title, textStyle: { color: '#8B93A1', fontSize: 12, fontWeight: 'normal' as const }, left: 4, top: 2 },
  grid: { left: 40, right: 12, top: 24, bottom: 20 },
  tooltip: { trigger: 'axis', backgroundColor: '#141922', borderColor: '#1F2632', textStyle: { color: '#E6E8EB' } },
  xAxis: {
    type: 'category',
    data: xs,
    axisLabel: { color: '#8B93A1', fontSize: 10 },
    axisLine: { lineStyle: { color: '#1F2632' } },
  },
  yAxis: {
    type: 'value',
    scale: true,
    axisLabel: { color: '#8B93A1', fontSize: 10 },
    splitLine: { lineStyle: { color: '#1F2632' } },
  },
  series: [
    {
      type: 'line',
      data: ys,
      smooth: true,
      symbol: 'circle',
      symbolSize: 4,
      lineStyle: { color: '#5B8CFF', width: 1.5 },
      itemStyle: { color: '#5B8CFF' },
      areaStyle: { color: 'rgba(91,140,255,0.12)' },
    },
  ],
})

const render = () => {
  // 按 report_date 升序
  const sorted = [...list.value].sort((a, b) => a.report_date.localeCompare(b.report_date))
  const xs = sorted.map((r) => r.report_date)
  const epsYs = sorted.map((r) => (r.eps ? Number(r.eps) : null))
  const profitYs = sorted.map((r) => (r.net_profit ? Number(r.net_profit) : null))

  if (epsChart) epsChart.setOption(buildLineOption('每股收益 EPS', xs, epsYs), true)
  if (profitChart) profitChart.setOption(buildLineOption('净利润', xs, profitYs), true)
}

onMounted(() => {
  if (epsRoot.value) epsChart = echarts.init(epsRoot.value)
  if (profitRoot.value) profitChart = echarts.init(profitRoot.value)
  render()
})

watch(list, () => render())

onBeforeUnmount(() => {
  epsChart?.dispose()
  profitChart?.dispose()
  epsChart = null
  profitChart = null
})

const columns: TableColumnsType<FinanceSnapshot> = [
  { title: '报告期', dataIndex: 'report_date', key: 'report_date' },
  { title: '总股本', key: 'total_shares', align: 'right' },
  { title: '流通股', key: 'float_shares', align: 'right' },
  { title: 'EPS', key: 'eps', align: 'right' },
  { title: 'BPS', key: 'bps', align: 'right' },
  { title: '净利润', key: 'net_profit', align: 'right' },
]
</script>

<template>
  <div class="wrap">
    <div class="charts">
      <div ref="epsRoot" class="chart" />
      <div ref="profitRoot" class="chart" />
    </div>
    <ATable
      :columns="columns"
      :data-source="list"
      :pagination="false"
      :loading="loading"
      row-key="report_date"
      size="small"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'total_shares'">
          <span class="mono">{{ record.total_shares ? fmtPrice(record.total_shares, 0) : '-' }}</span>
        </template>
        <template v-else-if="column.key === 'float_shares'">
          <span class="mono">{{ record.float_shares ? fmtPrice(record.float_shares, 0) : '-' }}</span>
        </template>
        <template v-else-if="column.key === 'eps'">
          <span class="mono">{{ record.eps ? fmtPrice(record.eps, 4) : '-' }}</span>
        </template>
        <template v-else-if="column.key === 'bps'">
          <span class="mono">{{ record.bps ? fmtPrice(record.bps, 4) : '-' }}</span>
        </template>
        <template v-else-if="column.key === 'net_profit'">
          <span class="mono">{{ record.net_profit ? fmtPrice(record.net_profit, 0) : '-' }}</span>
        </template>
      </template>
    </ATable>
  </div>
</template>

<style scoped>
.wrap { display: flex; flex-direction: column; gap: 12px; }
.charts {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}
.chart {
  height: 180px;
  background: var(--panel);
  border: 1px solid var(--divider);
  border-radius: 6px;
}
</style>
