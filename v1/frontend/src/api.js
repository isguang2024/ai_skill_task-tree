const API = '/v1'

export async function api(path, opts = {}) {
  const res = await fetch(API + path, {
    headers: { 'Content-Type': 'application/json', ...opts.headers },
    ...opts,
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ detail: res.statusText }))
    throw new Error(err.detail || res.statusText)
  }
  return res.json()
}

export function statusType(s) {
  switch ((s || '').toLowerCase()) {
    case 'done': return 'success'
    case 'running': return 'info'
    case 'blocked': case 'canceled': return 'error'
    case 'paused': return 'warning'
    default: return 'default'
  }
}

export function statusLabel(s) {
  const m = { ready: '就绪', running: '进行中', blocked: '阻塞', paused: '暂停', done: '完成', canceled: '已取消', closed: '已关闭' }
  return m[(s || '').toLowerCase()] || s || '未知'
}

export function stateLabel(status, result) {
  if ((result || '').toLowerCase() === 'mixed') return '混合关闭'
  return statusLabel(status)
}

export function pct(v) {
  let n = Math.round(v > 1 ? v : v * 100)
  return Math.max(0, Math.min(100, n))
}

export function shortTime(v) {
  if (!v) return ''
  let s = (v || '').replace('T', ' ').replace('Z', '')
  if (s.length >= 16) s = s.substring(0, 16)
  return s
}

export function excerpt(v, limit) {
  if (!v) return ''
  v = v.replace(/\n/g, ' ').replace(/\s+/g, ' ').trim()
  return v.length <= limit ? v : v.substring(0, limit) + '…'
}

export function eventTypeLabel(t) {
  const m = {
    task_created: '任务创建', task_updated: '任务更新', task_deleted: '任务删除',
    task_restored: '任务恢复', task_paused: '任务暂停', task_reopened: '任务恢复',
    task_canceled: '任务取消', node_created: '节点创建', node_retyped: '节点转型',
    node_updated: '节点更新', progress: '进度更新', complete: '完成',
    claim: '已领取', release: '已释放', blocked: '已阻塞', unblocked: '解除阻塞',
    paused: '暂停', reopened: '重开', canceled: '已取消', artifact: '产物', lease_sweep: '租约清扫',
  }
  return m[(t || '').toLowerCase()] || t || '事件'
}

export function leaseActive(node) {
  if (!node || !node.lease_until) return false
  try { return new Date(node.lease_until) > new Date() } catch { return false }
}

export function nodePct(node) {
  return pct(node.progress || 0)
}

function compareSiblings(a, b) {
  const so = (a.sort_order ?? Infinity) - (b.sort_order ?? Infinity)
  if (so !== 0) return so
  return (a.created_at || '').localeCompare(b.created_at || '')
}

export function buildTreeData(nodes) {
  const sorted = [...nodes].sort(compareSiblings)
  const byParent = {}
  const childCounts = {}
  for (const n of sorted) {
    const pid = n.parent_node_id || ''
    if (!byParent[pid]) byParent[pid] = []
    byParent[pid].push(n)
    if (pid) childCounts[pid] = (childCounts[pid] || 0) + 1
  }
  function walk(parentId) {
    const children = byParent[parentId] || []
    return children.map(n => {
      const hasChildren = (childCounts[n.id] || 0) > 0
      const item = {
        key: n.id,
        label: n.title,
        raw: { ...n, hasChildren, childCount: childCounts[n.id] || 0 },
      }
      if (hasChildren) {
        item.children = walk(n.id)
      } else {
        item.isLeaf = true
      }
      return item
    })
  }
  return walk('')
}

export const nodeTemplates = {
  dev: {
    instruction: '实现功能并补必要测试，标注改动文件与核心函数。',
    acceptance: ['功能按预期可用', '关键路径有测试或手工验证', '无明显回归风险'],
  },
  test: {
    instruction: '编写/补充测试用例，覆盖主流程与边界场景。',
    acceptance: ['测试可稳定通过', '覆盖主流程和异常流程', '报告包含失败与修复结论'],
  },
  accept: {
    instruction: '执行验收与回归检查，输出可交付结论。',
    acceptance: ['验收标准逐条确认', '关键截图/日志可追溯', '明确遗留项和下步建议'],
  },
}
