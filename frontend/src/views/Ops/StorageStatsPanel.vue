<script setup lang="ts">
// 数据存储统计卡：MySQL 各表行数 + Redis 各前缀 key 数
// - 数据来自 /api/storage-stats，轮询由父组件驱动
// - 单项失败只显示 error 文案，不影响其它项
import { computed } from 'vue'
import type { StorageStats, StorageStatItem } from '@/types'

const props = defineProps<{
  snapshot: StorageStats | null
  loading: boolean
}>()

const fmt = (n: number): string => {
  if (!Number.isFinite(n)) return '-'
  return n.toLocaleString('en-US')
}

const mysqlRows = computed<StorageStatItem[]>(() => props.snapshot?.postgres ?? [])
const redisRows = computed<StorageStatItem[]>(() => props.snapshot?.redis ?? [])
const scanLimit = computed<number>(() => props.snapshot?.scan_limit ?? 0)
</script>

<template>
  <section class="wrap">
    <header class="bar">
      <div class="title">数据存储统计</div>
      <div class="hint dim">
        MySQL 行数（COUNT(*)） · Redis 前缀 key 数（SCAN，上限 {{ fmt(scanLimit) }}）
      </div>
    </header>

    <div class="grid">
      <!-- MySQL -->
      <section class="panel">
        <header class="ph"><span>MySQL 表行数</span></header>
        <div v-if="!snapshot && loading" class="dim">加载中...</div>
        <table v-else class="tbl">
          <thead>
            <tr>
              <th>用途</th>
              <th>表</th>
              <th class="num">行数</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in mysqlRows" :key="r.table || r.label">
              <td>{{ r.label }}</td>
              <td class="mono dim">{{ r.table }}</td>
              <td class="num mono" :class="r.error ? 'err' : r.count === 0 ? 'warn' : ''">
                <span v-if="r.error" :title="r.error">ERR</span>
                <span v-else>{{ fmt(r.count) }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </section>

      <!-- Redis -->
      <section class="panel">
        <header class="ph"><span>Redis Key 数</span></header>
        <div v-if="!snapshot && loading" class="dim">加载中...</div>
        <table v-else class="tbl">
          <thead>
            <tr>
              <th>用途</th>
              <th>前缀 / Key</th>
              <th class="num">数量</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in redisRows" :key="(r.prefix || r.key || r.label)">
              <td>{{ r.label }}</td>
              <td class="mono dim">{{ r.prefix || r.key || '-' }}</td>
              <td class="num mono" :class="r.error ? 'err' : r.count === 0 ? 'warn' : ''">
                <span v-if="r.error" :title="r.error">ERR</span>
                <template v-else>
                  {{ fmt(r.count) }}
                  <span v-if="r.truncated" class="trunc" title="已达扫描上限，实际可能更多">+</span>
                </template>
              </td>
            </tr>
          </tbody>
        </table>
      </section>
    </div>
  </section>
</template>

<style scoped>
.wrap {
  background: var(--panel);
  border: 1px solid var(--divider);
  border-radius: 6px;
  padding: 12px 14px;
}
.bar { display: flex; align-items: baseline; justify-content: space-between; margin-bottom: 10px; gap: 12px; flex-wrap: wrap; }
.title { font-size: 13px; font-weight: 600; }
.hint { font-size: 12px; }
.dim { color: var(--text-dim); }

.grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}
@media (max-width: 900px) {
  .grid { grid-template-columns: 1fr; }
}
.panel {
  background: var(--panel-2, rgba(255,255,255,0.02));
  border: 1px solid var(--divider);
  border-radius: 4px;
  padding: 8px 10px;
}
.ph { font-size: 12px; color: var(--text-dim); margin-bottom: 6px; }

.tbl { width: 100%; border-collapse: collapse; font-size: 12px; }
.tbl th, .tbl td { padding: 4px 6px; text-align: left; border-bottom: 1px solid var(--divider); }
.tbl th { color: var(--text-dim); font-weight: 500; }
.tbl td.num, .tbl th.num { text-align: right; }
.err { color: var(--down); }
.warn { color: var(--warn); }
.trunc { color: var(--warn); margin-left: 2px; }
</style>
