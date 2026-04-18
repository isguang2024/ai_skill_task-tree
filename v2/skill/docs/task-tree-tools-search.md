# 搜索 / 全局 / 维护

| 工具 | 说明 |
|---|---|
| `task_tree_smart_search` | 全文搜索，推荐入口 |
| `task_tree_work_items` | 当前可执行工作项 |
| `task_tree_tree_view` | 缩进树文本视图 |
| `task_tree_import_plan` | 导入计划，`data` 必须是字符串 |
| `task_tree_sweep_leases` | 清理过期 lease |
| `task_tree_rebuild_index` | 重建全文索引 |

## 搜索建议

- `smart_search(q, scope=task/node/memory/all, task_id?, limit?)`
- 搜历史结论优先搜 `memory`。
- 任务内排查优先加 `task_id`。

## 旧入口

- `search` -> `smart_search`
- `block_node` -> `transition_node(action=block)`
