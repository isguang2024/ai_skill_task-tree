---
name: task-tree
description: Task Tree V2 — AI 任务管理系统。将复杂任务拆解为树形节点，逐个领取执行，通过 Memory 跨会话传递上下文。
---

# Task Tree V2

## 必守规则

- 节点有动词就执行，不要只写报告。
- 先 `claim_and_start_run`，再执行，最后 `complete`。
- 进度必须用 `progress(log_content=...)`，不要攒到最后。
- `group` 只做容器，`leaf` 承载执行。
- 任务过程中如果发现新增工作，直接追加节点，不要口头记住。
- 每个节点完成后必须记录用量；优先写节点增量用量，拿不到时写线程累计用量。
- 节点用量优先按“开始前 tokens_used”与“完成后 tokens_used”的差值计算。

## Codex 用量

- 线程累计用量可从 `~/.codex/state_5.sqlite` 的 `threads.tokens_used` 读取。
- 节点级用量没有原生字段，默认用节点执行前后两次线程累计差值近似。
- task-tree 内部节点差值流程，读 `docs/codex-usage.md`。
- 其他代理独立自取流程，读 `docs/codex-usage-self-fetch.md`。


## 连接

- MCP：`http://127.0.0.1:8880/mcp`
- HTTP API：`http://127.0.0.1:8880/v1/...`

## 按需读取

| 意图 | 文件 |
|---|---|
| 先判断该读什么 | `docs/task-tree-read-paths.md` |
| 任务拆解 / 编排 stage+node | `docs/task-tree-task-planning.md` |
| 执行流程 / self-advance | `docs/task-tree-execution.md` |
| Memory / 结构化字段 | `docs/task-tree-memory.md` |
| 阻塞 / 恢复 / lease | `docs/task-tree-recovery.md` |
| 扩展任务树 | `docs/task-tree-expansion.md` |
| 常见陷阱 / 旧写法 | `docs/task-tree-pitfalls.md` |
| Codex 用量获取（task-tree） | `docs/codex-usage.md` |
| Codex 用量获取（独立自取） | `docs/codex-usage-self-fetch.md` |
| MCP 工具参数 | `docs/task-tree-tools.md` |
| HTTP API | `docs/task-tree-api.md` |

> 先读最小文件，不要整套文档一起加载。
