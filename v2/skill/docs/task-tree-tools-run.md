# 执行 / Run / 产物

## 执行

| 工具 | 说明 |
|---|---|
| `task_tree_claim_and_start_run` | 推荐：领取 + 开始执行 |
| `task_tree_claim` | 仅领取 lease |
| `task_tree_release` | 释放 lease |
| `task_tree_progress` | 上报进度 + 记录过程 |
| `task_tree_complete` | 完成节点 |
| `task_tree_transition_node` | `action=block/pause/reopen/cancel/unblock` |
| `task_tree_patch_node_memory` | 更新节点 Memory |

## Run / Event

| 工具 | 说明 |
|---|---|
| `task_tree_start_run` | 创建 Run（通常用 `claim_and_start_run` 代替） |
| `task_tree_append_run_log` | 追加 Run 日志 |
| `task_tree_finish_run` | 结束 Run |
| `task_tree_list_node_runs` | Run 历史列表 |
| `task_tree_get_run` | Run 详情，日志需 `include_logs=true` |
| `task_tree_list_events` | 事件流 |

## 产物

| 工具 | 说明 |
|---|---|
| `task_tree_list_artifacts` | 产物列表 |
| `task_tree_create_artifact` | 链接型产物，使用 `uri` / `meta` |
| `task_tree_upload_artifact` | base64 上传外部文件产物 |

## 常用参数

- `task_tree_progress`：`progress` + `log_content`。
- `task_tree_complete`：`memory` 内联写入结构化字段。
- `task_tree_patch_node_memory`：`summary_text`、`decisions`、`evidence`、`next_actions`、`blockers`、`risks`。
- `task_tree_upload_artifact`：`content_base64`、`filename`。

## 约定

- `progress(1.0)` 不等于完成，必须再调 `complete`。
- AI 生成的内容优先写文件，再用 `create_artifact(uri=...)` 记链接。
- `upload_artifact` 只适合外部文件、截图、附件。
