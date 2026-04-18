# Codex 用量获取步骤（task-tree 内部）

> 如果是其他代理要独立读取，不经过 task-tree 运行链路，读 `codex-usage-self-fetch.md`。

## 可直接拿到的量

- 当前 Codex 线程的累计用量：`~/.codex/state_5.sqlite` -> `threads.tokens_used`
- 当前线程 ID：环境变量 `CODEX_THREAD_ID`

## 获取步骤

1. 打开 `~/.codex/state_5.sqlite`
2. 用 `CODEX_THREAD_ID` 查 `threads` 表
3. 读取该行的 `tokens_used`
4. 在节点开始前记一次，节点完成后再记一次
5. 两次差值就是这个节点的近似用量

## 建议写法

- 节点 Memory 里写：
  - `codex_tokens_start`
  - `codex_tokens_end`
  - `codex_tokens_delta`
- 如果没法在开始前记值，至少在完成后写线程累计 `tokens_used`

## 约束

- 这不是原生节点字段，不能当成绝对精确的节点账单。
- 同一 Codex 线程内的多节点连续执行，会让差值更接近真实节点用量。
- 如果一个节点里又嵌套了很多额外查询或编辑，差值会包含这些开销。
