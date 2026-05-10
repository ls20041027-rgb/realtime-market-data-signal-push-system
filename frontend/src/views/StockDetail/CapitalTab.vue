<script setup lang="ts">
// 资金流 tab：/api/capital 已由 useStockDetail 轮询；这里只做展示
import { computed } from 'vue'
import type { CapitalSnapshot } from '@/types'
import { D, fmtAmt, trendClass } from '@/composables/useDecimal'

const props = defineProps<{
  data: CapitalSnapshot | null
  loading?: boolean
}>()

// 买/卖力度条：按 abs(big_buy) 与 abs(big_sell) 归一化
const ratio = computed(() => {
  const d = props.data
  if (!d) return { buy: 50, sell: 50 }
  const b = D(d.big_buy).abs()
  const s = D(d.big_sell).abs()
  const sum = b.plus(s)
  if (!sum.gt(0)) return { buy: 50, sell: 50 }
  return {
    buy: Number(b.div(sum).mul(100).toFixed(0)),
    sell: Number(s.div(sum).mul(100).toFixed(0)),
  }
})

const netCls = computed(() => trendClass(props.data?.net_inflow ?? 0))
</script>

<template>
  <div class="wrap">
    <div v-if="!data && !loading" class="empty">暂无资金流数据</div>
    <template v-else-if="data">
      <div class="row top">
        <div class="stat">
          <div class="label">大单净流入</div>
          <div class="value mono big" :class="netCls">{{ fmtAmt(data.net_inflow) }}</div>
        </div>
        <div class="stat">
          <div class="label">大单买入</div>
          <div class="value mono color-up">{{ fmtAmt(data.big_buy) }}</div>
        </div>
        <div class="stat">
          <div class="label">大单卖出</div>
          <div class="value mono color-down">{{ fmtAmt(data.big_sell) }}</div>
        </div>
        <div class="stat">
          <div class="label">买 Tick</div>
          <div class="value mono">{{ data.buy_tick_count }}</div>
        </div>
        <div class="stat">
          <div class="label">卖 Tick</div>
          <div class="value mono">{{ data.sell_tick_count }}</div>
        </div>
        <div class="stat">
          <div class="label">重置日期</div>
          <div class="value mono dim">{{ data.last_reset_date }}</div>
        </div>
      </div>

      <div class="bar-group">
        <div class="bar-label dim">买卖力度</div>
        <div class="bar-track">
          <div class="bar-fill up" :style="{ width: `${ratio.buy}%` }" />
          <div class="bar-fill down" :style="{ width: `${ratio.sell}%` }" />
        </div>
        <div class="bar-legend mono dim">
          <span class="color-up">买 {{ ratio.buy }}%</span>
          <span class="color-down">卖 {{ ratio.sell }}%</span>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.wrap {
  background: var(--panel);
  border: 1px solid var(--divider);
  border-radius: 6px;
  padding: 16px 18px;
}
.empty { color: var(--text-dim); padding: 16px 0; text-align: center; }
.row.top {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 14px 24px;
}
.stat { display: flex; flex-direction: column; gap: 4px; }
.label { color: var(--text-dim); font-size: 12px; }
.value { color: var(--text); font-size: 15px; }
.value.big { font-size: 22px; font-weight: 600; }
.dim { color: var(--text-dim); }

.bar-group { margin-top: 18px; display: flex; flex-direction: column; gap: 6px; }
.bar-label { font-size: 12px; }
.bar-track {
  display: flex;
  height: 10px;
  border-radius: 2px;
  overflow: hidden;
  background: rgba(255, 255, 255, 0.04);
}
.bar-fill.up { background: var(--up); }
.bar-fill.down { background: var(--down); }
.bar-legend { display: flex; gap: 12px; font-size: 12px; }
</style>
