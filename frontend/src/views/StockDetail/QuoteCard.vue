<script setup lang="ts">
// 行情卡片：最新价 / 涨跌幅 / 成交量额 / 五档盘口
// 价格/金额严格走 Decimal，禁止 Number / parseFloat
import { computed } from 'vue'
import type { IndicatorSnapshot, QuoteSnapshot } from '@/types'
import { D, fmtPrice, fmtAmt, trendClass } from '@/composables/useDecimal'
import { fmtTime } from '@/utils/format'

const props = defineProps<{
  symbol: string
  name: string
  quote: QuoteSnapshot | null
  indicators: IndicatorSnapshot | null
}>()

interface LevelRow {
  label: string
  price: string
  volume: string
}

// 涨跌幅：优先后端 indicator.change_pct，否则本地用 last - prev 兜底
const change = computed(() => {
  const q = props.quote
  if (!q) return { pct: '0', amt: '0', cls: 'color-neutral' as const }
  const ind = props.indicators
  if (ind?.change_pct !== undefined && ind.change_amt !== undefined) {
    return {
      pct: D(ind.change_pct).toFixed(2),
      amt: D(ind.change_amt).toFixed(2),
      cls: trendClass(ind.change_pct),
    }
  }
  const prev = D(q.prev_close)
  const last = D(q.last_price)
  const amt = last.minus(prev)
  const pct = prev.gt(0) ? amt.div(prev).mul(100) : D(0)
  return { pct: pct.toFixed(2), amt: amt.toFixed(2), cls: trendClass(amt) }
})

// 五档（按"卖5..卖1 / 买1..买5"展示；后端 bid_levels/ask_levels 各 5 档）
const askRows = computed<LevelRow[]>(() => {
  const asks = props.quote?.ask_levels ?? []
  // 卖单 1..5 反向展示：卖5 在上、卖1 靠近买1
  return asks
    .slice(0, 5)
    .map((lv, i) => ({ label: `卖${i + 1}`, price: lv.price, volume: lv.volume }))
    .reverse()
})
const bidRows = computed<LevelRow[]>(() => {
  const bids = props.quote?.bid_levels ?? []
  return bids
    .slice(0, 5)
    .map((lv, i) => ({ label: `买${i + 1}`, price: lv.price, volume: lv.volume }))
})

// 五档价相对昨收的涨跌色
const levelCls = (price: string) => {
  const prev = D(props.quote?.prev_close ?? 0)
  if (!prev.gt(0)) return 'color-neutral'
  return trendClass(D(price).minus(prev))
}
</script>

<template>
  <div class="card">
    <div class="left">
      <div class="name-row">
        <span class="name">{{ name || symbol }}</span>
        <span class="symbol mono dim">{{ symbol }}</span>
      </div>
      <div class="price-row">
        <span class="last mono" :class="change.cls">
          {{ quote ? fmtPrice(quote.last_price) : '-' }}
        </span>
        <span class="chg mono" :class="change.cls">{{ change.amt }}</span>
        <span class="chg-pct mono" :class="change.cls">{{ change.pct }}%</span>
      </div>
      <div class="meta-row mono dim">
        <span>昨收 {{ quote ? fmtPrice(quote.prev_close) : '-' }}</span>
        <span>开盘 {{ quote ? fmtPrice(quote.open_price) : '-' }}</span>
        <span>最高 {{ quote ? fmtPrice(quote.high_price) : '-' }}</span>
        <span>最低 {{ quote ? fmtPrice(quote.low_price) : '-' }}</span>
        <span>量 {{ quote ? fmtAmt(quote.volume) : '-' }}</span>
        <span>额 {{ quote ? fmtAmt(quote.turnover) : '-' }}</span>
        <span v-if="indicators?.volume_ratio">量比 {{ indicators.volume_ratio }}</span>
        <span v-if="indicators?.turnover_rate">换手 {{ indicators.turnover_rate }}%</span>
        <span v-if="quote?.event_time">{{ fmtTime(quote.event_time) }}</span>
      </div>
    </div>

    <div class="book">
      <div class="book-col asks">
        <div v-for="row in askRows" :key="row.label" class="book-row">
          <span class="lb dim">{{ row.label }}</span>
          <span class="pr mono" :class="levelCls(row.price)">{{ fmtPrice(row.price) }}</span>
          <span class="vol mono dim">{{ row.volume }}</span>
        </div>
      </div>
      <div class="book-sep" />
      <div class="book-col bids">
        <div v-for="row in bidRows" :key="row.label" class="book-row">
          <span class="lb dim">{{ row.label }}</span>
          <span class="pr mono" :class="levelCls(row.price)">{{ fmtPrice(row.price) }}</span>
          <span class="vol mono dim">{{ row.volume }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.card {
  display: grid;
  grid-template-columns: 1fr 320px;
  gap: 24px;
  padding: 16px 20px;
  background: var(--panel);
  border: 1px solid var(--divider);
  border-radius: 6px;
}
.left { display: flex; flex-direction: column; gap: 10px; }
.name-row { display: flex; align-items: baseline; gap: 10px; }
.name { font-size: 18px; font-weight: 600; color: var(--text); }
.symbol { font-size: 13px; }
.dim { color: var(--text-dim); }

.price-row { display: flex; align-items: baseline; gap: 18px; }
.last { font-size: 34px; font-weight: 600; letter-spacing: 0.5px; }
.chg, .chg-pct { font-size: 16px; }

.meta-row {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 18px;
  font-size: 12px;
}

.book { display: grid; grid-template-rows: 1fr auto 1fr; }
.book-col { display: flex; flex-direction: column; gap: 2px; }
.book-row {
  display: grid;
  grid-template-columns: 40px 1fr 80px;
  align-items: center;
  height: 22px;
  font-size: 12px;
}
.book-row .lb { text-align: left; }
.book-row .pr { text-align: right; }
.book-row .vol { text-align: right; }
.book-sep { height: 1px; background: var(--divider); margin: 6px 0; }
</style>
