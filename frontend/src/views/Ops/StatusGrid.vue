<script setup lang="ts">
// 健康卡片网格：WS / Redis / MySQL / Runtime 四张卡
// - 不做派生计算，只格式化 uptime / latency
// - snapshot 为 null 时显示骨架
import { computed } from 'vue'
import type { StatusSnapshot } from '@/types'

const props = defineProps<{ snapshot: StatusSnapshot | null }>()

const uptimeLabel = computed(() => {
  const s = props.snapshot?.runtime.uptime_seconds ?? 0
  if (s <= 0) return '-'
  const d = Math.floor(s / 86400)
  const h = Math.floor((s % 86400) / 3600)
  const m = Math.floor((s % 3600) / 60)
  const sec = s % 60
  if (d > 0) return `${d}d ${h}h ${m}m`
  if (h > 0) return `${h}h ${m}m ${sec}s`
  if (m > 0) return `${m}m ${sec}s`
  return `${sec}s`
})
</script>

<template>
  <div class="grid">
    <!-- WS -->
    <section class="card">
      <header class="hd">
        <span class="title">WebSocket</span>
        <span v-if="snapshot" class="dot" :style="{ background: 'var(--up)' }" />
      </header>
      <div v-if="snapshot" class="kv">
        <div class="line"><span class="k">在线连接</span><span class="v mono">{{ snapshot.ws.clients }}</span></div>
        <div class="line"><span class="k">订阅 channel</span><span class="v mono">{{ snapshot.ws.channels }}</span></div>
        <div class="line">
          <span class="k">慢消费丢帧</span>
          <span
            class="v mono"
            :class="snapshot.ws.dropped_slow > 0 ? 'warn' : ''"
          >{{ snapshot.ws.dropped_slow }}</span>
        </div>
      </div>
      <div v-else class="dim">加载中...</div>
    </section>

    <!-- Redis -->
    <section class="card">
      <header class="hd">
        <span class="title">Redis</span>
        <span
          v-if="snapshot"
          class="dot"
          :style="{ background: snapshot.redis.up ? 'var(--up)' : 'var(--down)' }"
        />
      </header>
      <div v-if="snapshot" class="kv">
        <div class="line">
          <span class="k">存活</span>
          <span class="v mono" :class="snapshot.redis.up ? 'ok' : 'err'">
            {{ snapshot.redis.up ? 'UP' : 'DOWN' }}
          </span>
        </div>
        <div class="line"><span class="k">探活延迟</span><span class="v mono">{{ snapshot.redis.latency_ms }} ms</span></div>
        <div v-if="snapshot.redis.error" class="line err-line">
          <span class="k">错误</span><span class="v err mono">{{ snapshot.redis.error }}</span>
        </div>
      </div>
      <div v-else class="dim">加载中...</div>
    </section>

    <!-- MySQL -->
    <section class="card">
      <header class="hd">
        <span class="title">MySQL</span>
        <span
          v-if="snapshot"
          class="dot"
          :style="{ background: snapshot.postgres.up ? 'var(--up)' : 'var(--down)' }"
        />
      </header>
      <div v-if="snapshot" class="kv">
        <div class="line">
          <span class="k">存活</span>
          <span class="v mono" :class="snapshot.postgres.up ? 'ok' : 'err'">
            {{ snapshot.postgres.up ? 'UP' : 'DOWN' }}
          </span>
        </div>
        <div class="line"><span class="k">探活延迟</span><span class="v mono">{{ snapshot.postgres.latency_ms }} ms</span></div>
        <div v-if="snapshot.postgres.error" class="line err-line">
          <span class="k">错误</span><span class="v err mono">{{ snapshot.postgres.error }}</span>
        </div>
      </div>
      <div v-else class="dim">加载中...</div>
    </section>

    <!-- Runtime -->
    <section class="card">
      <header class="hd">
        <span class="title">Runtime</span>
      </header>
      <div v-if="snapshot" class="kv">
        <div class="line"><span class="k">PID</span><span class="v mono">{{ snapshot.runtime.pid }}</span></div>
        <div class="line"><span class="k">goroutines</span><span class="v mono">{{ snapshot.runtime.goroutines }}</span></div>
        <div class="line"><span class="k">uptime</span><span class="v mono">{{ uptimeLabel }}</span></div>
      </div>
      <div v-else class="dim">加载中...</div>
    </section>
  </div>
</template>

<style scoped>
.grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
}
@media (max-width: 1100px) {
  .grid { grid-template-columns: repeat(2, 1fr); }
}
.card {
  background: var(--panel);
  border: 1px solid var(--divider);
  border-radius: 6px;
  padding: 12px 14px;
}
.hd { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; }
.title { font-size: 13px; font-weight: 600; }
.dot {
  width: 8px; height: 8px; border-radius: 50%;
  box-shadow: 0 0 0 2px rgba(255, 255, 255, 0.05);
}
.kv { display: flex; flex-direction: column; gap: 6px; }
.line { display: flex; align-items: center; justify-content: space-between; font-size: 12px; }
.k { color: var(--text-dim); }
.v { color: var(--text); }
.ok { color: var(--up); }
.err { color: var(--down); }
.warn { color: var(--warn); }
.err-line .v { max-width: 160px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dim { color: var(--text-dim); font-size: 12px; }
</style>
