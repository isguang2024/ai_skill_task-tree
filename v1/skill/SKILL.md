---
name: task-tree-v1
description: Task Tree V1 (Core)。统一仓库中的 v1 版本，与本地 HTTP MCP `http://127.0.0.1:8879/mcp` 通信。前端开发地址 `http://127.0.0.1:5173/`。
---

# Task Tree V1 (Core)

这是统一 `task-tree` 仓库中的 V1 版本。默认连接：

```json
{
  "mcpServers": {
    "task-tree-v1": {
      "url": "http://127.0.0.1:8879/mcp"
    }
  }
}
```

同一个 `task-tree-service.exe serve` 进程提供：
- 前端：`http://127.0.0.1:8879`
- HTTP API：`http://127.0.0.1:8879/v1/...`
- MCP：`http://127.0.0.1:8879/mcp`

## 项目约定

- 仓库目录：`task-tree/v1/`（统一仓库的 v1 子目录）
- 后端：`backend/`
- 前端开发：`http://127.0.0.1:5173/`
- 这是 V1 / Core 版，不要把 V2 的 `8880/5174` 配置写回这里

## 使用原则

- 用户明确提到 `task-tree v1`、`core`、`V1`、`8879` 时，用这一套
- 修改业务 HTTP 接口时，必须同步 MCP 工具
- 后端改动后跑 `go test ./...`
- 前端改动后跑 `npm run build`

## 端口速记

- Core 后端 / MCP：`8879`
- Core 前端 dev：`5173`

## DXT 对应

- V1 DXT 目录：`task-tree/v1/task-tree-dxt/`
- 其中 `proxy.mjs` 必须指向 `http://127.0.0.1:8879/mcp`

## 注意

- 这个技能只描述 V1 版
- 统一仓库中 V2 版另有独立技能文件，不共用端口、不共用代理文件
