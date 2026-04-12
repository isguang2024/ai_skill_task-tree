# HTTP / MCP 对齐说明

## 架构约定

`task-tree-service.exe serve` 是唯一推荐的运行形态。

同一个进程同时提供：

- 前端 UI：`http://127.0.0.1:8879`
- HTTP API：`http://127.0.0.1:8879/v1/...`
- HTTP MCP：`http://127.0.0.1:8879/mcp`

三者都基于同一套 `App + SQLite` 逻辑层，不允许各自维护不同业务规则。

## 对齐原则

以下属于必须同步对齐的核心能力：

| 领域 | HTTP API | MCP Tool |
|---|---|---|
| 任务列表 / 详情 | `GET /v1/tasks` `GET /v1/tasks/{id}` | `task_tree_list_tasks` `task_tree_get_task` |
| 创建 / 更新任务 | `POST /v1/tasks` `PATCH /v1/tasks/{id}` | `task_tree_create_task` `task_tree_update_task` |
| 任务删除 / 恢复 | `DELETE /v1/tasks/{id}` `POST /v1/tasks/{id}/restore` `DELETE /v1/tasks/{id}/hard` `POST /admin/empty-trash` | `task_tree_delete_task` `task_tree_restore_task` `task_tree_hard_delete_task` `task_tree_empty_trash` |
| 任务状态流转 | `POST /v1/tasks/{id}/transition` | `task_tree_transition_task` |
| 项目列表 / 详情 | `GET /v1/projects` `GET /v1/projects/{id}` | `task_tree_list_projects` `task_tree_get_project` |
| 创建 / 更新 / 删除项目 | `POST /v1/projects` `PATCH /v1/projects/{id}` `DELETE /v1/projects/{id}` | `task_tree_create_project` `task_tree_update_project` `task_tree_delete_project` |
| 项目概览 | `GET /v1/projects/{id}/overview` `GET /v1/projects/{id}/tasks` | `task_tree_project_overview` `task_tree_list_tasks` |
| 节点列表 / 详情 | `GET /v1/tasks/{id}/nodes` `GET /v1/nodes/{id}` | `task_tree_list_nodes` `task_tree_get_node` |
| 创建 / 更新节点 | `POST /v1/tasks/{id}/nodes` `PATCH /v1/nodes/{id}` | `task_tree_create_node` `task_tree_update_node` |
| 节点推进 | `POST /v1/tasks/{id}/nodes/{id}/progress` `.../complete` `.../block` `.../transition` | `task_tree_progress` `task_tree_complete` `task_tree_block_node` `task_tree_transition_node` |
| 协作 | `POST /.../claim` `POST /.../release` | `task_tree_claim` `task_tree_release` |
| 上下文 | `GET /v1/tasks/{id}/remaining` `.../resume` `.../resume-context` `GET /v1/search` `GET /v1/work-items` `GET /v1/events` | `task_tree_get_remaining` `task_tree_resume` `task_tree_get_resume_context` `task_tree_search` `task_tree_work_items` `task_tree_list_events` |
| 产物 | `GET /v1/tasks/{id}/artifacts` `POST /v1/tasks/{id}/artifacts` `POST /v1/tasks/{id}/artifacts/upload` | `task_tree_list_artifacts` `task_tree_create_artifact` `task_tree_upload_artifact` |

规则：

1. 新增或修改任何后端业务 HTTP 能力时，必须同步新增或更新对应 MCP tool；如果无法直接一一对应，至少要提供等价 MCP 能力。
2. 如果 MCP 端因传输形态不同无法直接复用 HTTP 入参，至少要提供等价能力，例如文件上传通过 base64 版本对齐。
3. 新增能力要补测试，至少覆盖一个 HTTP 场景和一个 MCP 场景。
4. 静态资源、健康检查、页面路由和 MCP 传输层本身不计入业务对齐，但其对应的业务操作不能只存在于 HTTP。
5. 可以用 `backend/scripts/check-mcp-parity.ps1` 做本地自动校验，脚本会检查当前业务 HTTP 路由与 MCP tool 是否都已登记。
6. `POST /v1/tasks` / `task_tree_create_task` 现在都支持可选 `nodes` 初始节点树输入；单次创建可以直接带多层 `children`，也保留后续逐个 `create_node` 补节点的方式。

## 高效读取策略（新增）

默认推荐“先摘要、再下钻”：

1. 先读取摘要节点集合（低 payload）
2. 锁定目标节点后再读取 detail/events
3. 仅在需要完整树时读取全量

对应能力：

- HTTP:
  - `GET /v1/tasks/{id}/nodes?view_mode=summary`
  - `GET /v1/tasks/{id}/nodes?filter_mode=focus`
  - `GET /v1/tasks/{id}/resume?view_mode=summary&filter_mode=focus`
  - `GET /v1/events?view_mode=summary&type=...&q=...`
- MCP:
  - `task_tree_list_nodes_summary`
  - `task_tree_focus_nodes`
  - `task_tree_list_nodes`（支持 `view_mode/filter_mode/sort/cursor`）
  - `task_tree_resume`（支持节点筛选参数 + 事件筛选参数）

筛选参数（HTTP/MCP 对齐）：

- 节点：`status` `kind` `depth` `max_depth` `updated_after` `has_children` `q` `filter_mode` `view_mode` `sort_by` `sort_order` `cursor` `limit`
- 事件：`type` `q` `view_mode` `sort_order` `cursor` `before` `after` `limit`

`filter_mode=focus` 语义：
- 仅返回可执行叶子（默认 `ready/running`）及其祖先链，用于 AI 快速定位下一步，不扫全树。

## Streamable HTTP MCP

`/mcp` 当前支持：

- `POST /mcp`
  - JSON-RPC 请求入口
  - `Accept: application/json` 返回 JSON
  - `Accept: text/event-stream` 返回 SSE
- `Mcp-Session-Id`
  - 初始化时下发
  - 后续请求沿用
- `GET /mcp`
  - 基于 `Mcp-Session-Id` 读取回放流
  - `Last-Event-ID` 用于恢复断点之后的事件
- `DELETE /mcp`
  - 删除会话

本地 `/mcp` 只接受回环来源。

## 进度同步语义

以下语义必须保持一致，避免前端、HTTP 与 MCP 各自理解不同：

1. `summary_percent`、`summary_total`、`summary_done`、`summary_blocked` 只按叶子节点计算，`group` 节点只用于 rollup，不参与总量统计。
2. `remaining` 里的 `blocked_nodes`、`paused_nodes`、`canceled_nodes`、`remaining_nodes`、`remaining_estimate` 也只按叶子节点计算，不能把父级 `group` 重复计数。
3. `complete` 对带幂等 key 的重试必须稳定，不能因为重复请求制造新的版本变化或伪更新时间。
4. `release` / 过期 lease 清理时：
   - 零进度的 `running` 叶子节点要回到 `ready`
   - 已有进度的 `running` 节点保持 `running`
   - 任务 rollup 与更新时间要同步刷新
5. `block` / `complete` 会同步清理 lease，占用状态不能滞留在一个已经阻塞或已完成的节点上；`blocked` 节点也不能再次被 `claim`。
6. 前端显示必须以同一套后端计算结果为准，不能在 UI 内重复发明一套进度统计；“可领取工作”只展示真正可执行的 `ready/running` 节点，不把 `blocked` 混进来。

## UI 与接口边界

UI 不应该偷偷调用只有浏览器能用、而 MCP / HTTP 不可用的私有业务逻辑。

允许 UI 额外拥有的内容：

- 页面布局
- 前端局部状态
- 表单重定向与 flash
- 本地树展开状态缓存

不允许 UI 独占的内容：

- 任务 / 节点状态流转规则
- rollup 规则
- claim / lease 语义
- 产物存储语义

## 已明确排进后续路线

以下属于下一阶段计划，但当前未实现：

- 拖拽排序
- 节点移动 / 重挂
- 批量操作
- 优先级
- 截止时间

这些能力落地时，同样要遵守 HTTP / MCP 对齐规则。
