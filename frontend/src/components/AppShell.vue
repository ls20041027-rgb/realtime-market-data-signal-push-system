<script setup lang="ts">
// AppShell：侧边导航 + 顶栏 + 路由出口
import { RouterLink, RouterView, useRoute } from 'vue-router'
import { computed, onMounted } from 'vue'
import { Badge } from 'ant-design-vue'
import ConnectionBadge from './ConnectionBadge.vue'
import { useSignalStore } from '@/stores/signal'
import { useSystemStore } from '@/stores/system'

const route = useRoute()
const signalStore = useSignalStore()
const systemStore = useSystemStore()

interface NavItem {
  to: string
  label: string
  icon: string
}

const navs: NavItem[] = [
  { to: '/watchlist', label: '自选股', icon: '★' },
  { to: '/signals', label: '信号中心', icon: '◎' },
  { to: '/screener', label: '选股榜单', icon: '≡' },
  { to: '/ops', label: '运维大屏', icon: '⚙' },
]

const activePath = computed(() => route.path)

// 全局订阅：signal:ALL 累计未读红点；system:events 累计系统事件流
onMounted(() => {
  signalStore.bind()
  systemStore.bind()
})
</script>

<template>
  <div class="shell">
    <aside class="sidebar">
      <div class="logo">
        <span class="logo-mark">⚡</span>
        <span class="logo-text">Tornado Seeker</span>
      </div>
      <nav class="nav">
        <RouterLink
          v-for="n in navs"
          :key="n.to"
          :to="n.to"
          class="nav-item"
          :class="{ active: activePath.startsWith(n.to) }"
        >
          <span class="icon">{{ n.icon }}</span>
          <span>{{ n.label }}</span>
          <Badge
            v-if="n.to === '/signals' && signalStore.unread > 0"
            :count="signalStore.unread"
            :offset="[4, 0]"
            :number-style="{ backgroundColor: 'var(--critical)' }"
          />
        </RouterLink>
      </nav>
    </aside>
    <div class="main">
      <header class="topbar">
        <div class="crumb mono">{{ activePath }}</div>
        <ConnectionBadge />
      </header>
      <section class="content">
        <RouterView />
      </section>
    </div>
  </div>
</template>

<style scoped>
.shell {
  display: grid;
  grid-template-columns: 220px 1fr;
  height: 100vh;
  background: var(--bg);
}
.sidebar {
  border-right: 1px solid var(--divider);
  display: flex;
  flex-direction: column;
  background: var(--panel);
}
.logo {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 18px 16px;
  border-bottom: 1px solid var(--divider);
  font-weight: 600;
  letter-spacing: 0.5px;
}
.logo-mark { color: var(--info); }
.logo-text { font-size: 14px; }
.nav {
  display: flex;
  flex-direction: column;
  padding: 8px 6px;
  gap: 2px;
}
.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 6px;
  color: var(--text-dim);
  font-size: 13px;
  border-left: 2px solid transparent;
  transition: background 0.12s, color 0.12s;
}
.nav-item:hover { background: rgba(255, 255, 255, 0.03); color: var(--text); }
.nav-item.active {
  background: rgba(0, 200, 83, 0.08);
  color: var(--text);
  border-left-color: var(--up);
}
.icon { width: 16px; text-align: center; color: var(--text-dim); }
.main { display: flex; flex-direction: column; overflow: hidden; }
.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  border-bottom: 1px solid var(--divider);
  background: var(--panel);
}
.crumb { color: var(--text-dim); font-size: 12px; }
.content { flex: 1; overflow: auto; padding: 16px; }
</style>
