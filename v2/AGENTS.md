# Task Tree V2

本版本（V2）是统一 `task-tree` 仓库中的一部分，默认运行在：

| 入口 | 地址 |
|------|------|
| 前端工作台 / UI | `http://127.0.0.1:8880` |
| HTTP API | `http://127.0.0.1:8880/v1/...` |
| HTTP MCP | `http://127.0.0.1:8880/mcp` |
| 前端开发服务器 | `http://127.0.0.1:5174/` |

同一个后端 `serve` 进程同时提供已构建前端静态资源、HTTP API 和 HTTP MCP。

## 项目结构

```
task-tree/v2/
  backend/              # Go 后端（模块根）
    cmd/task-tree-service/
    internal/tasktree/
    migrations/
    docs/
    data/               # 运行时 SQLite 数据
  frontend/             # Vue 3 前端
    src/
    dist/               # npm run build 产物
  docs/                 # 项目级使用文档
  skill/                # AI 技能文档（面向 Claude Code / Codex）
  task-tree-dxt/        # Claude Desktop DXT 代理包
  启动后端.bat
  启动前端_前端开发测试服务.bat
```

## 运行方式

### 本地开发

```powershell
# 后端
cd backend
$env:TTS_ADDR="127.0.0.1:8880"
go run ./cmd/task-tree-service serve

# 前端
cd frontend
npm run dev
```

### 本地发布态

```powershell
cd frontend && npm run build
cd ..\backend
$env:TTS_ADDR="127.0.0.1:8880"
go run ./cmd/task-tree-service serve
```

浏览器访问 `http://127.0.0.1:8880`。不能直接打开 `frontend/dist/index.html`。

## 协作原则

1. **端口固定**：后端/MCP 用 `8880`，前端 dev 用 `5174`
2. **能力边界明确**：新增核心能力必须说明是否提供 HTTP API 和 MCP 工具；暂时只有 HTTP 的必须标注"HTTP only"
3. **MCP list_\* 形状契约**：任何返回集合的 MCP 工具（名字含 `list_` 或 `work_items` 这类聚合）必须返回 `{items, has_more, next_cursor}`。非分页时设 `has_more=false, next_cursor=""`；可直接调用 `wrapListResult(items)`。新增工具请同步扩展 `TestListToolsReturnUnifiedShape`（`backend/internal/tasktree/mcp_list_contract_test.go`），否则回归测试会红。HTTP 面的裸数组是历史兼容（`/v1/tasks`、`/v1/projects`、`/v1/tasks/{id}/stages`），新接口不应沿用。
4. **测试验证**：后端改动后 `go test ./...`，前端改动后 `npm run build`
5. **文档同步**：修改 MCP 工具、默认规则、技能工作流或能力边界后，同步更新以下文件：
   - `skill/SKILL.md` + `skill/docs/` 下的文档
   - `docs/技能与MCP说明.md`
   - `backend/docs/http-mcp-parity.md`
   - `backend/docs/mcp-open-manifest.txt`
6. **全局技能同步**：上述文件修改后，同步到全局技能目录：
   - `C:\Users\Administrator\.claude\skills\task-tree\`（Claude Code）
   - `C:\Users\Administrator\.codex\skills\task-tree\`（Codex）
   - 包括 `SKILL.md` 和 `skill/docs/` 下的所有文档

## 当前能力边界

### 已有 HTTP + MCP

- 项目：创建、读取、列表、更新、删除、概览
- 任务：创建（含 dry_run）、读取、列表、更新、回收站、状态流转、收尾总结
- 阶段：列出、创建、批量创建、激活
- 节点：创建、批量创建、读取、列表、摘要/focus/children/subtree、更新、排序、移动、进度、完成、阻塞、claim/release、retype、状态流转
- 上下文：resume、remaining、next-node、resume-context、task-context、node-context（preset 模式）
- 产物：列表、链接型、base64 上传
- Run：start/list/get/finish/log
- 全局：search（FTS5）、smart-search、work-items、tree-view、import-plan

### 当前仅 HTTP

- Task / Stage Memory 原生 PATCH 与 snapshot
- Node Memory snapshot

### 后端性能特性

- 递归 CTE 祖先链查询（替代 N+1 循环）
- 批量 Memory 查询（`WHERE IN` 替代逐条）
- 条件 Focus 树构建（仅在需要时执行）
- FTS5 全文搜索（BM25 排名）
- 精确列选择（无 `SELECT *`）

## DXT / Skill

- Skill 核心文档：`skill/SKILL.md`
- DXT 目录：`task-tree-dxt/`
- DXT 代理指向：`http://127.0.0.1:8880/mcp`

## 推荐文档

| 文档 | 内容 |
|------|------|
| `docs/README.md` | 项目文档总览 |
| `docs/技能与MCP说明.md` | 技能与 MCP 接入指南 |
| `docs/本地启动与使用.md` | 本地启动与使用 |
| `backend/docs/http-mcp-parity.md` | HTTP / MCP 能力边界 |
| `skill/SKILL.md` | AI 技能核心文档 |
| `skill/docs/task-tree-tools.md` | MCP 工具完整参考 |
| `skill/docs/task-tree-api.md` | HTTP API 参考 |
| `skill/docs/task-tree-best-practices.md` | 最佳实践与决策指南 |
