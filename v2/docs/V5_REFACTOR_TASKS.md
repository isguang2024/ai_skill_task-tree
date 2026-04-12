## 阶段进度

### 2026-04-12 P1/P2 修复收口（Phase 1-2）

**本次改动**
- 补齐后端 JSON 编解码错误处理、服务优雅关停和 SQLite WAL checkpoint，避免编码异常被吞掉以及服务退出时残留数据库句柄。
- 为任务/节点搜索相关 SQL 统一增加 LIKE 转义和 `ESCAPE '\'`，避免 `%`、`_` 被当作通配符误匹配。
- MCP session store 改为 `sync.RWMutex` 保护读写，前端补上 AI Chat `res.ok` 判断，以及 `TaskList`、`TrashPage`、`TaskDetail` 的销毁清理逻辑。
- 新增 `frontend/.gitignore`，并将 `backend/ai.env.txt` 替换为占位配置，移除工作区内旧密钥明文。

**下次方向**
- 在可支持 `-race` 的环境补跑并发竞态验证，确认 session store 无额外读写竞争。
- 如果还需要启用 AI，需在第三方平台完成真实密钥轮换后，再把新密钥写回本地环境文件。
- 继续推进 P3 的 Stage/Run/Memory UI 与 MCP 对齐，避免后续功能分支建立在旧行为之上。

### 2026-04-12 P3 首轮推进（Phase 3）

**本次改动**
- 收口了 Stage 管理面板现有实现，确认任务详情页已具备阶段列表、当前阶段高亮、激活切换和创建阶段弹窗。
- 在 `backend/internal/tasktree/mcp.go` 补齐 Run 执行层 MCP 暴露：`task_tree_start_run`、`task_tree_finish_run`、`task_tree_get_run`、`task_tree_list_node_runs`、`task_tree_append_run_log`。
- 新增 `task_tree_get_node_context` MCP tool，并同步更新 `backend/docs/http-mcp-parity.md` 与 `backend/docs/mcp-open-manifest.txt`。
- 通过 `go test ./...` 和 `npm run build` 验证当前后端与前端工作树可正常通过编译/构建。

**下次方向**
- 补齐节点详情中的完整 Run 历史列表，满足全部 run、状态、时间和 actor 信息展示要求。
- 完成 Run 详情与日志流 UI，支持按 seq 展示日志、查看 input/output，并处理长日志滚动。
- 扩展 Memory 面板，补全 conclusions、decisions、risks、blockers、next_actions 等字段的完整可视化。

### 2026-04-12 P3 全量收口（Phase 3）

**本次改动**
- 在 `frontend/src/pages/TaskDetail.vue` 补齐节点 Run 历史列表、Run 详情卡片和日志流展示，支持按节点查看全部 run、切换详情、查看 input/output 摘要与滚动日志。
- 将任务、阶段、节点三级 Memory 面板扩展为结构化视图，完整展示 `summary_text`、`conclusions`、`decisions`、`risks`、`blockers`、`next_actions`、`evidence` 等字段，并对空字段自动折叠。
- 保存 Memory 备注时前端携带 `expected_version`，后端 Memory PATCH 路由和存储层补齐乐观锁校验，出现并发覆盖时返回 `409 version mismatch` 并在 UI 中提示刷新后重试。
- 在 `backend/internal/tasktree/service_test.go` 新增 Memory PATCH 冲突用例，重新通过 `go test ./...` 和 `npm run build`。

**下次方向**
- P3 已完成，可转入 P4 的树导航组件抽取和工程化清理，降低 `TaskDetail.vue` 继续膨胀的风险。
- 如果后续继续扩展 Run/Memory 能力，优先拆分独立 composable 或子组件，避免详情页承担过多状态管理。
