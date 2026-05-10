<script setup lang="ts">
// 单榜组件：给定行集合 + 排序维度 + 正反向 + topN，渲染紧凑表格
// - 不再二次拉数据；父组件一次性算完 rows，所有榜共享
// - symbol 列链接到个股详情
import { computed } from 'vue'
import { Tag } from 'ant-design-vue'
import { fmtAmt, fmtPrice, trendClass } from '@/composables/useDecimal'
import { useMetaStore } from '@/stores/meta'
import type { BoardRow } from '@/composables/useScreener'

type Metric = 'change_pct_desc' | 'change_pct_asc' | 'turnover' | 'amplitude'

const props = defineProps<{
  title: string
  metric: Metric
  rows: BoardRow[]
  topN?: number
  valueLabel: string // 值列的表头（涨跌幅 / 成交额 / 振幅）
}>()

const metaStore = useMetaStore()

const sorted = computed<BoardRow[]>(() => {
  const n = props.topN ?? 20
  const list = [...props.rows]
  switch (props.metric) {
    case 'change_pct_desc':
      list.sort((a, b) => b.change_pct.cmp(a.change_pct))
      break
    case 'change_pct_asc':
      list.sort((a, b) => a.change_pct.cmp(b.change_pct))
      break
    case 'turnover':
      list.sort((a, b) => b.turnover_d.cmp(a.turnover_d))
      break
    case 'amplitude':
      list.sort((a, b) => b.amplitude.cmp(a.amplitude))
      break
  }
  return list.slice(0, n)
})

const renderValue = (r: BoardRow): string => {
  switch (props.metric) {
    case 'change_pct_desc':
    case 'change_pct_asc':
      return `${r.change_pct.mul(100).toFixed(2)}%`
    case 'turnover':
      return fmtAmt(r.turnover_d)
    case 'amplitude':
      return `${r.amplitude.mul(100).toFixed(2)}%`
  }
}

const valueClass = (r: BoardRow): string => {
  if (props.metric === 'turnover' || props.metric === 'amplitude') return 'mono'
  return `mono ${trendClass(r.change_pct)}`
}
</script>

<template>
  <section class="card">
    <header class="hd">
      <span class="title">{{ title }}</span>
      <Tag color="default">Top {{ sorted.length }}</Tag>
    </header>
    <table class="tbl">
      <thead>
        <tr>
          <th class="rank">#</th>
          <th>代码</th>
          <th>名称</th>
          <th class="num">最新</th>
          <th class="num">{{ valueLabel }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(r, i) in sorted" :key="r.symbol">
          <td class="rank mono dim">{{ i + 1 }}</td>
          <td>
            <router-link :to="`/stock/${r.symbol}`" class="mono link">
              {{ r.symbol }}
            </router-link>
          </td>
          <td class="name">{{ metaStore.nameOf(r.symbol) }}</td>
          <td class="num mono">{{ fmtPrice(r.last_price) }}</td>
          <td :class="['num', valueClass(r)]">{{ renderValue(r) }}</td>
        </tr>
        <tr v-if="sorted.length === 0">
          <td colspan="5" class="empty dim">暂无数据</td>
        </tr>
      </tbody>
    </table>
  </section>
</template>

<style scoped>
.card {
  background: var(--panel);
  border: 1px solid var(--divider);
  border-radius: 6px;
  padding: 10px 14px;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.hd {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}
.title { font-size: 14px; font-weight: 600; }
.tbl {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}
.tbl th {
  text-align: left;
  font-weight: 500;
  color: var(--text-dim);
  padding: 4px 6px;
  border-bottom: 1px solid var(--divider);
}
.tbl td {
  padding: 4px 6px;
  border-bottom: 1px dashed var(--divider);
}
.tbl tbody tr:hover { background: rgba(255, 255, 255, 0.02); }
.rank { width: 32px; text-align: right; }
.num { text-align: right; }
.name {
  max-width: 110px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.link { color: var(--info); text-decoration: none; }
.link:hover { text-decoration: underline; }
.dim { color: var(--text-dim); }
.empty { text-align: center; padding: 24px 0; }
.color-up { color: var(--up); }
.color-down { color: var(--down); }
.color-neutral { color: var(--neutral); }
</style>
