<script setup lang="ts">
// LiveTicker：订阅 signal:ALL，展示最近 30 条滚动信号（头插）
// signal store 内部已做 LIVE_MAX=30 + unread 计数
import { onMounted } from 'vue'
import { Tag, Empty } from 'ant-design-vue'
import { useSignalStore } from '@/stores/signal'
import { useMetaStore } from '@/stores/meta'
import { fmtPrice } from '@/composables/useDecimal'
import { fmtTime } from '@/utils/format'
import type { SignalAction, SignalSeverity } from '@/types'

const signalStore = useSignalStore()
const metaStore = useMetaStore()

onMounted(() => {
  signalStore.bind()
  signalStore.markRead()
  void metaStore.load()
})

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
</script>

<template>
  <div class="ticker">
    <div class="head">
      <span class="title">实时信号流</span>
      <span class="dim mono">WS signal:ALL · 最近 30 条</span>
    </div>
    <div v-if="signalStore.live.length === 0" class="empty">
      <Empty description="等待实时信号..." />
    </div>
    <ul v-else class="list">
      <li v-for="s in signalStore.live" :key="s.signal_id" class="item">
        <span class="mono dim time">{{ fmtTime(s.signal_time) }}</span>
        <Tag :color="actionColor[s.action]">{{ s.action }}</Tag>
        <router-link :to="`/stock/${s.symbol}`" class="symbol mono">{{ s.symbol }}</router-link>
        <span class="name">{{ metaStore.nameOf(s.symbol) }}</span>
        <span class="strat dim">{{ s.strategy_name }}</span>
        <span class="mono">触发 {{ fmtPrice(s.trigger_price) }}</span>
        <span class="mono dim">置信 {{ s.confidence }}</span>
        <Tag v-if="s.severity" :color="severityColor[s.severity]">{{ s.severity }}</Tag>
        <span class="reason">{{ s.reason }}</span>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.ticker {
  background: var(--panel);
  border: 1px solid var(--divider);
  border-radius: 6px;
  padding: 10px 14px;
}
.head { display: flex; align-items: baseline; justify-content: space-between; margin-bottom: 8px; }
.title { font-size: 14px; font-weight: 600; }
.dim { color: var(--text-dim); font-size: 12px; }
.empty { padding: 16px 0; }
.list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: 260px;
  overflow: auto;
}
.item {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 12px;
  padding: 2px 0;
}
.time { min-width: 72px; }
.symbol { color: var(--info); text-decoration: none; }
.symbol:hover { text-decoration: underline; }
.name { min-width: 80px; }
.strat { min-width: 100px; }
.reason { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text); }
</style>
