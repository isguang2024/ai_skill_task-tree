# 执行流程

## 标准流

1. `task_tree_claim_and_start_run(node_id)`
2. 实际执行
3. `task_tree_progress(progress, log_content="...")`
4. `task_tree_complete(node_id, memory:{...})`

## 规则

- 先 claim，再执行，再 complete。
- `progress(1.0)` 不等于完成。
- `summary_text` 必须有实质内容，不能只写“已完成”。
- `recommended_action` 只是建议，不是强制指令。
- 需要下一步时，优先 `task_tree_next_node` / `task_tree_focus_nodes` / `task_tree_list_nodes`。

## 读写边界

- 执行时尽量只读必要节点，不要为了确认一个点把整棵树展开。
- 已知 `node_id` 时，不要先走 `resume`。
