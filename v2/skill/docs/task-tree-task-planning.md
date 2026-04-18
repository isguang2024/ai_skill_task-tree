# 任务拆解

## 原则

- 树形优先，不要平铺同级。
- `group` 只组织结构，`leaf` 承载执行。
- 单个 `leaf` 尽量控制在 1 到 4 小时工作量。
- 有依赖就写 `depends_on_keys`。
- 大任务按阶段拆分，优先用 `task_tree_create_task(stages + nodes)` 一步建骨架。
- 追加节点用 `task_tree_batch_create_nodes`。
- 不确定结构时先 `dry_run=true`。

## 关键点

- 创建时优先设置稳定的 `node_key`、`stage_node_id`，便于后续引用。
- 需要拆分时，先把当前节点改成 `group`，再补子节点。
- 发现新的独立工作线，优先新增阶段，而不是把所有节点塞进一个平面列表。
