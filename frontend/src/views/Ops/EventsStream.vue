<script setup lang="ts">
// 系统事件流：订阅 system:events（由 system store 托管，全局单次 bind）
// - 级别染色：INFO 淡 / WARN 黄 / ERROR 橙 / CRITICAL 红
// - 支持按级别筛选 + 清空
import { computed, onMounted, ref } from 'vue'
import { Button, Empty, Select, SelectOption, Space, Tag } from 'ant-design-vue'
import { useSystemStore } from '@/stores/system'
import type { SystemLevel } from '@/types'

const systemStore = useSystemStore()

onMounted(() => systemStore.bind())

const filter = ref<SystemLevel | 'ALL'>('ALL')

const list = computed(() =>
  filter.value === 'ALL'
    ? systemStore.events
    : systemStore.events.filter((e) => e.level === filter.value),
)

const levelColor: Record<SystemLevel, string> = {
  INFO: 'default',
  WARN: 'orange',
  ERROR: 'volcano',
  CRITICAL: 'red',
}
</script>

<template>
  <section class="card">
    <header class="hd">
      <div class="left">
        <span class="title">系统事件流</span>
        <Tag color="default" class="mono">{{ list.length }} / {{ systemStore.events.length }}</Tag>
      </div>
      <Space>
        <Select v-model:value="filter" size="small" style="width: 120px">
          <SelectOption value="ALL">全部级别</SelectOption>
          <SelectOption value="INFO">INFO</SelectOption>
          <SelectOption value="WARN">WARN</SelectOption>
          <SelectOption value="ERROR">ERROR</SelectOption>
          <SelectOption value="CRITICAL">CRITICAL</SelectOption>
        </Select>
        <Button size="small" @click="systemStore.clear()">清空</Button>
      </Space>
    </header>

    <div v-if="list.length === 0" class="empty">
      <Empty description="暂无事件" />
    </div>
    <ul v-else class="list">
      <li v-for="e in list" :key="e.event_id" class="item" :class="`lvl-${e.level.toLowerCase()}`">
        <span class="mono dim type">{{ e.event_type }}</span>
        <Tag :color="levelColor[e.level]">{{ e.level }}</Tag>
        <span class="svc mono">{{ e.service }}</span>
        <span class="msg">{{ e.message }}</span>
        <span v-if="e.related_topic" class="mono dim topic">topic={{ e.related_topic }}</span>
        <span v-if="e.retry_count != null" class="mono dim">retry={{ e.retry_count }}</span>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.card {
  background: var(--panel);
  border: 1px solid var(--divider);
  border-radius: 6px;
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.hd { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; }
.left { display: flex; align-items: center; gap: 8px; }
.title { font-size: 13px; font-weight: 600; }
.empty { padding: 24px 0; }
.list {
  list-style: none; padding: 0; margin: 0;
  display: flex; flex-direction: column; gap: 4px;
  max-height: 420px; overflow: auto;
}
.item {
  display: flex; align-items: center; gap: 10px;
  font-size: 12px; padding: 4px 6px;
  border-left: 2px solid transparent;
  border-radius: 2px;
}
.item.lvl-warn { border-left-color: var(--warn); }
.item.lvl-error { border-left-color: var(--down); background: rgba(255, 59, 48, 0.04); }
.item.lvl-critical { border-left-color: var(--critical); background: rgba(255, 45, 85, 0.08); }
.type { min-width: 140px; }
.svc { min-width: 110px; color: var(--info); }
.msg { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.topic { max-width: 160px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dim { color: var(--text-dim); }
</style>
