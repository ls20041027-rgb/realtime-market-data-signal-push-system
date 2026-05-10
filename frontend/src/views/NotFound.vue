<script setup lang="ts">
import { useRouter } from 'vue-router'
const router = useRouter()
</script>

<template>
  <div class="nf">
    <h2>404</h2>
    <p class="dim">页面不存在</p>
    <a class="link" @click="router.push('/')">返回首页</a>
  </div>
</template>

<style scoped>
.nf { padding: 48px; text-align: center; color: var(--text); }
.dim { color: var(--text-dim); }
.link { color: var(--info); cursor: pointer; }
</style>
