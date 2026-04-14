# Task Tree V2 文档总览

Task Tree V2 同时提供前端工作台、HTTP API、MCP 接口和 Claude Desktop DXT 代理包。

## 端口约定

| 入口 | 地址 |
|------|------|
| 后端 / UI / API / MCP | `http://127.0.0.1:8880` |
| 前端开发服务器 | `http://127.0.0.1:5174/` |

## 文档导航

### 如果你是本地使用者

→ [本地启动与使用.md](./本地启动与使用.md)

### 如果你是 AI 编程助手（Claude Code / Codex）

→ [../skill/SKILL.md](../skill/SKILL.md)（核心技能文档，每次对话自动加载）

### 如果你想了解 MCP / 技能接入方式

→ [技能与MCP说明.md](./技能与MCP说明.md)

### 如果你要确认 HTTP 与 MCP 的能力边界

→ [../backend/docs/http-mcp-parity.md](../backend/docs/http-mcp-parity.md)

## 目录约定

```
task-tree/v2/
  backend/              # Go 后端（模块根）
  frontend/             # Vue 3 前端
  docs/                 # 项目级使用文档
  skill/                # AI 技能文档（面向 Claude Code / Codex）
  task-tree-dxt/        # Claude Desktop DXT 代理包
```

## 能力概览

### 已有 HTTP + MCP 的核心能力

项目、任务、阶段、节点（含 focus / children / subtree / context preset）、Run / 事件 / 产物 / 搜索 / resume / work-items、任务上下文快照、节点 Memory 结构化 patch、收尾总结。

### 当前仅 HTTP 的能力

- Task / Stage Memory 原生 PATCH 与 snapshot
- Node Memory snapshot

### 默认读取行为

| 接口 | 默认 | 需显式请求 |
|------|------|-----------|
| `resume` | 轻量包 | `include=events,runs,...` |
| `get_task` | 不带树 | `include_tree=true` |
| `list_nodes` | summary | `view_mode=detail` |
| `get_run` | 不带日志 | `include_logs=true` |
| `list_node_runs` / `list_artifacts` | summary + cursor | `view_mode=detail` |

## 推荐读取顺序

1. `resume` — 轻量恢复任务上下文
2. `focus_nodes` / `list_children` / `list_subtree_summary` — 锁定局部树
3. `get_node_context(preset=summary)` — 节点概要
4. 按需：`preset=memory/work`、`list_node_runs`、`get_run(include_logs=true)`、`list_events`

## 发布态访问方式

```
cd frontend && npm run build
# 启动后端
# 浏览器访问 http://127.0.0.1:8880
```

不能直接打开 `frontend/dist/index.html` — 页面依赖同源的 `/v1/...` 和 `/mcp` 接口。

## 文档同步约定

修改端口、接口、技能、DXT 或导航入口时，同步更新相关说明文件。修改 `skill/` 下文档时，同步到全局技能目录：

- `C:\Users\Administrator\.claude\skills\task-tree\`
- `C:\Users\Administrator\.codex\skills\task-tree\`
