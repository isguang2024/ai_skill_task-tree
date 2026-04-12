---
name: task-tree-core
description: Task Tree Core（V1）技能。适用于 `task-tree-core` 项目与本地 HTTP MCP `http://127.0.0.1:8879/mcp`。前端开发地址默认 `http://127.0.0.1:5173/`。
---

# Task Tree Core

这是 `task-tree-core` / V1 版本的技能文档。默认连接：

```json
{
  "mcpServers": {
    "task-tree-core": {
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

- 仓库目录：`task-tree-core/`
- 后端：`backend/`
- 前端开发：`http://127.0.0.1:5173/`
- 这是 V1 / Core 版，不要把 V2 的 `8880/5174` 配置写回这里

## 使用原则

- 用户明确提到 `task-tree-core`、`core`、`V1`、`8879` 时，用这一套
- 修改业务 HTTP 接口时，必须同步 MCP 工具
- 后端改动后跑 `go test ./...`
- 前端改动后跑 `npm run build`

## 端口速记

- Core 后端 / MCP：`8879`
- Core 前端 dev：`5173`

## DXT 对应

- Core DXT 目录：`task-tree-core/task-tree-dxt/`
- 其中 `proxy.mjs` 必须指向 `http://127.0.0.1:8879/mcp`

## 注意

- 这个技能只描述 Core 版
- `task-tree-v2` 另有独立技能和独立 DXT，不共用端口、不共用代理文件
