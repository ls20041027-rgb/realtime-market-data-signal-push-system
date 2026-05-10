<script setup lang="ts">
// FilterPanel：对齐 /api/signals 查询参数
// - symbol / signal_type / action / severity / from / to
// - 变更时 emit('change', SignalQuery)
import { computed, reactive, watch } from 'vue'
import {
  Button,
  Input,
  RangePicker,
  Select,
  SelectOption,
  Space,
} from 'ant-design-vue'
import type { SignalAction, SignalSeverity } from '@/types'
import type { SignalQuery } from '@/api/signal'
import { normalizeSymbol } from '@/utils/symbol'

interface FormState {
  symbol: string
  signal_type: string
  action: SignalAction | undefined
  severity: SignalSeverity | undefined
  range: [string, string] | undefined
}

const state = reactive<FormState>({
  symbol: '',
  signal_type: '',
  action: undefined,
  severity: undefined,
  range: undefined,
})

const emit = defineEmits<{ (e: 'change', q: SignalQuery): void }>()

const toQuery = (): SignalQuery => {
  const q: SignalQuery = {}
  const sym = normalizeSymbol(state.symbol)
  if (sym) q.symbol = sym
  if (state.signal_type.trim()) q.signal_type = state.signal_type.trim()
  if (state.action) q.action = state.action
  if (state.severity) q.severity = state.severity
  if (state.range && state.range.length === 2) {
    q.from = state.range[0]
    q.to = state.range[1]
  }
  return q
}

const apply = () => emit('change', toQuery())

const reset = () => {
  state.symbol = ''
  state.signal_type = ''
  state.action = undefined
  state.severity = undefined
  state.range = undefined
  emit('change', {})
}

// action/severity 即选即用；symbol / signal_type 文本需点击"查询"
watch(
  () => [state.action, state.severity, state.range],
  () => apply(),
)

// RangePicker 接受 dayjs 对象；这里用 string[] 兼容（from/to 格式 YYYY-MM-DD）
const rangeValue = computed<[string, string] | undefined>({
  get: () => state.range,
  set: (v) => {
    if (Array.isArray(v) && v.length === 2) {
      const fmt = (x: unknown): string => {
        if (!x) return ''
        if (typeof x === 'string') return x.slice(0, 10)
        const obj = x as { format?: (s: string) => string }
        if (typeof obj.format === 'function') return obj.format('YYYY-MM-DD')
        return ''
      }
      state.range = [fmt(v[0]), fmt(v[1])]
    } else {
      state.range = undefined
    }
  },
})
</script>

<template>
  <div class="panel">
    <Space :size="12" wrap>
      <Input
        v-model:value="state.symbol"
        placeholder="symbol，如 SH600000"
        allow-clear
        style="width: 180px"
        @press-enter="apply"
      />
      <Input
        v-model:value="state.signal_type"
        placeholder="signal_type"
        allow-clear
        style="width: 160px"
        @press-enter="apply"
      />
      <Select
        v-model:value="state.action"
        placeholder="动作"
        allow-clear
        style="width: 110px"
      >
        <SelectOption value="BUY">BUY</SelectOption>
        <SelectOption value="SELL">SELL</SelectOption>
        <SelectOption value="WATCH">WATCH</SelectOption>
        <SelectOption value="RISK">RISK</SelectOption>
      </Select>
      <Select
        v-model:value="state.severity"
        placeholder="等级"
        allow-clear
        style="width: 110px"
      >
        <SelectOption value="INFO">INFO</SelectOption>
        <SelectOption value="WARN">WARN</SelectOption>
        <SelectOption value="CRITICAL">CRITICAL</SelectOption>
      </Select>
      <RangePicker
        v-model:value="rangeValue"
        value-format="YYYY-MM-DD"
        :placeholder="['起始', '结束']"
      />
      <Button type="primary" @click="apply">查询</Button>
      <Button @click="reset">重置</Button>
    </Space>
  </div>
</template>

<style scoped>
.panel {
  background: var(--panel);
  border: 1px solid var(--divider);
  border-radius: 6px;
  padding: 12px 14px;
}
</style>
