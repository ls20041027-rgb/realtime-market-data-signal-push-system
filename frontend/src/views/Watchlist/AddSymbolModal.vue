<script setup lang="ts">
// 添加自选股弹窗：基于 meta.search 本地模糊匹配
import { computed, ref, watch } from 'vue'
import { Modal, Input, message } from 'ant-design-vue'
import { useMetaStore } from '@/stores/meta'
import { useWatchlistStore } from '@/stores/watchlist'
import { isValidSymbol, normalizeSymbol } from '@/utils/symbol'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ (e: 'update:open', v: boolean): void }>()

const metaStore = useMetaStore()
const watchStore = useWatchlistStore()

const keyword = ref('')
const results = computed(() => metaStore.search(keyword.value, 30))

watch(
  () => props.open,
  (v) => {
    if (v) {
      keyword.value = ''
      void metaStore.load()
    }
  },
)

const handleAdd = (symbol: string) => {
  const sym = normalizeSymbol(symbol)
  if (!isValidSymbol(sym)) {
    message.warning('invalid symbol')
    return
  }
  if (watchStore.has(sym)) {
    message.info('already in watchlist')
    return
  }
  watchStore.add(sym)
  message.success(`added ${sym}`)
}

// 直接添加（当输入恰好是合法 symbol 时）
const handleAddDirect = () => {
  const sym = normalizeSymbol(keyword.value)
  if (isValidSymbol(sym)) handleAdd(sym)
}

const close = () => emit('update:open', false)
</script>

<template>
  <Modal
    :open="props.open"
    title="添加自选股"
    :footer="null"
    width="520px"
    :body-style="{ padding: '16px' }"
    @cancel="close"
    @update:open="(v) => emit('update:open', v)"
  >
    <Input
      v-model:value="keyword"
      placeholder="输入代码（如 SH600519）或名称（如 贵州茅台）"
      allow-clear
      @press-enter="handleAddDirect"
    />
    <div class="hint">
      <span v-if="!metaStore.loaded">码表加载中…</span>
      <span v-else-if="!keyword">已收录 {{ metaStore.count }} 只</span>
      <span v-else-if="results.length === 0" class="dim">无匹配结果</span>
      <span v-else class="dim">共 {{ results.length }} 条</span>
    </div>
    <ul class="list">
      <li
        v-for="item in results"
        :key="item.symbol"
        class="row"
        :class="{ disabled: watchStore.has(item.symbol) }"
        @click="handleAdd(item.symbol)"
      >
        <span class="sym mono">{{ item.symbol }}</span>
        <span class="name">{{ item.name }}</span>
        <span class="tag">{{ watchStore.has(item.symbol) ? '已添加' : '+ 添加' }}</span>
      </li>
    </ul>
  </Modal>
</template>

<style scoped>
.hint { padding: 8px 4px; font-size: 12px; color: var(--text-dim); }
.list { list-style: none; margin: 0; padding: 0; max-height: 360px; overflow: auto; }
.row {
  display: grid;
  grid-template-columns: 100px 1fr 72px;
  align-items: center;
  gap: 12px;
  padding: 8px 10px;
  border-radius: 4px;
  cursor: pointer;
  transition: background 0.1s;
}
.row:hover { background: rgba(255, 255, 255, 0.04); }
.row.disabled { cursor: not-allowed; opacity: 0.5; }
.sym { color: var(--text); }
.name { color: var(--text-dim); font-size: 13px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.tag { color: var(--info); font-size: 12px; text-align: right; }
.dim { color: var(--text-dim); }
</style>
