<script setup lang="ts">
// ECharts 蜡烛图：接收原始 KLineBar[]（价格均为 string）
// - 内部用 decimal.js 计算 MA5/10/20
// - ECharts 渲染实例通过 shallowRef 管理，切换周期时销毁重建
import { onMounted, onBeforeUnmount, shallowRef, watch } from 'vue'
import * as echarts from 'echarts/core'
import { CandlestickChart, LineChart, BarChart } from 'echarts/charts'
import {
  GridComponent,
  TooltipComponent,
  LegendComponent,
  DataZoomComponent,
  AxisPointerComponent,
} from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import { D } from '@/composables/useDecimal'
import type { KLineBar } from '@/types'

echarts.use([
  CandlestickChart,
  LineChart,
  BarChart,
  GridComponent,
  TooltipComponent,
  LegendComponent,
  DataZoomComponent,
  AxisPointerComponent,
  CanvasRenderer,
])

const props = defineProps<{
  bars: KLineBar[]
  loading?: boolean
}>()

const rootEl = shallowRef<HTMLElement | null>(null)
const chart = shallowRef<echarts.ECharts | null>(null)

// 计算 MA（输入 close 数组为 string，中间用 Decimal）
const calcMA = (closes: string[], period: number): Array<number | null> => {
  const out: Array<number | null> = []
  for (let i = 0; i < closes.length; i++) {
    if (i < period - 1) {
      out.push(null)
      continue
    }
    let sum = D(0)
    for (let j = 0; j < period; j++) sum = sum.plus(D(closes[i - j]))
    out.push(+sum.div(period).toFixed(4))
  }
  return out
}

const buildOption = (bars: KLineBar[]) => {
  const categories = bars.map((b) => b.trade_date || b.trade_time || '')
  // ECharts 蜡烛图数据格式：[open, close, low, high]
  const ohlc = bars.map((b) => [
    Number(D(b.open).toFixed(4)),
    Number(D(b.close).toFixed(4)),
    Number(D(b.low).toFixed(4)),
    Number(D(b.high).toFixed(4)),
  ])
  const volumes = bars.map((b, i) => {
    const up = D(b.close).gte(D(b.open))
    return {
      value: Number(D(b.volume).toFixed(0)),
      itemStyle: { color: up ? 'rgba(0,200,83,0.55)' : 'rgba(255,59,48,0.55)' },
      xAxisIndex: 1,
      yAxisIndex: 1,
      name: categories[i],
    }
  })
  const closes = bars.map((b) => b.close)
  const ma5 = calcMA(closes, 5)
  const ma10 = calcMA(closes, 10)
  const ma20 = calcMA(closes, 20)

  return {
    backgroundColor: 'transparent',
    animation: false,
    legend: {
      data: ['K', 'MA5', 'MA10', 'MA20'],
      textStyle: { color: '#8B93A1' },
      top: 2,
    },
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'cross' },
      backgroundColor: '#141922',
      borderColor: '#1F2632',
      textStyle: { color: '#E6E8EB' },
    },
    axisPointer: { link: [{ xAxisIndex: 'all' }] },
    grid: [
      { left: 50, right: 16, top: 30, height: '60%' },
      { left: 50, right: 16, top: '76%', height: '18%' },
    ],
    xAxis: [
      {
        type: 'category',
        data: categories,
        axisLine: { lineStyle: { color: '#1F2632' } },
        axisLabel: { color: '#8B93A1' },
        splitLine: { show: false },
      },
      {
        type: 'category',
        gridIndex: 1,
        data: categories,
        axisLabel: { show: false },
        axisLine: { lineStyle: { color: '#1F2632' } },
      },
    ],
    yAxis: [
      {
        scale: true,
        splitLine: { lineStyle: { color: '#1F2632' } },
        axisLine: { lineStyle: { color: '#1F2632' } },
        axisLabel: { color: '#8B93A1' },
      },
      {
        scale: true,
        gridIndex: 1,
        splitNumber: 2,
        axisLabel: { color: '#8B93A1' },
        axisLine: { lineStyle: { color: '#1F2632' } },
        splitLine: { show: false },
      },
    ],
    dataZoom: [
      { type: 'inside', xAxisIndex: [0, 1], start: 60, end: 100 },
      { type: 'slider', xAxisIndex: [0, 1], height: 18, bottom: 0 },
    ],
    series: [
      {
        name: 'K',
        type: 'candlestick',
        data: ohlc,
        itemStyle: {
          color: '#00C853',
          color0: '#FF3B30',
          borderColor: '#00C853',
          borderColor0: '#FF3B30',
        },
      },
      { name: 'MA5', type: 'line', data: ma5, smooth: true, symbol: 'none', lineStyle: { width: 1, color: '#5B8CFF' } },
      { name: 'MA10', type: 'line', data: ma10, smooth: true, symbol: 'none', lineStyle: { width: 1, color: '#FFB300' } },
      { name: 'MA20', type: 'line', data: ma20, smooth: true, symbol: 'none', lineStyle: { width: 1, color: '#FF2D55' } },
      { name: 'VOL', type: 'bar', xAxisIndex: 1, yAxisIndex: 1, data: volumes },
    ],
  }
}

const render = () => {
  if (!chart.value) return
  // 空数据时清空画布，避免切换 symbol / 周期残留旧图
  if (!Array.isArray(props.bars) || props.bars.length === 0) {
    chart.value.clear()
    return
  }
  chart.value.setOption(buildOption(props.bars), true)
}

const handleResize = () => chart.value?.resize()

onMounted(() => {
  if (!rootEl.value) return
  chart.value = echarts.init(rootEl.value)
  render()
  window.addEventListener('resize', handleResize)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', handleResize)
  chart.value?.dispose()
  chart.value = null
})

watch(
  () => props.bars,
  () => render(),
)
</script>

<template>
  <div class="kline-wrap">
    <div ref="rootEl" class="kline" />
    <div v-if="loading" class="loading-mask">加载中...</div>
    <div v-if="!loading && bars.length === 0" class="empty">暂无数据</div>
  </div>
</template>

<style scoped>
.kline-wrap { position: relative; width: 100%; height: 420px; }
.kline { width: 100%; height: 100%; }
.loading-mask,
.empty {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-dim);
  font-size: 13px;
  background: rgba(11, 14, 19, 0.4);
}
</style>
