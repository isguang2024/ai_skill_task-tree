<template>
  <n-spin :show="loading">

    <!-- Project view -->
    <template v-if="!selectedProject">
      <!-- Filter bar -->
      <n-card size="small" style="margin-bottom:12px;">
        <n-space align="center" justify="space-between">
          <n-space align="center">
          <n-input v-model:value="searchQuery" placeholder="搜索项目" clearable style="width:220px;"
            @update:value="debouncedLoad" />
          <n-tag type="info" size="small">{{ projects.length }} 个项目</n-tag>
          </n-space>
          <n-space align="center" :size="4">
            <n-button size="small" type="primary" @click="openCreateProject">新建项目</n-button>
          </n-space>
        </n-space>
      </n-card>

      <n-empty v-if="!loading && projects.length===0" description="暂无项目" style="padding:60px 0;" />

      <n-grid :cols="3" :x-gap="12" :y-gap="12">
        <n-gi v-for="proj in projects" :key="proj.id">
          <n-card hoverable size="small" style="cursor:pointer;height:100%;" @click="openProject(proj)">
            <template #header>
              <n-space align="center" :size="6">
                <span style="font-size:18px;">📁</span>
                <n-tag v-if="proj.project_key" size="small">{{ proj.project_key }}</n-tag>
                <n-tag v-if="proj.is_default" size="small" type="info">默认</n-tag>
              </n-space>
            </template>
            <template #header-extra>
              <n-space :size="4" @click.stop>
                <n-button size="tiny" quaternary @click.stop="startEdit(proj)">编辑</n-button>
                <n-button size="tiny" quaternary type="error" @click.stop="deleteProject(proj)">删除</n-button>
                <n-text depth="3" style="font-size:11px;">{{ shortTime(proj.updated_at) }}</n-text>
              </n-space>
            </template>
            <div style="font-size:15px;font-weight:600;margin-bottom:4px;">{{ proj.name }}</div>
            <n-text v-if="proj.description" depth="3" style="font-size:12px;display:block;margin-bottom:8px;">
              {{ excerpt(proj.description, 80) }}
            </n-text>
            <!-- Mini stats from overview -->
            <n-space v-if="proj._summary" :size="4" wrap>
              <n-tag size="small" round>总 {{ proj._summary.total }}</n-tag>
              <n-tag v-if="proj._summary.running > 0" type="success" size="small" round>运行 {{ proj._summary.running }}</n-tag>
              <n-tag v-if="proj._summary.blocked > 0" type="error" size="small" round>阻塞 {{ proj._summary.blocked }}</n-tag>
              <n-tag v-if="proj._summary.paused > 0" type="warning" size="small" round>暂停 {{ proj._summary.paused }}</n-tag>
              <n-tag v-if="proj._summary.done > 0" size="small" round>完成 {{ proj._summary.done }}</n-tag>
            </n-space>
          </n-card>
        </n-gi>
      </n-grid>

      <!-- Edit Project Modal -->
      <n-modal v-model:show="showEdit" preset="card" title="编辑项目" style="max-width:480px;" @click.stop>
        <n-form label-placement="top">
          <n-form-item label="项目名称">
            <n-input v-model:value="editForm.name" placeholder="项目名称" />
          </n-form-item>
          <n-form-item label="项目 Key">
            <n-input v-model:value="editForm.project_key" placeholder="例如：PROJ" />
          </n-form-item>
          <n-form-item label="描述">
            <n-input v-model:value="editForm.description" type="textarea" :rows="3" placeholder="项目描述" />
          </n-form-item>
        </n-form>
        <template #action>
          <n-button @click="showEdit=false">取消</n-button>
          <n-button type="primary" :loading="saving" @click="saveEdit">保存</n-button>
        </template>
      </n-modal>

      <!-- Create Project Modal -->
      <n-modal v-model:show="showCreate" preset="card" title="新建项目" style="max-width:480px;" @click.stop>
        <n-form label-placement="top">
          <n-form-item label="项目名称" required>
            <n-input v-model:value="createForm.name" placeholder="例如：AI 任务重构" />
          </n-form-item>
          <n-form-item label="项目 Key">
            <n-input v-model:value="createForm.project_key" placeholder="例如：REFACTOR" />
          </n-form-item>
          <n-form-item label="描述">
            <n-input v-model:value="createForm.description" type="textarea" :rows="3" placeholder="项目描述" />
          </n-form-item>
          <n-form-item label="默认项目">
            <n-switch v-model:value="createForm.is_default" />
          </n-form-item>
        </n-form>
        <template #action>
          <n-button @click="showCreate=false">取消</n-button>
          <n-button type="primary" :loading="creating" @click="saveCreate">创建</n-button>
        </template>
      </n-modal>
    </template>

    <!-- Tasks inside a project -->
    <template v-else>
      <!-- Header with back -->
      <n-card size="small" style="margin-bottom:12px;">
        <n-space align="center" justify="space-between">
          <n-space align="center" :size="8">
            <n-button size="small" quaternary @click="closeProject">← 返回项目</n-button>
            <n-divider vertical />
            <span style="font-size:16px;">📁</span>
            <n-text strong>{{ selectedProject.name }}</n-text>
            <n-tag v-if="selectedProject.project_key" size="small">{{ selectedProject.project_key }}</n-tag>
          </n-space>
          <n-space align="center" :size="6" wrap>
            <n-input v-model:value="searchQuery" placeholder="搜索任务" clearable style="width:180px;"
              @update:value="debouncedLoadTasks" />
            <n-select v-model:value="statusFilter" :options="statusOpts" style="width:130px;"
              @update:value="loadProjectTasks" clearable placeholder="全部状态" />
            <n-button size="small" tertiary @click="toggleSelectionMode">
              {{ selectionMode ? '退出多选' : '多选任务' }}
            </n-button>
            <template v-if="selectionMode">
              <n-tag type="warning" size="small">已选 {{ selectedTaskIds.length }} 项</n-tag>
              <n-button size="small" :disabled="tasks.length===0" @click="toggleSelectAllTasks">
                {{ allTasksSelected ? '取消全选' : '全选当前页' }}
              </n-button>
              <n-button
                size="small"
                type="error"
                :disabled="selectedTaskIds.length===0"
                :loading="batchRecycling"
                @click="confirmBatchRecycle"
              >
                批量回收
              </n-button>
            </template>
          </n-space>
        </n-space>
      </n-card>

      <!-- Summary stats -->
      <n-grid v-if="summary" :cols="8" :x-gap="8" :y-gap="8" style="margin-bottom:12px;">
        <n-gi v-for="s in summaryItems" :key="s.label">
          <n-card size="small">
            <n-statistic :label="s.label" :value="s.value" />
          </n-card>
        </n-gi>
      </n-grid>

      <n-empty v-if="!loading && tasks.length===0" description="暂无任务" style="padding:40px 0;" />

      <!-- Task grid 3 per row -->
      <n-grid :cols="3" :x-gap="12" :y-gap="12">
        <n-gi v-for="task in tasks" :key="task.id">
          <n-card hoverable size="small" @click="handleTaskCardClick(task)" :style="taskCardStyle(task)">
            <template #header>
              <n-space align="center" :size="6">
                <n-tag :type="statusType(task.status)" size="small" :bordered="false">{{ stateLabel(task.status, task.result) }}</n-tag>
                <n-tag size="small">{{ task.task_key || task.id.substring(0,8) }}</n-tag>
                <n-tag type="info" size="small">{{ pct(task.summary_percent) }}%</n-tag>
              </n-space>
            </template>
            <template #header-extra>
              <n-space align="center" :size="6" @click.stop>
                <n-tag v-if="selectionMode && isTaskSelected(task.id)" type="primary" size="small" round>已选</n-tag>
                <n-checkbox
                  v-if="selectionMode"
                  :checked="isTaskSelected(task.id)"
                  @update:checked="setTaskSelection(task.id, $event)"
                  @click.stop
                />
                <n-text depth="3" style="font-size:11px;">{{ shortTime(task.updated_at) }}</n-text>
              </n-space>
            </template>
            <div style="font-size:14px;font-weight:600;margin-bottom:4px;">{{ task.title }}</div>
            <n-text v-if="task.goal" depth="3" style="font-size:12px;display:block;margin-bottom:8px;">
              {{ excerpt(task.goal, 100) }}
            </n-text>
            <n-progress :percentage="pct(task.summary_percent)" :show-indicator="false" :height="5"
              :border-radius="3" style="margin-bottom:6px;" />
            <n-space justify="space-between" align="center">
              <n-space :size="4">
                <n-tag size="small" round>剩余 {{ task._remaining || 0 }}</n-tag>
                <n-tag v-if="task._blocked > 0" type="error" size="small" round>阻塞 {{ task._blocked }}</n-tag>
                <n-tag v-if="task._paused > 0" type="warning" size="small" round>暂停 {{ task._paused }}</n-tag>
              </n-space>
            </n-space>
            <div v-if="task.current_stage?.title" style="font-size:11px;color:var(--n-text-color-3);margin-top:4px;">
              当前阶段: <strong>{{ task.current_stage.path || task.current_stage.title }}</strong>
            </div>
            <div v-if="task._summary" style="font-size:11px;color:var(--n-text-color-2);margin-top:4px;line-height:1.6;">
              {{ excerpt(task._summary, 72) }}
            </div>
            <div v-if="task._nextTitle" style="font-size:11px;color:var(--n-text-color-3);margin-top:4px;">
              下一步: <strong>{{ task._nextPath }}</strong> {{ task._nextTitle }}
            </div>
          </n-card>
        </n-gi>
      </n-grid>
    </template>

  </n-spin>
</template>

<script setup>
import { ref, computed, onMounted, onActivated, onDeactivated, onUnmounted, watch, inject } from 'vue'
import { useRouter } from 'vue-router'
import { api, statusType, stateLabel, pct, shortTime, excerpt, fetchProjectOverview } from '../api.js'

const props = defineProps({ projectId: String })
const router = useRouter()
const breadcrumb = inject('breadcrumb', ref([]))

// Project view state
const projects = ref([])
const selectedProject = ref(null)

// Task view state
const tasks = ref([])
const summary = ref(null)
const loading = ref(true)
const searchQuery = ref('')
const statusFilter = ref(null)
const selectionMode = ref(false)
const selectedTaskIds = ref([])
const batchRecycling = ref(false)

const statusOpts = [
  { label: '就绪', value: 'ready' },
  { label: '进行中', value: 'running' },
  { label: '阻塞', value: 'blocked' },
  { label: '暂停', value: 'paused' },
  { label: '完成', value: 'done' },
  { label: '已取消', value: 'canceled' },
  { label: '已关闭', value: 'closed' },
]

const summaryItems = computed(() => {
  if (!summary.value) return []
  const s = summary.value
  return [
    { label: '总数', value: s.total },
    { label: '就绪', value: s.ready },
    { label: '进行中', value: s.running },
    { label: '阻塞', value: s.blocked },
    { label: '暂停', value: s.paused },
    { label: '完成', value: s.done },
    { label: '取消', value: s.canceled },
    { label: '关闭', value: s.closed },
  ]
})

const allTasksSelected = computed(() => tasks.value.length > 0 && selectedTaskIds.value.length === tasks.value.length)

// Load all projects with mini stats
async function loadProjects() {
  loading.value = true
  breadcrumb.value = [{ label: '任务总览', path: '/' }]
  try {
    const q = new URLSearchParams()
    if (searchQuery.value) q.set('q', searchQuery.value)
    q.set('view_mode', 'summary_with_stats')
    projects.value = await api('/projects?' + q)
  } catch (e) {
    window.$message?.error('加载项目失败: ' + e.message)
  } finally {
    loading.value = false
  }
}

let debounceTimer = null
function debouncedLoad() {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(loadProjects, 300)
}
function debouncedLoadTasks() {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(loadProjectTasks, 300)
}

async function openProject(proj) {
  selectedProject.value = proj
  searchQuery.value = ''
  statusFilter.value = null
  resetTaskSelection({ exitMode: true })
  router.push('/projects/' + proj.id)
  // loadProjectTasks will be triggered by watcher on projectId prop
}

function closeProject() {
  selectedProject.value = null
  searchQuery.value = ''
  statusFilter.value = null
  resetTaskSelection({ exitMode: true })
  router.push('/')
}

async function loadProjectTasks() {
  if (!selectedProject.value) return
  loading.value = true
  try {
    const overview = await fetchProjectOverview(selectedProject.value.id)
    const list = (overview.tasks || []).filter(t => {
      if (statusFilter.value && t.status !== statusFilter.value) return false
      if (searchQuery.value) {
        const q = searchQuery.value.toLowerCase()
        const text = [t.title, t.task_key, t.memory?.summary, t.current_stage?.title, t.current_stage?.path].filter(Boolean).join(' ').toLowerCase()
        return text.includes(q)
      }
      return true
    })
    tasks.value = list.map(normalizeOverviewTask)
    summary.value = overview.summary || null
    breadcrumb.value = [
      { label: '任务总览', path: '/' },
      { label: selectedProject.value.name },
    ]
  } catch (e) {
    window.$message?.error('加载任务失败: ' + e.message)
  } finally {
    loading.value = false
  }
}

function goTask(task) {
  router.push({
    path: '/tasks/' + task.id,
    state: {
      projectId: selectedProject.value?.id,
      projectName: selectedProject.value?.name,
      backPath: selectedProject.value?.id ? '/projects/' + selectedProject.value.id : '/',
    },
  })
}

function resetTaskSelection(options = {}) {
  selectedTaskIds.value = []
  if (options.exitMode) selectionMode.value = false
}

function isTaskSelected(taskId) {
  return selectedTaskIds.value.includes(taskId)
}

function setTaskSelection(taskId, checked) {
  if (!selectionMode.value) return
  if (checked) {
    if (!isTaskSelected(taskId)) selectedTaskIds.value = [...selectedTaskIds.value, taskId]
    return
  }
  selectedTaskIds.value = selectedTaskIds.value.filter(id => id !== taskId)
}

function toggleTaskSelection(taskId) {
  setTaskSelection(taskId, !isTaskSelected(taskId))
}

function toggleSelectionMode() {
  if (selectionMode.value) {
    resetTaskSelection({ exitMode: true })
    return
  }
  selectionMode.value = true
}

function toggleSelectAllTasks() {
  if (allTasksSelected.value) {
    selectedTaskIds.value = []
    return
  }
  selectedTaskIds.value = tasks.value.map(task => task.id)
}

function handleTaskCardClick(task) {
  if (selectionMode.value) {
    toggleTaskSelection(task.id)
    return
  }
  goTask(task)
}

function taskCardStyle(task) {
  const selected = selectionMode.value && isTaskSelected(task.id)
  return {
    cursor: 'pointer',
    height: '100%',
    borderColor: selected ? 'var(--n-primary-color)' : '',
    boxShadow: selected ? '0 0 0 1px var(--n-primary-color)' : '',
    background: selected ? 'rgba(24, 160, 88, 0.06)' : '',
  }
}

function confirmBatchRecycle() {
  if (!selectedTaskIds.value.length || batchRecycling.value) return
  const count = selectedTaskIds.value.length
  window.$dialog?.warning({
    title: '批量回收任务',
    content: count === 1
      ? '选中的任务将移入回收站，可在回收站恢复。确定继续？'
      : `选中的 ${count} 个任务将移入回收站，可在回收站恢复。确定继续？`,
    positiveText: '确认回收',
    negativeText: '取消',
    onPositiveClick: runBatchRecycle,
  })
}

async function runBatchRecycle() {
  if (!selectedTaskIds.value.length || batchRecycling.value) return
  const ids = [...selectedTaskIds.value]
  batchRecycling.value = true
  const failedIds = []
  try {
    for (const id of ids) {
      try {
        await api('/tasks/' + id, { method: 'DELETE' })
      } catch {
        failedIds.push(id)
      }
    }
    const successCount = ids.length - failedIds.length
    if (successCount > 0) {
      window.$message?.success(
        failedIds.length === 0
          ? `已回收 ${successCount} 个任务`
          : `已回收 ${successCount} 个任务，${failedIds.length} 个失败`
      )
    }
    if (failedIds.length > 0) {
      window.$message?.error('部分任务回收失败，请重试')
    }
    await loadProjectTasks()
    selectionMode.value = failedIds.length > 0
    selectedTaskIds.value = failedIds.filter(id => tasks.value.some(task => task.id === id))
  } finally {
    batchRecycling.value = false
  }
}

// Edit / Delete project
const showEdit = ref(false)
const showCreate = ref(false)
const saving = ref(false)
const creating = ref(false)
const editTarget = ref(null)
const editForm = ref({ name: '', project_key: '', description: '' })
const createForm = ref({ name: '', project_key: '', description: '', is_default: false })

function openCreateProject() {
  createForm.value = { name: '', project_key: '', description: '', is_default: false }
  showCreate.value = true
}

function startEdit(proj) {
  editTarget.value = proj
  editForm.value = { name: proj.name || '', project_key: proj.project_key || '', description: proj.description || '' }
  showEdit.value = true
}

async function saveEdit() {
  if (!editTarget.value || !editForm.value.name.trim()) return
  saving.value = true
  try {
    await api('/projects/' + editTarget.value.id, {
      method: 'PATCH',
      body: JSON.stringify({ name: editForm.value.name, project_key: editForm.value.project_key || null, description: editForm.value.description || null }),
    })
    showEdit.value = false
    window.$message?.success('已保存')
    loadProjects()
  } catch (e) {
    window.$message?.error('保存失败: ' + e.message)
  } finally {
    saving.value = false
  }
}

async function saveCreate() {
  if (!createForm.value.name.trim()) return
  creating.value = true
  try {
    await api('/projects', {
      method: 'POST',
      body: JSON.stringify({
        name: createForm.value.name,
        project_key: createForm.value.project_key || null,
        description: createForm.value.description || null,
        is_default: createForm.value.is_default,
      }),
    })
    showCreate.value = false
    window.$message?.success('项目已创建')
    loadProjects()
  } catch (e) {
    window.$message?.error('创建失败: ' + e.message)
  } finally {
    creating.value = false
  }
}

function deleteProject(proj) {
  const total = proj._summary?.total || 0
  const hasTask = total > 0
  window.$dialog?.error({
    title: '删除项目',
    content: hasTask
      ? `项目「${proj.name}」包含 ${total} 个任务，删除后这些任务将全部移入回收站，可从回收站恢复。确定继续？`
      : `确定删除项目「${proj.name}」？`,
    positiveText: '确认删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await api('/projects/' + proj.id, { method: 'DELETE' })
        window.$message?.success('已删除')
        loadProjects()
      } catch (e) {
        window.$message?.error('删除失败: ' + e.message)
      }
    },
  })
}

// Light background refresh for task status/percent
let refreshTimer = null
async function lightRefresh() {
  if (loading.value || !selectedProject.value) return
  try {
    const ov = await api('/projects/' + selectedProject.value.id + '/overview')
    if (!ov.tasks) return
    tasks.value = tasks.value.map(t => {
      const fresh = ov.tasks.find(f => f.id === t.id)
      return fresh ? normalizeOverviewTask(fresh) : t
    })
    if (ov.summary) summary.value = ov.summary
  } catch {}
}

function normalizeOverviewTask(task) {
  return {
    ...task,
    _remaining: task.remaining?.nodes || 0,
    _blocked: task.remaining?.blocked || 0,
    _paused: task.remaining?.paused || 0,
    _nextPath: task.current_stage?.path || '',
    _nextTitle: task.memory?.next_actions?.[0] || '',
    _summary: task.memory?.summary || '',
  }
}

function startAutoRefresh() {
  stopAutoRefresh()
  refreshTimer = setInterval(lightRefresh, 8000)
}
function stopAutoRefresh() {
  if (refreshTimer) { clearInterval(refreshTimer); refreshTimer = null }
}

async function initByProp() {
  if (props.projectId) {
    try {
      loading.value = true
      // Find project in already-loaded list, else fetch full list
      let proj = projects.value.find(p => p.id === props.projectId)
      if (!proj) {
        const list = await api('/projects')
        projects.value = list
        proj = list.find(p => p.id === props.projectId)
      }
      if (!proj) { router.replace('/'); return }
      selectedProject.value = proj
      searchQuery.value = ''
      statusFilter.value = null
      resetTaskSelection({ exitMode: true })
      await loadProjectTasks()
    } catch {
      router.replace('/')
    }
  } else {
    selectedProject.value = null
    resetTaskSelection({ exitMode: true })
    loadProjects()
  }
}

watch(() => props.projectId, initByProp)
watch(tasks, currentTasks => {
  const validIds = new Set(currentTasks.map(task => task.id))
  selectedTaskIds.value = selectedTaskIds.value.filter(id => validIds.has(id))
  if (selectionMode.value && currentTasks.length === 0) {
    resetTaskSelection({ exitMode: true })
  }
})

onMounted(() => { initByProp(); startAutoRefresh() })
onActivated(() => { initByProp(); startAutoRefresh() })
onDeactivated(stopAutoRefresh)
onUnmounted(() => {
  stopAutoRefresh()
  if (debounceTimer) {
    clearTimeout(debounceTimer)
    debounceTimer = null
  }
})
</script>
