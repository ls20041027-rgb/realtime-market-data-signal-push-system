<script setup lang="ts">
// K 线 tab：日 / 5 分钟切换
// - 切周期时重新拉取；/api/kline 按 trade_date 倒序，这里正序后喂 ECharts
import { ref, watch } from 'vue'
import { Radio, RadioGroup } from 'ant-design-vue'
import KLineChart from '@/components/charts/KLineChart.vue'
import { fetchDailyKline, fetch5MinKline } from '@/api/kline'
import { createLogger } from '@/utils/logger'
import type { KLineBar } from '@/types'

const log = createLogger('stock:kline')

const props = defineProps<{ symbol: string }>()

type Period = 'daily' | '5m'
const period = ref<Period>('daily')
const bars = ref<KLineBar[]>([])
const loading = ref(false)

const sortAsc = (list: KLineBar[]): KLineBar[] => {
  if (!Array.isArray(list) || list.length === 0) return []
  return [...list].sort((a, b) => {
    const ka = a.trade_date || a.trade_time || ''
    const kb = b.trade_date || b.trade_time || ''
    return ka.localeCompare(kb)
  })
}

const load = async () => {
  if (!props.symbol) return
  loading.value = true
  try {
    const list = period.value === 'daily'
      ? await fetchDailyKline(props.symbol, { limit: 120 })
      : await fetch5MinKline(props.symbol, { limit: 240 })
    bars.value = sortAsc(list)
  } catch (err) {
    log.error('fetchKline failed', err)
    bars.value = []
  } finally {
    loading.value = false
  }
}

watch(() => [props.symbol, period.value], () => void load(), { immediate: true })
</script>

<template>
  <div class="kline-tab">
    <div class="bar">
      <RadioGroup v-model:value="period" button-style="solid" size="small">
        <Radio value="daily">日 K</Radio>
        <Radio value="5m">5 分钟</Radio>
      </RadioGroup>
    </div>
    <KLineChart :bars="bars" :loading="loading" />
  </div>
</template>

<style scoped>
.kline-tab { display: flex; flex-direction: column; gap: 8px; }
.bar { display: flex; justify-content: flex-end; }
</style>
