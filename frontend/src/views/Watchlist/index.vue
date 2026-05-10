<script setup lang="ts">
// 自选股页总装：顶栏（搜索/添加/刷新） + 表格 + 空状态
import { ref } from 'vue'
import { Button, Empty, Space } from 'ant-design-vue'
import { useWatchlistStore } from '@/stores/watchlist'
import { useWatchlistQuotes } from '@/composables/useWatchlistQuotes'
import WatchlistTable from './WatchlistTable.vue'
import AddSymbolModal from './AddSymbolModal.vue'

const watchStore = useWatchlistStore()
const { rows, refreshing, refresh } = useWatchlistQuotes()

const addOpen = ref(false)
const openAdd = () => (addOpen.value = true)
const handleRemove = (symbol: string) => watchStore.remove(symbol)
</script>

<template>
  <div class="page">
    <header class="bar">
      <div class="title">
        <h2>自选股</h2>
        <span class="count mono">{{ watchStore.symbols.length }}</span>
      </div>
      <Space>
        <Button :loading="refreshing" @click="refresh">刷新</Button>
        <Button type="primary" @click="openAdd">＋ 添加</Button>
      </Space>
    </header>

    <WatchlistTable v-if="rows.length > 0" :rows="rows" @remove="handleRemove" />
    <Empty v-else description="暂无自选股，点击右上角添加" class="empty" />

    <AddSymbolModal v-model:open="addOpen" />
  </div>
</template>

<style scoped>
.page { padding: 16px; color: var(--text); }
.bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}
.title { display: flex; align-items: baseline; gap: 12px; }
.title h2 { margin: 0; font-size: 18px; font-weight: 600; }
.count { color: var(--text-dim); font-size: 13px; }
.empty { margin-top: 64px; }
</style>
