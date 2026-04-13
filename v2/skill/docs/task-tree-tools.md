# Task Tree V2 — 完整 MCP 工具清单

> 本文件是按需查阅的参考文档，不在每次对话中加载。需要时用 `Read` 工具读取。

## 项目管理

| 工具 | 说明 |
|------|------|
| `task_tree_list_projects` | 列出所有项目 |
| `task_tree_create_project` | 创建项目 |
| `task_tree_get_project` | 获取项目详情 |
| `task_tree_update_project` | 更新项目 |
| `task_tree_delete_project` | 删除项目 |
| `task_tree_project_overview` | 项目概览（含任务统计） |

## 任务管理

| 工具 | 说明 |
|------|------|
| `task_tree_list_tasks` | 列出任务（支持 project_id 筛选） |
| `task_tree_create_task` | 创建任务（可含初始 stages + 节点树，一步到位） |
| `task_tree_create_task(dry_run=true)` | 仅预演校验，不落库，返回 preview_tree 和依赖解析结果 |
| `task_tree_get_task` | 获取任务详情 |
| `task_tree_update_task` | 更新任务 |
| `task_tree_delete_task` | 软删除任务（进回收站） |
| `task_tree_hard_delete_task` | 硬删除任务 |
| `task_tree_restore_task` | 从回收站恢复 |
| `task_tree_transition_task` | 任务状态流转（pause/reopen/cancel） |
| `task_tree_resume` | **核心**：获取任务完整上下文 |
| `task_tree_next_node` | **核心**：获取下一个可执行节点 |
| `task_tree_get_remaining` | 获取任务剩余进度统计 |
| `task_tree_get_resume_context` | 获取节点级恢复上下文（祖先链 + Memory + 最近运行） |

## 节点操作

| 工具 | 说明 |
|------|------|
| `task_tree_list_nodes` | 列出节点（view_mode: slim/summary/detail/events） |
| `task_tree_create_node` | 创建节点（支持 depends_on / depends_on_keys） |
| `task_tree_batch_create_nodes` | **优化**：批量创建节点（N→1，事务原子） |
| `task_tree_get_node` | 获取节点详情（`include_context=true` 含完整上下文） |
| `task_tree_update_node` | 更新节点（支持 depends_on / depends_on_keys） |
| `task_tree_get_node_context` | **核心**：节点完整上下文 |
| `task_tree_focus_nodes` | 获取可执行的焦点节点 |
| `task_tree_move_node` | 移动节点 |
| `task_tree_reorder_nodes` | 重新排序同级节点 |
| `task_tree_retype_node` | 节点类型转换（group ↔ leaf） |

## 节点执行层

| 工具 | 说明 |
|------|------|
| `task_tree_claim_and_start_run` | **优化**：领取 + 开始运行（2→1） |
| `task_tree_claim` | 领取节点执行权 |
| `task_tree_release` | 释放节点执行权 |
| `task_tree_progress` | 上报进度（支持 `log_content` 内联日志） |
| `task_tree_complete` | **优化**：完成节点（自动 finish run + 内联 memory + 返回 next_node，支持 `result_payload`） |
| `task_tree_transition_node` | 节点状态流转（block/pause/reopen/unblock/cancel） |
| `task_tree_next_node` | 获取下一个可执行节点（含 `alternatives` 并行候选） |

## 运行层

| 工具 | 说明 |
|------|------|
| `task_tree_start_run` | 为节点创建运行记录 |
| `task_tree_append_run_log` | 追加运行日志 |
| `task_tree_finish_run` | 结束运行（complete 已自动 finish，通常不需要手动调用） |
| `task_tree_get_run` | 获取运行详情（含完整日志） |
| `task_tree_list_node_runs` | 列出节点的运行历史 |

## 阶段管理

| 工具 | 说明 |
|------|------|
| `task_tree_list_stages` | 列出阶段 |
| `task_tree_create_stage` | 创建阶段（可选立即激活） |
| `task_tree_batch_create_stages` | 批量创建阶段（事务原子） |
| `task_tree_activate_stage` | 激活指定阶段 |

## Memory 与搜索

| 工具 | 说明 |
|------|------|
| `task_tree_patch_node_memory` | **核心**：更新节点 Memory。支持 execution_log（覆盖）和 append_execution_log（追加）两种模式 |
| `task_tree_smart_search` | **核心**：全文检索（FTS5 + BM25） |
| `task_tree_get_task_context` | 读取任务上下文快照（架构决策、参考文件、上下文文档） |
| `task_tree_patch_task_context` | 更新任务上下文快照字段 |
| `task_tree_tree_view` | 输出树形缩进视图（可按 stage 过滤） |
| `task_tree_import_plan` | 导入 Markdown/YAML/JSON 计划（支持 dry-run/apply） |
| `task_tree_rebuild_index` | 重建全文检索索引 |

## 其他

| 工具 | 说明 |
|------|------|
| `task_tree_work_items` | 获取当前可执行的工作项 |
| `task_tree_list_events` | 列出任务事件流 |
| `task_tree_create_artifact` | 创建产物链接 |
| `task_tree_upload_artifact` | 上传产物文件 |
| `task_tree_list_artifacts` | 列出产物 |
| `task_tree_wrapup` | 写入任务收尾总结（本次改动、验证结果、遗留问题） |
| `task_tree_get_wrapup` | 获取任务收尾总结，通过 task_id 直接获取 |
| `task_tree_sweep_leases` | 清理过期 lease |
| `task_tree_empty_trash` | 清空回收站 |
| `task_tree.batch_create_nodes` | `task_tree_batch_create_nodes` 的短别名 |
| `task_tree.activate_stage` | `task_tree_activate_stage` 的短别名 |
| `task_tree.list_nodes_summary` | `task_tree_list_nodes_summary` 的短别名 |

## 已废弃工具（保留向后兼容）

| 工具 | 替代方案 |
|------|---------|
| `task_tree_list_nodes_summary` | `list_nodes(view_mode: 'summary')` |
| `task_tree_search` | `smart_search` |
| `task_tree_block_node` | `transition_node(action: 'block')` |
