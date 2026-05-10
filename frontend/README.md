# Tornado Seeker Frontend

前端单页应用，对接 `push_gateway` 的 WebSocket 与 REST。架构与约束以
[FRONTEND_ARCHITECTURE.md](./FRONTEND_ARCHITECTURE.md) 为准。

## 启动

```bash
# 安装依赖（首次）
pnpm install

# 开发服务器（默认端口 5173，代理 /api 与 /ws 到 http://localhost:8080）
pnpm dev

# 构建
pnpm build

# 预览构建产物
pnpm preview
```

## 环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `VITE_API_BASE` | `http://localhost:8080`（开发） | push_gateway HTTP 入口 |
| `VITE_WS_URL`   | `ws://localhost:8080/ws`        | push_gateway WebSocket 入口 |

## 目录约定

见 [FRONTEND_ARCHITECTURE.md §2](./FRONTEND_ARCHITECTURE.md#2-目录结构)。
