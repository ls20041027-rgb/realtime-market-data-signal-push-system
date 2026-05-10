<script setup lang="ts">
// 选股榜单：2x2（涨幅 / 跌幅 / 成交额 / 振幅）客户端聚合
// - 候选池：自选股 / 全市场（默认前 500 只）
// - 批量 /api/quotes 分片并发；Decimal 派生 change_pct / amplitude
// - 全市场大候选池默认不自动刷；自选股 15s 自动刷
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Button, InputNumber, Radio, RadioGroup, Space, Tag } from 'ant-design-vue'
import BoardCard from './BoardCard.vue'
import { useScreener } from '@/composables/useScreener'
import { useMetaStore } from '@/stores/meta'
import { useWatchlistStore } from '@/stores/watchlist'
import { fmtTime } from '@/utils/format'

type PoolMode = 'watchlist' | 'market'

const metaStore = useMetaStore()
const watchStore = useWatchlistStore()
const { rows, loading, updatedAt, error, load } = useScreener()

const poolMode = ref<PoolMode>('watchlist')
/** 全市场模式下的最大候选规模，默认 500，避免打爆 push_gateway */
const marketLimit = ref<number>(500)

const poolSymbols = computed<string[]>(() => {
  if (poolMode.value === 'watchlist') return [...watchStore.symbols]
  const all = Object.keys(metaStore.nameMap)
  return all.slice(0, Math.max(1, marketLimit.value))
})

const refresh = () => load(poolSymbols.value)

// 自动刷新：只在 watchlist 模式开启（全市场太重，手动触发）
let timer: ReturnType<typeof setInterval> | null = null
const startAuto = () => {
  stopAuto()
  timer = setInterval(() => {
    if (document.visibilityState === 'visible' && poolMode.value === 'watchlist') {
      refresh()
    }
  }, 15000)
}
const stopAuto = () => {
  if (timer) {
    clearInterval(timer)
    timer = null
  }
}

onMounted(async () => {
  await metaStore.load()
  refresh()
  startAuto()
})
onBeforeUnmount(stopAuto)

watch(poolMode, () => refresh())
</script>

<template>
  <div class="page">
    <header class="bar">
      <div class="title">
        <h2>选股榜单</h2>
        <Tag color="default">候选池 {{ poolSymbols.length }} 只</Tag>
        <Tag v-if="updatedAt > 0" color="default" class="mono">
          更新 {{ fmtTime(new Date(updatedAt).toISOString()) }}
        </Tag>
        <Tag v-if="error" color="red">加载失败：{{ error }}</Tag>
      </div>
      <Space>
        <RadioGroup v-model:value="poolMode" button-style="solid" size="small">
          <Radio value="watchlist">自选股</Radio>
          <Radio value="market">全市场</Radio>
        </RadioGroup>
        <InputNumber
          v-if="poolMode === 'market'"
          v-model:value="marketLimit"
          :min="50"
          :max="2000"
          :step="50"
          size="small"
          style="width: 110px"
        />
        <Button :loading="loading" type="primary" @click="refresh">刷新</Button>
      </Space>
    </header>

    <div class="grid">
      <BoardCard title="涨幅榜" metric="change_pct_desc" value-label="涨跌幅" :rows="rows" :top-n="20" />
      <BoardCard title="跌幅榜" metric="change_pct_asc" value-label="涨跌幅" :rows="rows" :top-n="20" />
      <BoardCard title="成交额榜" metric="turnover" value-label="成交额" :rows="rows" :top-n="20" />
      <BoardCard title="振幅榜" metric="amplitude" value-label="振幅" :rows="rows" :top-n="20" />
    </div>
  </div>
</template>

<style scoped>
.page { padding: 16px; color: var(--text); }
.bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  gap: 12px;
  flex-wrap: wrap;
}
.title { display: flex; align-items: center; gap: 10px; }
.title h2 { margin: 0; font-size: 18px; font-weight: 600; }
.grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  grid-auto-rows: minmax(0, 1fr);
  gap: 12px;
}
@media (max-width: 1100px) {
  .grid { grid-template-columns: 1fr; }
}
</style>
