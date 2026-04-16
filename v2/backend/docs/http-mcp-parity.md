# HTTP / MCP 边界说明

本文档描述 Task Tree V2 中 HTTP 与 MCP 的能力对齐状态，并固化"轻默认、按需展开"的读取规则。

## 架构约定

同一个后端 `serve` 进程提供：

| 入口 | 地址 |
|------|------|
| UI | `http://127.0.0.1:8880` |
| HTTP API | `http://127.0.0.1:8880/v1/...` |
| HTTP MCP | `http://127.0.0.1:8880/mcp` |
| 前端开发 | `http://127.0.0.1:5174/`（仅开发态） |

## 对齐原则

### 必须 HTTP + MCP 对齐的核心能力

项目、任务、阶段、节点、Run / Event / Artifact、Search / Resume / Remaining / Work Items、Task / Node Context

### 当前仅 HTTP 的增强能力

- Task / Stage Memory 原生 PATCH 与 snapshot
- Node Memory snapshot

这些必须在文档中明确标记为 `HTTP only`。

## 能力矩阵

| 领域 | HTTP API | MCP Tool | 状态 |
|------|---------|----------|------|
| **项目** | `GET/POST /v1/projects`，`GET/PATCH/DELETE /v1/projects/{id}`，`GET .../overview` | `list_projects`，`create_project`，`get_project`，`update_project`，`delete_project`，`project_overview` | 已对齐 |
| **任务 CRUD** | `GET/POST /v1/tasks`，`GET/PATCH/DELETE /v1/tasks/{id}`，`DELETE .../hard`，`POST .../restore` | `list_tasks`，`create_task`（含 dry_run），`get_task`，`update_task`，`delete_task`，`hard_delete_task`，`restore_task`，`empty_trash` | 已对齐 |
| **任务流转** | `POST /v1/tasks/{id}/transition` | `transition_task` | 已对齐 |
| **恢复/导航** | `GET .../resume`，`GET .../remaining`，`GET .../next-node`，`GET .../context`，`PATCH .../context` | `resume`，`get_remaining`，`next_node`，`get_task_context`，`patch_task_context` | 已对齐 |
| **收尾** | `GET/POST .../wrapup` | `wrapup`，`get_wrapup` | 已对齐 |
| **阶段** | `GET/POST .../stages`，`POST .../stages/batch`，`POST .../stages/{sid}/activate` | `list_stages`，`create_stage`，`batch_create_stages`，`activate_stage` | 已对齐 |
| **节点 CRUD** | `GET/POST .../nodes`，`POST .../nodes/batch`，`GET/PATCH /v1/nodes/{id}`，`POST .../reorder`，`POST .../move`，`POST .../retype` | `list_nodes`，`list_nodes_summary`，`focus_nodes`，`create_node`，`batch_create_nodes`，`get_node`，`update_node`，`reorder_nodes`，`move_node`，`retype_node` | 已对齐 |
| **节点上下文** | `GET /v1/nodes/{id}/context`，`GET .../resume-context` | `get_node_context`（preset=summary/memory/work/full），`get_resume_context` | 已对齐 |
| **节点执行** | `POST .../claim-and-start-run`，`POST .../claim`，`POST .../release`，`POST .../progress`，`POST .../complete`，`POST .../block`，`POST .../transition` | `claim_and_start_run`，`claim`，`release`，`progress`，`complete`，`block_node`，`transition_node` | 已对齐 |
| **节点 Memory** | `GET/PATCH /v1/nodes/{id}/memory` | `patch_node_memory` | 已对齐（patch） |
| **节点 Memory snapshot** | `POST /v1/nodes/{id}/memory/snapshot` | 无 | **HTTP only** |
| **Run** | `POST .../runs`，`GET .../runs`，`GET /v1/runs/{id}`，`POST .../finish`，`POST .../logs` | `start_run`，`list_node_runs`，`get_run`，`finish_run`，`append_run_log` | 已对齐 |
| **产物** | `GET .../artifacts`，`POST .../artifacts`，`POST .../artifacts/upload`，`GET .../download` | `list_artifacts`，`create_artifact`，`upload_artifact` | 已对齐 |
| **搜索/全局** | `GET /v1/search`，`GET /v1/smart-search`，`GET /v1/work-items`，`GET .../tree-view`，`POST .../import-plan` | `search`，`smart_search`，`work_items`，`tree_view`，`import_plan` | 已对齐 |
| **管理** | `POST .../sweep-leases`，`POST .../empty-trash`，`POST .../rebuild-index` | `sweep_leases`，`empty_trash`，`rebuild_index` | 已对齐 |
| **Task Memory** | `GET/PATCH /v1/tasks/{id}/memory`，`POST .../snapshot` | 无 | **HTTP only** |
| **Stage Memory** | `GET/PATCH /v1/stages/{id}/memory`，`POST .../snapshot` | 无 | **HTTP only** |

## 默认读取规则（HTTP + MCP 统一）

`resume` 只用于恢复工作现场，不是默认查询入口。

| 接口 | 默认行为 | 需显式请求 |
|------|---------|-----------|
| `resume` | 轻量包（task + tree + remaining + next_node_summary） | `include=events,runs,artifacts,next_node_context,task_memory,stage_memory` |
| `get_task` | 不带树 | `include_tree=true` |
| `list_nodes` | `view_mode=summary` | `view_mode=detail/events` |
| `get_node_context` | 建议 `preset=summary` | `preset=memory/work/full` |
| `get_run` | 不带日志 | `include_logs=true` |
| `list_node_runs` | `summary + cursor` | `view_mode=detail` |
| `list_artifacts` | `summary + cursor` | `view_mode=detail` |

## 推荐读取路径

```
恢复现场：
1. resume — 轻量恢复
2. focus_nodes / list_nodes_summary — 锁定可执行节点
3. get_node_context(preset=summary) — 节点概要

普通查询：
1. get_node / get_node_context / next_node / focus_nodes / list_nodes
2. 按需补充：preset=memory/work，list_node_runs，get_run(include_logs=true)
3. 任务级参考：get_task_context
```

## 后端性能优化说明

当前后端已实施以下性能优化：

- **递归 CTE 祖先链查询**：`fetchAncestorChain` 用单条 SQL 递归查询替代 N+1 `findNode` 循环
- **批量 Memory 查询**：`batchGetNodeMemorySummaries` 用 `WHERE IN` 替代逐条查询
- **条件 Focus 树构建**：`buildResumeV2` 仅在需要完整树时才调用 `buildFocusNodes`
- **FTS5 全文搜索**：`smartSearch` 使用 BM25 排名，`search` 已统一转发到 `smartSearch`
- **精确列选择**：列表查询指定具体字段，不使用 `SELECT *`

## Streamable HTTP MCP

`/mcp` 支持：

- `POST /mcp`、`GET /mcp`、`DELETE /mcp`
- `Mcp-Session-Id` 会话管理
- SSE 推送 / 回放 / 恢复流
- 仅接受回环来源

## 文档维护要求

1. 新增 MCP 工具或默认规则后，同步更新 `mcp-open-manifest.txt`
2. HTTP only 能力必须在本文件明确标记
3. 给 Task / Stage Memory 补 MCP 时，本文件、技能文档和外部说明要一起更新
