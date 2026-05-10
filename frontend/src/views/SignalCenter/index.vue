<script setup lang="ts">
// 信号中心：实时流 + 历史筛选 + 详情抽屉
import { computed, onMounted, ref, watch } from 'vue'
import { Badge, Descriptions, DescriptionsItem, Drawer, Tag } from 'ant-design-vue'
import FilterPanel from './FilterPanel.vue'
import SignalList from './SignalList.vue'
import LiveTicker from './LiveTicker.vue'
import { usePagination } from '@/composables/usePagination'
import { fetchSignals, fetchSignalById, type SignalQuery } from '@/api/signal'
import { useSignalStore } from '@/stores/signal'
import { useMetaStore } from '@/stores/meta'
import { createLogger } from '@/utils/logger'
import { fmtPrice } from '@/composables/useDecimal'
import type { SignalAction, SignalSeverity, TradingSignal } from '@/types'

const log = createLogger('signal-center')

const signalStore = useSignalStore()
const metaStore = useMetaStore()

const query = ref<SignalQuery>({})

const pager = usePagination<TradingSignal>(
  (page, pageSize) =>
    fetchSignals({ ...query.value, page, page_size: pageSize }),
  { pageSize: 20 },
)
const { items, total, page, pageSize, loading } = pager

const onFilterChange = (q: SignalQuery) => {
  query.value = q
  void pager.load(1)
}
const onPageChange = (p: number, ps: number) => {
  pageSize.value = ps
  void pager.load(p)
}

// 详情抽屉
const drawerOpen = ref(false)
const detail = ref<TradingSignal | null>(null)
const detailLoading = ref(false)

const openDetail = async (id: string) => {
  drawerOpen.value = true
  detail.value = null
  detailLoading.value = true
  try {
    detail.value = await fetchSignalById(id)
  } catch (err) {
    log.error('fetchSignalById failed', err)
  } finally {
    detailLoading.value = false
  }
}

onMounted(() => {
  signalStore.bind()
  void metaStore.load()
  void pager.load(1)
})

// 切页面离开时读掉未读
watch(
  () => signalStore.unread,
  () => {
    // 进到信号中心则清空红点
    if (signalStore.unread > 0) signalStore.markRead()
  },
  { immediate: true },
)

const actionColor: Record<SignalAction, string> = {
  BUY: 'green',
  SELL: 'red',
  WATCH: 'blue',
  RISK: 'orange',
}
const severityColor: Record<SignalSeverity, string> = {
  INFO: 'default',
  WARN: 'orange',
  CRITICAL: 'red',
}

const detailName = computed(() =>
  detail.value ? metaStore.nameOf(detail.value.symbol) : '',
)
</script>

<template>
  <div class="page">
    <header class="bar">
      <div class="title">
        <h2>信号中心</h2>
        <Badge
          v-if="signalStore.unread > 0"
          :count="signalStore.unread"
          :offset="[4, -2]"
        />
      </div>
    </header>

    <LiveTicker />

    <FilterPanel @change="onFilterChange" />

    <SignalList
      :items="items"
      :total="total"
      :page="page"
      :page-size="pageSize"
      :loading="loading"
      @page-change="onPageChange"
      @detail="openDetail"
    />

    <Drawer
      v-model:open="drawerOpen"
      title="信号详情"
      width="520"
      :body-style="{ padding: '16px' }"
    >
      <div v-if="detailLoading" class="dim">加载中...</div>
      <div v-else-if="!detail" class="dim">未找到信号</div>
      <template v-else>
        <Descriptions :column="1" size="small" bordered>
          <DescriptionsItem label="ID">
            <span class="mono">{{ detail.signal_id }}</span>
          </DescriptionsItem>
          <DescriptionsItem label="代码">
            <router-link :to="`/stock/${detail.symbol}`" class="mono link">
              {{ detail.symbol }}
            </router-link>
            <span class="dim"> · {{ detailName }}</span>
          </DescriptionsItem>
          <DescriptionsItem label="时间">
            <span class="mono">{{ detail.signal_time }}</span>
          </DescriptionsItem>
          <DescriptionsItem label="动作">
            <Tag :color="actionColor[detail.action]">{{ detail.action }}</Tag>
          </DescriptionsItem>
          <DescriptionsItem label="类型">{{ detail.signal_type }}</DescriptionsItem>
          <DescriptionsItem label="策略">{{ detail.strategy_name }}</DescriptionsItem>
          <DescriptionsItem label="触发价">
            <span class="mono">{{ fmtPrice(detail.trigger_price) }}</span>
          </DescriptionsItem>
          <DescriptionsItem label="置信度">
            <span class="mono">{{ detail.confidence }}</span>
          </DescriptionsItem>
          <DescriptionsItem v-if="detail.severity" label="等级">
            <Tag :color="severityColor[detail.severity]">{{ detail.severity }}</Tag>
          </DescriptionsItem>
          <DescriptionsItem v-if="detail.summary" label="摘要">
            {{ detail.summary }}
          </DescriptionsItem>
          <DescriptionsItem label="原因">{{ detail.reason }}</DescriptionsItem>
          <DescriptionsItem v-if="detail.indicators" label="指标">
            <pre class="ind mono">{{ JSON.stringify(detail.indicators, null, 2) }}</pre>
          </DescriptionsItem>
        </Descriptions>
      </template>
    </Drawer>
  </div>
</template>

<style scoped>
.page { display: flex; flex-direction: column; gap: 14px; color: var(--text); }
.bar { display: flex; align-items: center; justify-content: space-between; }
.title { display: flex; align-items: center; gap: 10px; }
.title h2 { margin: 0; font-size: 18px; font-weight: 600; }
.dim { color: var(--text-dim); font-size: 13px; }
.link { color: var(--info); text-decoration: none; }
.link:hover { text-decoration: underline; }
.ind {
  margin: 0;
  padding: 8px;
  font-size: 12px;
  background: rgba(255, 255, 255, 0.03);
  border-radius: 4px;
  max-height: 200px;
  overflow: auto;
}
</style>
