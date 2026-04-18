# Memory

## 规则

- `execution_log` 已废弃，系统从 `progress(log_content=...)` 自动聚合。
- 只写结构化字段，不手写长日志。
- `preset=full` 才会看到完整聚合日志。

## 必写字段

- `summary_text`：做了什么 + 量化结果，不能只写“已完成”。
- `decisions`：关键决策及理由。
- `evidence`：文件路径、命令输出、验证结果。
- `next_actions`：下一步，方便后续 agent 接手。
- `blockers`：当前阻塞项。
- `risks`：已知风险和注意事项。

## 记录方式

```json
{
  "summary_text": "删除 5 个废弃文件，更新 2 处引用，npm run build 通过",
  "decisions": ["保留兼容别名 3 个月"],
  "evidence": ["frontend/src/main.js:L12", "npm run build exit 0"],
  "next_actions": ["继续处理详情页拆分"],
  "risks": ["兼容别名到期后需再次清理"]
}
```
