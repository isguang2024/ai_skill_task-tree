<template>
  <n-spin :show="loading">
    <n-card size="small" style="margin-bottom:12px;">
      <n-space align="center" justify="space-between">
        <n-space align="center">
          <n-input v-model:value="searchQuery" placeholder="搜索已删除任务" clearable style="width:240px;"
            @update:value="debouncedLoad" />
          <n-tag type="info" size="small">共 {{ tasks.length }} 项</n-tag>
        </n-space>
        <n-popconfirm @positive-click="emptyTrash">
          <template #trigger>
            <n-button type="error" size="small" :disabled="tasks.length===0">清空回收站</n-button>
          </template>
          确认彻底删除回收站中的所有任务？此操作不可恢复！
        </n-popconfirm>
      </n-space>
    </n-card>

    <n-empty v-if="!loading && tasks.length===0" description="回收站为空" style="padding:60px 0;" />

    <div v-for="task in tasks" :key="task.id" style="margin-bottom:8px;">
      <n-card size="small">
        <template #header>
          <n-space align="center" :size="6">
            <n-tag type="error" size="small" :bordered="false">已删除</n-tag>
            <n-tag :type="statusType(task.status)" size="small">{{ statusLabel(task.status) }}</n-tag>
            <n-tag v-if="task.task_key" size="small">{{ task.task_key }}</n-tag>
          </n-space>
        </template>
        <template #header-extra>
          <n-space :size="4">
            <n-popconfirm @positive-click="restoreTask(task.id)">
              <template #trigger>
                <n-button size="small" type="info" secondary>恢复</n-button>
              </template>
              确认恢复该任务？
            </n-popconfirm>
            <n-popconfirm @positive-click="hardDelete(task.id)">
              <template #trigger>
                <n-button size="small" type="error" secondary>彻底删除</n-button>
              </template>
              确认彻底删除该任务？此操作不可恢复！
            </n-popconfirm>
          </n-space>
        </template>
        <div style="font-size:15px;font-weight:600;margin-bottom:4px;">{{ task.title }}</div>
        <n-text v-if="task.goal" depth="3" style="font-size:13px;">
          {{ excerpt(task.goal, 150) }}
        </n-text>
      </n-card>
    </div>
  </n-spin>
</template>

<script setup>
import { ref, onMounted, onActivated, onUnmounted } from 'vue'
import { api, statusType, statusLabel, excerpt } from '../api.js'

const tasks = ref([])
const loading = ref(true)
const searchQuery = ref('')

let debounceTimer = null
function debouncedLoad() {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(loadTrash, 300)
}

async function loadTrash() {
  loading.value = true
  try {
    const params = new URLSearchParams()
    params.set('include_deleted', 'true')
    if (searchQuery.value) params.set('q', searchQuery.value)
    params.set('limit', '80')
    const list = await api('/tasks?' + params)
    // Only show deleted tasks (those with deleted_at)
    tasks.value = list.filter(t => t.deleted_at)
  } catch (e) {
    window.$message?.error('加载失败: ' + e.message)
  } finally {
    loading.value = false
  }
}

async function restoreTask(id) {
  try {
    await api('/tasks/' + id + '/restore', { method: 'POST' })
    window.$message?.success('任务已恢复')
    loadTrash()
  } catch (e) {
    window.$message?.error('恢复失败: ' + e.message)
  }
}

async function hardDelete(id) {
  try {
    await api('/tasks/' + id + '/hard', { method: 'DELETE' })
    window.$message?.success('已彻底删除')
    loadTrash()
  } catch (e) {
    window.$message?.error('删除失败: ' + e.message)
  }
}

async function emptyTrash() {
  try {
    const res = await api('/admin/empty-trash', { method: 'POST' })
    window.$message?.success(`已清空 ${res.deleted || 0} 个任务`)
    loadTrash()
  } catch (e) {
    window.$message?.error('清空失败: ' + e.message)
  }
}

onMounted(loadTrash)
onActivated(loadTrash)
onUnmounted(() => {
  if (debounceTimer) {
    clearTimeout(debounceTimer)
    debounceTimer = null
  }
})
</script>
