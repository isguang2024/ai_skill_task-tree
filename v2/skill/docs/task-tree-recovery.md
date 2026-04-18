# 阻塞与恢复

## 阻塞

1. `task_tree_transition_node(node_id, action=block, message="...")`
2. `task_tree_patch_node_memory(node_id, blockers="...")`
3. `task_tree_release(node_id)`

## 解除 / 调整状态

- 解除阻塞：`task_tree_transition_node(action=unblock)`
- 暂停：`pause`
- 恢复：`reopen`
- 取消：`cancel`

## Lease 机制

- `claim` / `claim_and_start_run` 会占用 lease。
- `complete` 会自动释放。
- 异常退出后，lease 过期或 `sweep_leases` 可回收。
- 需要暂停时，先 `release`，后续再重新 claim。

## 故障处理

- 中途失败：写 Memory 记录进度，再释放 lease。
- 需求变更：先改 `instruction`，再重新执行。
- 节点不再需要：直接 `transition_node(action=cancel)`。
