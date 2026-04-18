# 常见陷阱

| 陷阱 | 正确做法 |
|---|---|
| 创建 stage 时用 `name` | 用 `title` |
| 创建 project 时用 `title` / `key` | 用 `name` / `project_key` |
| `transition` 传 `status` | 传 `action` |
| `move_node` 传 `target_parent_node_id` / `position` | 传 `before_node_id` / `after_node_id` |
| `create_artifact` 传 `url` / `content` | 传 `uri` / `meta` |
| `import_plan.data` 传对象 | `data` 必须是字符串 |
| 对 `group` 节点调 `complete` | 只有 `leaf` 能完成 |
| `progress(1.0)` 后以为节点已完成 | 仍然必须调 `complete` |
| 已知节点还先调 `resume` | `resume` 只用于恢复现场 |
| 手写 `execution_log` | 用 `progress(log_content=...)` 自动记录 |
| AI 生成内容后直接 `upload_artifact` | 先写文件，再 `create_artifact(uri=...)` 记链接 |
| 读取任务树时默认展开整棵树 | 先用 `focus_nodes` / `list_nodes` / `tree_view` |

## 额外检查

- `get_task` 默认不返回树；要看树请用 `list_nodes` / `tree_view`。
- `get_run` 默认不带日志；要看日志请传 `include_logs=true`。
