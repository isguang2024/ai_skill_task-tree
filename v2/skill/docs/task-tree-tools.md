# Task Tree V2 — MCP 工具完整参考

> 本文件按需查阅，不在每次对话中加载。需要工具细节时再读取。

## 场景速查索引

| 我想... | 用这个工具 |
|---------|-----------|
| 恢复任务上下文（仅恢复现场） | `task_tree_resume` |
| 看当前该做什么 | `task_tree_focus_nodes` / `task_tree_next_node` |
| 看某个父节点的子节点 | `task_tree_list_nodes(parent_node_id=...)` |
| 看某个子树 | `task_tree_list_nodes(subtree_root_node_id=..., max_relative_depth=...)` |
| 领取节点并开始执行 | `task_tree_claim_and_start_run` |
| 上报执行进度 | `task_tree_progress` |
| 完成节点 | `task_tree_complete` |
| 写入/追加 Memory | `task_tree_patch_node_memory` |
| 搜索历史结论 | `task_tree_smart_search` |
| 创建完整任务骨架 | `task_tree_create_task(stages + nodes)` |
| 批量追加节点 | `task_tree_batch_create_nodes` |
| 导入外部计划 | `task_tree_import_plan` |
| 标记阻塞 | `task_tree_transition_node(action=block)` |
| 看节点记忆 | `task_tree_get_node_context(preset=memory)` |
| 看执行证据 | `task_tree_get_node_context(preset=work)` |
| 项目全局概览 | `task_tree_project_overview(view_mode=summary_with_stats)` |

---

## 项目管理

| 工具 | 说明 | 关键参数 |
|------|------|---------|
| `task_tree_list_projects` | 列出所有项目 | `q`（关键词过滤） |
| `task_tree_create_project` | 创建项目 | `title`，`key`，`description`，`is_default` |
| `task_tree_get_project` | 项目详情 | `project_id` |
| `task_tree_update_project` | 更新项目 | `project_id`，可改 `title`/`key`/`description`/`is_default` |
| `task_tree_delete_project` | 删除项目 | `project_id` |
| `task_tree_project_overview` | 项目概览 + 任务列表 | `project_id`，`view_mode=summary_with_stats`（推荐） |

## 任务管理

| 工具 | 说明 | 关键参数 |
|------|------|---------|
| `task_tree_list_tasks` | 列出任务 | `status`，`project_id`，`q` |
| `task_tree_create_task` | 创建任务（可含骨架） | `title`，`goal`，`project_id`，`stages`，`nodes`，`dry_run=true` 仅预演 |
| `task_tree_get_task` | 任务详情 | `task_id`，`include_tree=true` 才返回树 |
| `task_tree_update_task` | 更新任务 | `task_id`，可改 `title`/`task_key`/`goal` |
| `task_tree_delete_task` | 软删除（移入回收站） | `task_id` |
| `task_tree_hard_delete_task` | 硬删除（不可恢复） | `task_id`（必须已在回收站） |
| `task_tree_restore_task` | 从回收站恢复 | `task_id` |
| `task_tree_empty_trash` | 清空回收站 | 无 |
| `task_tree_transition_task` | 任务状态流转 | `task_id`，`action=pause/reopen/cancel` |

### 恢复 / 导航 / 收尾

| 工具 | 说明 | 关键参数 |
|------|------|---------|
| `task_tree_resume` | **恢复现场专用**：轻量恢复包 | `task_id`，`include=events,runs,artifacts,next_node_context,task_memory,stage_memory`，支持全部树过滤参数 |
| `task_tree_next_node` | 推荐下一可执行节点 | `task_id`，返回 `node` + `alternatives` |
| `task_tree_get_remaining` | 剩余统计 | `task_id`，返回剩余数/阻塞数/暂停数/剩余估时 |
| `task_tree_get_task_context` | 任务上下文快照 | `task_id`，含 architecture_decisions/reference_files/context_doc_text |
| `task_tree_patch_task_context` | 更新任务上下文快照 | `task_id`，支持部分更新 |
| `task_tree_wrapup` | 写入任务收尾总结 | `task_id`，`summary`，`conclusions` 等 |
| `task_tree_get_wrapup` | 读取任务收尾总结 | `task_id` |

> `task_tree_resume` 只用于恢复工作现场。已知 `node_id` 时改用 `task_tree_get_node` / `task_tree_get_node_context`；只想找下一步时改用 `task_tree_next_node`；只想看可执行节点或局部树时改用 `task_tree_focus_nodes` / `task_tree_list_nodes` / `task_tree_work_items`。

## 阶段管理

| 工具 | 说明 | 关键参数 |
|------|------|---------|
| `task_tree_list_stages` | 列出任务下的阶段 | `task_id` |
| `task_tree_create_stage` | 创建阶段 | `task_id`，`title`（**不是** `name`） |
| `task_tree_batch_create_stages` | 批量创建阶段（事务） | `task_id`，`stages` 数组 |
| `task_tree_activate_stage` | 激活阶段 | `task_id`，`stage_node_id` |

## 节点读取

| 工具 | 说明 | 关键参数 |
|------|------|---------|
| `task_tree_list_nodes` | 节点列表（最灵活） | `task_id`，`view_mode=slim/summary/detail/events`，`filter_mode=all/focus/active/blocked/done`，`parent_node_id`，`subtree_root_node_id`，`max_relative_depth`，`kind`，`status`，`depth`，`max_depth`，`has_children`，`sort_by`，`sort_order`，`limit`，`cursor` |
| `task_tree_list_nodes_summary` | 轻量节点列表 | `task_id`（等同 `list_nodes(view_mode=summary)`） |
| `task_tree_focus_nodes` | 可执行节点 + 祖先链 | `task_id` |
| `task_tree_get_node` | 单个节点详情 | `node_id` |
| `task_tree_get_node_context` | 节点上下文聚合 | `node_id`，`preset=summary/memory/work/full` |
| `task_tree_get_resume_context` | 节点级 resume 上下文 | `task_id`，`node_id` |

### preset 说明

| preset | 包含内容 | execution_log | 适用场景 |
|--------|---------|:---:|---------|
| `summary` | 节点基础信息 + 祖先链 + 同级摘要 | ❌ | 默认入口，了解节点位置 |
| `work` | summary + memory（结构化字段） + runs/events/artifacts | ❌ | 需要执行证据和决策 |
| `memory` | summary + node/stage/task memory（结构化字段） | ❌ | 需要历史决策和结论 |
| `full` | 全部字段，**包括完整 execution_log** | ✅ | 仅用于接手节点或深度调试 |

> **重要**：`execution_log` 可能非常长（数百行），只有 `preset=full` 才返回。正常工作流用 `summary` 或 `work` 即可。

## 节点写入

| 工具 | 说明 | 关键参数 |
|------|------|---------|
| `task_tree_create_node` | 创建单个节点 | `task_id`，`title`，`kind=leaf/group`，`parent_node_id`，`instruction`，`acceptance_criteria`，`depends_on_keys` |
| `task_tree_batch_create_nodes` | 批量创建（事务原子） | `task_id`，`nodes` 数组（支持多层 `children`） |
| `task_tree_update_node` | 更新节点 | `node_id`，可改 `title`/`instruction`/`acceptance_criteria`/`estimate`/`sort_order` |
| `task_tree_reorder_nodes` | 批量重排同级节点 | `node_ids` 数组（新顺序） |
| `task_tree_move_node` | 移动节点 | `node_id`，`target_parent_node_id`，`position` |
| `task_tree_retype_node` | group → leaf 转换 | `node_id`（仅无子节点的 group 可转） |

## 节点执行

| 工具 | 说明 | 关键参数 |
|------|------|---------|
| `task_tree_claim_and_start_run` | **推荐**：领取 + 开始 | `node_id`，可选 `actor`/`trigger_kind`/`input_summary` |
| `task_tree_claim` | 仅领取 lease | `node_id` |
| `task_tree_release` | 释放 lease | `node_id` |
| `task_tree_progress` | **上报进度 + 记录执行过程** | `node_id`，`progress`（0.0-1.0），`log_content`（自动写入 run_logs，替代手写 execution_log） |
| `task_tree_complete` | **完成节点** | `node_id`，`memory:{...}` 内联 Memory（只需结构化字段），`result_payload`，返回 `.next` 建议 |
| `task_tree_transition_node` | 状态流转 | `node_id`，`action=block/pause/reopen/cancel/unblock`，`message` |
| `task_tree_block_node` | 标记阻塞（旧入口） | 推荐改用 `transition_node(action=block)` |
| `task_tree_patch_node_memory` | 更新节点 Memory | `node_id`，结构化字段（execution_log 已废弃，系统自动从 run_logs 聚合） |

## Run / Event / Artifact

| 工具 | 说明 | 关键参数 |
|------|------|---------|
| `task_tree_start_run` | 创建 Run（通常用 claim_and_start_run 代替） | `node_id` |
| `task_tree_append_run_log` | 追加 Run 日志 | `run_id`，`kind`，`content` |
| `task_tree_finish_run` | 结束 Run | `run_id`，`status`，`result=done/canceled` |
| `task_tree_list_node_runs` | Run 历史列表 | `node_id`，`view_mode=summary/detail`，`cursor` |
| `task_tree_get_run` | Run 详情 | `run_id`，`include_logs=true` 才返回日志 |
| `task_tree_list_events` | 事件流 | `task_id` 或 `node_id`，`limit`，`cursor` |
| `task_tree_list_artifacts` | 产物列表 | `task_id` 或 `node_id`，`view_mode=summary/detail`，`cursor` |
| `task_tree_create_artifact` | 创建链接型产物 | `task_id`，`node_id`，`title`，`url`/`content` |
| `task_tree_upload_artifact` | base64 上传文件产物 | `task_id`，`node_id`，`filename`，`content_base64` |

## 搜索 / 全局

| 工具 | 说明 | 关键参数 |
|------|------|---------|
| `task_tree_smart_search` | **推荐**：全文搜索（FTS5 + BM25） | `q`，`scope=task/node/memory`，`task_id`，`limit` |
| `task_tree_search` | 旧入口（内部转发 smart_search） | 已废弃，用 `smart_search` |
| `task_tree_work_items` | 当前可执行工作项 | 无参数 |
| `task_tree_tree_view` | 缩进树文本视图 | `task_id`，支持阶段过滤和仅可执行筛选 |
| `task_tree_import_plan` | 导入计划 | `data`（**必须是字符串**），`apply=false` 仅预演 |
| `task_tree_sweep_leases` | 清理过期 lease | 无 |
| `task_tree_rebuild_index` | 重建 FTS5 索引 | 无 |

## 已废弃入口

| 旧工具 | 替代方案 |
|--------|---------|
| `task_tree_list_nodes_summary` | `task_tree_list_nodes(view_mode=summary)` |
| `task_tree_search` | `task_tree_smart_search` |
| `task_tree_block_node` | `task_tree_transition_node(action=block)` |

## 短别名

| 别名 | 等同于 |
|------|--------|
| `task_tree.batch_create_nodes` | `task_tree_batch_create_nodes` |
| `task_tree.activate_stage` | `task_tree_activate_stage` |
| `task_tree.list_nodes_summary` | `task_tree_list_nodes_summary` |

## 通用参数模式

### 分页（cursor）

所有列表接口返回 `{ items, has_more, next_cursor }`。下一页传 `cursor=<next_cursor>` 即可。

### view_mode

- `slim`：最少字段
- `summary`：默认，关键字段
- `detail`：全部字段
- `events`：含事件流（仅 list_nodes）

### filter_mode（仅 list_nodes）

- `all`：全部节点
- `focus`：可执行节点 + 祖先链
- `active`：running/ready 状态
- `blocked`：blocked 状态
- `done`：done 状态

### sort_by / sort_order

- `sort_by=path/updated_at/created_at/status/progress`
- `sort_order=asc/desc`
