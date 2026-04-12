# Task Tree V1 (Core)

本版本（V1 / Core）提供一套本地任务树服务，面向三类入口同时工作：

- 前端工作台：`http://127.0.0.1:8879`
- HTTP API：`http://127.0.0.1:8879/v1/...`
- HTTP MCP：`http://127.0.0.1:8879/mcp`

同一个 `serve` 进程同时提供前端、HTTP API 和 HTTP MCP。

## 项目结构

```
task-tree/v1/       # 统一仓库中的 V1 版本
  backend/          # Go 后端源码
    *.go            # 所有 Go 源文件
    migrations/     # SQLite 数据库迁移文件（.sql）
    docs/           # 后端相关文档
    ai.env.txt      # AI 服务配置（本地，不提交）
    mcp-config.example.json  # MCP 配置示例
  frontend/         # Vue 2 + Naive UI 前端
    src/            # 前端源码
    dist/           # 构建产物（由 npm run build 生成）
    package.json
    vite.config.js
  skill/            # 技能文档
  data/             # 运行时数据（SQLite DB，自动创建，不提交）
  go.mod / go.sum   # Go 模块定义（须在根目录）
  AGENTS.md         # 本文件，AI 协作约束说明
  task-tree-service  # 构建产物
  启动后端.bat / 启动前端.bat
```

## 构建与启动

```powershell
# 构建后端
go build -o task-tree-service.exe ./cmd/task-tree-service

# 开发运行（不需要先构建）
go run ./cmd/task-tree-service serve

# 构建前端
cd frontend && npm run build

# 启动服务（前端已构建）
.\task-tree-service.exe serve
```

## 核心原则

1. 长任务、跨轮任务、可续跑任务必须使用任务树，不要在对话里维护 TODO。
2. Claude / Codex 默认通过 HTTP MCP 使用任务树，地址固定为 `http://127.0.0.1:8879/mcp`。
3. 前端、HTTP API、MCP 基于同一套 `App + SQLite` 能力层；新增核心任务能力时，HTTP 与 MCP 必须同步对齐，不能出现一边能做、一边做不了。
4. 新增或修改后端业务接口时，必须同时补齐对应 MCP tool，并把能力登记到 `backend/docs/mcp-open-manifest.txt`；如果某个 HTTP 业务接口暂时没有 MCP 等价能力，视为未完成。
5. 提交前建议运行 `backend/scripts/check-mcp-parity.ps1`，它会自动检查当前业务 HTTP 路由与 MCP tool 是否对齐。
4. UI 是工作台，不是演示页。优先清楚的层级、密度和操作闭环，而不是装饰性卡片。

## HTTP / MCP 对齐要求

以下属于必须保持对齐的核心能力：

- 任务：创建、读取、列表、更新、移入回收站、恢复、彻底删除、清空回收站、状态流转
- 节点：创建、读取、列表、更新、进度、完成、阻塞、claim、release、状态流转
- 上下文：remaining、resume、resume-context、search、work-items、events
- 产物：列表、挂链接、上传文件
- 项目：创建、读取、列表、更新、删除、概览

如果只在 HTTP 层新增了核心能力，而 MCP 没补同名能力，视为未完成。纯静态资源、健康检查、页面路由、MCP 传输层本身不在这个业务对齐范围内；但它们承载的业务操作仍要有 MCP 等价能力。

## Streamable HTTP MCP

`/mcp` 采用 Streamable HTTP 语义：

- `POST /mcp`
  - 发送 JSON-RPC 请求
  - `Accept: application/json` 时返回普通 JSON
  - `Accept: text/event-stream` 时返回 SSE
- `Mcp-Session-Id`
  - 初始化时返回会话 ID
  - 后续请求可带上同一会话
- `GET /mcp`
  - 用于 SSE 回放 / 恢复流
  - 需要 `Mcp-Session-Id`
  - 可选 `Last-Event-ID`，从指定事件之后继续回放
- `DELETE /mcp`
  - 关闭会话

本地 `/mcp` 只接受回环来源，避免非本机来源直接访问本地 MCP。

## 任务内容质量

### 任务

- `title`：短、可检索的名词短语
- `goal`：2-4 句说清交付标准、范围外项、约束

### 节点

- `title`：动词短语
- `instruction`：写清文件、函数、步骤、命令
- `acceptance_criteria`：2-5 条可验证条件
- `estimate`：诚实工时

### progress / complete

- 禁止 `done`、`ok`、`完成一半` 这类无效 message
- progress 要说明已做内容与剩余内容
- complete 要写清：
  - 改了什么
  - 证据是什么
  - 有哪些偏差
  - 明确留下了什么后续项

服务端会对 message 做软校验，warnings 会挂在事件 payload 里，UI 也会高亮。

## Skill 技能文件

`skill/` 目录存放 Claude Code Skill（技能）文件，格式为 Markdown（`.md`）。

- 每个 Skill 文件描述一项可复用的操作流程或提示词模板
- 文件命名使用小写短横线，如 `create-task.md`、`review-pr.md`
- 新增 Skill 时在文件头部写清：触发条件、输入、步骤、输出

## UI 方向

当前主界面是任务工作台：

- 首页：任务总览、筛选、快速操作
- 详情页：左侧递归树，右侧当前节点工作台
- 移动端：树抽屉化，默认先看工作台

继续迭代时优先保证：

- 树层级清楚
- 当前节点与左树选中状态一一对应
- 事件流可读
- 产物区有层次
- 移动端抽屉和主面板操作闭环

## 已纳入后续路线

以下能力已明确进入后续规划，但当前未实现：

- 拖拽排序
- 节点移动 / 重挂
- 批量操作
- 优先级
- 截止时间

新增这些能力时，仍需遵守“HTTP 与 MCP 同步对齐”的规则。

## API 文档

- 首页工作台：`http://127.0.0.1:8879`
- API 文档：`http://127.0.0.1:8879/docs`
