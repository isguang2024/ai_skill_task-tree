<template>
  <n-spin :show="loading">
    <n-card size="small" style="margin-bottom:12px;">
      <n-input-group>
        <n-input v-model:value="query" placeholder="搜索任务或节点…" clearable style="flex:1;"
          @keydown.enter="doSearch" />
        <n-select v-model:value="kind" :options="kindOpts" style="width:120px;" />
        <n-button type="primary" @click="doSearch" :loading="loading">搜索</n-button>
      </n-input-group>
    </n-card>

    <n-empty v-if="searched && !loading && results.tasks.length===0 && results.nodes.length===0"
      description="没有找到匹配的结果" style="padding:60px 0;" />

    <!-- Task Results -->
    <div v-if="results.tasks.length > 0">
      <n-h6 prefix="bar" style="margin-bottom:8px;">
        任务 ({{ results.tasks.length }})
      </n-h6>
      <div v-for="task in results.tasks" :key="task.id" style="margin-bottom:8px;">
        <n-card hoverable size="small" style="cursor:pointer;" @click="goTask(task.id)">
          <template #header>
            <n-space align="center" :size="6">
              <n-tag :type="statusType(task.status)" size="small" :bordered="false">{{ statusLabel(task.status) }}</n-tag>
              <n-tag v-if="task.task_key" size="small">{{ task.task_key }}</n-tag>
            </n-space>
          </template>
          <template #header-extra>
            <n-text depth="3" style="font-size:12px;">{{ shortTime(task.updated_at) }}</n-text>
          </template>
          <div style="font-size:15px;font-weight:600;margin-bottom:4px;">{{ task.title }}</div>
          <n-text v-if="task.goal" depth="3" style="font-size:13px;">
            {{ excerpt(task.goal, 120) }}
          </n-text>
        </n-card>
      </div>
    </div>

    <!-- Node Results -->
    <div v-if="results.nodes.length > 0" style="margin-top:16px;">
      <n-h6 prefix="bar" style="margin-bottom:8px;">
        节点 ({{ results.nodes.length }})
      </n-h6>
      <div v-for="node in results.nodes" :key="node.id" style="margin-bottom:8px;">
        <n-card hoverable size="small" style="cursor:pointer;" @click="goNode(node)">
          <template #header>
            <n-space align="center" :size="6">
              <n-tag :type="statusType(node.status)" size="small" :bordered="false">{{ statusLabel(node.status) }}</n-tag>
              <n-text depth="3" style="font-size:12px;">{{ node.path }}</n-text>
            </n-space>
          </template>
          <div style="font-size:15px;font-weight:600;margin-bottom:4px;">{{ node.title }}</div>
          <n-tag size="small" round type="info">{{ node.task_title || '未知任务' }}</n-tag>
        </n-card>
      </div>
    </div>
  </n-spin>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { api, statusType, statusLabel, shortTime, excerpt } from '../api.js'

const router = useRouter()
const route = useRoute()
const query = ref('')
const kind = ref('all')
const loading = ref(false)
const searched = ref(false)
const results = ref({ tasks: [], nodes: [] })

const kindOpts = [
  { label: '全部', value: 'all' },
  { label: '仅任务', value: 'tasks' },
  { label: '仅节点', value: 'nodes' },
]

async function doSearch() {
  const q = query.value.trim()
  if (!q) return
  loading.value = true
  searched.value = true
  try {
    const params = new URLSearchParams({ q, kind: kind.value, limit: '40' })
    const res = await api('/search?' + params)
    results.value = {
      tasks: res.tasks || [],
      nodes: res.nodes || [],
    }
  } catch (e) {
    window.$message?.error('搜索失败: ' + e.message)
  } finally {
    loading.value = false
  }
}

function goTask(id) { router.push('/tasks/' + id) }
function goNode(node) { router.push('/tasks/' + node.task_id + '?node=' + node.id) }

onMounted(() => {
  const q = route.query.q
  if (q) {
    query.value = q
    doSearch()
  }
})
</script>
