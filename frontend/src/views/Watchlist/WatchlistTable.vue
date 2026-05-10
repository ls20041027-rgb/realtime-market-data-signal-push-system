<script setup lang="ts">
// 自选股表格：< 50 行用 a-table，>= 50 行切虚拟滚动
// R11: UI 密度优先，固定列 + 涨跌色 + 右键删除
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { Table as ATable, Popconfirm } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import { useVirtualizer } from '@tanstack/vue-virtual'
import { shallowRef, watchEffect } from 'vue'
import type { WatchRow } from '@/composables/useWatchlistQuotes'
import { fmtPrice, fmtAmt, trendClass } from '@/composables/useDecimal'
import { fmtTime } from '@/utils/format'

const props = defineProps<{ rows: WatchRow[] }>()
const emit = defineEmits<{ (e: 'remove', symbol: string): void }>()

const router = useRouter()
const VIRTUAL_THRESHOLD = 50
const ROW_HEIGHT = 40

const useVirtual = computed(() => props.rows.length >= VIRTUAL_THRESHOLD)
const rowsRef = computed(() => props.rows)

// a-table 列定义
const columns: TableColumnsType<WatchRow> = [
  { title: '代码', dataIndex: 'symbol', key: 'symbol', width: 110 },
  { title: '名称', dataIndex: 'name', key: 'name', width: 140, ellipsis: true },
  { title: '最新价', key: 'last_price', align: 'right', width: 100 },
  { title: '涨跌幅', key: 'change_pct', align: 'right', width: 100 },
  { title: '涨跌额', key: 'change_amt', align: 'right', width: 100 },
  { title: '成交额', key: 'turnover', align: 'right', width: 120 },
  { title: '更新', key: 'event_time', align: 'right', width: 90 },
  { title: '', key: 'ops', width: 60, align: 'center' },
]

const goDetail = (symbol: string) => router.push(`/stock/${symbol}`)

// 虚拟滚动
const parentRef = shallowRef<HTMLElement | null>(null)
const virtualizer = useVirtualizer(
  computed(() => ({
    count: rowsRef.value.length,
    getScrollElement: () => parentRef.value,
    estimateSize: () => ROW_HEIGHT,
    overscan: 8,
  })),
)

const virtualItems = computed(() => virtualizer.value.getVirtualItems())
const totalSize = computed(() => virtualizer.value.getTotalSize())

watchEffect(() => {
  // rows 变化时刷新尺寸，避免残留 itemMeasure
  virtualizer.value.measure()
})
</script>

<template>
  <div class="wrap">
    <!-- 小数据量：Ant Design 表格，功能最完整 -->
    <ATable
      v-if="!useVirtual"
      :columns="columns"
      :data-source="rows"
      :pagination="false"
      row-key="symbol"
      size="middle"
      :custom-row="(r: WatchRow) => ({ onClick: () => goDetail(r.symbol), style: { cursor: 'pointer' } })"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'last_price'">
          <span class="mono">{{ record.last_price === '-' ? '-' : fmtPrice(record.last_price) }}</span>
        </template>
        <template v-else-if="column.key === 'change_pct'">
          <span class="mono" :class="trendClass(record.change_pct)">
            {{ record.change_pct }}%
          </span>
        </template>
        <template v-else-if="column.key === 'change_amt'">
          <span class="mono" :class="trendClass(record.change_amt)">{{ record.change_amt }}</span>
        </template>
        <template v-else-if="column.key === 'turnover'">
          <span class="mono">{{ record.turnover === '-' ? '-' : fmtAmt(record.turnover) }}</span>
        </template>
        <template v-else-if="column.key === 'event_time'">
          <span class="mono dim">{{ fmtTime(record.event_time) }}</span>
        </template>
        <template v-else-if="column.key === 'ops'">
          <Popconfirm
            title="移除自选？"
            :ok-text="'确认'"
            :cancel-text="'取消'"
            @confirm.stop="emit('remove', record.symbol)"
            @click.stop
          >
            <a class="del" @click.stop>×</a>
          </Popconfirm>
        </template>
      </template>
    </ATable>

    <!-- 大数据量：虚拟滚动，仅渲染可见行 -->
    <div v-else class="virtual-wrap">
      <div class="vrow header mono">
        <div class="c sym">代码</div>
        <div class="c name">名称</div>
        <div class="c num">最新价</div>
        <div class="c num">涨跌幅</div>
        <div class="c num">涨跌额</div>
        <div class="c num">成交额</div>
        <div class="c num">更新</div>
        <div class="c ops"></div>
      </div>
      <div ref="parentRef" class="vscroll">
        <div class="vinner" :style="{ height: `${totalSize}px` }">
          <div
            v-for="vi in virtualItems"
            :key="String(vi.key)"
            class="vrow"
            :style="{
              position: 'absolute',
              top: 0,
              left: 0,
              right: 0,
              height: `${vi.size}px`,
              transform: `translateY(${vi.start}px)`,
            }"
            @click="goDetail(rows[vi.index].symbol)"
          >
            <div class="c sym mono">{{ rows[vi.index].symbol }}</div>
            <div class="c name">{{ rows[vi.index].name }}</div>
            <div class="c num mono">
              {{ rows[vi.index].last_price === '-' ? '-' : fmtPrice(rows[vi.index].last_price) }}
            </div>
            <div class="c num mono" :class="trendClass(rows[vi.index].change_pct)">
              {{ rows[vi.index].change_pct }}%
            </div>
            <div class="c num mono" :class="trendClass(rows[vi.index].change_amt)">
              {{ rows[vi.index].change_amt }}
            </div>
            <div class="c num mono">
              {{ rows[vi.index].turnover === '-' ? '-' : fmtAmt(rows[vi.index].turnover) }}
            </div>
            <div class="c num mono dim">{{ fmtTime(rows[vi.index].event_time) }}</div>
            <div class="c ops" @click.stop>
              <Popconfirm
                title="移除自选？"
                :ok-text="'确认'"
                :cancel-text="'取消'"
                @confirm="emit('remove', rows[vi.index].symbol)"
              >
                <a class="del">×</a>
              </Popconfirm>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.wrap { background: var(--panel); border: 1px solid var(--divider); border-radius: 6px; overflow: hidden; }
.dim { color: var(--text-dim); }
.del { color: var(--text-dim); font-size: 16px; line-height: 1; cursor: pointer; }
.del:hover { color: var(--down); }

.virtual-wrap { display: flex; flex-direction: column; }
.vrow {
  display: grid;
  grid-template-columns: 110px 140px 100px 100px 100px 120px 90px 60px;
  align-items: center;
  padding: 0 12px;
  border-bottom: 1px solid var(--divider);
  cursor: pointer;
}
.vrow:hover { background: rgba(255, 255, 255, 0.03); }
.vrow.header {
  background: rgba(255, 255, 255, 0.02);
  color: var(--text-dim);
  font-size: 12px;
  height: 36px;
  cursor: default;
}
.c { padding: 0 6px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.c.num { text-align: right; }
.c.ops { text-align: center; }

.vscroll { height: 560px; overflow: auto; position: relative; }
.vinner { position: relative; width: 100%; }
</style>
