# 执行中扩展任务树

## 场景

- 当前节点需要拆分子任务。
- 发现同级别的新工作。
- 发现新的独立工作线。
- 现有节点描述需要调整。

## 做法

- 拆分当前节点时，先 `retype_node` 变成 `group`，再 `batch_create_nodes`。
- 新增同级工作时，直接 `create_node`。
- 新增工作线时，先 `create_stage`，再补节点。
- 说明变更时，用 `update_node` 修改 `title` / `instruction` / `acceptance_criteria`。
- 扩展原因要写进 `progress(log_content=...)`。
