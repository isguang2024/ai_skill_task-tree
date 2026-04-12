<template>
  <n-spin :show="loading" style="min-height:300px;">
    <template v-if="task">
      <!-- Task Header -->
      <n-page-header @back="$router.back()">
        <template #title>{{ task.title }}</template>
        <template #header>
          <n-space :size="4">
            <n-tag :type="statusType(task.status)" :bordered="false">{{ stateLabel(task.status, task.result) }}</n-tag>
            <n-tag>{{ task.task_key || task.id.substring(0,8) }}</n-tag>
            <n-tag type="info">{{ taskPct }}%</n-tag>
          </n-space>
        </template>
        <template #extra>
          <n-space>
            <n-button v-if="nextNode" type="primary" size="small" @click="selectNode(nextNode.id || nextNode.node_id)">下一步</n-button>
            <n-button size="small" @click="exportBrief" quaternary>导出简报</n-button>
            <n-button size="small" @click="copyId(task.id)" quaternary>复制 ID</n-button>
            <n-button size="small" @click="showSettings=true">任务设置</n-button>
            <n-dropdown :options="taskActions" @select="onTaskAction">
              <n-button size="small">操作</n-button>
            </n-dropdown>
          </n-space>
        </template>
        <n-text v-if="task.goal" depth="3" style="font-size:13px;display:block;margin:4px 0 8px;white-space:pre-wrap;">
          {{ task.goal }}
        </n-text>
        <n-progress :percentage="taskPct" :height="8" :border-radius="4" style="margin-bottom:8px;" />
        <n-grid :cols="6" :x-gap="8">
          <n-gi><n-statistic label="剩余" :value="remaining" /></n-gi>
          <n-gi><n-statistic label="阻塞" :value="blockCount" /></n-gi>
          <n-gi><n-statistic label="暂停" :value="pausedCount" /></n-gi>
          <n-gi><n-statistic label="取消" :value="canceledCount" /></n-gi>
          <n-gi><n-statistic label="节点" :value="nodes.length" /></n-gi>
          <n-gi><n-statistic label="估时" :value="estimate" /></n-gi>
        </n-grid>
      </n-page-header>

      <n-divider style="margin:12px 0;" />

      <!-- Main Content: Tree + Detail -->
      <n-grid :cols="24" :x-gap="12">
        <!-- Node Tree (Left) -->
        <n-gi :span="8">
          <n-card title="任务树" size="small" style="position:sticky;top:0;">
            <template #header-extra>
              <n-space :size="4">
                <n-tag size="small">{{ nodes.length }} 节点</n-tag>
                <n-button size="tiny" quaternary @click="myViewOnly=!myViewOnly">
                  {{ myViewOnly ? '我的视图' : '全部' }}
                </n-button>
              </n-space>
            </template>
            <!-- Favorites -->
            <div v-if="favorites.length" style="margin-bottom:8px;">
              <n-space :size="4">
                <n-tag v-for="fav in favorites" :key="fav.id" size="small" round closable
                  @close="removeFav(fav.id)" @click="selectNode(fav.id)" style="cursor:pointer;">
                  {{ fav.path || fav.title }}
                </n-tag>
              </n-space>
            </div>
            <!-- Search -->
            <n-input v-model:value="treeSearch" placeholder="搜索节点…" size="small" clearable style="margin-bottom:8px;" />
            <n-space :size="4" style="margin-bottom:8px;">
              <n-button size="tiny" quaternary @click="expandAll">全部展开</n-button>
              <n-button size="tiny" quaternary @click="collapseAll">全部折叠</n-button>
            </n-space>
            <!-- Tree -->
            <n-scrollbar style="max-height:calc(100vh - 380px);">
              <n-tree :data="filteredTreeData" :selected-keys="selectedNodeId ? [selectedNodeId] : []"
                :expanded-keys="expandedKeys" :pattern="treeSearch" :show-irrelevant-nodes="false"
                :render-label="renderTreeLabel" :render-prefix="renderTreePrefix" :render-suffix="renderTreeSuffix"
                :node-props="nodeProps" block-line
                @update:selected-keys="onTreeSelect" @update:expanded-keys="k=>expandedKeys=k" />
              <n-empty v-if="treeData.length===0" description="还没有节点" size="small">
                <template #extra>
                  <n-button size="tiny" type="primary" @click="openNodeCreate('')">新增根节点</n-button>
                </template>
              </n-empty>
            </n-scrollbar>
          </n-card>
        </n-gi>

        <!-- Right Content -->
        <n-gi :span="16">
          <n-tabs v-model:value="activeTab" type="line" animated>
            <!-- Node Detail Tab -->
            <n-tab-pane name="node" tab="当前节点">
              <template v-if="selectedNode">
                <n-card size="small" style="margin-bottom:12px;">
                  <template #header>
                    <n-space align="center" :size="6">
                      <n-tag :type="statusType(selectedNode.status)" :bordered="false">{{ stateLabel(selectedNode.status, selectedNode.result) }}</n-tag>
                      <n-tag size="small">{{ selectedNode.kind==='group'?'分组':'叶子' }}</n-tag>
                      <n-tag type="info" size="small">{{ nodePct(selectedNode) }}%</n-tag>
                      <n-tag v-if="isNodeClaimed" type="success" size="small">已领取 {{ claimedBy }}</n-tag>
                    </n-space>
                  </template>
                  <template #header-extra>
                    <n-space :size="4">
                      <n-button size="tiny" quaternary @click="toggleFav(selectedNode)">{{ isFav ? '★' : '☆' }}</n-button>
                      <n-button size="tiny" quaternary @click="copyId(selectedNode.id)">复制 ID</n-button>
                    </n-space>
                  </template>

                  <div style="font-size:16px;font-weight:600;margin-bottom:2px;">{{ selectedNode.title }}</div>
                  <n-text depth="3" style="font-size:12px;">{{ selectedNode.path }}</n-text>
                  <n-progress :percentage="nodePct(selectedNode)" :height="6" :border-radius="3" style="margin:10px 0;" />

                  <!-- Node Actions -->
                  <n-space :size="6" style="margin-bottom:12px;">
                    <n-button v-if="canClaim" size="small" type="primary" @click="claimNode">领取</n-button>
                    <n-button v-if="canRelease" size="small" @click="releaseNode">释放</n-button>
                    <n-button v-if="canProgress" size="small" type="info" @click="showProgressModal=true">上报进度</n-button>
                    <n-button v-if="canComplete" size="small" type="success" @click="showCompleteModal=true">完成</n-button>
                    <n-button v-if="canBlock" size="small" type="error" ghost @click="blockNode">阻塞</n-button>
                    <n-button v-if="canUnblock" size="small" @click="nodeTransition('unblock')">解除阻塞</n-button>
                    <n-button v-if="canPause" size="small" type="warning" ghost @click="nodeTransition('pause')">暂停</n-button>
                    <n-button v-if="canReopen" size="small" @click="nodeTransition('reopen')">重开</n-button>
                    <n-button v-if="canCancel" size="small" type="error" ghost @click="nodeTransition('cancel')">取消</n-button>
                    <n-button v-if="canConvertToLeaf" size="small" @click="retypeNode">转回执行节点</n-button>
                    <n-button size="small" quaternary @click="openNodeCreate(selectedNode.id)">添加子节点</n-button>
                    <n-button v-if="selectedNode.parent_node_id" size="small" quaternary @click="openNodeCreate(selectedNode.parent_node_id)">添加同级</n-button>
                  </n-space>

                  <!-- Instruction -->
                  <div v-if="selectedNode.instruction" style="margin-bottom:12px;">
                    <n-text depth="3" style="font-size:11px;font-weight:600;">INSTRUCTION</n-text>
                    <div style="font-size:13px;white-space:pre-wrap;margin-top:4px;padding:8px;background:var(--n-color-modal);border-radius:4px;">{{ selectedNode.instruction }}</div>
                  </div>

                  <!-- Acceptance -->
                  <div v-if="selectedNode.acceptance_criteria?.length" style="margin-bottom:12px;">
                    <n-text depth="3" style="font-size:11px;font-weight:600;">验收标准</n-text>
                    <n-list size="small" bordered style="margin-top:4px;">
                      <n-list-item v-for="(c,i) in selectedNode.acceptance_criteria" :key="i">
                        <n-text>{{ c }}</n-text>
                      </n-list-item>
                    </n-list>
                  </div>

                  <!-- Node Info -->
                  <n-descriptions :column="2" label-placement="top" bordered size="small">
                    <n-descriptions-item label="节点 ID">
                      <n-text code style="font-size:11px;cursor:pointer;" @click="copyId(selectedNode.id)">{{ selectedNode.id }}</n-text>
                    </n-descriptions-item>
                    <n-descriptions-item label="任务 ID">
                      <n-text code style="font-size:11px;cursor:pointer;" @click="copyId(props.id)">{{ props.id }}</n-text>
                    </n-descriptions-item>
                    <n-descriptions-item label="估时">{{ selectedNode.estimate || 0 }}h</n-descriptions-item>
                    <n-descriptions-item label="类型">{{ selectedNode.kind==='group'?'分组节点':'叶子节点' }}</n-descriptions-item>
                    <n-descriptions-item label="路径">{{ selectedNode.path }}</n-descriptions-item>
                    <n-descriptions-item label="租约" v-if="selectedNode.lease_until">{{ shortTime(selectedNode.lease_until) }}</n-descriptions-item>
                  </n-descriptions>
                </n-card>

                <!-- Edit Node -->
                <n-collapse style="margin-bottom:12px;">
                  <n-collapse-item title="编辑节点" name="edit">
                    <n-form :model="editForm" label-placement="top" size="small">
                      <n-grid :cols="2" :x-gap="8">
                        <n-gi>
                          <n-form-item label="标题">
                            <n-input v-model:value="editForm.title" />
                          </n-form-item>
                        </n-gi>
                        <n-gi>
                          <n-form-item label="估时(h)">
                            <n-input-number v-model:value="editForm.estimate" :min="0" :step="0.5" />
                          </n-form-item>
                        </n-gi>
                      </n-grid>
                      <n-form-item label="Instruction">
                        <n-input v-model:value="editForm.instruction" type="textarea" :rows="4" />
                      </n-form-item>
                      <n-form-item label="验收标准（一行一条）">
                        <n-input v-model:value="editForm.acceptance" type="textarea" :rows="3" />
                      </n-form-item>
                      <n-button type="primary" size="small" :loading="submitting" @click="saveNode">保存节点</n-button>
                    </n-form>
                  </n-collapse-item>
                </n-collapse>

                <!-- Children -->
                <n-card v-if="selectedChildren.length" size="small" title="子节点" style="margin-bottom:12px;">
                  <template #header-extra>
                    <n-space :size="4" align="center">
                      <n-tag size="small">直接 {{ selectedChildren.length }}</n-tag>
                      <n-tag v-if="selectedDescendantLeaves.length !== selectedChildren.length" size="small" type="info">叶子 {{ selectedDescendantLeaves.length }}</n-tag>
                    </n-space>
                  </template>
                  <n-list size="small" hoverable clickable>
                    <n-list-item v-for="child in selectedChildren" :key="child.id" @click="selectNode(child.id)">
                      <template #prefix>
                        <n-tag :type="statusType(child.status)" size="small" :bordered="false">{{ stateLabel(child.status, child.result) }}</n-tag>
                      </template>
                      <div>
                        <div style="font-size:13px;font-weight:500;">{{ child.title }}</div>
                        <n-space :size="4" align="center" style="margin-top:2px;">
                          <n-text depth="3" style="font-size:11px;">{{ child.path }}</n-text>
                          <n-text code style="font-size:10px;cursor:pointer;color:var(--n-text-color-3);" @click.stop="copyId(child.id)">{{ child.id.substring(0, 15) }}…</n-text>
                        </n-space>
                      </div>
                      <template #suffix>
                        <n-space :size="4" align="center">
                          <n-tag v-if="getChildCount(child.id) > 0" size="small" round>{{ getChildCount(child.id) }}</n-tag>
                          <n-tag size="small" type="info">{{ nodePct(child) }}%</n-tag>
                        </n-space>
                      </template>
                    </n-list-item>
                  </n-list>
                </n-card>

              </template>
              <n-empty v-else description="选择左侧节点查看详情" />
            </n-tab-pane>

            <!-- Events Tab -->
            <n-tab-pane name="events" :tab="'事件 (' + events.length + ')'">
              <n-card size="small">
                <!-- Event scope nav -->
                <n-space align="center" justify="space-between" style="margin-bottom:10px;">
                  <n-radio-group v-model:value="eventScope" size="small">
                    <n-radio-button value="all">全部事件 ({{ events.length }})</n-radio-button>
                    <n-radio-button value="node" :disabled="!selectedNodeId">
                      {{ selectedNodeId ? (selectedNode?.title?.substring(0,12) || '当前节点') + (nodeEvents.length ? ' (' + nodeEvents.length + ')' : '') : '未选节点' }}
                    </n-radio-button>
                  </n-radio-group>
                  <n-checkbox v-if="eventScope==='node'" v-model:checked="eventsWarnOnly" size="small">仅异常</n-checkbox>
                </n-space>

                <!-- All events -->
                <template v-if="eventScope==='all'">
                  <n-empty v-if="events.length===0" description="暂无事件" />
                  <n-scrollbar v-else style="max-height:calc(100vh - 360px);">
                    <n-timeline>
                      <n-timeline-item v-for="ev in events" :key="ev.id"
                        :type="ev.type==='complete'?'success':ev.type==='blocked'?'error':'default'"
                        :title="eventTypeLabel(ev.type)" :time="shortTime(ev.created_at)">
                        <n-space :size="4" align="center" style="margin-bottom:2px;">
                          <n-text depth="3" style="font-size:11px;">{{ ev.node_id ? nodes.find(n=>n.id===ev.node_id)?.path || ev.node_id.substring(0,12) : '' }}</n-text>
                        </n-space>
                        <div v-if="ev.message" style="font-size:12px;white-space:pre-wrap;color:var(--n-text-color-2);">{{ ev.message }}</div>
                        <div v-if="ev.actor_type||ev.actor_id" style="font-size:11px;color:var(--n-text-color-3);">{{ (ev.actor_type||'') + ' ' + (ev.actor_id||'') }}</div>
                      </n-timeline-item>
                    </n-timeline>
                  </n-scrollbar>
                </template>

                <!-- Node events (with descendants) -->
                <template v-else>
                  <n-empty v-if="!selectedNodeId" description="请先在左侧选择一个节点" />
                  <n-empty v-else-if="filteredNodeEvents.length===0" description="暂无事件" />
                  <n-scrollbar v-else style="max-height:calc(100vh - 360px);">
                    <n-timeline>
                      <n-timeline-item v-for="ev in filteredNodeEvents" :key="ev.id"
                        :type="ev.type==='complete'?'success':ev.type==='blocked'?'error':'default'"
                        :title="eventTypeLabel(ev.type)" :time="shortTime(ev.created_at)">
                        <n-space :size="4" align="center" style="margin-bottom:2px;">
                          <n-text depth="3" style="font-size:11px;">{{ ev.node_id ? nodes.find(n=>n.id===ev.node_id)?.path || ev.node_id.substring(0,12) : '' }}</n-text>
                        </n-space>
                        <div v-if="ev.message" style="font-size:12px;white-space:pre-wrap;color:var(--n-text-color-2);">{{ ev.message }}</div>
                        <div v-if="ev.actor_type||ev.actor_id" style="font-size:11px;color:var(--n-text-color-3);">{{ (ev.actor_type||'') + ' ' + (ev.actor_id||'') }}</div>
                      </n-timeline-item>
                    </n-timeline>
                  </n-scrollbar>
                </template>
              </n-card>
            </n-tab-pane>

            <!-- Artifacts Tab -->
            <n-tab-pane name="artifacts" :tab="'产物 (' + artifacts.length + ')'">
              <n-card size="small">
                <!-- Add Artifact -->
                <n-collapse style="margin-bottom:12px;">
                  <n-collapse-item title="添加产物" name="add">
                    <n-form :model="artifactForm" label-placement="left" size="small">
                      <n-grid :cols="2" :x-gap="8">
                        <n-gi><n-form-item label="标题"><n-input v-model:value="artifactForm.title" placeholder="可选" /></n-form-item></n-gi>
                        <n-gi><n-form-item label="Kind"><n-input v-model:value="artifactForm.kind" /></n-form-item></n-gi>
                      </n-grid>
                      <n-form-item label="URI">
                        <n-input v-model:value="artifactForm.uri" placeholder="https://... 或 file:///..." />
                      </n-form-item>
                      <n-button type="primary" size="small" @click="createArtifact">添加链接</n-button>
                    </n-form>
                    <n-divider />
                    <n-upload :action="'/v1/tasks/'+task.id+'/artifacts/upload'" :data="{node_id:selectedNode?.id||''}"
                      name="file" @finish="onUploadFinish">
                      <n-button size="small">上传文件</n-button>
                    </n-upload>
                  </n-collapse-item>
                </n-collapse>
                <n-empty v-if="artifacts.length===0" description="暂无产物" />
                <n-list v-else size="small">
                  <n-list-item v-for="art in artifacts" :key="art.id">
                    <n-thing :title="art.title||art.id" :description="art.uri" content-style="font-size:12px;">
                      <template #header-extra>
                        <n-space :size="4">
                          <n-tag size="small">{{ art.kind || 'link' }}</n-tag>
                          <n-tag size="small">{{ shortTime(art.created_at) }}</n-tag>
                          <n-button v-if="(art.uri||'').startsWith('local://')" size="tiny" type="primary" quaternary
                            tag="a" :href="'/v1/artifacts/'+art.id+'/download'" target="_blank">下载</n-button>
                        </n-space>
                      </template>
                    </n-thing>
                  </n-list-item>
                </n-list>
              </n-card>
            </n-tab-pane>
          </n-tabs>
        </n-gi>
      </n-grid>
    </template>
  </n-spin>

  <!-- Progress Modal -->
  <n-modal v-model:show="showProgressModal" preset="card" title="上报进度" style="max-width:480px;">
    <n-form :model="progressForm" label-placement="top">
      <n-grid :cols="2" :x-gap="8">
        <n-gi>
          <n-form-item label="进度增量 (0.0–1.0)">
            <n-input-number v-model:value="progressForm.delta" :min="0" :max="1" :step="0.05" />
          </n-form-item>
        </n-gi>
        <n-gi>
          <n-form-item label="快捷百分比 (1-99)">
            <n-input-group>
              <n-input-number v-model:value="progressForm.targetPct" :min="1" :max="99" />
              <n-button @click="applyPct">应用</n-button>
            </n-input-group>
          </n-form-item>
        </n-gi>
      </n-grid>
      <n-form-item label="说明">
        <n-input v-model:value="progressForm.message" type="textarea" :rows="5"
          placeholder="做了什么 / 证据 / 偏差 / 遗留" />
      </n-form-item>
    </n-form>
    <template #action>
      <n-button @click="showProgressModal=false">取消</n-button>
      <n-button type="primary" :loading="submitting" @click="doProgress">写入进度</n-button>
    </template>
  </n-modal>

  <!-- Complete Modal -->
  <n-modal v-model:show="showCompleteModal" preset="card" title="完成节点" style="max-width:480px;">
    <n-form :model="completeForm" label-placement="top">
      <n-form-item label="完成说明（必填）">
        <n-input v-model:value="completeForm.message" type="textarea" :rows="6"
          placeholder="做了什么 / 证据 / 偏差 / 遗留" />
      </n-form-item>
    </n-form>
    <template #action>
      <n-button @click="showCompleteModal=false">取消</n-button>
      <n-button type="success" :loading="submitting" @click="doComplete">标记完成</n-button>
    </template>
  </n-modal>

  <!-- Create Node Modal -->
  <n-modal v-model:show="showNodeCreate" preset="card" :title="nodeCreateParent?'新增子节点':'新增根节点'" style="max-width:560px;">
    <n-form :model="newNodeForm" label-placement="top">
      <n-form-item label="节点标题" :rule="{required:true}">
        <n-input v-model:value="newNodeForm.title" placeholder="动词短语" />
      </n-form-item>
      <n-grid :cols="2" :x-gap="8">
        <n-gi><n-form-item label="Key（可选）"><n-input v-model:value="newNodeForm.node_key" /></n-form-item></n-gi>
        <n-gi><n-form-item label="估时(h)"><n-input-number v-model:value="newNodeForm.estimate" :min="0" :step="0.5" /></n-form-item></n-gi>
      </n-grid>
      <n-form-item label="Instruction">
        <n-input v-model:value="newNodeForm.instruction" type="textarea" :rows="4" />
      </n-form-item>
      <n-form-item label="节点模板">
        <n-select v-model:value="newNodeForm.template" :options="templateOpts" clearable @update:value="applyTemplate" />
      </n-form-item>
      <n-form-item label="验收标准（一行一条）">
        <n-input v-model:value="newNodeForm.acceptance" type="textarea" :rows="3" />
      </n-form-item>
    </n-form>
    <template #action>
      <n-button @click="showNodeCreate=false">取消</n-button>
      <n-button type="primary" :loading="submitting" @click="doCreateNode">创建节点</n-button>
    </template>
  </n-modal>

  <!-- Task Settings Modal -->
  <n-modal v-model:show="showSettings" preset="card" title="任务设置" style="max-width:560px;">
    <n-form :model="settingsForm" label-placement="top">
      <n-grid :cols="2" :x-gap="8">
        <n-gi><n-form-item label="标题"><n-input v-model:value="settingsForm.title" /></n-form-item></n-gi>
        <n-gi><n-form-item label="Key"><n-input v-model:value="settingsForm.task_key" /></n-form-item></n-gi>
      </n-grid>
      <n-form-item label="目标 (Goal)">
        <n-input v-model:value="settingsForm.goal" type="textarea" :rows="5" />
      </n-form-item>
    </n-form>
    <template #action>
      <n-button @click="showSettings=false">取消</n-button>
      <n-button type="primary" :loading="submitting" @click="saveTask">保存</n-button>
    </template>
  </n-modal>
</template>

<script setup>
import { ref, computed, watch, onMounted, h, reactive, inject } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { NTag, NRadioGroup, NRadioButton } from 'naive-ui'
import { api, statusType, statusLabel, stateLabel, pct, shortTime, excerpt, eventTypeLabel, leaseActive, nodePct as nodePctCalc, buildTreeData, nodeTemplates } from '../api.js'

const props = defineProps(['id'])
const router = useRouter()
const route = useRoute()
const breadcrumb = inject('breadcrumb', ref([]))

const loading = ref(true)
const submitting = ref(false)
const task = ref(null)
const nodes = ref([])
const events = ref([])
const artifacts = ref([])
const remaining = ref(0)
const blockCount = ref(0)
const pausedCount = ref(0)
const canceledCount = ref(0)
const estimate = ref('0h')
const nextNode = ref(null)
const selectedNodeId = ref('')
const selectedNode = ref(null)
const selectedChildren = ref([])
const selectedDescendantLeaves = ref([])
const nodeEvents = ref([])
const expandedKeys = ref([])
const treeSearch = ref('')
const activeTab = ref('node')
const myViewOnly = ref(false)
const eventsWarnOnly = ref(false)
const eventScope = ref('all') // 'all' | 'node'
const favorites = ref([])

// Modals
const showProgressModal = ref(false)
const showCompleteModal = ref(false)
const showNodeCreate = ref(false)
const showSettings = ref(false)
const nodeCreateParent = ref('')

// Forms
const editForm = reactive({ title: '', estimate: null, instruction: '', acceptance: '' })
const progressForm = reactive({ delta: 0.1, targetPct: null, message: '' })
const completeForm = reactive({ message: '' })
const newNodeForm = reactive({ title: '', node_key: '', estimate: null, instruction: '', acceptance: '', template: null })
const settingsForm = reactive({ title: '', task_key: '', goal: '' })
const artifactForm = reactive({ title: '', kind: 'link', uri: '' })

const templateOpts = [
  { label: '无模板', value: null },
  { label: '开发模板', value: 'dev' },
  { label: '测试模板', value: 'test' },
  { label: '验收模板', value: 'accept' },
]

const taskPct = computed(() => task.value ? pct(task.value.summary_percent) : 0)
const treeData = computed(() => buildTreeData(nodes.value))
const filteredTreeData = computed(() => {
  if (!myViewOnly.value) return treeData.value
  function filter(items) {
    return items.filter(item => {
      const st = (item.raw.status || '').toLowerCase()
      const childrenFiltered = item.children ? filter(item.children) : []
      return st === 'ready' || st === 'running' || childrenFiltered.length > 0
    }).map(item => {
      if (!item.children) return item
      return { ...item, children: filter(item.children) }
    })
  }
  return filter(treeData.value)
})

// Aggregate progress: for group nodes, compute based on ALL descendant leaf nodes
const nodeAggPct = computed(() => {
  const map = {}
  const byParent = {}
  for (const n of nodes.value) {
    const pid = n.parent_node_id || ''
    if (!byParent[pid]) byParent[pid] = []
    byParent[pid].push(n)
  }
  function calc(nodeId) {
    if (map[nodeId] !== undefined) return map[nodeId]
    const children = byParent[nodeId] || []
    if (children.length === 0) {
      // leaf node: use its own progress
      const node = nodes.value.find(n => n.id === nodeId)
      const v = node ? pct(node.progress || 0) : 0
      map[nodeId] = v
      return v
    }
    // group node: average of all descendant leaves
    let sum = 0, count = 0
    function collectLeaves(pid) {
      for (const child of (byParent[pid] || [])) {
        const grandChildren = byParent[child.id]
        if (grandChildren && grandChildren.length > 0) {
          collectLeaves(child.id)
        } else {
          sum += pct(child.progress || 0)
          count++
        }
      }
    }
    collectLeaves(nodeId)
    const v = count > 0 ? Math.round(sum / count) : 0
    map[nodeId] = v
    return v
  }
  for (const n of nodes.value) calc(n.id)
  return map
})

function nodePct(node) {
  if (nodeAggPct.value[node.id] !== undefined) return nodeAggPct.value[node.id]
  return nodePctCalc(node)
}

// Node capabilities
const isNodeClaimed = computed(() => selectedNode.value && leaseActive(selectedNode.value))
const claimedBy = computed(() => {
  if (!selectedNode.value) return ''
  return [(selectedNode.value.claimed_by_type || ''), (selectedNode.value.claimed_by_id || '')].filter(Boolean).join('/')
})
const isLeaf = computed(() => selectedNode.value?.kind === 'leaf')
const isClosed = computed(() => {
  if (!selectedNode.value) return true
  const r = selectedNode.value.result, s = selectedNode.value.status
  return r === 'done' || r === 'canceled' || s === 'closed'
})
const canClaim = computed(() => isLeaf.value && !isClosed.value && selectedNode.value?.status !== 'paused' && selectedNode.value?.status !== 'blocked' && !isNodeClaimed.value)
const canRelease = computed(() => isNodeClaimed.value)
const canProgress = computed(() => isLeaf.value && selectedNode.value?.status !== 'blocked' && selectedNode.value?.status !== 'paused' && !isClosed.value)
const canComplete = computed(() => isLeaf.value && !isClosed.value)
const canBlock = computed(() => isLeaf.value && selectedNode.value?.status !== 'blocked' && !isClosed.value)
const canUnblock = computed(() => isLeaf.value && selectedNode.value?.status === 'blocked')
const canPause = computed(() => isLeaf.value && !isClosed.value && selectedNode.value?.status !== 'paused')
const canReopen = computed(() => isLeaf.value && (selectedNode.value?.status === 'paused' || selectedNode.value?.result === 'done' || selectedNode.value?.result === 'canceled'))
const canCancel = computed(() => isLeaf.value && !selectedNode.value?.result && selectedNode.value?.status !== 'closed')
const canConvertToLeaf = computed(() => selectedNode.value?.kind === 'group' && !selectedChildren.value.length)

const isFav = computed(() => favorites.value.some(f => f.id === selectedNodeId.value))

const filteredNodeEvents = computed(() => {
  if (!eventsWarnOnly.value) return nodeEvents.value
  return nodeEvents.value.filter(ev => {
    const t = (ev.type || '').toLowerCase()
    return t === 'blocked' || t === 'reopened' || t === 'error' || (ev.payload?.warnings?.length > 0)
  })
})

// Task actions dropdown
const taskActions = computed(() => {
  const items = []
  const s = (task.value?.status || '').toLowerCase()
  if (s === 'ready' || s === 'running' || s === 'blocked') {
    items.push({ label: '暂停任务', key: 'pause' })
    items.push({ label: '取消任务', key: 'cancel' })
  }
  if (s === 'paused' || s === 'canceled' || s === 'closed') items.push({ label: '恢复任务', key: 'reopen' })
  if (s === 'paused') items.push({ label: '取消任务', key: 'cancel' })
  items.push({ label: '回收任务', key: 'delete' })
  return items
})

// Favorites
function loadFavs() {
  try {
    const v = JSON.parse(localStorage.getItem('task-tree-favs::' + props.id) || '[]')
    return Array.isArray(v) ? v : []
  } catch { return [] }
}
function saveFavs(items) {
  try { localStorage.setItem('task-tree-favs::' + props.id, JSON.stringify(items.slice(0, 20))) } catch {}
}
function toggleFav(node) {
  const favs = loadFavs()
  const idx = favs.findIndex(f => f.id === node.id)
  if (idx >= 0) favs.splice(idx, 1)
  else favs.unshift({ id: node.id, title: node.title, path: node.path })
  saveFavs(favs)
  favorites.value = favs
}
function removeFav(id) {
  const favs = loadFavs().filter(f => f.id !== id)
  saveFavs(favs)
  favorites.value = favs
}

// Tree render functions
function renderTreePrefix({ option }) {
  const raw = option.raw
  return h(NTag, { type: statusType(raw.status), size: 'small', bordered: false, style: 'font-size:10px;' },
    { default: () => stateLabel(raw.status, raw.result) })
}
function renderTreeLabel({ option }) {
  return h('span', { style: 'font-size:13px;' }, option.label)
}
function renderTreeSuffix({ option }) {
  const raw = option.raw
  const items = []
  const aggPct = nodeAggPct.value[raw.id] !== undefined ? nodeAggPct.value[raw.id] : nodePctCalc(raw)
  items.push(h(NTag, { size: 'tiny', type: 'info', round: true, style: 'font-size:10px;padding:0 5px;' }, { default: () => aggPct + '%' }))
  return h('span', { style: 'display:inline-flex;gap:4px;' }, items)
}
function nodeProps({ option }) {
  return { style: selectedNodeId.value === option.key ? 'background:var(--n-node-color-active);border-radius:4px;' : '' }
}

function expandAll() {
  expandedKeys.value = nodes.value.map(n => n.id)
}
function collapseAll() {
  expandedKeys.value = []
}

async function onTreeSelect(keys) {
  if (keys.length > 0) await selectNode(keys[0])
}

async function selectNode(nodeId, opts = {}) {
  if (!nodeId) return
  selectedNodeId.value = nodeId
  try {
    const node = (!opts.forceFetch && (opts.node || nodes.value.find(n => n.id === nodeId))) || await api('/nodes/' + nodeId)
    selectedNode.value = node
    editForm.title = node.title || ''
    editForm.estimate = node.estimate || null
    editForm.instruction = node.instruction || ''
    editForm.acceptance = (node.acceptance_criteria || []).join('\n')
    selectedChildren.value = nodes.value.filter(n => n.parent_node_id === nodeId)
    // Collect all descendant leaves
    const leaves = []
    function collectLeaves(parentId) {
      for (const n of nodes.value) {
        if (n.parent_node_id !== parentId) continue
        const hasKids = nodes.value.some(c => c.parent_node_id === n.id)
        if (hasKids) collectLeaves(n.id)
        else leaves.push(n)
      }
    }
    collectLeaves(nodeId)
    selectedDescendantLeaves.value = leaves
    if (!opts.forceFetch && Array.isArray(opts.events)) {
      nodeEvents.value = opts.events
      return
    }
    const hasChildren = selectedChildren.value.length > 0
    const params = new URLSearchParams({ task_id: props.id, node_id: nodeId, limit: '20' })
    if (hasChildren) params.set('include_descendants', 'true')
    const evWrap = await api('/events?' + params)
    nodeEvents.value = evWrap.items || evWrap || []
  } catch (e) {
    window.$message?.error('加载节点失败: ' + e.message)
  }
}

async function load() {
  loading.value = true
  try {
    const resume = await api('/tasks/' + props.id + '/resume?view_mode=detail&limit=10000')
    const taskInfo = resume.task || {}
    task.value = {
      id: props.id,
      title: taskInfo.title || '',
      goal: taskInfo.goal || '',
      status: taskInfo.status || '',
      task_key: taskInfo.task_key || '',
      project_id: taskInfo.project_id || '',
      summary_percent: taskInfo.summary_percent || 0,
      result: taskInfo.result || '',
    }
    nodes.value = resume.tree || []
    // Update breadcrumb with project context from history state first, then fallback to task.project_id.
    const st = window.history.state || {}
    const crumb = [{ label: '任务总览', path: '/' }]
    if (st.projectId && st.projectName) {
      crumb.push({ label: st.projectName, path: '/projects/' + st.projectId })
    } else if (task.value.project_id) {
      try {
        const project = await api('/projects/' + task.value.project_id)
        crumb.push({ label: project.name || '项目', path: '/projects/' + task.value.project_id })
      } catch {}
    }
    crumb.push({ label: task.value.title })
    breadcrumb.value = crumb
    remaining.value = resume.remaining?.remaining_nodes || 0
    blockCount.value = resume.remaining?.blocked_nodes || 0
    pausedCount.value = resume.remaining?.paused_nodes || 0
    canceledCount.value = resume.remaining?.canceled_nodes || 0
    estimate.value = ((resume.remaining?.remaining_estimate || 0)).toFixed(1) + 'h'
    events.value = resume.recent_events || []
    artifacts.value = resume.artifacts || []
    settingsForm.title = task.value.title || ''
    settingsForm.task_key = task.value.task_key || ''
    settingsForm.goal = task.value.goal || ''
    favorites.value = loadFavs()

    // Init expanded keys (all non-leaf nodes for full tree visibility)
    const parentIds = new Set(nodes.value.filter(n => n.parent_node_id).map(n => n.parent_node_id))
    if (expandedKeys.value.length === 0) expandedKeys.value = [...parentIds]

    // Select next node or first
    if (resume.next_node?.node) {
      nextNode.value = resume.next_node.node
      const nextId = resume.next_node.node.id || resume.next_node.node.node_id
      const node = nodes.value.find(n => n.id === nextId)
      const selectOpts = { events: resume.next_node.recent_events || [] }
      if (node) selectOpts.node = node
      await selectNode(nextId, selectOpts)
    } else if (nodes.value.length > 0) {
      const sorted = [...nodes.value].sort((a, b) => {
        const so = (a.sort_order ?? Infinity) - (b.sort_order ?? Infinity)
        if (so !== 0) return so
        return (a.created_at || '').localeCompare(b.created_at || '')
      })
      await selectNode(sorted[0].id, { node: sorted[0] })
    }

    // Start SSE
    startSSE()
  } catch (e) {
    window.$message?.error('加载失败: ' + e.message)
  } finally {
    loading.value = false
  }
}

// SSE Live Events
let eventSource = null
// Debounced node+task refresh to batch rapid events
let refreshTimer = null
function scheduleRefresh() {
  if (refreshTimer) return
  refreshTimer = setTimeout(async () => {
    refreshTimer = null
    try {
      const [t, ns] = await Promise.all([
        api('/tasks/' + props.id),
        api('/tasks/' + props.id + '/nodes'),
      ])
      task.value = t
      nodes.value = ns
      // Re-select current node to refresh its detail (lease, progress, status)
      if (selectedNodeId.value) {
        const updated = ns.find(n => n.id === selectedNodeId.value)
        if (updated) selectedNode.value = updated
        selectedChildren.value = ns.filter(n => n.parent_node_id === selectedNodeId.value)
      }
    } catch {}
  }, 600)
}

function startSSE() {
  if (eventSource) { eventSource.close(); eventSource = null }
  if (!task.value) return
  const url = '/v1/tasks/' + props.id + '/events/stream'
  eventSource = new EventSource(url)
  eventSource.addEventListener('task_event', (e) => {
    try {
      const ev = JSON.parse(e.data)

      // 1. Insert into global events list (deduplicated)
      if (!events.value.some(x => x.id === ev.id)) {
        events.value = [ev, ...events.value]
      }

      // 2. Insert into node events if relevant to selected node or its descendants
      if (selectedNodeId.value) {
        const nodeIds = new Set([selectedNodeId.value, ...selectedDescendantLeaves.value.map(n => n.id)])
        if (nodeIds.has(ev.node_id) && !nodeEvents.value.some(x => x.id === ev.id)) {
          nodeEvents.value = [ev, ...nodeEvents.value]
        }
      }

      // 3. Optimistically patch the affected node in-place from event payload
      if (ev.node_id && ev.node_snapshot) {
        const snap = ev.node_snapshot
        nodes.value = nodes.value.map(n => n.id === ev.node_id ? { ...n, ...snap } : n)
        if (selectedNodeId.value === ev.node_id) selectedNode.value = { ...selectedNode.value, ...snap }
      }

      // 4. Debounced full refresh for accurate rollup (task summary + all nodes)
      scheduleRefresh()
    } catch {}
  })
  eventSource.onerror = () => {
    if (eventSource) { eventSource.close(); eventSource = null }
    setTimeout(() => { if (task.value) startSSE() }, 5000)
  }
}

// Actions
async function taskTransition(action) {
  try {
    await api('/tasks/' + props.id + '/transition', { method: 'POST', body: JSON.stringify({ action }) })
    window.$message?.success('操作成功')
    await load()
  } catch (e) { window.$message?.error(e.message) }
}

function navigateAfterTaskDelete() {
  const st = window.history.state || {}
  const backPath = typeof st.backPath === 'string' ? st.backPath : ''
  if (backPath && backPath !== ('/tasks/' + props.id)) {
    router.push(backPath)
    return
  }
  if (task.value?.project_id) {
    router.push('/projects/' + task.value.project_id)
    return
  }
  router.push('/')
}

function onTaskAction(key) {
  const confirmMap = {
    pause: { title: '暂停任务', content: '确认暂停该任务？暂停后所有节点将不可操作。', type: 'warning' },
    cancel: { title: '取消任务', content: '确认取消该任务？取消后所有未完成节点将被标记为取消。', type: 'error' },
    reopen: { title: '恢复任务', content: '确认恢复该任务？', type: 'info' },
    delete: { title: '回收任务', content: '确认将该任务移入回收站？', type: 'warning' },
  }
  const conf = confirmMap[key] || { title: '确认', content: '确认执行该操作？', type: 'warning' }
  const dialogFn = conf.type === 'error' ? 'error' : conf.type === 'info' ? 'info' : 'warning'
  window.$dialog?.[dialogFn]({
    title: conf.title, content: conf.content, positiveText: '确认', negativeText: '取消',
    onPositiveClick: async () => {
      try {
        if (key === 'delete') {
          await api('/tasks/' + props.id, { method: 'DELETE' })
          window.$message?.success('已回收')
          navigateAfterTaskDelete()
        } else {
          await taskTransition(key)
        }
      } catch (e) { window.$message?.error(e.message) }
    },
  })
}

async function claimNode() {
  try {
    await api('/tasks/' + props.id + '/nodes/' + selectedNodeId.value + '/claim', {
      method: 'POST', body: JSON.stringify({ actor: { tool: 'web_ui' } })
    })
    window.$message?.success('领取成功')
    await selectNode(selectedNodeId.value, { forceFetch: true })
  } catch (e) { window.$message?.error(e.message) }
}

async function releaseNode() {
  try {
    await api('/tasks/' + props.id + '/nodes/' + selectedNodeId.value + '/release', { method: 'POST' })
    window.$message?.success('已释放')
    await selectNode(selectedNodeId.value, { forceFetch: true })
  } catch (e) { window.$message?.error(e.message) }
}

async function blockNode() {
  try {
    await api('/tasks/' + props.id + '/nodes/' + selectedNodeId.value + '/block', {
      method: 'POST', body: JSON.stringify({ reason: '' })
    })
    window.$message?.success('已阻塞')
    await load()
  } catch (e) { window.$message?.error(e.message) }
}

async function nodeTransition(action) {
  try {
    await api('/tasks/' + props.id + '/nodes/' + selectedNodeId.value + '/transition', {
      method: 'POST', body: JSON.stringify({ action })
    })
    window.$message?.success('操作成功')
    await load()
  } catch (e) { window.$message?.error(e.message) }
}

async function retypeNode() {
  try {
    await api('/tasks/' + props.id + '/nodes/' + selectedNodeId.value + '/retype', { method: 'POST', body: '{}' })
    window.$message?.success('已转型')
    await load()
  } catch (e) { window.$message?.error(e.message) }
}

function applyPct() {
  if (!progressForm.targetPct || !selectedNode.value) return
  const current = nodePctCalc(selectedNode.value) / 100
  const delta = (progressForm.targetPct / 100) - current
  if (delta <= 0) { window.$message?.warning('目标需大于当前进度'); return }
  progressForm.delta = Math.round(delta * 100) / 100
}

async function doProgress() {
  submitting.value = true
  try {
    const body = { delta_progress: progressForm.delta || 0.1 }
    if (progressForm.message) body.message = progressForm.message
    await api('/tasks/' + props.id + '/nodes/' + selectedNodeId.value + '/progress', {
      method: 'POST', body: JSON.stringify(body)
    })
    showProgressModal.value = false
    progressForm.delta = 0.1; progressForm.message = ''; progressForm.targetPct = null
    window.$message?.success('进度已更新')
    await load()
  } catch (e) { window.$message?.error(e.message) }
  finally { submitting.value = false }
}

async function doComplete() {
  if (!completeForm.message.trim()) { window.$message?.warning('请填写完成说明'); return }
  submitting.value = true
  try {
    await api('/tasks/' + props.id + '/nodes/' + selectedNodeId.value + '/complete', {
      method: 'POST', body: JSON.stringify({ message: completeForm.message })
    })
    showCompleteModal.value = false
    completeForm.message = ''
    window.$message?.success('节点已完成')
    await load()
  } catch (e) { window.$message?.error(e.message) }
  finally { submitting.value = false }
}

async function saveNode() {
  submitting.value = true
  try {
    const body = { title: editForm.title }
    if (editForm.estimate != null) body.estimate = editForm.estimate
    if (editForm.instruction !== undefined) body.instruction = editForm.instruction
    const criteria = editForm.acceptance.split('\n').map(s => s.trim()).filter(Boolean)
    if (criteria.length) body.acceptance_criteria = criteria
    await api('/nodes/' + selectedNodeId.value, { method: 'PATCH', body: JSON.stringify(body) })
    window.$message?.success('已保存')
    await load()
  } catch (e) { window.$message?.error(e.message) }
  finally { submitting.value = false }
}

function openNodeCreate(parentId) {
  nodeCreateParent.value = parentId
  newNodeForm.title = ''; newNodeForm.node_key = ''; newNodeForm.estimate = null
  newNodeForm.instruction = ''; newNodeForm.acceptance = ''; newNodeForm.template = null
  showNodeCreate.value = true
}

function applyTemplate(val) {
  if (!val || !nodeTemplates[val]) return
  const tpl = nodeTemplates[val]
  if (!newNodeForm.instruction.trim()) newNodeForm.instruction = tpl.instruction
  if (!newNodeForm.acceptance.trim()) newNodeForm.acceptance = tpl.acceptance.join('\n')
}

async function doCreateNode() {
  if (!newNodeForm.title.trim()) return
  submitting.value = true
  try {
    const body = { title: newNodeForm.title, kind: 'leaf' }
    if (nodeCreateParent.value) body.parent_node_id = nodeCreateParent.value
    if (newNodeForm.node_key) body.node_key = newNodeForm.node_key
    if (newNodeForm.estimate) body.estimate = newNodeForm.estimate
    if (newNodeForm.instruction) body.instruction = newNodeForm.instruction
    const criteria = (newNodeForm.acceptance || '').split('\n').map(s => s.trim()).filter(Boolean)
    if (criteria.length) body.acceptance_criteria = criteria
    const result = await api('/tasks/' + props.id + '/nodes', { method: 'POST', body: JSON.stringify(body) })
    showNodeCreate.value = false
    window.$message?.success('节点已创建')
    await load()
    if (result.id) selectNode(result.id)
  } catch (e) { window.$message?.error(e.message) }
  finally { submitting.value = false }
}

async function saveTask() {
  submitting.value = true
  try {
    const body = { title: settingsForm.title }
    if (settingsForm.task_key) body.task_key = settingsForm.task_key
    if (settingsForm.goal !== undefined) body.goal = settingsForm.goal
    await api('/tasks/' + props.id, { method: 'PATCH', body: JSON.stringify(body) })
    showSettings.value = false
    window.$message?.success('已保存')
    await load()
  } catch (e) { window.$message?.error(e.message) }
  finally { submitting.value = false }
}

async function createArtifact() {
  try {
    const body = { uri: artifactForm.uri }
    if (artifactForm.title) body.title = artifactForm.title
    if (artifactForm.kind) body.kind = artifactForm.kind
    if (selectedNode.value) body.node_id = selectedNode.value.id
    await api('/tasks/' + props.id + '/artifacts', { method: 'POST', body: JSON.stringify(body) })
    artifactForm.title = ''; artifactForm.uri = ''
    window.$message?.success('已添加')
    await load()
  } catch (e) { window.$message?.error(e.message) }
}

function onUploadFinish() {
  window.$message?.success('上传成功')
  load()
}

function copyId(id) {
  navigator.clipboard?.writeText(id).then(() => window.$message?.success('已复制: ' + id))
}

function getChildCount(nodeId) {
  return nodes.value.filter(n => n.parent_node_id === nodeId).length
}

function exportBrief() {
  if (!task.value) return
  const brief = [
    '任务简报',
    '标题: ' + task.value.title,
    '状态: ' + stateLabel(task.value.status, task.value.result),
    '概述: ' + (task.value.goal || '无'),
    '进度: ' + taskPct.value + '%',
    '风险: ' + (blockCount.value > 0 ? '阻塞 ' + blockCount.value : '无'),
    '下一步: ' + (nextNode.value ? (nextNode.value.path + ' · ' + nextNode.value.title) : '暂无'),
  ].join('\n')
  navigator.clipboard?.writeText(brief).then(() => window.$message?.success('简报已复制'))
}

onMounted(load)
watch(() => props.id, (n, o) => { if (n && n !== o) { expandedKeys.value = []; load() } })

// Cleanup SSE on unmount
import { onUnmounted } from 'vue'
onUnmounted(() => { if (eventSource) { eventSource.close(); eventSource = null } })
</script>
