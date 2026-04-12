# Task Tree V2

本版本（V2）是统一 `task-tree` 仓库中的一部分，默认运行在：

- 前端工作台：`http://127.0.0.1:8880`
- HTTP API：`http://127.0.0.1:8880/v1/...`
- HTTP MCP：`http://127.0.0.1:8880/mcp`
- 前端开发服务器：`http://127.0.0.1:5174/`

同一个后端 `serve` 进程同时提供：

- 已构建前端静态资源
- HTTP API
- HTTP MCP

开发态下前端可单独跑 `5174`；发布态下浏览器只访问后端 `8880`。

## 项目结构

```text
task-tree/v2/           # 统一仓库中的 V2 版本
  backend/              # Go 后端（模块根）
    cmd/task-tree-service/
    internal/tasktree/
    migrations/
    docs/
    data/               # 运行时 SQLite 数据
  frontend/             # Vue 3 前端
    src/
    dist/               # npm run build 产物
  docs/                 # 项目级使用文档与变更记录
  skill/                # V2 技能文档
  task-tree-dxt/        # V2 DXT 代理与打包文件
  启动后端.bat
  启动前端_前端开发测试服务.bat
```

## 运行方式

### 本地开发

```powershell
# 后端
cd backend
$env:TTS_ADDR="127.0.0.1:8880"
go run ./cmd/task-tree-service serve

# 前端
cd frontend
npm run dev
```

### 本地发布态

```powershell
cd frontend
npm run build

cd ..\backend
$env:TTS_ADDR="127.0.0.1:8880"
go run ./cmd/task-tree-service serve
```

然后浏览器访问 `http://127.0.0.1:8880`。

不要直接双击 `frontend/dist/index.html`，这不是纯静态离线页；它依赖同源的 `/v1/...` 和 `/mcp`。

## 协作原则

1. 本仓库所有文档、脚本、代理都应使用以下端口：
   - 后端 / MCP：`8880`
   - 前端 dev：`5174`
2. 新增核心业务能力时，必须明确写清：
   - 是否提供 HTTP API
   - 是否提供 MCP 工具
   - 如果暂时只提供 HTTP，要在文档里明确标注”HTTP only”
3. 后端改动后运行 `go test ./...`，前端改动后运行 `npm run build`。
4. 改动关键路径、端口、接口、DXT、技能、导航或使用方式时，顺手同步更新相关说明文件，保证文档始终和最新状态一致。
5. **技能文件修改或新增 API/MCP 时，必须同步到全局技能文件夹：**
   - 修改或新增 `skill/SKILL.md` 内容后，同时更新：
     - `C:\Users\Administrator\.claude\skills\task-tree\SKILL.md`（Claude Code 全局）
     - `C:\Users\Administrator\.codex\skills\task-tree\SKILL.md`（Codex 全局）
   - 新增 HTTP 接口时在文档中说明，同步更新全局技能的”完整 HTTP API”部分
   - 新增 MCP 工具时，同步更新全局技能的”完整工具清单”和”工具速查表”部分

## V2 当前能力边界

### 已同时提供 HTTP + MCP 的核心能力

- 项目：创建、读取、列表、更新、删除、概览
- 任务：创建、读取、列表、更新、回收站、状态流转
- 阶段：列出、创建、激活
- 节点：创建、读取、列表、摘要读取、focus 读取、更新、排序、移动、进度、完成、阻塞、claim、release、retype、状态流转
- 上下文：remaining、resume、resume-context、events、search、work-items
- 产物：列表、链接型产物、base64 上传

### 当前以 HTTP 为主的能力

- Run 执行层：开始 run、结束 run、追加 run 日志、读取 run、列出节点 runs
- Memory / 读模型增强：任务、阶段、节点 memory 读写；节点 context；overview 衍生字段

这部分已经有 HTTP 路由和前端接入，但当前 MCP 还没有完全对齐，文档必须如实说明。

## DXT / Skill 对应

- Skill：`skill/SKILL.md`
- DXT：`task-tree-dxt/`
- DXT 代理必须指向：`http://127.0.0.1:8880/mcp`

如果安装 Claude Desktop 扩展，应使用 `task-tree-dxt/task-tree.dxt`。

## 推荐文档

- 项目总览：`docs/README.md`
- 技能与 MCP：`docs/技能与MCP说明.md`
- 本地启动与发布：`docs/本地启动与使用.md`
- HTTP / MCP 边界：`backend/docs/http-mcp-parity.md`
