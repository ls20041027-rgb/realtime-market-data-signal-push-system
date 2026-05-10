<script setup lang="ts">
// 技术指标当前值面板：
// 后端 /api/indicators 返回的是"最新一帧"，并非时间序列，
// 所以这里不画副图曲线，而是用紧凑的数值卡片 + MACD 柱表达当前状态。
// 对齐 CONTRACT §二 tech:{symbol} 字段。
import { computed } from 'vue'
import type { IndicatorSnapshot } from '@/types'
import { D, fmtPrice, trendClass } from '@/composables/useDecimal'

const props = defineProps<{
  data: IndicatorSnapshot | null
  loading?: boolean
}>()

interface Cell {
  label: string
  value: string
  cls?: string
}

const mainCells = computed<Cell[]>(() => {
  const d = props.data
  if (!d) return []
  return [
    { label: 'MA5', value: d.ma5 ? fmtPrice(d.ma5) : '-' },
    { label: 'MA10', value: d.ma10 ? fmtPrice(d.ma10) : '-' },
    { label: 'MA20', value: d.ma20 ? fmtPrice(d.ma20) : '-' },
    { label: 'MA60', value: d.ma60 ? fmtPrice(d.ma60) : '-' },
    { label: 'RSI14', value: d.rsi14 ? fmtPrice(d.rsi14) : '-' },
    { label: 'KDJ-K', value: d.kdj_k ? fmtPrice(d.kdj_k) : '-' },
    { label: 'KDJ-D', value: d.kdj_d ? fmtPrice(d.kdj_d) : '-' },
    { label: 'KDJ-J', value: d.kdj_j ? fmtPrice(d.kdj_j) : '-' },
    { label: 'BOLL-上', value: d.boll_up ? fmtPrice(d.boll_up) : '-' },
    { label: 'BOLL-中', value: d.boll_mid ? fmtPrice(d.boll_mid) : '-' },
    { label: 'BOLL-下', value: d.boll_low ? fmtPrice(d.boll_low) : '-' },
  ]
})

const macdCells = computed<Cell[]>(() => {
  const d = props.data
  if (!d) return []
  return [
    { label: 'DIF', value: d.macd_dif ? fmtPrice(d.macd_dif, 4) : '-', cls: trendClass(d.macd_dif ?? 0) },
    { label: 'DEA', value: d.macd_dea ? fmtPrice(d.macd_dea, 4) : '-', cls: trendClass(d.macd_dea ?? 0) },
    { label: 'HIST', value: d.macd_hist ? fmtPrice(d.macd_hist, 4) : '-', cls: trendClass(d.macd_hist ?? 0) },
  ]
})

// MACD hist 柱形图（单一柱，表达当前多空强度）
const macdBarWidth = computed(() => {
  const h = D(props.data?.macd_hist ?? 0)
  // 经验范围：|hist| <= 2 左右，映射到 0~100%
  const pct = h.abs().div(2).mul(100)
  const n = Number(pct.toFixed(0))
  return Math.min(100, Math.max(4, n))
})

const macdBarClass = computed(() => trendClass(props.data?.macd_hist ?? 0))
</script>

<template>
  <div class="panel">
    <div class="head">
      <span class="title">技术指标</span>
      <span v-if="loading" class="dim">更新中...</span>
      <span v-else-if="data?.updated_at" class="dim mono">{{ data.updated_at }}</span>
    </div>

    <div v-if="!data" class="empty">暂无指标数据</div>

    <template v-else>
      <div class="grid">
        <div v-for="c in mainCells" :key="c.label" class="cell">
          <div class="label">{{ c.label }}</div>
          <div class="value mono">{{ c.value }}</div>
        </div>
      </div>

      <div class="macd">
        <div class="macd-cells">
          <div v-for="c in macdCells" :key="c.label" class="cell">
            <div class="label">{{ c.label }}</div>
            <div class="value mono" :class="c.cls">{{ c.value }}</div>
          </div>
        </div>
        <div class="bar-wrap">
          <div class="bar-label dim">MACD 柱</div>
          <div class="bar-track">
            <div class="bar-fill" :class="macdBarClass" :style="{ width: `${macdBarWidth}%` }" />
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.panel {
  background: var(--panel);
  border: 1px solid var(--divider);
  border-radius: 6px;
  padding: 12px 14px;
}
.head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 10px;
}
.title { font-size: 14px; font-weight: 600; color: var(--text); }
.dim { color: var(--text-dim); font-size: 12px; }
.empty { color: var(--text-dim); font-size: 13px; padding: 16px 0; }
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(110px, 1fr));
  gap: 8px 16px;
}
.cell { display: flex; flex-direction: column; gap: 2px; }
.label { color: var(--text-dim); font-size: 12px; }
.value { color: var(--text); font-size: 14px; }

.macd {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px dashed var(--divider);
}
.macd-cells {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px 16px;
  margin-bottom: 10px;
}
.bar-wrap { display: flex; align-items: center; gap: 8px; }
.bar-label { font-size: 12px; min-width: 56px; }
.bar-track {
  flex: 1;
  height: 8px;
  background: rgba(255, 255, 255, 0.04);
  border-radius: 2px;
  overflow: hidden;
}
.bar-fill { height: 100%; transition: width 0.2s ease; }
.bar-fill.color-up { background: var(--up); }
.bar-fill.color-down { background: var(--down); }
.bar-fill.color-neutral { background: var(--neutral); }
</style>
