<script setup lang="ts">
// 入口接收统计卡：按 message_type / filedata 子类型展示累计条数和 QPS
// - 数据来自 /api/ingest-stats（stream_engine Pathway 侧实时写 Redis Hash）
// - QPS 由前端 store 基于两次拉取累计差 / 时间差本地计算，后端不维护滑窗
import { computed } from 'vue'
import type { IngestStats, IngestStatItem } from '@/types'

const props = defineProps<{
  snapshot: IngestStats | null
  qps: Record<string, number>
  totalQps: number
  loading: boolean
}>()

const fmt = (n: number): string => {
  if (!Number.isFinite(n)) return '-'
  return n.toLocaleString('en-US')
}

const fmtQps = (n: number): string => {
  if (!Number.isFinite(n) || n <= 0) return '0'
  if (n >= 100) return n.toFixed(0)
  if (n >= 10) return n.toFixed(1)
  return n.toFixed(2)
}

const messageTypes = computed<IngestStatItem[]>(
  () => props.snapshot?.message_types ?? [],
)
const fileDataTypes = computed<IngestStatItem[]>(
  () => props.snapshot?.file_data_types ?? [],
)

const totalCount = computed<number>(() => props.snapshot?.total_count ?? 0)

const uptimeLabel = computed<string>(() => {
  const started = props.snapshot?.started_at_ms ?? 0
  const now = props.snapshot?.now_ms ?? 0
  if (started <= 0 || now <= 0) return '-'
  const s = Math.max(0, Math.floor((now - started) / 1000))
  const d = Math.floor(s / 86400)
  const h = Math.floor((s % 86400) / 3600)
  const m = Math.floor((s % 3600) / 60)
  const sec = s % 60
  if (d > 0) return `${d}d ${h}h ${m}m`
  if (h > 0) return `${h}h ${m}m`
  if (m > 0) return `${m}m ${sec}s`
  return `${sec}s`
})

const mtQps = (mt: string): number => props.qps[`mt:${mt}`] ?? 0
const ftQps = (ft: string): number => props.qps[`ft:${ft}`] ?? 0
</script>

<template>
  <section class="wrap">
    <header class="bar">
      <div class="title">入口接收统计</div>
      <div class="hint dim">
        Pathway groupby+count · 总量 {{ fmt(totalCount) }} · 总 QPS
        <span class="mono">{{ fmtQps(totalQps) }}</span>
        · stream_engine 已运行 <span class="mono">{{ uptimeLabel }}</span>
      </div>
    </header>

    <div class="grid">
      <!-- message_type -->
      <section class="panel">
        <header class="ph"><span>按 message_type</span></header>
        <div v-if="!snapshot && loading" class="dim">加载中...</div>
        <table v-else class="tbl">
          <thead>
            <tr>
              <th>类型</th>
              <th class="mono">message_type</th>
              <th class="num">累计</th>
              <th class="num">QPS</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="messageTypes.length === 0">
              <td colspan="4" class="dim">暂无数据</td>
            </tr>
            <tr v-for="r in messageTypes" :key="r.message_type">
              <td>{{ r.label }}</td>
              <td class="mono dim">{{ r.message_type }}</td>
              <td class="num mono" :class="r.count === 0 ? 'warn' : ''">{{ fmt(r.count) }}</td>
              <td class="num mono">{{ fmtQps(mtQps(r.message_type ?? '')) }}</td>
            </tr>
          </tbody>
        </table>
      </section>

      <!-- file_data_type -->
      <section class="panel">
        <header class="ph"><span>RCV_FILEDATA 小类</span></header>
        <div v-if="!snapshot && loading" class="dim">加载中...</div>
        <table v-else class="tbl">
          <thead>
            <tr>
              <th>子类</th>
              <th class="mono">file_data_type</th>
              <th class="num">累计</th>
              <th class="num">QPS</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="fileDataTypes.length === 0">
              <td colspan="4" class="dim">暂无数据</td>
            </tr>
            <tr v-for="r in fileDataTypes" :key="r.file_data_type">
              <td>{{ r.label }}</td>
              <td class="mono dim">{{ r.file_data_type }}</td>
              <td class="num mono" :class="r.count === 0 ? 'warn' : ''">{{ fmt(r.count) }}</td>
              <td class="num mono">{{ fmtQps(ftQps(r.file_data_type ?? '')) }}</td>
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
.tbl th.num, .tbl td.num { text-align: right; }
.tbl .mono { font-family: var(--mono-font, ui-monospace, SFMono-Regular, Menlo, monospace); }
.tbl td.warn { color: var(--warn); }
</style>
