# Codex 用量自取流程

> 这个流程不依赖 task-tree 的节点、run 或 Memory。任何能访问本机 Codex 状态库的代理，都可以自己执行。

## 目标

- 读取当前 Codex 线程的累计 token。
- 如需节点级近似值，先记开始快照，再在节点结束时再记一次，差值就是该节点的近似用量。

## 前提

- 能读取本机文件系统。
- 能拿到当前线程 ID。
- 默认优先读 `CODEX_STATE_DB_PATH`，否则回退到 `CODEX_HOME/.codex/state_5.sqlite`，再不行读用户目录下的 `.codex/state_5.sqlite`。

## 流程

1. 先确定 sqlite 路径。
2. 读取 `CODEX_THREAD_ID`。
3. 打开 sqlite，查 `threads` 表里 `id = CODEX_THREAD_ID` 的那一行。
4. 取出 `tokens_used`，这就是当前线程累计用量。
5. 如果要算节点用量，在节点开始前保存一次 `tokens_used`，结束后再读一次。
6. 用结束值减开始值，得到这个节点的近似用量。

## 最小查询

- 表：`threads`
- 条件：`id = CODEX_THREAD_ID`
- 字段：`tokens_used`

## 建议输出

- `thread_id`
- `tokens_used`
- `captured_at`
- 如果有节点边界，再加 `start_tokens`、`end_tokens`、`delta_tokens`

## 约束

- 这不是原生节点账单，只是线程累计差值。
- 同一线程里如果穿插了别的操作，差值会把这些开销一起算进去。
- 如果读不到 `CODEX_THREAD_ID`，就不要猜，直接报告无法定位线程。
