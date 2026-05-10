<script setup lang="ts">
// 个股详情页总装：顶栏（返回/代码/名称/交易所） + 行情卡片 + 7 个 tab
import { computed, onMounted, ref, toRefs } from 'vue'
import { useRouter } from 'vue-router'
import { Button, Tabs, Tag, TabPane } from 'ant-design-vue'
import { useMetaStore } from '@/stores/meta'
import { useStockDetail } from '@/composables/useStockDetail'
import { detectExchange, isValidSymbol, normalizeSymbol } from '@/utils/symbol'
import QuoteCard from './QuoteCard.vue'
import KLineTab from './KLineTab.vue'
import IndicatorsPanel from './IndicatorsPanel.vue'
import FenbiTab from './FenbiTab.vue'
import CapitalTab from './CapitalTab.vue'
import FinanceTab from './FinanceTab.vue'
import SignalTab from './SignalTab.vue'

const props = defineProps<{ symbol: string }>()
const { symbol: symbolProp } = toRefs(props)

const router = useRouter()
const metaStore = useMetaStore()

// 规范化 symbol（大写 + 去空格）；非法 symbol 不订阅
const normalized = computed(() => normalizeSymbol(symbolProp.value))
const valid = computed(() => isValidSymbol(normalized.value))
const exchange = computed(() => detectExchange(normalized.value))

const name = computed(() => metaStore.nameOf(normalized.value))

const { quote, indicators, capital, liveSignals } = useStockDetail(normalized)

onMounted(() => {
  void metaStore.load()
})

const activeKey = ref<string>('kline')

const goBack = () => {
  if (window.history.length > 1) router.back()
  else router.push('/watchlist')
}

const exchangeLabel: Record<string, string> = {
  SSE: '上交所',
  SZSE: '深交所',
  BSE: '北交所',
}
</script>

<template>
  <div class="page">
    <header class="bar">
      <Button size="small" @click="goBack">← 返回</Button>
      <div class="title">
        <span class="name">{{ name }}</span>
        <span class="symbol mono dim">{{ normalized }}</span>
        <Tag v-if="exchange" color="blue">{{ exchangeLabel[exchange] }}</Tag>
      </div>
    </header>

    <div v-if="!valid" class="invalid">非法的 symbol：{{ symbolProp }}</div>

    <template v-else>
      <QuoteCard
        :symbol="normalized"
        :name="name"
        :quote="quote"
        :indicators="indicators"
      />

      <Tabs v-model:active-key="activeKey" class="tabs">
        <TabPane key="kline" tab="K 线">
          <KLineTab :symbol="normalized" />
        </TabPane>
        <TabPane key="indicators" tab="技术指标">
          <IndicatorsPanel :data="indicators" />
        </TabPane>
        <TabPane key="fenbi" tab="分笔">
          <FenbiTab :symbol="normalized" />
        </TabPane>
        <TabPane key="capital" tab="资金流">
          <CapitalTab :data="capital" />
        </TabPane>
        <TabPane key="finance" tab="财务">
          <FinanceTab :symbol="normalized" />
        </TabPane>
        <TabPane key="signal" tab="信号">
          <SignalTab :symbol="normalized" :live-signals="liveSignals" />
        </TabPane>
      </Tabs>
    </template>
  </div>
</template>

<style scoped>
.page { display: flex; flex-direction: column; gap: 14px; color: var(--text); }
.bar { display: flex; align-items: center; gap: 14px; }
.title { display: flex; align-items: baseline; gap: 10px; }
.name { font-size: 16px; font-weight: 600; }
.symbol { font-size: 13px; }
.dim { color: var(--text-dim); }
.invalid { color: var(--down); padding: 24px 0; text-align: center; }
.tabs :deep(.ant-tabs-nav) { margin-bottom: 12px; }
</style>
