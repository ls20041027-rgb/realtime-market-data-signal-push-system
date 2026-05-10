import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  { path: '/', redirect: '/watchlist' },
  {
    path: '/watchlist',
    name: 'watchlist',
    component: () => import('@/views/Watchlist/index.vue'),
    meta: { title: '自选股' },
  },
  {
    path: '/stock/:symbol',
    name: 'stock-detail',
    component: () => import('@/views/StockDetail/index.vue'),
    props: true,
    meta: { title: '个股详情' },
  },
  {
    path: '/signals',
    name: 'signals',
    component: () => import('@/views/SignalCenter/index.vue'),
    meta: { title: '信号中心' },
  },
  {
    path: '/screener',
    name: 'screener',
    component: () => import('@/views/Screener/index.vue'),
    meta: { title: '选股榜单' },
  },
  {
    path: '/ops',
    name: 'ops',
    component: () => import('@/views/Ops/index.vue'),
    meta: { title: '运维大屏' },
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/views/NotFound.vue'),
    meta: { title: '未找到' },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.afterEach((to) => {
  const title = (to.meta?.title as string | undefined) || 'Tornado Seeker'
  document.title = `${title} · Tornado Seeker`
})

export default router
