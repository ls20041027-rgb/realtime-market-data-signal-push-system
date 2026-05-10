<script setup lang="ts">
// 指标 tab：当前只包一层 IndicatorSub（后端口径是"最新一帧"，无时间序列）
import IndicatorSub from '@/components/charts/IndicatorSub.vue'
import type { IndicatorSnapshot } from '@/types'

defineProps<{
  data: IndicatorSnapshot | null
  loading?: boolean
}>()
</script>

<template>
  <div class="wrap">
    <IndicatorSub :data="data" :loading="loading" />
  </div>
</template>

<style scoped>
.wrap { display: flex; flex-direction: column; gap: 12px; }
</style>
