# HTTP / MCP 边界说明

本文档描述本项目（Task Tree V2）当前版本里，哪些能力已经同时提供 HTTP 与 MCP，哪些能力仍然是 HTTP 优先。

## 架构约定

同一个后端 `serve` 进程提供：

- UI：`http://127.0.0.1:8880`
- HTTP API：`http://127.0.0.1:8880/v1/...`
- HTTP MCP：`http://127.0.0.1:8880/mcp`

开发态前端单独跑在：

- `http://127.0.0.1:5174/`

## 对齐原则

### 必须保持 HTTP + MCP 对齐的核心能力

- 项目
- 任务
- 阶段
- 节点
- 事件
- 搜索 / work-items / remaining / resume
- 产物

### 可以暂时作为 HTTP only 的增强能力

- Node/Stage/Task 级别的 Memory 原生 PATCH 接口（manual_note、full patch）

但这类能力必须在文档里明确标注，不能写成“已经有 MCP 工具”。

## 当前能力矩阵

| 领域 | HTTP API | MCP Tool | 当前状态 |
|---|---|---|---|
| 项目列表 / 详情 | `GET /v1/projects` `GET /v1/projects/{id}` | `task_tree_list_projects` `task_tree_get_project` | 已对齐 |
| 项目创建 / 更新 / 删除 | `POST /v1/projects` `PATCH /v1/projects/{id}` `DELETE /v1/projects/{id}` | `task_tree_create_project` `task_tree_update_project` `task_tree_delete_project` | 已对齐 |
| 项目概览 | `GET /v1/projects/{id}/overview` | `task_tree_project_overview` | 已对齐 |
| 任务列表 / 详情 | `GET /v1/tasks` `GET /v1/tasks/{id}` | `task_tree_list_tasks` `task_tree_get_task` | 已对齐 |
| 任务创建 / 更新 | `POST /v1/tasks` `PATCH /v1/tasks/{id}` | `task_tree_create_task`（支持 `dry_run`） `task_tree_update_task` | 已对齐 |
| 任务回收站 | `DELETE /v1/tasks/{id}` `POST /v1/tasks/{id}/restore` `DELETE /v1/tasks/{id}/hard` `POST /admin/empty-trash` | `task_tree_delete_task` `task_tree_restore_task` `task_tree_hard_delete_task` `task_tree_empty_trash` | 已对齐 |
| 任务状态流转 | `POST /v1/tasks/{id}/transition` | `task_tree_transition_task` | 已对齐 |
| 阶段列表 / 创建 / 批量创建 / 激活 | `GET /v1/tasks/{id}/stages` `POST /v1/tasks/{id}/stages` `POST /v1/tasks/{id}/stages/batch` `POST /v1/tasks/{id}/stages/{stageNodeId}/activate` | `task_tree_list_stages` `task_tree_create_stage` `task_tree_batch_create_stages` `task_tree_activate_stage` | 已对齐 |
| 节点列表 / 摘要 / focus / 详情 | `GET /v1/tasks/{id}/nodes` `GET /v1/nodes/{id}` | `task_tree_list_nodes` `task_tree_list_nodes_summary` `task_tree_focus_nodes` `task_tree_get_node` | 已对齐 |
| 节点创建 / 更新 / 排序 / 移动 | `POST /v1/tasks/{id}/nodes` `PATCH /v1/nodes/{id}` `POST /v1/nodes/reorder` `POST /v1/nodes/{id}/move` | `task_tree_create_node`（支持 `depends_on_keys`） `task_tree_update_node`（支持 `depends_on_keys`） `task_tree_reorder_nodes` `task_tree_move_node` | 已对齐 |
| 节点批量创建 | `POST /v1/tasks/{id}/nodes/batch` | `task_tree_batch_create_nodes` | 已对齐（事务原子） |
| 节点推进 / 阻塞 / claim / release / retype / 状态流转 | `POST /.../progress` `.../complete` `.../block` `.../claim` `.../release` `.../retype` `.../transition` | `task_tree_progress` `task_tree_complete` `task_tree_block_node` `task_tree_claim` `task_tree_release` `task_tree_retype_node` `task_tree_transition_node` | 已对齐 |
| 上下文 / 搜索 | `GET /v1/tasks/{id}/remaining` `.../resume` `.../context` `GET /v1/events` `GET /v1/search` `GET /v1/work-items` | `task_tree_get_remaining` `task_tree_resume` `task_tree_get_resume_context` `task_tree_list_events` `task_tree_search` `task_tree_work_items` | 已对齐 |
| 任务上下文快照 | `GET/PATCH /v1/tasks/{id}/context` | `task_tree_get_task_context` `task_tree_patch_task_context` | 已对齐 |
| 树视图 / 计划导入 | `GET /v1/tasks/{id}/tree-view` `POST /v1/import-plan` | `task_tree_tree_view` `task_tree_import_plan` | 已对齐 |
| 产物 | `GET /v1/tasks/{id}/artifacts` `POST /v1/tasks/{id}/artifacts` `POST /v1/tasks/{id}/artifacts/upload` | `task_tree_list_artifacts` `task_tree_create_artifact` `task_tree_upload_artifact` | 已对齐 |
| Run 执行层 | `POST/GET /v1/nodes/{nodeId}/runs` `GET /v1/runs/{runId}` `POST /v1/runs/{runId}/finish` `POST /v1/runs/{runId}/logs` | `task_tree_start_run` `task_tree_list_node_runs` `task_tree_get_run` `task_tree_finish_run` `task_tree_append_run_log` | 已对齐 |
| Memory（原生 patch） | `GET/PATCH /v1/tasks/{id}/memory` `GET/PATCH /v1/stages/{id}/memory` `GET/PATCH /v1/nodes/{id}/memory` | 无（但 `task_tree_patch_node_memory` 存在） | 部分 HTTP only |
| 节点 context 读模型 | `GET /v1/nodes/{id}/context` | `task_tree_get_node_context` | 已对齐 |
| 工具短别名 | N/A（MCP 能力） | `task_tree.batch_create_nodes` `task_tree.activate_stage` `task_tree.list_nodes_summary` | MCP 已提供 |

## 高效读取策略

默认推荐“先摘要、再下钻”：

1. 先用摘要列表或 focus 树锁定目标节点
2. 再按需读取节点 detail、resume-context、events、artifacts
3. 需要执行层和记忆层信息时，再补 HTTP 请求

### 推荐的 MCP 读取顺序

1. `task_tree_list_tasks` 或 `task_tree_project_overview`
2. `task_tree_focus_nodes` 或 `task_tree_list_nodes_summary`
3. `task_tree_get_node`
4. `task_tree_get_node_context` 或 `task_tree_get_resume_context`
5. `task_tree_get_task_context`（需要任务级决策/参考文件时）
6. `task_tree_list_events`
7. `task_tree_list_artifacts`

### 需要 HTTP 补充的场景

- 编辑 task/stage/node memory 的 `manual_note` 或 full patch
- 任务/阶段 memory 的原生 PATCH（当前仍是 HTTP only）

## Streamable HTTP MCP

`/mcp` 支持：

- `POST /mcp`
- `GET /mcp`
- `DELETE /mcp`
- `Mcp-Session-Id`
- SSE / 回放 / 恢复流

本地 `/mcp` 仅接受回环来源。

## 文档维护要求

1. 新增 MCP 工具后，要同步更新 `mcp-open-manifest.txt`。
2. 如果某个 HTTP 能力暂时没有 MCP 对应，必须在本文件里继续标记为 `HTTP only`。
3. 如果未来给 Run / Memory 补了 MCP，本文件要先改，再更新技能文档与外部接入说明。
