---
# 前端架构（FRONTEND_ARCHITECTURE）

> 本文档是 **前端** 的单一架构事实源。后端契约见
> [services/push_gateway/CONTRACT.md](./services/push_gateway/CONTRACT.md)，
> 功能范围与接口清单见 [FRONTEND_SCOPE.md](./FRONTEND_SCOPE.md)，
> UI 视觉见 [FRONTEND_UI_PROMPTS.md](./FRONTEND_UI_PROMPTS.md)。
>
> **事实来源优先级**（冲突以上位为准）：
> 1. `shared/topic_contract.yaml`
> 2. `services/push_gateway/CONTRACT.md`
> 3. `FRONTEND_SCOPE.md`
> 4. 本文件
>
> 原则：契约优先 / 奥卡姆剃刀 / 精度安全（`string` 透传）/ 单一 WS 连接 / 纯只读客户端。

---

## 1. 技术栈选型（定型）

| 维度 | 选型 | 理由 |
|---|---|---|
| 框架 | **Vue 3 + `<script setup>` + TypeScript** | 模板语法更适合高密度仪表盘；TS 对契约字段强约束 |
| 构建 | **Vite 5** | 冷启快、HMR 友好 |
| 路由 | **Vue Router 4** | 配合懒加载按页切分 chunk |
| 状态 | **Pinia** | 轻量，够表达 WS 订阅计数 + 多路行情合并 |
| HTTP | **Axios**（拦截器统一 `{code,message,data}` 解包） | 和 REST 信封天然匹配 |
| WebSocket | **自写 `WSClient` 单例 + `useChannel()` composable** | 业务语义：subscribe/unsubscribe 计数、断线指数退避、页面切换不断连 |
| 图表 | **ECharts 5**（K 线 / 副图 / 折线 / 柱状全覆盖） | 一个依赖搞定全站；备选 Lightweight-Charts 仅 K 线时用 |
| 大数 | **`decimal.js`**（统一包装 `D(x)`） | 后端金额 `string` 透传，禁 `Number()` / `parseFloat` 参与运算 |
| UI 库 | **Ant Design Vue 4** | 表格 / 分页 / 弹窗 / 筛选器完整；暗色主题成熟 |
| 样式 | **UnoCSS + CSS Variables** | 暗色主题色板走 CSS 变量，见 §7 |
| 国际化 | 暂不做 | 毕设阶段单语（中文） |
| 测试 | **Vitest + Testing Library**（R11 精简原则） | 只测 WS 订阅计数、Decimal 格式化、分页 hook 三条关键路径 |
| 代码规范 | ESLint + Prettier + typescript-eslint | — |
| 包管理 | **pnpm** | workspace 友好；速度快 |

---

## 2. 目录结构

```
frontend/
├── index.html
├── vite.config.ts
├── tsconfig.json
├── uno.config.ts
├── package.json
├── .env.development            # VITE_API_BASE / VITE_WS_URL
├── .env.production
└── src/
    ├── main.ts                  # 应用入口：挂 pinia / router / antd / WS 单例
    ├── App.vue
    ├── router/
    │   └── index.ts             # 6 个顶级路由 + 懒加载
    ├── api/                     # REST 客户端（只读）
    │   ├── http.ts              # axios 实例 + 响应信封拦截器
    │   ├── quote.ts             # /api/quote、/api/quotes、/api/fenbi
    │   ├── indicator.ts         # /api/indicators、/api/capital
    │   ├── kline.ts             # /api/kline、/api/kline5m
    │   ├── finance.ts           # /api/finance
    │   ├── signal.ts            # /api/signals、/api/signals/:id
    │   ├── meta.ts              # /api/stock-list、/api/stock/:symbol
    │   └── status.ts            # /healthz、/api/status
    ├── ws/                      # WebSocket 层
    │   ├── client.ts            # WSClient 单例（连接 / 重连 / 心跳 / 分发）
    │   ├── channels.ts          # channel 常量与校验（白名单）
    │   └── useChannel.ts        # 组件内订阅 composable（引用计数）
    ├── stores/                  # Pinia stores
    │   ├── quote.ts             # 行情快照：Map<symbol, Quote>
    │   ├── indicator.ts         # 指标缓存
    │   ├── signal.ts            # 信号流（全市场滚动 + 未读红点）
    │   ├── watchlist.ts         # 自选股（localStorage 持久化）
    │   ├── meta.ts              # 股票码表（首屏拉一次）
    │   ├── system.ts            # 系统事件 + 连接状态
    │   └── status.ts            # /api/status 运维大屏数据
    ├── composables/             # 通用 hook
    │   ├── useAutoRefresh.ts    # 可见性感知的 interval（页面隐藏暂停）
    │   ├── usePagination.ts     # 对齐 {items,total,page,page_size}
    │   ├── useRetry.ts          # 指数退避（WS / HTTP 复用）
    │   └── useDecimal.ts        # decimal.js 包装：D(x) / fmtPrice / fmtPct
    ├── utils/
    │   ├── symbol.ts            # normalize、detectExchange（对齐后端）
    │   ├── format.ts            # 时间 / 量能（万 / 亿）/ 颜色类（涨跌色）
    │   ├── envelope.ts          # 后端 {code,message,data} 解包
    │   └── logger.ts            # 前端日志（开发态 console，生产态屏蔽）
    ├── components/              # 通用 UI 组件（不含业务状态）
    │   ├── AppShell.vue         # 左侧导航 + 顶栏 + <router-view/>
    │   ├── ConnectionBadge.vue  # WS 连接状态徽标
    │   ├── SystemEventBadge.vue # 订阅 system:events 的顶栏告警
    │   ├── PriceCell.vue        # 等宽数字 + 涨跌色
    │   ├── ChangeBar.vue        # 涨跌幅填充条
    │   ├── OrderBookMini.vue    # 五档盘口迷你件
    │   ├── SignalBadge.vue      # BUY/SELL 徽标
    │   ├── ConfidenceBar.vue    # 信号置信度进度条
    │   └── charts/
    │       ├── KLineChart.vue   # ECharts 蜡烛 + MA 叠加 + tooltip
    │       ├── IndicatorSub.vue # MACD / KDJ / RSI 副图（组合）
    │       └── Sparkline.vue    # 财务趋势 / 量比小折线
    ├── views/                   # 路由页面
    │   ├── StockDetail/
    │   │   ├── index.vue        # 总装
    │   │   ├── QuoteCard.vue
    │   │   ├── KLineTab.vue
    │   │   ├── IndicatorsPanel.vue
    │   │   ├── FenbiTab.vue
    │   │   ├── CapitalTab.vue
    │   │   ├── FinanceTab.vue
    │   │   └── SignalTab.vue
    │   ├── Watchlist/
    │   │   └── index.vue
    │   ├── SignalCenter/
    │   │   ├── index.vue
    │   │   ├── FilterPanel.vue
    │   │   ├── SignalList.vue
    │   │   └── LiveTicker.vue   # 订阅 signal:ALL
    │   ├── Screener/
    │   │   └── index.vue        # 2x2 榜单
    │   └── Ops/
    │       └── index.vue        # 运维大屏
    ├── types/                   # TS 契约类型（对齐后端字段）
    │   ├── envelope.ts
    │   ├── quote.ts
    │   ├── signal.ts
    │   └── index.ts
    └── styles/
        ├── vars.css             # 色板 + 字号
        └── global.css
```

**原则**：`api/` 只负责 HTTP、`ws/` 只负责连接、`stores/` 只负责状态合并；页面组件不直接 `axios` 或 `new WebSocket`。

---

## 3. 分层架构

```
┌────────────────────────── Views (路由页面) ─────────────────────────┐
│  StockDetail / Watchlist / SignalCenter / Screener / Ops          │
└───────────────▲────────────────────────────▲──────────────────────┘
                │ 读 / 订阅                    │ 用
                │                             │
┌──────── Stores (Pinia) ────────┐    ┌─ Composables / Components ─┐
│ quote / signal / watchlist /   │    │ useChannel / usePagination │
│ meta / system / status         │    │ KLineChart / SignalBadge   │
└───────▲──────────────▲─────────┘    └────────────▲───────────────┘
        │ 快照 + 增量   │ 事件                      │
┌──────── api/ ────────┐┌──── ws/ ────┐
│  Axios REST client   ││  WSClient   │
└──────────▲───────────┘└──────▲──────┘
           │                    │
           └────── push_gateway ────────（唯一后端出口）
```

**数据流**：
- **首帧用 REST**（快照）：进页 / 首屏都先拉 `/api/quote` 或 `/api/quotes` 拿一次全量，避免 WS 建连期的白屏；
- **增量用 WS**（推送）：订阅 `quote:*` / `signal:*` / `system:events`，通过 store 合并覆盖；
- **轮询仅用于非推送资源**：`/api/indicators`、`/api/capital`（3~5s）、`/api/status`（2~5s，且只在运维大屏页激活）；
- **历史资源一次性拉**：`/api/kline*`、`/api/finance`、`/api/signals` 进页 / 切周期时拉。

---

## 4. WebSocket 层设计（`src/ws/`）

### 4.1 WSClient 单例职责

- **单一长连接**：整个 SPA 只维护一条 `new WebSocket(VITE_WS_URL)`，跨路由不断连；
- **订阅计数**：`Map<channel, refCount>`；组件挂载 `+1`，卸载 `-1`，降到 0 才发 `unsubscribe`；
- **订阅状态机**：`IDLE → CONNECTING → OPEN → CLOSING → CLOSED`；重连期间的订阅请求进入 `pendingQueue`，`OPEN` 后批量 `subscribe`；
- **心跳**：每 25s 发 `{action:"ping"}`，40s 未收到任何帧主动 `close(4000)`；
- **重连**：指数退避（1s → 2 → 4 → 8 → 16，封顶 30s；连续成功 10 帧后重置）；
- **分发**：收到帧按 `channel` 前缀匹配分发到对应 store 的 handler；未知帧（没 `channel` 且不是 `pong`/`error`）丢弃并 `warn`；
- **慢消费保护**：前端侧不做缓冲，`onmessage` 内仅做极轻操作（写入 store 的 `Map`），重计算一律进 `requestAnimationFrame` / `requestIdleCallback`。

### 4.2 `useChannel(channel)` composable 契约

```ts
// 组件内使用
const { data, status } = useChannel<QuoteFrame>('quote:SH600000', {
  onMessage: (frame) => quoteStore.apply(frame),
  immediate: true,      // mount 时订阅；unmount 自动 -1
})
```

- 同一 channel 被 N 个组件订阅 → 只发一次 `subscribe`；最后一个组件卸载才 `unsubscribe`；
- `status` 反映整条连接状态（`CONNECTING / OPEN / RECONNECTING / CLOSED`），用于全局徽标。

### 4.3 Channel 白名单（硬编码常量，禁散落）

```ts
// src/ws/channels.ts
export const CH = {
  quote:  (s: string) => `quote:${s}` as const,
  signal: (s: string) => `signal:${s}` as const,
  signalAll: 'signal:ALL' as const,
  systemEvents: 'system:events' as const,
}
```

非 `quote:*` / `signal:*` / `signal:ALL` / `system:events` 的订阅一律被 `WSClient.validate()` 拒绝，不打到网络层。

---

## 5. 状态管理（Pinia stores）

| Store | 持久化 | 结构 | 消费来源 |
|---|---|---|---|
| `quote` | ❌ | `Map<symbol, QuoteSnapshot>` | REST 首帧 + `quote:*` WS |
| `indicator` | ❌ | `Map<symbol, IndicatorSnapshot>` | `/api/indicators` 轮询 |
| `signal` | ❌ | `{ live: Signal[]（滚动 30）, history: Signal[], unread: number }` | `signal:*` / `signal:ALL` WS + `/api/signals` REST |
| `watchlist` | ✅ `localStorage` | `string[]`（symbols） | 纯前端；批量 `/api/quotes?symbols=` |
| `meta` | ✅ `sessionStorage` | `Map<symbol, {name, exchange, lot_size}>` | `/api/stock-list` 首屏一次 |
| `system` | ❌ | `{ events: SystemEvent[], connStatus }` | `system:events` WS |
| `status` | ❌ | `StatusSnapshot`（仅运维页拉取） | `/api/status` 轮询 |

**内存边界**：
- `quote` / `indicator` Map 超过 500 条时 LRU 淘汰（不在自选股且 60s 无更新的优先淘汰）；
- `signal.live` 固定最大 30 条，新帧头插 + 尾弹；
- `signal.history` 不常驻 store，分页 hook 内持有，页面卸载释放。

---

## 6. 路由

```ts
// src/router/index.ts
const routes = [
  { path: '/',             redirect: '/watchlist' },
  { path: '/watchlist',    component: () => import('@/views/Watchlist/index.vue') },
  { path: '/stock/:symbol', component: () => import('@/views/StockDetail/index.vue'), props: true },
  { path: '/signals',      component: () => import('@/views/SignalCenter/index.vue') },
  { path: '/screener',     component: () => import('@/views/Screener/index.vue') },
  { path: '/ops',          component: () => import('@/views/Ops/index.vue') },
  { path: '/:pathMatch(.*)*', component: () => import('@/views/NotFound.vue') },
]
```

- 每个页面独立 chunk（懒加载）；
- `AppShell` 套在 `<router-view>` 外，维护侧边栏高亮、顶栏连接徽标、系统告警徽标。

---

## 7. 主题与设计令牌（对齐 [FRONTEND_UI_PROMPTS.md](./FRONTEND_UI_PROMPTS.md)）

```css
/* src/styles/vars.css */
:root {
  --bg:        #0B0E13;
  --panel:     #141922;
  --divider:   #1F2632;
  --text:      #E6E8EB;
  --text-dim:  #8B93A1;
  --up:        #00C853;   /* 涨 */
  --down:      #FF3B30;   /* 跌 */
  --neutral:   #8B93A1;
  --warn:      #FFB300;
  --info:      #5B8CFF;
  --critical:  #FF2D55;
  --font-mono: 'JetBrains Mono', 'Menlo', monospace;
  --font-sans: 'Inter', 'PingFang SC', sans-serif;
}
```

涨跌色用工具函数统一产出 class，禁止页面内硬编码 `color: #xxx`：

```ts
// src/utils/format.ts
export function trendClass(change: Decimal): 'text-up' | 'text-down' | 'text-neutral' {
  if (change.gt(0)) return 'text-up'
  if (change.lt(0)) return 'text-down'
  return 'text-neutral'
}
```

---

## 8. 精度安全（强约束）

1. 后端金额 / 价格 / 比率**全部以 `string` 下发**，前端 **禁止** `Number(x)` / `parseFloat(x)` 参与任何运算；
2. 统一用 `D(x)`（`decimal.js` 包装）：

```ts
// src/composables/useDecimal.ts
import Decimal from 'decimal.js'
Decimal.set({ precision: 20, rounding: Decimal.ROUND_HALF_UP })
export const D = (x: string | number | Decimal) => new Decimal(x ?? 0)
export const fmtPrice = (x: string, digits = 2) => D(x).toFixed(digits)
export const fmtPct   = (x: string, digits = 2) => `${D(x).mul(100).toFixed(digits)}%`
export const fmtAmt   = (x: string) => {                 // 万 / 亿
  const v = D(x)
  if (v.abs().gte(1e8)) return `${v.div(1e8).toFixed(2)}亿`
  if (v.abs().gte(1e4)) return `${v.div(1e4).toFixed(2)}万`
  return v.toFixed(0)
}
```

3. TS 类型中所有金额字段用 `string` 而非 `number`：

```ts
// src/types/quote.ts
export interface QuoteSnapshot {
  symbol: string
  last_price: string
  prev_close: string
  change_pct: string
  volume: string
  turnover: string
  bid_levels: Array<{ price: string; volume: string }>
  ask_levels: Array<{ price: string; volume: string }>
  event_time: string
}
```

---

## 9. 错误处理

### 9.1 HTTP（对齐 `CONTRACT §5.2`）

```ts
// src/api/http.ts 响应拦截
http.interceptors.response.use((resp) => {
  const { code, message, data } = resp.data
  if (code === 0) return data
  switch (code) {
    case 40001: throw new NotFoundError(message)     // 友好提示
    case 40002: throw new ValidationError(message)
    case 50001: case 50002: throw new ServiceDownError(message) // 降级 UI
    default: throw new UnknownError(message, code)
  }
})
```

页面层用 `try/catch` → Ant Design `message.error` 或局部兜底（骨架图 / 空状态）。

### 9.2 WS

- `{type:"error"}` 帧：toast 一次，不断连；
- 连接异常：顶栏徽标变色（`RECONNECTING` 黄色 / `CLOSED` 红色）；
- 连续 3 次重连失败：弹 `notification` 提示用户检查网络（仍持续重连）。

---

## 10. 性能要点

| 场景 | 策略 |
|---|---|
| 自选股表格 200 行 × `quote:*` 订阅 | WS 帧用 `shallowRef` + 手动触发；表格用 `v-memo` 或虚拟滚动（`@tanstack/vue-virtual`） |
| K 线切周期 | 关闭上一个 ECharts 实例再挂新实例；tooltip 用 `throttle(16)` |
| 高频 `quote:*` | 后端已 200ms 节流；前端再用 `requestAnimationFrame` 合并同一 tick 内的多次 store 更新 |
| 页面不可见 | `useAutoRefresh` 监听 `document.visibilitychange`，隐藏时暂停轮询；WS 保持连接 |
| 大表分页 | 走后端 `total/page/page_size`，前端不做全量缓存 |
| 首屏 | 路由级代码分割 + antd 按需引入 + ECharts 按需 `use()` |

---

## 11. 环境变量

```bash
# .env.development
VITE_API_BASE=http://localhost:8080
VITE_WS_URL=ws://localhost:8080/ws

# .env.production
VITE_API_BASE=/api-proxy
VITE_WS_URL=wss://<host>/ws
```

运行期 `import.meta.env.VITE_*` 读取；禁止在代码里硬编码 URL（对齐 R3）。

---

## 12. 开发 / 构建脚本

```json
// package.json（节选）
{
  "scripts": {
    "dev":     "vite",
    "build":   "vue-tsc --noEmit && vite build",
    "preview": "vite preview --port 5173",
    "lint":    "eslint --ext .ts,.vue src",
    "test":    "vitest run"
  }
}
```

---

## 13. 与后端联调的强约束（清单）

- [ ] **不调用** `FRONTEND_SCOPE §1` 之外的任何 URL / channel；
- [ ] 所有数字类字段 TS 类型是 `string`，`decimal.js` 参与运算；
- [ ] WS 订阅 / 退订严格走 `useChannel()`，不在组件里直接 `ws.send()`；
- [ ] 列表组件使用 `{items,total,page,page_size}` 原样透传，不估算总数；
- [ ] `symbol` 一律大写前缀（`SH/SZ/BJ`），来自后端的值不 `.toLowerCase()`；
- [ ] 错误码按 `CONTRACT §5.2` 映射到 UI 文案；
- [ ] 禁止把接口 URL 放进组件，统一在 `src/api/`；
- [ ] 不发任何写请求（前端只应出现 `GET`）。

---

## 14. 排期建议

| 阶段 | 产出 | 对应 FRONTEND_SCOPE |
|---|---|---|
| P1 骨架 | Vite + Vue3 + Router + Pinia + AppShell + WSClient + HTTP 拦截 + decimal 包装 | §1.2.6、§2.6 |
| P2 自选股页 | watchlist store + `/api/quotes` 批量 + 多路 `quote:*` 订阅 | §2.2 |
| P3 个股详情页 | QuoteCard + KLine + Indicators + Fenbi + Capital + Finance + SignalTab | §2.1 |
| P4 信号中心 | FilterPanel + SignalList 分页 + LiveTicker（`signal:ALL`） | §2.3 |
| P5 选股榜单 | 客户端聚合 2x2 榜单 | §2.4 |
| P6 运维大屏 | `/api/status` 轮询 + `system:events` | §2.5 |

---

## 15. 相关文件索引

- 后端契约：[services/push_gateway/CONTRACT.md](./services/push_gateway/CONTRACT.md)
- 后端设计：[services/push_gateway/DESIGN.md](./services/push_gateway/DESIGN.md)
- 功能范围：[FRONTEND_SCOPE.md](./FRONTEND_SCOPE.md)
- UI 视觉：[FRONTEND_UI_PROMPTS.md](./FRONTEND_UI_PROMPTS.md)
- 项目规则：`.codebuddy/skills/tornado-seeker-rules/SKILL.md`
