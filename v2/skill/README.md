# skill — Task Tree V2 技能文档

本目录存放面向 AI 编程助手（Claude Code、Codex 等）的技能文档。

## 文件说明

| 文件 | 用途 | 加载时机 |
|------|------|---------|
| [SKILL.md](SKILL.md) | 核心技能文档：连接信息、数据模型、工作流、行为规则、工具速查表 | **每次对话自动加载** |
| [docs/task-tree-tools.md](docs/task-tree-tools.md) | 完整 MCP 工具清单：场景速查索引、全部参数说明、通用参数模式 | 需要工具细节时按需读取 |
| [docs/task-tree-api.md](docs/task-tree-api.md) | 完整 HTTP API 参考：端点列表、请求/响应示例、错误码、分页说明 | 需要 HTTP 调用细节时按需读取 |
| [docs/task-tree-best-practices.md](docs/task-tree-best-practices.md) | 最佳实践：任务拆解、执行流程、性能优化、并发控制、故障恢复 | 需要决策参考时按需读取 |

## 使用方式

AI 每次对话只加载 `SKILL.md`（约 200 行）。当需要以下细节时，按需 `Read` 对应文档：

- **不知道该用哪个工具** → 读 `docs/task-tree-tools.md` 的场景速查索引
- **需要 HTTP 调用细节** → 读 `docs/task-tree-api.md`
- **不确定怎么拆解任务或处理故障** → 读 `docs/task-tree-best-practices.md`

## 同步约定

修改本目录下任何文件后，必须同步到全局技能目录：

- `C:\Users\Administrator\.claude\skills\task-tree\`
- `C:\Users\Administrator\.codex\skills\task-tree\`
