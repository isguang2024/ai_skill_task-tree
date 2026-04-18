# 写入 / 结构

## 项目 / 任务

| 工具 | 关键参数 |
|---|---|
| `task_tree_create_project` | `name`，`project_key`，`description`，`is_default` |
| `task_tree_update_project` | `project_id`，`name`，`project_key`，`description`，`is_default` |
| `task_tree_project_overview` | `project_id` |
| `task_tree_create_task` | `title`，`goal`，`project_id`，`stages`，`nodes`，`task_key`，`tags`，`dry_run` |
| `task_tree_update_task` | `task_id`，`title`，`task_key`，`goal`，`project_id` |
| `task_tree_transition_task` | `task_id`，`action=pause/reopen/cancel` |
| `task_tree_delete_task` / `hard_delete_task` / `restore_task` / `empty_trash` | 任务回收相关 |

## 阶段

| 工具 | 关键参数 |
|---|---|
| `task_tree_create_stage` | `task_id`，`title`，`node_key`，`instruction`，`acceptance_criteria`，`estimate`，`activate` |
| `task_tree_batch_create_stages` | `task_id`，`stages` |
| `task_tree_activate_stage` | `stage_node_id` |
| `task_tree_list_stages` | `task_id` |

## 节点写入

| 工具 | 关键参数 |
|---|---|
| `task_tree_create_node` | `task_id`，`title`，`kind`，`parent_node_id`，`stage_node_id`，`node_key`，`instruction`，`acceptance_criteria`，`depends_on_keys` |
| `task_tree_batch_create_nodes` | `task_id`，`nodes`，支持嵌套 `children` |
| `task_tree_update_node` | `node_id`，`title`，`instruction`，`acceptance_criteria`，`estimate`，`sort_order` |
| `task_tree_reorder_nodes` | `node_ids` |
| `task_tree_move_node` | `node_id`，`before_node_id`，`after_node_id` |
| `task_tree_retype_node` | `node_id` |

## 约定

- `batch_create_nodes` 适合一次性补整棵子树。
- `move_node` 只是在同级里换位置，不是改父节点。
- `create_task` 适合一步创建完整骨架，配合 `dry_run=true` 先验收。
