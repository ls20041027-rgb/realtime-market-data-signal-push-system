<script setup lang="ts">
import { onMounted } from 'vue'
import { ConfigProvider, theme } from 'ant-design-vue'
import AppShell from '@/components/AppShell.vue'
import { useMetaStore } from '@/stores/meta'
import { useSystemStore } from '@/stores/system'
import { useSignalStore } from '@/stores/signal'
import { wsClient } from '@/ws/client'

// 应用启动：加载码表 + 建立 WS 单例连接 + 绑定全局 store
onMounted(() => {
  useMetaStore().load()
  wsClient.connect()
  useSystemStore().bind()
  useSignalStore().bind()
})

// ant-design-vue 暗色主题：对齐 styles/vars.css 中的设计令牌，
// 避免 reset.css 注入亮色默认值造成"黑底黑字"。
const antTheme = {
  algorithm: theme.darkAlgorithm,
  token: {
    colorPrimary: '#5b8cff',
    colorBgBase: '#0b0e13',
    colorBgContainer: '#141922',
    colorBgElevated: '#141922',
    colorBgLayout: '#0b0e13',
    colorBorder: '#1f2632',
    colorBorderSecondary: '#1f2632',
    colorText: '#e6e8eb',
    colorTextSecondary: '#8b93a1',
    colorTextTertiary: '#8b93a1',
    colorSuccess: '#00c853',
    colorError: '#ff3b30',
    colorWarning: '#ffb300',
    colorInfo: '#5b8cff',
    fontFamily: "Inter, 'PingFang SC', 'Microsoft YaHei', sans-serif",
  },
}
</script>

<template>
  <ConfigProvider :theme="antTheme">
    <AppShell />
  </ConfigProvider>
</template>
