<script setup lang="ts">
// 顶栏 WebSocket 连接状态徽标
import { computed } from 'vue'
import { wsClient, type ConnStatus } from '@/ws/client'
import { ref } from 'vue'

const status = ref<ConnStatus>(wsClient.getStatus())
wsClient.onStatus((s) => (status.value = s))

const label = computed(() => {
  switch (status.value) {
    case 'OPEN':
      return '实时已连接'
    case 'CONNECTING':
      return '正在连接'
    case 'RECONNECTING':
      return '重连中'
    case 'CLOSED':
      return '已断开'
    default:
      return '未连接'
  }
})

const dotColor = computed(() => {
  switch (status.value) {
    case 'OPEN':
      return 'var(--up)'
    case 'CONNECTING':
    case 'RECONNECTING':
      return 'var(--warn)'
    default:
      return 'var(--down)'
  }
})
</script>

<template>
  <div class="conn-badge">
    <span class="dot" :style="{ background: dotColor }" />
    <span class="text">{{ label }}</span>
  </div>
</template>

<style scoped>
.conn-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border: 1px solid var(--divider);
  border-radius: 999px;
  font-size: 12px;
  color: var(--text-dim);
}
.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  box-shadow: 0 0 0 2px rgba(255, 255, 255, 0.05);
}
</style>
