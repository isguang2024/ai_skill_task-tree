<template>
  <n-config-provider :theme="isDark ? darkTheme : null">
    <n-message-provider>
      <n-dialog-provider>
        <n-notification-provider>
          <GlobalApiProvider />
          <n-layout has-sider position="absolute" style="height:100vh;">
            <n-layout-sider bordered :width="200" collapse-mode="width" :collapsed-width="64"
              :collapsed="siderCollapsed" show-trigger @collapse="siderCollapsed=true" @expand="siderCollapsed=false">
              <div style="padding:16px 16px 8px;text-align:center;">
                <n-text strong style="font-size:18px;" v-if="!siderCollapsed">Task Tree</n-text>
                <n-text strong style="font-size:16px;" v-else>TT</n-text>
              </div>
              <n-menu :options="menuOptions" :value="activeMenuKey" @update:value="onMenuSelect" :collapsed="siderCollapsed" />
            </n-layout-sider>
            <n-layout>
              <n-layout-header bordered style="height:52px;padding:0 16px;display:flex;align-items:center;gap:12px;">
                <n-breadcrumb style="font-size:15px;">
                  <n-breadcrumb-item v-for="(item, i) in breadcrumbItems" :key="i"
                    @click="item.path ? router.push(item.path) : undefined"
                    :style="item.path && i < breadcrumbItems.length-1 ? 'cursor:pointer;' : ''">
                    {{ item.label }}
                  </n-breadcrumb-item>
                </n-breadcrumb>
                <div style="flex:1" />
                <n-button size="small" @click="showCreateTask=true" type="primary">新建任务</n-button>
                <n-button size="small" quaternary @click="isDark=!isDark">{{ isDark ? '☀️' : '🌙' }}</n-button>
                <n-button v-if="aiEnabled" size="small" quaternary @click="showAI=!showAI">AI 助手</n-button>
              </n-layout-header>
              <n-layout-content content-style="padding:16px;background:var(--n-color);" :native-scrollbar="false">
                <router-view />
              </n-layout-content>
            </n-layout>
          </n-layout>

          <!-- Create Task Modal -->
          <n-modal v-model:show="showCreateTask" preset="card" title="新建任务" style="max-width:560px;">
            <n-form ref="createFormRef" :model="createForm" label-placement="top">
              <n-form-item label="任务标题" path="title" :rule="{required:true,message:'请输入标题'}">
                <n-input v-model:value="createForm.title" placeholder="例如：重构订单同步链路" />
              </n-form-item>
              <n-form-item label="任务 Key（可选）" path="task_key">
                <n-input v-model:value="createForm.task_key" placeholder="例如：SYNC" />
              </n-form-item>
              <n-form-item label="任务目标 (Goal)" path="goal">
                <n-input v-model:value="createForm.goal" type="textarea" :rows="4"
                  placeholder="2-4句，说清交付标准、约束和范围外项" />
              </n-form-item>
            </n-form>
            <template #action>
              <n-button @click="showCreateTask=false">取消</n-button>
              <n-button type="primary" :loading="creating" @click="doCreateTask">创建任务</n-button>
            </template>
          </n-modal>

          <!-- AI Drawer -->
          <n-drawer v-model:show="showAI" :width="420" placement="right">
            <n-drawer-content title="AI 工作助手" :native-scrollbar="false">
              <template #header-extra>
                <n-space>
                  <n-select v-model:value="aiModel" :options="aiModels" size="small" style="width:160px;" placeholder="默认模型" clearable />
                  <n-button size="small" quaternary @click="clearAI">清空</n-button>
                </n-space>
              </template>
              <div ref="aiMessagesRef" style="min-height:200px;">
                <div v-if="aiMessages.length===0" style="color:var(--n-text-color-3);padding:20px;">
                  可以直接让我分析当前任务、补节点结构，或者推进节点操作。
                </div>
                <div v-for="(msg,i) in aiMessages" :key="i" style="margin-bottom:12px;">
                  <n-tag :type="msg.role==='user'?'info':'success'" size="small">{{ msg.role==='user'?'你':'AI' }}</n-tag>
                  <div style="white-space:pre-wrap;margin-top:4px;font-size:13px;">{{ msg.content }}</div>
                </div>
                <div v-if="aiLoading" style="padding:8px 0;">
                  <n-spin size="small" /> 思考中...
                </div>
              </div>
              <template #footer>
                <n-input-group>
                  <n-input v-model:value="aiInput" type="textarea" :autosize="{minRows:1,maxRows:4}"
                    placeholder="向 AI 提问或下达操作指令…" @keydown.enter.ctrl="sendAI" />
                  <n-button type="primary" @click="sendAI" :loading="aiLoading" style="align-self:flex-end;">发送</n-button>
                </n-input-group>
              </template>
            </n-drawer-content>
          </n-drawer>
        </n-notification-provider>
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup>
import { ref, computed, watch, onMounted, h, nextTick, defineComponent, provide } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { darkTheme, useMessage, useDialog, useNotification } from 'naive-ui'
import { api } from './api.js'

const GlobalApiProvider = defineComponent({
  setup() {
    window.$message = useMessage()
    window.$dialog = useDialog()
    window.$notification = useNotification()
    return () => null
  }
})

const router = useRouter()
const route = useRoute()

const breadcrumbItems = ref([{ label: '任务总览', path: '/' }])
provide('breadcrumb', breadcrumbItems)

const siderCollapsed = ref(false)
const isDark = ref(false)
const showCreateTask = ref(false)
const creating = ref(false)
const createForm = ref({ title: '', task_key: '', goal: '' })
const createFormRef = ref(null)

// AI
const aiEnabled = ref(false)
const showAI = ref(false)
const aiInput = ref('')
const aiMessages = ref([])
const aiLoading = ref(false)
const aiModel = ref(null)
const aiMessagesRef = ref(null)
const aiModels = [
  { label: 'gpt-5.4', value: 'gpt-5.4' },
  { label: 'gpt-5.3-codex', value: 'gpt-5.3-codex' },
  { label: 'gpt-4o', value: 'gpt-4o' },
  { label: 'claude-opus-4-5', value: 'claude-opus-4-5' },
  { label: 'claude-sonnet-4-5', value: 'claude-sonnet-4-5' },
]

const menuOptions = [
  { label: '任务总览', key: 'overview', icon: renderMenuIcon('overview') },
  { label: '工作池', key: 'work', icon: renderMenuIcon('work') },
  { label: '搜索', key: 'search', icon: renderMenuIcon('search') },
  { label: '文档', key: 'docs', icon: renderMenuIcon('docs') },
  { label: '回收站', key: 'trash', icon: renderMenuIcon('trash') },
]

const activeMenuKey = computed(() => {
  if (route.path === '/' || route.path.startsWith('/projects/')) return 'overview'
  if (route.path === '/work') return 'work'
  if (route.path === '/search') return 'search'
  if (route.path === '/docs') return 'docs'
  if (route.path === '/trash') return 'trash'
  return ''
})

const currentTitle = computed(() => {
  const titles = { overview: '任务总览', work: '工作池', search: '搜索', trash: '回收站' }
  return titles[activeMenuKey.value] || '任务详情'
})

function onMenuSelect(key) {
  const routes = { overview: '/', work: '/work', search: '/search', docs: '/docs', trash: '/trash' }
  if (routes[key]) router.push(routes[key])
}

function renderMenuIcon(name) {
  const paths = {
    overview: 'M4 6h16M4 12h16M4 18h16',
    work: 'M4 7h6l2 2h8v10H4z',
    search: 'M11 4a7 7 0 1 0 4.24 12.6l4.08 4.09 1.42-1.42-4.09-4.08A7 7 0 0 0 11 4z',
    docs: 'M6 4h9l3 3v13H6zM9 9h6M9 13h6M9 17h4',
    trash: 'M6 7h12M9 7V5h6v2m-7 0 .7 11h6.6L16 7',
  }
  return () =>
    h(
      'svg',
      {
        viewBox: '0 0 24 24',
        width: '18',
        height: '18',
        fill: 'none',
        stroke: 'currentColor',
        'stroke-width': '1.8',
        'stroke-linecap': 'round',
        'stroke-linejoin': 'round',
        style: 'display:block;',
      },
      [h('path', { d: paths[name] || paths.overview })],
    )
}

async function doCreateTask() {
  if (!createForm.value.title.trim()) return
  creating.value = true
  try {
    const body = { title: createForm.value.title }
    if (createForm.value.task_key) body.task_key = createForm.value.task_key
    if (createForm.value.goal) body.goal = createForm.value.goal
    const result = await api('/tasks', { method: 'POST', body: JSON.stringify(body) })
    showCreateTask.value = false
    createForm.value = { title: '', task_key: '', goal: '' }
    router.push('/tasks/' + result.id)
  } catch (e) {
    window.$message?.error('创建失败: ' + e.message)
  } finally {
    creating.value = false
  }
}

async function checkAI() {
  try {
    const res = await fetch('/ai/status')
    const data = await res.json()
    aiEnabled.value = data.enabled === true
  } catch { aiEnabled.value = false }
}

async function sendAI() {
  const text = aiInput.value.trim()
  if (!text || aiLoading.value) return
  aiMessages.value.push({ role: 'user', content: text })
  aiInput.value = ''
  aiLoading.value = true
  try {
    const body = { message: text }
    if (aiModel.value) body.model = aiModel.value
    // Get current task context from route
    if (route.params.id) body.task_id = route.params.id
    const res = await fetch('/ai/chat', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
    if (!res.ok) {
      const err = await res.json().catch(() => ({ detail: res.statusText }))
      throw new Error(err.detail || err.error || res.statusText)
    }
    const data = await res.json()
    aiMessages.value.push({ role: 'assistant', content: data.reply || data.message || JSON.stringify(data) })
  } catch (e) {
    aiMessages.value.push({ role: 'assistant', content: '请求失败: ' + e.message })
  } finally {
    aiLoading.value = false
    await nextTick()
    if (aiMessagesRef.value) aiMessagesRef.value.scrollTop = aiMessagesRef.value.scrollHeight
  }
}

async function clearAI() {
  try { await fetch('/ai/clear', { method: 'POST' }) } catch {}
  aiMessages.value = []
}

onMounted(checkAI)
</script>
