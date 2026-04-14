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

export function buildQuery(params = {}) {
  const query = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === '') continue
    if (Array.isArray(value)) {
      if (value.length === 0) continue
      query.set(key, value.join(','))
      continue
    }
    query.set(key, String(value))
  }
  return query.toString()
}

export function normalizeNode(node) {
  if (!node) return null
  return {
    ...node,
    id: node.id || node.node_id || '',
    node_id: node.node_id || node.id || '',
    parent_node_id: node.parent_node_id || '',
    stage_node_id: node.stage_node_id || '',
  }
}

export function normalizeNodeList(nodes = []) {
  return (nodes || []).map(normalizeNode).filter(Boolean)
}

export function fetchTaskResume(taskId, params = {}) {
  const query = buildQuery(params)
  return api('/tasks/' + taskId + '/resume' + (query ? '?' + query : ''))
}

export function fetchNodeContext(nodeId, params = {}) {
  const query = buildQuery(params)
  return api('/nodes/' + nodeId + '/context' + (query ? '?' + query : ''))
}

export function fetchProjectOverview(projectId, params = {}) {
  const query = buildQuery(params)
  return api('/projects/' + projectId + '/overview' + (query ? '?' + query : ''))
}

export function fetchTaskMemory(taskId) {
  return api('/tasks/' + taskId + '/memory')
}

export function patchTaskMemory(taskId, body) {
  return api('/tasks/' + taskId + '/memory', { method: 'PATCH', body: JSON.stringify(body) })
}

export function fetchStageMemory(stageId) {
  return api('/stages/' + stageId + '/memory')
}

export function patchStageMemory(stageId, body) {
  return api('/stages/' + stageId + '/memory', { method: 'PATCH', body: JSON.stringify(body) })
}

export function fetchNodeMemory(nodeId, level = 'structured') {
  const q = level ? '?level=' + level : ''
  return api('/nodes/' + nodeId + '/memory' + q)
}

// Fetch only execution_log field (lazy load for long content)
export function fetchNodeExecutionLog(nodeId) {
  return api('/nodes/' + nodeId + '/memory?level=full').then(m => m?.execution_log || '')
}

export function patchNodeMemory(nodeId, body) {
  return api('/nodes/' + nodeId + '/memory', { method: 'PATCH', body: JSON.stringify(body) })
}

export function listNodeDetails(taskId, limit = 10000) {
  return api('/tasks/' + taskId + '/nodes?' + buildQuery({ limit, view_mode: 'detail' }))
}

export const listAllNodes = listNodeDetails

export function listEvents(params = {}) {
  const query = buildQuery(params)
  return api('/events' + (query ? '?' + query : '')).then(res => res.items || res || [])
}

export function listTaskArtifacts(taskId, params = {}) {
  const query = buildQuery(params)
  return api('/tasks/' + taskId + '/artifacts' + (query ? '?' + query : '')).then(res => res.items || res || [])
}

export function listStages(taskId) {
  return api('/tasks/' + taskId + '/stages')
}

export function createStage(taskId, body) {
  return api('/tasks/' + taskId + '/stages', { method: 'POST', body: JSON.stringify(body) })
}

export function activateStage(taskId, stageNodeId) {
  return api('/tasks/' + taskId + '/stages/' + stageNodeId + '/activate', { method: 'POST', body: JSON.stringify({}) })
}

export function listNodeRuns(nodeId, limit = 20) {
  return api('/nodes/' + nodeId + '/runs?' + buildQuery({ limit, view_mode: 'summary' })).then(res => res.items || res || [])
}

export function startNodeRun(nodeId, body = {}) {
  return api('/nodes/' + nodeId + '/runs', { method: 'POST', body: JSON.stringify(body) })
}

export function fetchRun(runId, params = {}) {
  const query = buildQuery(params)
  return api('/runs/' + runId + (query ? '?' + query : ''))
}

export function finishRun(runId, body = {}) {
  return api('/runs/' + runId + '/finish', { method: 'POST', body: JSON.stringify(body) })
}

export function addRunLog(runId, body = {}) {
  return api('/runs/' + runId + '/logs', { method: 'POST', body: JSON.stringify(body) })
}

export function createTaskArtifact(taskId, body) {
  return api('/tasks/' + taskId + '/artifacts', { method: 'POST', body: JSON.stringify(body) })
}

export function dirtyTargets(envelope) {
  if (!envelope || typeof envelope !== 'object') return []
  return Array.isArray(envelope.dirty) ? envelope.dirty : []
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
  const raw = Number(v || 0)
  const n = Math.round(raw > 1 ? raw : raw * 100)
  return Math.max(0, Math.min(100, n))
}

export function shortTime(v) {
  if (!v) return ''
  const d = new Date(v)
  if (isNaN(d.getTime())) return v
  const pad = n => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
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
    run_started: '运行开始', run_finished: '运行结束', run_log: '运行日志',
  }
  return m[(t || '').toLowerCase()] || t || '事件'
}

export function leaseActive(node) {
  if (!node || !node.lease_until) return false
  try { return new Date(node.lease_until) > new Date() } catch { return false }
}

export function nodePct(node) {
  return pct(node?.progress || 0)
}

function compareSiblings(a, b) {
  const so = (a.sort_order ?? Infinity) - (b.sort_order ?? Infinity)
  if (so !== 0) return so
  return (a.created_at || '').localeCompare(b.created_at || '')
}

export function buildTreeData(nodes) {
  const normalized = normalizeNodeList(nodes)
  const sorted = [...normalized].sort(compareSiblings)
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
      if (hasChildren) item.children = walk(n.id)
      else item.isLeaf = true
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
