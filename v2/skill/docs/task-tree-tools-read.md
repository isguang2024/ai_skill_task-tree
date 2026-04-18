# 读取 / 导航

## 常用读取入口

| 意图 | 工具 |
|---|---|
| 恢复现场 | `task_tree_resume` |
| 看下一步 | `task_tree_next_node` |
| 看可执行节点 | `task_tree_focus_nodes` |
| 看父节点子节点 | `task_tree_list_nodes(parent_node_id=...)` |
| 看子树 | `task_tree_list_nodes(subtree_root_node_id=..., max_relative_depth=...)` |
| 只看节点详情 | `task_tree_get_node` |
| 看节点上下文 | `task_tree_get_node_context(preset=summary/memory/work/full)` |
| 看节点级恢复上下文 | `task_tree_get_resume_context` |
| 看剩余统计 | `task_tree_get_remaining` |
| 看任务上下文 | `task_tree_get_task_context` |
| 看整棵树 | `task_tree_tree_view` |
| 看可执行工作项 | `task_tree_work_items` |
| 列项目 / 任务 / 阶段 | `task_tree_list_projects` / `task_tree_list_tasks` / `task_tree_list_stages` |

## 读取原则

- `resume` 只用于恢复现场，同一 `task_id` 默认最多一次。
- `resume` 默认轻量；`include` 按需开启 `events,runs,artifacts,next_node_context,task_memory,stage_memory`。
- `get_task` 不返回树；要看树请用 `list_nodes` 或 `tree_view`。
- `list_*` 和 `work_items` 统一返回 `{ items, has_more, next_cursor }`。
- `get_node_context` 默认用 `summary`，不要一上来就 `full`。

## 常用参数

- `task_tree_list_nodes`：`view_mode=slim/summary/detail/events`，`filter_mode=all/focus/active/blocked/done`。
- `task_tree_get_node_context`：`preset=summary/memory/work/full`。
- `task_tree_get_task`：`includeCustomerNeeds`、`includeRelations`。
