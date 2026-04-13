# Task Tree V2 文档总览

本项目（Task Tree V2）同时提供：

- 前端工作台
- HTTP API
- HTTP MCP
- Claude Desktop DXT 代理包
- 技能文档

## 先看什么

### 如果你是本地使用者

看 [本地启动与使用.md](./本地启动与使用.md)。

### 如果你想先了解项目用途和用法

看 [项目说明页说明.md](./项目说明页说明.md)。

### 如果你是技能 / MCP 使用者

看 [技能与MCP说明.md](./技能与MCP说明.md)。

### 如果你要确认 HTTP 与 MCP 的真实边界

看 [backend/docs/http-mcp-parity.md](../backend/docs/http-mcp-parity.md)。

## 端口约定

- 后端 / UI / API / MCP：`http://127.0.0.1:8880`
- 前端开发服务器：`http://127.0.0.1:5174/`

## 当前目录约定

- 后端：`backend/`
- 前端：`frontend/`
- 技能：`skill/SKILL.md`
- DXT：`task-tree-dxt/`
- 项目文档：`docs/`

## 文档同步约定

当你修改关键路径、端口、接口、DXT、技能、导航或使用方式时，要顺手同步更新说明文件，确保文档和最新状态一致。

另外，凡是改动 `skill/` 下文档，不仅要更新仓库内文件，还要同步到全局技能目录：

- `C:\Users\Administrator\.claude\skills\task-tree\`
- `C:\Users\Administrator\.codex\skills\task-tree\`

其中除 `SKILL.md` 外，`skill/docs/task-tree-api.md`、`skill/docs/task-tree-best-practices.md`、`skill/docs/task-tree-tools.md` 也必须同步。

## 当前对外能力

### 已有 HTTP + MCP

- 项目
- 任务
- 阶段
- 节点
- 事件 / 产物 / 搜索 / resume / work-items

### 当前主要是 HTTP

- Memory 读写

### 已有 MCP、但仍需注意读写边界

- Run 执行层
- 节点 context / overview 增强读模型

这不是缺陷说明，而是当前版本的真实边界。对接自动化或技能时应按这个边界来。

## 发布态访问方式

前端 build 后不会变成“直接打开 html 文件”的离线站点。正确方式是：

1. `frontend` 先执行 `npm run build`
2. 启动后端服务
3. 浏览器访问 `http://127.0.0.1:8880`

后端会托管 `frontend/dist`，并同时提供 `/v1/...` 与 `/mcp`。
