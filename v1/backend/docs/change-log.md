## 2026-04-11 前端新增项目入口

### 本次改动
- 在任务总览页补了“新建项目”按钮和弹窗表单，直接调用现有 `POST /v1/projects` 接口创建项目。
- 新建项目支持名称、项目 Key、描述和默认项目开关，创建完成后会自动刷新项目列表。
- 这次没有新增后端重复接口，而是复用已有项目创建能力，保持 HTTP / MCP 对齐不变。

### 下次方向
- 如果后面还要继续完善项目层能力，可以把创建项目也补进更显眼的全局入口，而不只放在总览页。
- 可以继续补项目级别的搜索、默认项目提示和创建成功后的自动跳转逻辑。

## 2026-04-11 MCP 自动对齐校验脚本

### 本次改动
- 新增 [backend/scripts/check-mcp-parity.ps1](/Users/Administrator/Desktop/Ai任务步骤记录/task-tree-core/backend/scripts/check-mcp-parity.ps1)，自动比对当前业务 HTTP 路由与 MCP tool 清单，缺项会直接退出非零。
- 同步把脚本使用方式写进 [AGENTS.md](/Users/Administrator/Desktop/Ai任务步骤记录/task-tree-core/AGENTS.md) 和 [backend/docs/http-mcp-parity.md](/Users/Administrator/Desktop/Ai任务步骤记录/task-tree-core/backend/docs/http-mcp-parity.md)，避免后续只靠人工检查。
- 脚本已在当前仓库验证通过，覆盖 36 个业务端点。

### 下次方向
- 后续新增业务接口时，先补脚本里的对照表，再补 MCP tool 和测试。
- 如果再想减少人工维护，可以把对照表拆成可机器读取的 YAML 或 JSON，再由脚本和文档共用同一份源数据。

## 2026-04-11 MCP 接口对齐与开放清单

### 本次改动
- 补齐了项目层面的 MCP 能力，新增 `task_tree_update_project` 和 `task_tree_delete_project`，让 HTTP 的项目更新/删除与 MCP 对齐。
- 在 [AGENTS.md](/Users/Administrator/Desktop/Ai任务步骤记录/task-tree-core/AGENTS.md) 和 [backend/docs/http-mcp-parity.md](/Users/Administrator/Desktop/Ai任务步骤记录/task-tree-core/backend/docs/http-mcp-parity.md) 里写明了约束：后端新增或修改业务接口时，必须同步 MCP。
- 新建了 [backend/docs/mcp-open-manifest.txt](/Users/Administrator/Desktop/Ai任务步骤记录/task-tree-core/backend/docs/mcp-open-manifest.txt)，按“一行一个”登记当前开放的 MCP tool 和简介，方便后续核对。

### 下次方向
- 如果后续继续新增 `/v1` 业务接口，先补 MCP 再合入，避免只改一侧。
- 如果想把“对齐检查”做得更硬，可以再补一个脚本，自动比对 HTTP 路由和 MCP tool 清单，减少人工遗漏。

## 2026-04-11 本地请求链路性能优化

### 本次改动
- 后端把 SQLite 连接池从单连接放宽到 4 个连接，并设置了对应的 idle 连接数，避免本地多个独立 `GET` 请求被同一连接串行排队。
- 任务详情页初始加载改成优先走一次 `resume?view_mode=detail&limit=10000`，复用返回的任务、树、remaining、事件和产物数据，减少首屏请求数。
- 详情页选中节点时优先使用已加载的节点数据，只在必要时才回退到单节点接口；领取/释放后仍强制刷新，避免因为本地缓存显示旧状态。
- 已验证 `go test ./...` 和前端 `npm run build` 通过。

### 下次方向
- 任务总览页的每任务 `remaining/resume` fanout 仍然偏多，下一轮可以继续收敛成更少的汇总请求。
- 如果后面还要继续压响应时间，可以给关键 API 加一次轻量计时日志，直接定位最慢的 SQL 和最慢的页面请求。

## 2026-04-11 数据目录迁移到 backend

### 本次改动
- 将运行时数据库目录从仓库根的 `data/` 迁移到 `backend/data/`，和 Go 模块根、启动脚本保持一致。
- 清理了根目录下的旧数据库文件，避免后续调试时再把数据写回旧位置。
- `.gitignore` 已同步更新为忽略 `backend/data/`，防止运行态数据进版本库。

### 下次方向
- 如果后面还想继续收敛结构，可以把数据库路径也显式写进启动脚本或配置项，减少对默认相对路径的依赖。
- 若要保留历史调试数据，建议单独做一次备份目录，而不是让根目录和 `backend/` 同时存在两份数据库。

## 2026-04-11 后端代码结构化迁移

### 本次改动
- 将原本平铺在 `backend/` 根目录的 Go 源码收拢到 `backend/internal/tasktree`，并新增 `backend/cmd/task-tree-service/main.go` 作为唯一启动入口。
- 迁移后保留了现有功能边界，`ai.env.txt`、`migrations/` 和前端静态目录改为通过向上查找定位，兼容 `go test` 在包目录下执行的工作目录。
- `go test ./...` 已在 `backend/` 下通过，说明拆分后的 `cmd` + `internal` 结构仍然可用。

### 下次方向
- 如果后面还要继续收敛结构，可以再把 `internal/tasktree` 进一步拆成 `app`、`store`、`ui`、`mcp` 等更小包。
- 需要补一份简短的运行说明，明确现在应该从 `backend/` 进入并通过 `go run ./cmd/task-tree-service` 或对应二进制启动。

## 2026-04-11 Go 模块迁移到 backend

### 本次改动
- 将 `go.mod` 和 `go.sum` 迁移到了 `backend/`，让 Go 模块根与后端源码目录对齐。
- 同步修正了后端代码里的相对路径：`ai.env.txt`、`migrations/` 和前端静态目录都改成基于 `backend/` 的实际位置。
- 更新了烟测断言，改为校验当前 SPA 壳而不是旧的服务器渲染文案；`go test ./...` 已在 `backend/` 下通过。

### 下次方向
- 如果后续还要继续收敛结构，可以把根目录的启动说明和构建说明也改成“进入 `backend/` 执行 Go 命令”的版本。
- 若前端仍要继续保持独立构建，建议再补一份根目录级说明，明确 `backend/` 和 `frontend/` 的职责边界。

## 2026-04-11 Git 仓库初始化并推送 GitHub

### 本次改动
- 在 `task-tree-core` 初始化了新的 Git 仓库，并将 `main` 分支推送到 `https://github.com/isguang2024/ai_skill_task-tree`。
- 补充了忽略规则，排除了本地密钥文件、代理本地配置、前端依赖目录和构建产物，避免把环境态与编译输出带入仓库历史。
- 首次提交已完成，远端分支已建立跟踪关系；推送时 Git 提示本机缺少 `credential-manager-core`，但不影响本次推送结果。

### 下次方向
- 后续若继续开发，直接在当前远端仓库上追加提交即可，不需要重新初始化。
- 如果前端还需要重新生成静态产物，建议先确认哪些目录应纳入源码、哪些只保留在本地构建输出，再决定是否重新提交。

## 2026-04-10 Task Tree 关闭态 UI 收口与空分组转回执行节点

### 本次改动
- 后端新增 `retypeNodeToLeaf`，允许无子节点的 `group` 节点显式转回 `leaf`，会重置为 `ready + 空 result`，并写入 `node_retyped` 事件；HTTP API、UI 路由和 MCP tool 已一起补齐。
- 工作台 UI 开始真正消费 `result` 语义：`closed + mixed` 现在显示为“混合关闭”，任务总览新增 `已取消 / 混合关闭` 筛选项，顶部汇总也补了取消数和关闭数。
- 任务详情、树节点、搜索结果和直接子节点列表都改成按 `status + result` 联合展示；`canceled` 从灰态改成红态，避免再和 `closed` 混在一起。
- `service_test.go` 新增“空 group 转回 leaf 成功 / 有子节点时禁止转回”回归，`go test ./...` 已通过。

### 下次方向
- 继续把 `mixed` 的构成来源做得更直观，例如在详情页补“完成 X / 取消 Y”的关闭拆解，而不只是显示单个标签。
- 如果后面要支持节点删除或移动，记得把 `group -> leaf` 的可用性判断也纳入相同规则，避免出现孤儿 group 或错误回退。

## 2026-04-10 Task Tree 状态结果正确重构

### 本次改动
- 任务与节点新增 `result` 语义层，用于承载 `done` / `canceled` / `mixed` 这类结束结果；旧数据会在迁移时自动补齐，取消节点也不再被错误算作“已完成”。
- 后端状态机重写为“过程状态 + 结束结果”双层模型：`leaf` 仍是唯一可执行单元，`group` 只做结构汇总；`leaf` 在新增子节点后会自动降级为 `group`，并清空旧的完成结果与占用状态。
- rollup、remaining、resume、任务取消/重开都按新语义收口：全部取消会得到 `canceled`，完成与取消混合关闭会得到 `closed + mixed`，任务级 `reopen` 只会恢复被取消的叶子，不会误伤已完成节点。
- `service_test.go` 回归测试已同步改写，并补了“完成后再拆分会回退”“mixed 关闭态”“任务级 reopen 恢复取消叶子”等场景；`go test ./...` 已通过。

### 下次方向
- 补一轮 UI 文案和筛选项，让 `closed / mixed` 在任务总览、搜索和事件流里更显式，不再只靠状态标签区分。
- 如果后面要做更彻底的领域模型收敛，可以继续把 `status` 的终态从 `done/canceled/closed` 收到统一 `closed`，把所有结束差异完全放到 `result` 上。

## 2026-04-10 Task Tree lease 与可领取语义继续收紧

### 本次改动
- `block` 和 `complete` 现在会同步清理节点 lease，避免 UI 和 API 里出现“节点已阻塞/完成，但仍显示被某个 agent 占用”的假同步状态。
- `claim` 不再允许落在 `blocked` 节点上，前端工作台里的 `CanClaim` 与“可领取工作”页也同步改成只面向真正可执行的 `ready/running` 节点。
- `service_test.go` 新增回归：验证 blocked 节点 claim 冲突、block/complete 自动释放 claim，以及 `/work` 页面不再混入 blocked 节点；`go test ./...` 与 `go build .` 已通过。

### 下次方向
- 继续补多层树里“部分 running + 部分 blocked + 部分 paused”的组合状态测试，确认 resume 和 rollup 在复杂场景下仍然稳定。
- 如果要再收紧协作语义，下一步可以把 claim actor、lease 续租和抢占策略也纳入统一状态机，而不是只靠事件记录。

## 2026-04-10 Task Tree 进度同步语义收紧

### 本次改动
- 收紧了后端进度同步语义：`remaining` 与任务汇总统计统一改为只按叶子节点计算，避免 group 节点重复计入阻塞、暂停和剩余数量。
- `release` 与过期 lease 清理现在会同步修正零进度 `running` 节点回到 `ready`，并立即刷新任务 rollup，避免前端看到“无人认领但仍在进行中”的假状态。
- `complete` 带幂等 key 的重试现在真正稳定，不再因为重复提交制造新的节点版本或伪更新时间；UI 侧也补了扁平节点的子节点计数标注，减少树与右侧列表信息不一致。
- `service_test.go` 新增了 group 不重复计数、release 归位、complete 幂等和过期 lease 回位等同步场景回归测试；`go test ./...` 与 `go build .` 已通过。

### 下次方向
- 继续补更细的同步边界测试，例如多层 group 混合 `paused/blocked/running` 的 rollup 组合。
- 如果要进一步提高前端一致性，下一步应把任务总览和详情页的局部刷新也统一走同一份状态源，而不是依赖整页重载。

## 2026-04-10 Task Tree HTTP 对齐与工作台收敛

### 本次改动
- `/mcp` 补齐成更完整的 Streamable HTTP：支持 `Mcp-Session-Id`、SSE 响应、`GET /mcp` 基于 `Last-Event-ID` 的恢复流，以及 `DELETE /mcp` 删除会话。
- HTTP API 与 MCP tool 的核心任务能力已拉齐，新增读取任务/节点详情、列出节点、remaining、resume-context、事件、产物、删除/恢复/彻底删除、清空回收站、artifact base64 上传、lease sweep 等能力。
- 任务工作台继续收紧：任务列表改成更紧凑的行式结构，详情页头部改成“左信息右态势”，右侧收成任务动态和产物概览，移动端树抽屉补了遮罩与收起交互。
- 新增 [http-mcp-parity.md](/Users/Administrator/Desktop/Ai任务步骤记录/task-tree-service/docs/http-mcp-parity.md)，明确以后 HTTP 与 MCP 的能力对齐规则；`AGENTS.md` 和 `skill/SKILL.md` 也改成统一的 URL 型 MCP 文案。

### 下次方向
- 继续打磨移动端工作台、事件流的扫描效率，以及产物区的分组和排序。
- 把树展开状态、筛选状态和当前 tab 的恢复再做细一点，减少刷新后的跳变。
- 推进下一阶段能力：拖拽排序、节点移动 / 重挂、批量操作，以及优先级 / 截止时间。

## 2026-04-10 Task Tree UI 与状态流转重构

### 本次改动
- 将主界面重构为新的任务工作台，首页改成高密度任务总览，详情页改成左树右侧节点工作台，支持递归树、tab、内联编辑和任务/节点主按钮操作。
- 新增任务与节点编辑接口，以及统一的 `transition` 状态流转接口；补齐 `paused` / `canceled` 语义，并重写 rollup、remaining、resume 的聚合规则。
- 补齐 UI 表单链路，支持在主工作台里直接保存任务、保存节点、暂停/恢复/取消、阻塞/解除阻塞、重开、上传产物和挂链接。
- MCP 同步新增更新与状态流转工具，测试覆盖 API、UI、深层树展示和状态闭环；`go test ./...` 与 `go build .` 已通过。

### 下次方向
- 把树节点的展开状态从纯本地缓存继续延伸到更细的交互，例如懒加载大树、折叠快捷键和批量展开/收起。
- 继续打磨任务详情的内容密度与视觉节奏，尤其是移动端树抽屉、事件流可读性和产物管理区的信息层次。
- 评估是否补二期能力：拖拽排序、节点移动/重挂、批量操作，以及优先级/截止时间这类计划维度。

## 2026-04-10 Task Tree HTTP MCP 接入

### 本次改动
- 给现有服务新增了 `http://127.0.0.1:8879/mcp` 端点，同一个 `task-tree-service.exe serve` 进程现在同时提供前端、HTTP API 和 HTTP MCP。
- MCP 初始化协议版本统一到 `2025-11-25`，并补了最小可用的 Streamable HTTP 处理：`POST /mcp` 返回 JSON-RPC，`GET /mcp` 明确返回 `405`，通知请求返回 `202`。
- 增加了本地回环 `Origin` 校验，阻止非本机来源直接访问本地 MCP 端点。
- 更新了仓库内和已安装的 `task-tree` 技能文档、示例配置与 `AGENTS.md`，默认推荐用 URL `http://127.0.0.1:8879/mcp` 配置 MCP。

### 下次方向
- 如果客户端需要更完整的 Streamable HTTP 特性，可以继续补 SSE、会话 ID 和可恢复流，而不是只用 JSON 响应模式。
- 把 Claude / Codex 当前已有的本地进程配置切换到 HTTP URL 配置，并验证两边都能直接通过 `/mcp` 工作。
- 清理剩余的旧文档和脚本说明，避免继续混用旧接入方式和 HTTP 接入方式。

## 2026-04-10 Task Tree 工作台三栏布局调整

### 本次改动
- 任务详情页重排为全屏三栏工作台：左侧任务树加宽，中间保留当前节点工作区，最右侧改成随当前选中节点切换的事件日志与产物区，任务头上移为横跨主区的顶层摘要。
- 原本内联挂在页面里的“任务设置”已改成 `TESTRUN` 头部按钮触发的弹窗表单，避免把任务元信息和节点操作混在同一纵向流里。
- 前端交互补齐了新的任务设置弹窗打开/关闭逻辑，并通过 `go test ./...` 与真实页面 `HTTP 200` 请求验证模板和静态资源已正常生效。

### 下次方向
- 继续压缩中间工作区的折叠块数量，把常用推进动作进一步集中，减少首屏信息噪音。
- 如果还要提升扫描效率，下一步最值得做的是右侧事件流筛选、日志高亮和节点切换时的局部刷新，而不是整页重载。

## 2026-04-10 Task Tree Workspace 视觉重构

### 本次改动
- 首页与详情页统一成新的 `Task Tree Workspace` 视觉语言：重做品牌头、导航、总览引导区、指标卡片、任务列表和任务详情抬头，整体从默认白底卡片堆叠改成更明确的工作台层级。
- CSS 主题切到青绿色主色配合橙色动作色，统一了按钮、状态胶囊、树选中态、事件卡片、弹窗和 AI 面板的交互反馈，并引入 `Fira Sans / Fira Code` 作为工作台字体基线。
- 前端脚本补了任务树的移动端闭环：详情页在手机端默认先显示工作面板，树抽屉改为按钮唤起；同时修正树搜索高亮的正则转义问题，并把键盘焦点态与当前选中态拆开。
- 通过 `go test ./...`、桌面端首页/详情页截图，以及移动端详情页与树抽屉截图完成回归，浏览器控制台无报错。

### 下次方向
- 继续压缩详情页折叠块的首屏占比，把“推进节点”和“编辑节点”进一步前置，减少滚动成本。
- 为首页和工作池补更细的排序、筛选和空态策略，提升任务较多时的扫描效率。
- 如果后续继续打磨移动端，可以补树抽屉的局部刷新、手势关闭和当前节点固定定位，避免深层任务切换时反复查找。

## 2026-04-10 Task Tree Workspace 移动端收尾与回归

### 本次改动
- 移动端详情页顶栏改成“品牌区 + 动作组”两层结构，`新建任务 / 任务树 / AI 助手` 不再压缩品牌区；同时在手机端隐藏“上次任务”和固定 AI FAB，改成顶栏内联 AI 入口。
- AI 面板触发器统一为 `data-ai-toggle`，并补上与任务树抽屉的互斥、`Escape` 关闭和窗口放大时自动收起抽屉的逻辑，避免移动端状态残留。
- 本轮重新完成桌面首页、桌面详情、移动端详情与移动端树抽屉回归；`go test ./...` 通过，当前页面控制台无报错，相关截图包含 `uiv2-verify-home-desktop.png`、`uiv2-verify-detail-default-desktop.png`、`uiv2-verify-detail-leaf-desktop.png`、`uiv2-surface-mobile-after.png` 和 `uiv2-surface-mobile-tree-open-after.png`。

### 下次方向
- 默认进入任务详情时，当前选中节点仍可能回落到父分组；下一轮应优先梳理 `resume` 结果到默认 `selected node` 的映射逻辑。
- 如果继续打磨移动端，可把 AI 面板头部再压缩一点，并补树抽屉点击遮罩关闭的显式自动化回归，减少交互回归成本。

## 2026-04-11 创建任务时一次性建树

### 本次改动
- `POST /v1/tasks` 与 `task_tree_create_task` 现在都支持可选 `nodes` 初始节点树输入；每个节点可继续带多层 `children`，创建任务时就能一次性落完整父子结构。
- 后端新增递归 `taskNodeSeed` 输入模型，创建任务后会复用现有 `createNode` 规则逐层建树；如果节点带子节点但未显式指定 `kind`，会按 `group` 处理，旧的逐个 `create_node` 追加方式保持不变。
- `service_test.go` 与 `mcp_test.go` 新增一次性建树回归，覆盖 HTTP 和 MCP 两条链路；`go test ./...` 与 `backend/scripts/check-mcp-parity.ps1` 已通过。
- 同步更新了 `backend/docs/http-mcp-parity.md`、`backend/docs/mcp-open-manifest.txt`，以及仓库版 / Codex 版 / Claude 版的 task-tree 技能说明。

### 下次方向
- 如果后面任务模板越来越复杂，可以继续补“批量创建节点但延迟 rollup”优化，减少超大树初始化时的重复聚合开销。
- 前端后续可补一个“创建任务时直接编辑初始节点树”的表单，而不是只在接口层支持这项能力。

## 2026-04-11 删除项目后恢复任务可见性修复

### 本次改动
- 修复了删除项目后任务进入回收站、再恢复时仍保留已删除 `project_id` 的问题；`restoreTask` 现在会检查原项目是否仍有效，不存在时自动回挂默认项目。
- `task_restored` 事件现在会在发生项目迁移时带上恢复 payload，记录是否回挂默认项目、旧项目 ID 与新项目 ID，便于后续排查。
- 新增后端回归测试，覆盖“删除项目 -> 任务进回收站 -> 恢复任务 -> 重新出现在有效项目 overview”整条链路；`go test ./...` 已通过。

### 下次方向
- 如果后续要支持“恢复项目”能力，可以再评估是否允许任务恢复时优先回原项目，而不是直接回默认项目。
- 前端如果后面增加独立任务总览页，可再补一个弱依赖项目可见性的列表视图，减少项目删改对任务浏览的耦合。
