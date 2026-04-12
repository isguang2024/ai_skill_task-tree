<template>
  <n-spin :show="loading" style="min-height:300px;">
    <template v-if="task">
      <!-- Task Header -->
      <n-page-header @back="$router.back()">
        <template #title>{{ task.title }}</template>
        <template #header>
            <n-space :size="4">
            <n-tooltip>
              <template #trigger>
                <n-tag :type="statusType(task.status)" :bordered="false">{{ stateLabel(task.status, task.result) }}</n-tag>
              </template>
              <span>任务当前状态；用于表示整体执行是否就绪、进行中或已关闭</span>
            </n-tooltip>
            <n-tag>{{ task.task_key || task.id.substring(0,8) }}</n-tag>
            <n-tag type="info">{{ taskPct }}%</n-tag>
          </n-space>
        </template>
        <template #extra>
          <n-space>
            <n-tooltip v-if="nextNode">
              <template #trigger>
                <n-button type="primary" size="small" @click="selectNode(nextNode.id || nextNode.node_id)">下一步</n-button>
              </template>
              <span>跳转到系统推荐的下一可执行节点</span>
            </n-tooltip>
            <n-tooltip>
              <template #trigger>
                <n-button size="small" @click="exportBrief" quaternary>导出简报</n-button>
              </template>
              <span>导出当前任务的简要说明</span>
            </n-tooltip>
            <n-tooltip>
              <template #trigger>
                <n-button size="small" @click="copyId(task.id)" quaternary>复制 ID</n-button>
              </template>
              <span>复制任务 ID 便于外部引用</span>
            </n-tooltip>
            <n-tooltip>
              <template #trigger>
                <n-button size="small" @click="showSettings=true">任务设置</n-button>
              </template>
              <span>编辑任务标题、Key 和目标</span>
            </n-tooltip>
            <n-dropdown :options="taskActions" @select="onTaskAction">
              <template #default>
                <n-tooltip>
                  <template #trigger>
                    <n-button size="small">操作</n-button>
                  </template>
                  <span>打开任务级状态流转菜单</span>
                </n-tooltip>
              </template>
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

      <n-divider style="margin:8px 0;" />



      <!-- Main Content: Tree + Detail -->
      <n-grid :cols="24" :x-gap="12">
        <!-- Node Tree (Left) -->
        <n-gi :span="8">
            <n-card title="任务树" size="small" style="position:sticky;top:0;">
              <template #header-extra>
                <n-space :size="4">
                  <n-tag size="small">{{ nodes.length }} 节点</n-tag>
                  <n-tag size="small" type="info">{{ treeMode === 'focus' ? 'Focus' : 'Full' }}</n-tag>
                  <n-button size="tiny" quaternary @click="myViewOnly=!myViewOnly">
                    {{ myViewOnly ? '我的视图' : '全部' }}
                  </n-button>
                  <n-button size="tiny" quaternary @click="toggleTreeMode">
                    {{ treeMode === 'focus' ? '查看全量' : '聚焦模式' }}
                  </n-button>
                </n-space>
              </template>
            <!-- Stages compact -->
            <div v-if="stages.length > 0" style="margin-bottom:8px;padding-bottom:8px;border-bottom:1px solid var(--n-border-color);">
              <n-space :size="4" align="center" wrap>
                <n-text depth="3" style="font-size:11px;font-weight:600;">阶段</n-text>
                <n-tooltip v-for="stage in stages" :key="stage.id">
                  <template #trigger>
                    <n-tag size="small" round
                      :type="currentStage?.id === stage.id ? 'success' : 'default'"
                      :bordered="currentStage?.id !== stage.id"
                      style="cursor:pointer;"
                      @click="currentStage?.id !== stage.id ? doActivateStage(stage.id) : undefined">
                      {{ stage.title }}
                    </n-tag>
                  </template>
                  <div style="font-size:12px;line-height:1.6;">
                    <div>{{ stage.path || stage.id }}</div>
                    <div>状态: {{ stateLabel(stage.status, stage.result) }}</div>
                    <div v-if="currentStage?.id === stage.id">当前激活阶段</div>
                    <div v-else>点击激活</div>
                  </div>
                </n-tooltip>
                <n-tooltip>
                  <template #trigger>
                    <n-button size="tiny" quaternary @click="showStageCreate=true">+</n-button>
                  </template>
                  <span>新增阶段</span>
                </n-tooltip>
              </n-space>
              <n-text v-if="currentStage" depth="3" style="font-size:11px;display:block;margin-top:4px;">
                当前: {{ currentStage.title }} · {{ stateLabel(currentStage.status, currentStage.result) }}
              </n-text>
            </div>
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
                      <n-tooltip>
                        <template #trigger>
                          <n-tag :type="statusType(selectedNode.status)" :bordered="false">{{ stateLabel(selectedNode.status, selectedNode.result) }}</n-tag>
                        </template>
                        <span>节点当前状态；决定是否可领取、完成或流转</span>
                      </n-tooltip>
                      <n-tag size="small">{{ selectedNode.kind==='group'?'分组':'叶子' }}</n-tag>
                      <n-tag type="info" size="small">{{ nodePct(selectedNode) }}%</n-tag>
                      <n-tag v-if="isNodeClaimed" type="success" size="small">已领取 {{ claimedBy }}</n-tag>
                    </n-space>
                  </template>
                  <template #header-extra>
                    <n-space :size="4">
                      <n-tooltip>
                        <template #trigger>
                          <n-button size="tiny" quaternary @click="toggleFav(selectedNode)">{{ isFav ? '★' : '☆' }}</n-button>
                        </template>
                        <span>收藏或取消收藏当前节点</span>
                      </n-tooltip>
                      <n-tooltip>
                        <template #trigger>
                          <n-button size="tiny" quaternary @click="copyId(selectedNode.id)">复制 ID</n-button>
                        </template>
                        <span>复制节点 ID 便于外部引用</span>
                      </n-tooltip>
                    </n-space>
                  </template>

                  <div style="font-size:16px;font-weight:600;margin-bottom:2px;">{{ selectedNode.title }}</div>
                  <n-text depth="3" style="font-size:12px;">{{ selectedNode.path }}</n-text>
                  <n-progress :percentage="nodePct(selectedNode)" :height="6" :border-radius="3" style="margin:10px 0;" />

                  <!-- Node Actions -->
                  <n-space :size="6" style="margin-bottom:12px;">
                    <n-tooltip v-if="canClaim"><template #trigger><n-button size="small" type="primary" @click="confirmClaimNode">领取</n-button></template><span>领取当前节点的执行权</span></n-tooltip>
                    <n-tooltip v-if="canRelease"><template #trigger><n-button size="small" @click="confirmReleaseNode">释放</n-button></template><span>归还当前节点的执行权</span></n-tooltip>
                    <n-tooltip v-if="canProgress"><template #trigger><n-button size="small" type="info" @click="confirmOpenProgressModal">上报进度</n-button></template><span>记录当前节点进展</span></n-tooltip>
                    <n-tooltip v-if="canComplete"><template #trigger><n-button size="small" type="success" @click="confirmOpenCompleteModal">完成</n-button></template><span>将当前节点标记完成</span></n-tooltip>
                    <n-tooltip v-if="canBlock"><template #trigger><n-button size="small" type="error" ghost @click="confirmBlockNode">阻塞</n-button></template><span>将节点标记为阻塞</span></n-tooltip>
                    <n-tooltip v-if="canUnblock"><template #trigger><n-button size="small" @click="confirmNodeTransition('unblock')">解除阻塞</n-button></template><span>恢复节点为可执行状态</span></n-tooltip>
                    <n-tooltip v-if="canPause"><template #trigger><n-button size="small" type="warning" ghost @click="confirmNodeTransition('pause')">暂停</n-button></template><span>暂停当前节点</span></n-tooltip>
                    <n-tooltip v-if="canReopen"><template #trigger><n-button size="small" @click="confirmNodeTransition('reopen')">重开</n-button></template><span>把已关闭节点重新打开</span></n-tooltip>
                    <n-tooltip v-if="canCancel"><template #trigger><n-button size="small" type="error" ghost @click="confirmNodeTransition('cancel')">取消</n-button></template><span>取消当前节点</span></n-tooltip>
                    <n-tooltip v-if="canConvertToLeaf"><template #trigger><n-button size="small" @click="confirmRetypeNode">转回执行节点</n-button></template><span>把无子节点的分组改回执行节点</span></n-tooltip>
                    <n-tooltip><template #trigger><n-button size="small" quaternary @click="confirmOpenNodeCreate(selectedNode.id)">添加子节点</n-button></template><span>在当前节点下新增子节点</span></n-tooltip>
                    <n-tooltip v-if="selectedNode.parent_node_id"><template #trigger><n-button size="small" quaternary @click="confirmOpenNodeCreate(selectedNode.parent_node_id)">添加同级</n-button></template><span>在同级位置新增兄弟节点</span></n-tooltip>
                    <n-tooltip v-if="!activeRun"><template #trigger><n-button size="small" type="info" ghost @click="confirmOpenRunStart">开始 Run</n-button></template><span>为该节点创建一次运行记录</span></n-tooltip>
                    <n-tooltip v-if="activeRun"><template #trigger><n-button size="small" type="success" ghost @click="confirmOpenRunFinish">结束 Run</n-button></template><span>结束当前运行记录</span></n-tooltip>
                    <n-tooltip v-if="activeRun"><template #trigger><n-button size="small" quaternary @click="confirmOpenRunLog">追加日志</n-button></template><span>向当前 Run 追加日志</span></n-tooltip>
                  </n-space>

                  <!-- Instruction -->
                  <div v-if="selectedNode.instruction" style="margin-bottom:12px;">
                    <n-text depth="3" style="font-size:11px;font-weight:600;">INSTRUCTION</n-text>
                    <div style="font-size:13px;white-space:pre-wrap;margin-top:4px;padding:8px;background:var(--n-color-modal);border-radius:4px;">{{ selectedNode.instruction }}</div>
                  </div>

                  <!-- Acceptance + Node Info (collapsible) -->
                  <n-collapse style="margin-bottom:0;">
                    <n-collapse-item v-if="selectedNode.acceptance_criteria?.length" title="验收标准" name="acceptance">
                      <n-list size="small" bordered>
                        <n-list-item v-for="(c,i) in selectedNode.acceptance_criteria" :key="i">
                          <n-text>{{ c }}</n-text>
                        </n-list-item>
                      </n-list>
                    </n-collapse-item>
                    <n-collapse-item title="节点信息" name="info">
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
                    </n-collapse-item>
                  </n-collapse>

                  <n-grid :cols="2" :x-gap="8" style="margin-top:12px;">
                    <n-gi>
                      <n-card size="small" title="节点摘要">
                        <div style="font-size:12px;line-height:1.7;white-space:pre-wrap;">{{ selectedMemoryText || '暂无节点摘要' }}</div>
                      </n-card>
                    </n-gi>
                    <n-gi>
                      <n-card size="small" title="节点运行历史">
                        <n-empty v-if="nodeRuns.length===0" description="暂无运行记录" size="small" />
                        <n-list v-else size="small" hoverable clickable>
                          <n-list-item
                            v-for="run in nodeRuns"
                            :key="run.id || run.run_id"
                            @click="selectRun(run.id || run.run_id)"
                            :style="selectedRunId === (run.id || run.run_id) ? 'background:var(--n-color-hover);border-radius:6px;' : ''"
                          >
                            <n-space vertical :size="4" style="width:100%;">
                              <n-space justify="space-between" style="width:100%;" align="center">
                                <n-space :size="4" align="center">
                                  <n-tooltip>
                                    <template #trigger>
                                      <n-tag :type="run.status==='running' ? 'info' : 'default'" size="small" :bordered="false">
                                        {{ run.result || run.status || 'run' }}
                                      </n-tag>
                                    </template>
                                    <span>Run 状态；running 表示执行中，其他值表示已结束</span>
                                  </n-tooltip>
                                  <n-text strong style="font-size:12px;">{{ run.trigger_kind || 'manual' }}</n-text>
                                </n-space>
                                <n-text depth="3" style="font-size:11px;">{{ shortTime(run.started_at || run.created_at) }}</n-text>
                              </n-space>
                              <n-space :size="8" wrap style="font-size:11px;">
                                <n-text depth="3">开始 {{ shortTime(run.started_at || run.created_at) }}</n-text>
                                <n-text depth="3">结束 {{ shortTime(run.finished_at || run.updated_at) || '进行中' }}</n-text>
                                <n-text depth="3">执行者 {{ formatRunActor(run) }}</n-text>
                              </n-space>
                            </n-space>
                          </n-list-item>
                        </n-list>
                      </n-card>
                    </n-gi>
                  </n-grid>
                  <n-card v-if="selectedRun" size="small" title="Run 详情" style="margin-top:12px;">
                    <template #header-extra>
                      <n-space :size="4" align="center">
                        <n-tooltip>
                          <template #trigger>
                            <n-tag size="small">{{ selectedRun.id || selectedRun.run_id }}</n-tag>
                          </template>
                          <span>Run 的唯一 ID</span>
                        </n-tooltip>
                        <n-tooltip>
                          <template #trigger>
                            <n-tag :type="selectedRun.status==='running' ? 'info' : 'default'" size="small" :bordered="false">
                              {{ selectedRun.result || selectedRun.status || 'run' }}
                            </n-tag>
                          </template>
                          <span>Run 的状态或结果</span>
                        </n-tooltip>
                      </n-space>
                    </template>
                    <n-spin :show="runDetailLoading">
                      <n-grid :cols="2" :x-gap="8" style="margin-bottom:12px;">
                        <n-gi>
                          <n-card size="small" title="输入摘要">
                            <div style="font-size:12px;line-height:1.7;white-space:pre-wrap;">{{ selectedRun.input_summary || '无' }}</div>
                          </n-card>
                        </n-gi>
                        <n-gi>
                          <n-card size="small" title="输出摘要">
                            <div style="font-size:12px;line-height:1.7;white-space:pre-wrap;">{{ selectedRun.output_preview || selectedRun.error_text || '无' }}</div>
                          </n-card>
                        </n-gi>
                      </n-grid>
                      <n-space :size="8" wrap style="margin-bottom:12px;">
                        <n-tooltip><template #trigger><n-tag size="small">触发 {{ selectedRun.trigger_kind || 'manual' }}</n-tag></template><span>Run 的触发来源</span></n-tooltip>
                        <n-tooltip><template #trigger><n-tag size="small">开始 {{ shortTime(selectedRun.started_at || selectedRun.created_at) }}</n-tag></template><span>Run 开始执行的时间</span></n-tooltip>
                        <n-tooltip><template #trigger><n-tag size="small">结束 {{ shortTime(selectedRun.finished_at || selectedRun.updated_at) || '进行中' }}</n-tag></template><span>Run 结束时间；未结束时显示进行中</span></n-tooltip>
                        <n-tooltip><template #trigger><n-tag size="small">执行者 {{ formatRunActor(selectedRun) }}</n-tag></template><span>执行该 Run 的 actor 信息</span></n-tooltip>
                      </n-space>
                      <n-card size="small" title="日志流">
                        <n-empty v-if="selectedRunLogs.length===0" description="暂无日志" size="small" />
                        <n-scrollbar v-else style="max-height:260px;">
                          <n-list size="small">
                            <n-list-item v-for="log in selectedRunLogs" :key="log.id || log.log_id">
                              <n-space vertical :size="4" style="width:100%;">
                                <n-space justify="space-between" style="width:100%;" align="center">
                                  <n-space :size="4" align="center">
                                    <n-tag size="small" type="info" :bordered="false">#{{ log.seq }}</n-tag>
                                    <n-tag size="small">{{ log.kind || 'log' }}</n-tag>
                                  </n-space>
                                  <n-text depth="3" style="font-size:11px;">{{ shortTime(log.created_at) }}</n-text>
                                </n-space>
                                <div v-if="log.content" style="font-size:12px;line-height:1.7;white-space:pre-wrap;">{{ log.content }}</div>
                                <div v-else-if="log.payload" style="font-size:12px;line-height:1.7;white-space:pre-wrap;">{{ JSON.stringify(log.payload, null, 2) }}</div>
                              </n-space>
                            </n-list-item>
                          </n-list>
                        </n-scrollbar>
                      </n-card>
                    </n-spin>
                  </n-card>
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
                          <n-tooltip>
                            <template #trigger>
                              <n-tag :type="statusType(child.status)" size="small" :bordered="false">{{ stateLabel(child.status, child.result) }}</n-tag>
                            </template>
                            <span>子节点状态</span>
                          </n-tooltip>
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

            <!-- Memory Tab -->
            <n-tab-pane name="memory" tab="Memory">
              <n-collapse :default-expanded-names="['node']">
                <n-collapse-item title="节点 Memory" name="node">
                  <template v-if="selectedNode">
                    <div v-if="nodeMemorySummary" class="memory-summary">{{ nodeMemorySummary }}</div>
                    <div v-if="nodeMemorySections.length" class="memory-sections">
                      <div v-for="section in nodeMemorySections" :key="'node-' + section.key" class="memory-section">
                        <n-text depth="3" class="memory-section-title">{{ section.label }}</n-text>
                        <n-list size="small" bordered class="memory-list">
                          <n-list-item v-for="(item, index) in section.items" :key="'node-' + section.key + '-' + index">
                            <div class="memory-item">{{ formatMemoryItem(item) }}</div>
                          </n-list-item>
                        </n-list>
                      </div>
                    </div>
                    <n-input v-model:value="memoryForm.node_note" type="textarea" :rows="2" placeholder="当前节点备注" style="margin-top:8px;" />
                    <n-button size="tiny" type="primary" class="memory-save-btn" :loading="submitting" @click="saveMemoryNote('node')">保存节点备注</n-button>
                  </template>
                  <n-empty v-else description="请先选择节点" size="small" />
                </n-collapse-item>
                <n-collapse-item title="阶段 Memory" name="stage">
                  <template v-if="currentStage">
                    <div v-if="stageMemorySummary" class="memory-summary">{{ stageMemorySummary }}</div>
                    <div v-if="stageMemorySections.length" class="memory-sections">
                      <div v-for="section in stageMemorySections" :key="'stage-' + section.key" class="memory-section">
                        <n-text depth="3" class="memory-section-title">{{ section.label }}</n-text>
                        <n-list size="small" bordered class="memory-list">
                          <n-list-item v-for="(item, index) in section.items" :key="'stage-' + section.key + '-' + index">
                            <div class="memory-item">{{ formatMemoryItem(item) }}</div>
                          </n-list-item>
                        </n-list>
                      </div>
                    </div>
                    <n-input v-model:value="memoryForm.stage_note" type="textarea" :rows="2" placeholder="当前阶段备注" style="margin-top:8px;" />
                    <n-button size="tiny" type="primary" class="memory-save-btn" :loading="submitting" @click="saveMemoryNote('stage')">保存阶段备注</n-button>
                  </template>
                  <n-empty v-else description="尚未进入阶段模式" size="small" />
                </n-collapse-item>
                <n-collapse-item title="任务 Memory" name="task">
                  <div v-if="taskMemorySummary" class="memory-summary">{{ taskMemorySummary }}</div>
                  <div v-if="taskMemorySections.length" class="memory-sections">
                    <div v-for="section in taskMemorySections" :key="'task-' + section.key" class="memory-section">
                      <n-text depth="3" class="memory-section-title">{{ section.label }}</n-text>
                      <n-list size="small" bordered class="memory-list">
                        <n-list-item v-for="(item, index) in section.items" :key="'task-' + section.key + '-' + index">
                          <div class="memory-item">{{ formatMemoryItem(item) }}</div>
                        </n-list-item>
                      </n-list>
                    </div>
                  </div>
                  <n-input v-model:value="memoryForm.task_note" type="textarea" :rows="2" placeholder="人工备注写入 task memory.manual_note_text" style="margin-top:8px;" />
                  <n-button size="tiny" type="primary" class="memory-save-btn" :loading="submitting" @click="saveMemoryNote('task')">保存任务备注</n-button>
                </n-collapse-item>
              </n-collapse>
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
                <n-empty v-if="selectedArtifacts.length===0" description="暂无产物" />
                <n-list v-else size="small">
                  <n-list-item v-for="art in selectedArtifacts" :key="art.id">
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
      <n-tooltip><template #trigger><n-button @click="confirmCloseProgressModal">取消</n-button></template><span>关闭，不保存进度</span></n-tooltip>
      <n-tooltip><template #trigger><n-button type="primary" :loading="submitting" @click="confirmDoProgress">写入进度</n-button></template><span>保存本次进度说明</span></n-tooltip>
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
      <n-tooltip><template #trigger><n-button @click="confirmCloseCompleteModal">取消</n-button></template><span>关闭，不标记完成</span></n-tooltip>
      <n-tooltip><template #trigger><n-button type="success" :loading="submitting" @click="confirmDoComplete">标记完成</n-button></template><span>将节点状态改为完成</span></n-tooltip>
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
      <n-tooltip><template #trigger><n-button @click="confirmCloseNodeCreate">取消</n-button></template><span>关闭，不创建节点</span></n-tooltip>
      <n-tooltip><template #trigger><n-button type="primary" :loading="submitting" @click="confirmDoCreateNode">创建节点</n-button></template><span>保存当前节点配置并创建</span></n-tooltip>
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
      <n-tooltip><template #trigger><n-button @click="confirmCloseSettings">取消</n-button></template><span>关闭，不保存任务设置</span></n-tooltip>
      <n-tooltip><template #trigger><n-button type="primary" :loading="submitting" @click="confirmSaveTask">保存</n-button></template><span>保存任务标题、Key 和目标</span></n-tooltip>
    </template>
  </n-modal>

  <n-modal v-model:show="showStageCreate" preset="card" title="新增阶段" style="max-width:520px;">
    <n-form label-placement="top">
      <n-form-item label="阶段标题">
        <n-input v-model:value="stageForm.title" placeholder="例如：执行层收尾" />
      </n-form-item>
      <n-form-item label="阶段说明">
        <n-input v-model:value="stageForm.instruction" type="textarea" :rows="3" />
      </n-form-item>
      <n-form-item label="创建后立即激活">
        <n-switch v-model:value="stageForm.activate" />
      </n-form-item>
    </n-form>
    <template #action>
      <n-tooltip><template #trigger><n-button @click="confirmCloseStageCreate">取消</n-button></template><span>关闭，不创建阶段</span></n-tooltip>
      <n-tooltip><template #trigger><n-button type="primary" :loading="submitting" @click="confirmDoCreateStage">创建</n-button></template><span>创建新阶段节点</span></n-tooltip>
    </template>
  </n-modal>

  <n-modal v-model:show="showRunStart" preset="card" title="开始 Run" style="max-width:520px;">
    <n-form label-placement="top">
      <n-form-item label="输入摘要">
        <n-input v-model:value="runStartForm.input_summary" type="textarea" :rows="3" placeholder="本次执行准备做什么" />
      </n-form-item>
    </n-form>
    <template #action>
      <n-tooltip><template #trigger><n-button @click="confirmCloseRunStart">取消</n-button></template><span>关闭，不创建 Run</span></n-tooltip>
      <n-tooltip><template #trigger><n-button type="primary" :loading="submitting" @click="confirmDoStartRun">开始</n-button></template><span>创建新的运行记录</span></n-tooltip>
    </template>
  </n-modal>

  <n-modal v-model:show="showRunFinish" preset="card" title="结束 Run" style="max-width:520px;">
    <n-form label-placement="top">
      <n-form-item label="结果">
        <n-radio-group v-model:value="runFinishForm.result">
          <n-space>
            <n-radio-button value="done">完成</n-radio-button>
            <n-radio-button value="canceled">取消</n-radio-button>
            <n-radio-button value="">仅结束 Run</n-radio-button>
          </n-space>
        </n-radio-group>
      </n-form-item>
      <n-form-item label="输出摘要">
        <n-input v-model:value="runFinishForm.output_preview" type="textarea" :rows="3" placeholder="这次执行产出了什么" />
      </n-form-item>
      <n-form-item label="错误信息">
        <n-input v-model:value="runFinishForm.error_text" type="textarea" :rows="2" placeholder="可选" />
      </n-form-item>
    </n-form>
    <template #action>
      <n-tooltip><template #trigger><n-button @click="confirmCloseRunFinish">取消</n-button></template><span>关闭，不结束当前 Run</span></n-tooltip>
      <n-tooltip><template #trigger><n-button type="primary" :loading="submitting" @click="confirmDoFinishRun">结束</n-button></template><span>结束当前 Run 并写入结果</span></n-tooltip>
    </template>
  </n-modal>

  <n-modal v-model:show="showRunLog" preset="card" title="追加 Run 日志" style="max-width:520px;">
    <n-form label-placement="top">
      <n-form-item label="日志类型">
        <n-input v-model:value="runLogForm.kind" placeholder="例如：note / command / output" />
      </n-form-item>
      <n-form-item label="内容">
        <n-input v-model:value="runLogForm.content" type="textarea" :rows="3" />
      </n-form-item>
    </n-form>
    <template #action>
      <n-tooltip><template #trigger><n-button @click="confirmCloseRunLog">取消</n-button></template><span>关闭，不追加日志</span></n-tooltip>
      <n-tooltip><template #trigger><n-button type="primary" :loading="submitting" @click="confirmDoAddRunLog">追加</n-button></template><span>给当前 Run 追加一条日志</span></n-tooltip>
    </template>
  </n-modal>
</template>

<script setup>
import { ref, computed, watch, onMounted, h, reactive, inject, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { NTag, NRadioGroup, NRadioButton } from 'naive-ui'
import {
  api,
  statusType,
  statusLabel,
  stateLabel,
  pct,
  shortTime,
  excerpt,
  eventTypeLabel,
  leaseActive,
  nodePct as nodePctCalc,
  buildTreeData,
  nodeTemplates,
  normalizeNode,
  normalizeNodeList,
  fetchTaskResume,
  fetchNodeContext,
  fetchProjectOverview,
  patchTaskMemory,
  patchStageMemory,
  patchNodeMemory,
  listNodeRuns,
  fetchRun,
  createTaskArtifact,
  listAllNodes,
  listStages,
  createStage,
  activateStage,
  startNodeRun,
  finishRun,
  addRunLog,
  dirtyTargets,
} from '../api.js'

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
const taskMemory = ref(null)
const currentStage = ref(null)
const currentStageMemory = ref(null)
const recentRuns = ref([])
const stages = ref([])
const treeMode = ref('focus')
const summaryMode = ref(true)
const selectedNodeId = ref('')
const selectedNode = ref(null)
const selectedChildren = ref([])
const selectedDescendantLeaves = ref([])
const nodeEvents = ref([])
const nodeArtifacts = ref([])
const nodeRuns = ref([])
const selectedNodeMemory = ref(null)
const selectedStageSummary = ref(null)
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
const showStageCreate = ref(false)
const showRunStart = ref(false)
const showRunFinish = ref(false)
const showRunLog = ref(false)
const nodeCreateParent = ref('')

// Forms
const editForm = reactive({ title: '', estimate: null, instruction: '', acceptance: '' })
const progressForm = reactive({ delta: 0.1, targetPct: null, message: '' })
const completeForm = reactive({ message: '' })
const newNodeForm = reactive({ title: '', node_key: '', estimate: null, instruction: '', acceptance: '', template: null })
const settingsForm = reactive({ title: '', task_key: '', goal: '' })
const artifactForm = reactive({ title: '', kind: 'link', uri: '' })
const stageForm = reactive({ title: '', instruction: '', activate: true })
const runStartForm = reactive({ input_summary: '' })
const runFinishForm = reactive({ result: 'done', output_preview: '', error_text: '' })
const runLogForm = reactive({ kind: 'note', content: '' })
const memoryForm = reactive({ task_note: '', stage_note: '', node_note: '' })
const selectedRunId = ref('')
const selectedRunDetail = ref(null)
const runDetailLoading = ref(false)

const templateOpts = [
  { label: '无模板', value: null },
  { label: '开发模板', value: 'dev' },
  { label: '测试模板', value: 'test' },
  { label: '验收模板', value: 'accept' },
]

const taskPct = computed(() => task.value ? pct(task.value.summary_percent) : 0)
const treeData = computed(() => buildTreeData(nodes.value))
const selectedArtifacts = computed(() => nodeArtifacts.value.length ? nodeArtifacts.value : artifacts.value)
const taskSummaryText = computed(() => excerpt(taskMemory.value?.summary || task.value?.goal || '', 140))
const stageSummaryText = computed(() => excerpt(currentStageMemory.value?.summary || currentStage.value?.title || '', 120))
const selectedMemoryText = computed(() => excerpt(selectedNodeMemory.value?.summary || selectedNode.value?.instruction || '', 160))
const activeRun = computed(() => nodeRuns.value.find(run => run.status === 'running') || null)
const selectedRun = computed(() => {
  if (!selectedRunId.value) return null
  if (selectedRunDetail.value && (selectedRunDetail.value.id === selectedRunId.value || selectedRunDetail.value.run_id === selectedRunId.value)) {
    return selectedRunDetail.value
  }
  return nodeRuns.value.find(run => (run.id || run.run_id) === selectedRunId.value) || null
})
const selectedRunLogs = computed(() => selectedRun.value?.logs || [])
const taskMemorySummary = computed(() => memorySummary(taskMemory.value))
const stageMemorySummary = computed(() => memorySummary(currentStageMemory.value))
const nodeMemorySummary = computed(() => memorySummary(selectedNodeMemory.value))
const taskMemorySections = computed(() => buildMemorySections(taskMemory.value))
const stageMemorySections = computed(() => buildMemorySections(currentStageMemory.value))
const nodeMemorySections = computed(() => buildMemorySections(selectedNodeMemory.value))
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

function applyTaskSnapshot(taskInfo = {}) {
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
  settingsForm.title = task.value.title || ''
  settingsForm.task_key = task.value.task_key || ''
  settingsForm.goal = task.value.goal || ''
}

function updateSelectedRelationships(nodeId) {
  selectedChildren.value = nodes.value.filter(n => n.parent_node_id === nodeId)
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
}

async function applyBreadcrumb() {
  const st = window.history.state || {}
  const crumb = [{ label: '任务总览', path: '/' }]
  if (st.projectId && st.projectName) {
    crumb.push({ label: st.projectName, path: '/projects/' + st.projectId })
  } else if (task.value?.project_id) {
    try {
      const overview = await fetchProjectOverview(task.value.project_id)
      crumb.push({ label: overview.project?.name || '项目', path: '/projects/' + task.value.project_id })
    } catch {
      try {
        const project = await api('/projects/' + task.value.project_id)
        crumb.push({ label: project.name || '项目', path: '/projects/' + task.value.project_id })
      } catch {}
    }
  }
  crumb.push({ label: task.value?.title || '任务详情' })
  breadcrumb.value = crumb
}

async function loadStages() {
  try {
    stages.value = (await listStages(props.id)).map(normalizeNode)
  } catch {
    stages.value = []
  }
}

function syncMemoryForms() {
  memoryForm.task_note = taskMemory.value?.manual_note_text || ''
  memoryForm.stage_note = currentStageMemory.value?.manual_note_text || ''
  memoryForm.node_note = selectedNodeMemory.value?.manual_note_text || ''
}

function memorySummary(memory) {
  if (!memory) return ''
  return memory.summary_text || memory.summary || ''
}

function normalizeMemoryItems(items) {
  if (!Array.isArray(items)) return []
  return items.filter(item => {
    if (item == null) return false
    if (typeof item === 'string') return item.trim() !== ''
    return true
  })
}

function buildMemorySections(memory) {
  if (!memory) return []
  const specs = [
    { key: 'conclusions', label: '结论' },
    { key: 'decisions', label: '决策' },
    { key: 'risks', label: '风险' },
    { key: 'blockers', label: '阻塞' },
    { key: 'next_actions', label: '下一步' },
    { key: 'evidence', label: '证据' },
  ]
  return specs
    .map(spec => ({ ...spec, items: normalizeMemoryItems(memory[spec.key]) }))
    .filter(section => section.items.length > 0)
}

function formatMemoryItem(item) {
  if (typeof item === 'string') return item
  return JSON.stringify(item, null, 2)
}

function formatRunActor(run) {
  const parts = [run?.actor_type, run?.actor_id].filter(Boolean)
  return parts.length ? parts.join('/') : '未知'
}

async function selectRun(runId) {
  if (!runId) {
    selectedRunId.value = ''
    selectedRunDetail.value = null
    return
  }
  selectedRunId.value = runId
  runDetailLoading.value = true
  try {
    selectedRunDetail.value = await fetchRun(runId)
  } catch (e) {
    selectedRunDetail.value = nodeRuns.value.find(run => (run.id || run.run_id) === runId) || null
    window.$message?.error('加载 Run 详情失败: ' + e.message)
  } finally {
    runDetailLoading.value = false
  }
}

async function onTreeSelect(keys) {
  if (keys.length > 0) await selectNode(keys[0])
}

async function selectNode(nodeId, opts = {}) {
  if (!nodeId) return
  selectedNodeId.value = nodeId
  try {
    const fallbackNode = normalizeNode((!opts.forceFetch && (opts.node || nodes.value.find(n => n.id === nodeId))) || null)
    const context = await fetchNodeContext(nodeId).catch(() => null)
    const contextNode = normalizeNode(context?.node || null)
    const node = contextNode || fallbackNode || normalizeNode(await api('/nodes/' + nodeId))
    selectedNode.value = node
    editForm.title = node.title || ''
    editForm.estimate = node.estimate || null
    editForm.instruction = node.instruction || ''
    editForm.acceptance = (node.acceptance_criteria || []).join('\n')
    updateSelectedRelationships(nodeId)
    selectedNodeMemory.value = context?.memory || null
    selectedStageSummary.value = context?.stage_summary || null
    nodeRuns.value = context?.recent_runs || await listNodeRuns(nodeId, 10).catch(() => [])
    nodeArtifacts.value = context?.artifacts || []
    const initialRunId = activeRun.value?.id || activeRun.value?.run_id || nodeRuns.value[0]?.id || nodeRuns.value[0]?.run_id || ''
    if (initialRunId) await selectRun(initialRunId)
    else {
      selectedRunId.value = ''
      selectedRunDetail.value = null
    }
    if (selectedStageSummary.value?.memory && currentStage.value?.id === selectedStageSummary.value?.stage?.id) {
      currentStageMemory.value = selectedStageSummary.value.memory
    }
    syncMemoryForms()
    if (!opts.forceFetch && Array.isArray(opts.events)) nodeEvents.value = opts.events
    else nodeEvents.value = context?.recent_events || []
  } catch (e) {
    window.$message?.error('加载节点失败: ' + e.message)
  }
}

async function load() {
  loading.value = true
  try {
    activeTab.value = typeof route.query.tab === 'string' ? route.query.tab : 'node'
    const resume = await fetchTaskResume(props.id, {
      limit: treeMode.value === 'focus' ? 200 : 10000,
      include_full_tree: treeMode.value === 'full',
      debug: 1,
    })
    applyTaskSnapshot(resume.task || {})
    taskMemory.value = resume.task_memory || null
    currentStage.value = normalizeNode(resume.current_stage || null)
    currentStageMemory.value = resume.current_stage_memory || null
    recentRuns.value = resume.recent_runs || []
    await loadStages()
    if (treeMode.value === 'full') {
      const allNodesResult = await listAllNodes(props.id)
      nodes.value = normalizeNodeList(allNodesResult.items || allNodesResult || [])
    } else {
      nodes.value = normalizeNodeList(resume.tree || [])
    }
    syncMemoryForms()
    await applyBreadcrumb()
    remaining.value = resume.remaining?.remaining_nodes || 0
    blockCount.value = resume.remaining?.blocked_nodes || 0
    pausedCount.value = resume.remaining?.paused_nodes || 0
    canceledCount.value = resume.remaining?.canceled_nodes || 0
    estimate.value = ((resume.remaining?.remaining_estimate || 0)).toFixed(1) + 'h'
    events.value = resume.recent_events || []
    artifacts.value = resume.artifacts || []
    favorites.value = loadFavs()

    // Init expanded keys (all non-leaf nodes for full tree visibility)
    const parentIds = new Set(nodes.value.filter(n => n.parent_node_id).map(n => n.parent_node_id))
    if (expandedKeys.value.length === 0) expandedKeys.value = [...parentIds]

    // Select next node or first
    if (resume.next_node?.node) {
      nextNode.value = normalizeNode(resume.next_node.node)
      const nextId = nextNode.value.id
      const node = nodes.value.find(n => n.id === nextId)
      const selectOpts = { events: resume.next_node.recent_events || [] }
      if (node) selectOpts.node = node
      await selectNode(nextId, selectOpts)
    } else if (typeof route.query.node === 'string' && route.query.node) {
      await selectNode(route.query.node)
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
function scheduleRefresh(mode = 'resume') {
  if (refreshTimer) return
  refreshTimer = setTimeout(async () => {
    refreshTimer = null
    try {
      if (mode === 'node' && selectedNodeId.value) {
        await selectNode(selectedNodeId.value, { forceFetch: true })
        return
      }
      await load()
    } catch {}
  }, 600)
}

function toggleTreeMode() {
  treeMode.value = treeMode.value === 'focus' ? 'full' : 'focus'
  load()
}

async function doCreateStage() {
  if (!stageForm.title.trim()) {
    window.$message?.warning('请填写阶段标题')
    return
  }
  submitting.value = true
  try {
    await createStage(props.id, {
      title: stageForm.title,
      instruction: stageForm.instruction || undefined,
      activate: stageForm.activate,
    })
    showStageCreate.value = false
    stageForm.title = ''
    stageForm.instruction = ''
    stageForm.activate = true
    window.$message?.success('阶段已创建')
    await load()
  } catch (e) {
    const text = String(e.message || '')
    if (text.toLowerCase().includes('version') || text.includes('并发')) {
      window.$message?.error('保存失败：数据已被其他更新覆盖，请刷新后重试')
    } else {
      window.$message?.error(text)
    }
  } finally {
    submitting.value = false
  }
}

async function doActivateStage(stageId) {
  try {
    await activateStage(props.id, stageId)
    window.$message?.success('阶段已激活')
    await load()
  } catch (e) {
    window.$message?.error(e.message)
  }
}

async function doStartRun() {
  if (!selectedNodeId.value) return
  submitting.value = true
  try {
    await startNodeRun(selectedNodeId.value, {
      actor: { tool: 'web_ui' },
      input_summary: runStartForm.input_summary || undefined,
    })
    showRunStart.value = false
    runStartForm.input_summary = ''
    window.$message?.success('Run 已开始')
    await selectNode(selectedNodeId.value, { forceFetch: true })
    scheduleRefresh('resume')
  } catch (e) {
    window.$message?.error(e.message)
  } finally {
    submitting.value = false
  }
}

async function doFinishRun() {
  if (!activeRun.value) return
  submitting.value = true
  try {
    await finishRun(activeRun.value.id || activeRun.value.run_id, {
      result: runFinishForm.result || undefined,
      output_preview: runFinishForm.output_preview || undefined,
      error_text: runFinishForm.error_text || undefined,
    })
    showRunFinish.value = false
    runFinishForm.result = 'done'
    runFinishForm.output_preview = ''
    runFinishForm.error_text = ''
    window.$message?.success('Run 已结束')
    await load()
  } catch (e) {
    window.$message?.error(e.message)
  } finally {
    submitting.value = false
  }
}

async function doAddRunLog() {
  if (!activeRun.value) return
  submitting.value = true
  try {
    await addRunLog(activeRun.value.id || activeRun.value.run_id, {
      kind: runLogForm.kind,
      content: runLogForm.content || undefined,
    })
    showRunLog.value = false
    runLogForm.kind = 'note'
    runLogForm.content = ''
    window.$message?.success('日志已追加')
    await selectNode(selectedNodeId.value, { forceFetch: true })
  } catch (e) {
    window.$message?.error(e.message)
  } finally {
    submitting.value = false
  }
}

async function saveMemoryNote(level) {
  submitting.value = true
  try {
    if (level === 'task') {
      taskMemory.value = await patchTaskMemory(props.id, {
        manual_note_text: memoryForm.task_note,
        expected_version: taskMemory.value?.version,
      })
    } else if (level === 'stage') {
      if (!currentStage.value?.id) return
      currentStageMemory.value = await patchStageMemory(currentStage.value.id, {
        manual_note_text: memoryForm.stage_note,
        expected_version: currentStageMemory.value?.version,
      })
    } else {
      if (!selectedNodeId.value) return
      selectedNodeMemory.value = await patchNodeMemory(selectedNodeId.value, {
        manual_note_text: memoryForm.node_note,
        expected_version: selectedNodeMemory.value?.version,
      })
    }
    syncMemoryForms()
    window.$message?.success('备注已保存')
    scheduleRefresh('resume')
  } catch (e) {
    const text = String(e.message || '')
    if (text.toLowerCase().includes('version') || text.includes('并发')) {
      window.$message?.error('保存失败：数据已被其他更新覆盖，请刷新后重试')
    } else {
      window.$message?.error(text)
    }
  } finally {
    submitting.value = false
  }
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

      // 2. Dirty envelope 优先做局部刷新，回退时再整页刷新
      const dirty = dirtyTargets(ev)
      if (dirty.length) {
        if (dirty.includes('events') && selectedNodeId.value) {
          const nodeIds = new Set([selectedNodeId.value, ...selectedDescendantLeaves.value.map(n => n.id)])
          if (nodeIds.has(ev.node_id) && !nodeEvents.value.some(x => x.id === ev.id)) {
            nodeEvents.value = [ev, ...nodeEvents.value]
          }
        }
        if (dirty.includes('node') && selectedNodeId.value === ev.node_id) {
          scheduleRefresh('node')
        }
        if (dirty.includes('resume') || dirty.includes('task') || dirty.includes('runs') || dirty.includes('artifacts')) {
          scheduleRefresh('resume')
        }
        return
      }

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

function askConfirm({ title, content, type = 'warning', onConfirm }) {
  const dialogFn = type === 'error' ? 'error' : type === 'info' ? 'info' : 'warning'
  window.$dialog?.[dialogFn]({
    title,
    content,
    positiveText: '确认',
    negativeText: '取消',
    onPositiveClick: onConfirm,
  })
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

function confirmClaimNode() {
  askConfirm({ title: '领取节点', content: '确认领取当前节点吗？', type: 'info', onConfirm: claimNode })
}

function confirmReleaseNode() {
  askConfirm({ title: '释放节点', content: '确认释放当前节点吗？', type: 'warning', onConfirm: releaseNode })
}

function confirmBlockNode() {
  askConfirm({ title: '阻塞节点', content: '确认将当前节点标记为阻塞吗？', type: 'warning', onConfirm: blockNode })
}

function confirmNodeTransition(action) {
  const titles = { unblock: '解除阻塞', pause: '暂停节点', reopen: '重开节点', cancel: '取消节点' }
  const contents = {
    unblock: '确认解除当前节点的阻塞状态吗？',
    pause: '确认暂停当前节点吗？',
    reopen: '确认重新打开当前节点吗？',
    cancel: '确认取消当前节点吗？',
  }
  askConfirm({ title: titles[action] || '确认操作', content: contents[action] || '确认执行该操作吗？', type: action === 'cancel' ? 'error' : 'warning', onConfirm: () => nodeTransition(action) })
}

function confirmRetypeNode() {
  askConfirm({ title: '转回执行节点', content: '确认把当前分组节点转回执行节点吗？', type: 'warning', onConfirm: retypeNode })
}

function confirmOpenNodeCreate(parentId) {
  askConfirm({ title: '新增节点', content: '确认打开新增节点窗口吗？', type: 'info', onConfirm: () => openNodeCreate(parentId) })
}

function confirmOpenRunStart() {
  askConfirm({ title: '开始 Run', content: '确认开始新的 Run 记录吗？', type: 'info', onConfirm: () => { showRunStart.value = true } })
}

function confirmOpenRunFinish() {
  askConfirm({ title: '结束 Run', content: '确认结束当前 Run 记录吗？', type: 'warning', onConfirm: () => { showRunFinish.value = true } })
}

function confirmOpenRunLog() {
  askConfirm({ title: '追加日志', content: '确认要给当前 Run 追加日志吗？', type: 'info', onConfirm: () => { showRunLog.value = true } })
}

function confirmOpenProgressModal() {
  askConfirm({ title: '上报进度', content: '确认打开进度上报窗口吗？', type: 'info', onConfirm: () => { showProgressModal.value = true } })
}

function confirmOpenCompleteModal() {
  askConfirm({ title: '完成节点', content: '确认打开完成节点窗口吗？', type: 'warning', onConfirm: () => { showCompleteModal.value = true } })
}

function confirmCloseProgressModal() { askConfirm({ title: '关闭窗口', content: '确认关闭进度窗口吗？未填写内容会丢失。', onConfirm: () => { showProgressModal.value = false } }) }
function confirmDoProgress() { askConfirm({ title: '写入进度', content: '确认提交本次进度吗？', onConfirm: doProgress }) }
function confirmCloseCompleteModal() { askConfirm({ title: '关闭窗口', content: '确认关闭完成窗口吗？未填写内容会丢失。', onConfirm: () => { showCompleteModal.value = false } }) }
function confirmDoComplete() { askConfirm({ title: '标记完成', content: '确认将当前节点标记完成吗？', type: 'warning', onConfirm: doComplete }) }
function confirmCloseNodeCreate() { askConfirm({ title: '关闭窗口', content: '确认关闭创建节点窗口吗？未填写内容会丢失。', onConfirm: () => { showNodeCreate.value = false } }) }
function confirmDoCreateNode() { askConfirm({ title: '创建节点', content: '确认创建这个节点吗？', onConfirm: doCreateNode }) }
function confirmCloseSettings() { askConfirm({ title: '关闭窗口', content: '确认关闭任务设置窗口吗？未保存内容会丢失。', onConfirm: () => { showSettings.value = false } }) }
function confirmSaveTask() { askConfirm({ title: '保存任务', content: '确认保存任务设置吗？', onConfirm: saveTask }) }
function confirmCloseStageCreate() { askConfirm({ title: '关闭窗口', content: '确认关闭阶段创建窗口吗？未填写内容会丢失。', onConfirm: () => { showStageCreate.value = false } }) }
function confirmDoCreateStage() { askConfirm({ title: '创建阶段', content: '确认创建这个阶段吗？', onConfirm: doCreateStage }) }
function confirmCloseRunStart() { askConfirm({ title: '关闭窗口', content: '确认关闭 Run 创建窗口吗？未填写内容会丢失。', onConfirm: () => { showRunStart.value = false } }) }
function confirmDoStartRun() { askConfirm({ title: '开始 Run', content: '确认创建新的 Run 记录吗？', onConfirm: doStartRun }) }
function confirmCloseRunFinish() { askConfirm({ title: '关闭窗口', content: '确认关闭 Run 结束窗口吗？未填写内容会丢失。', onConfirm: () => { showRunFinish.value = false } }) }
function confirmDoFinishRun() { askConfirm({ title: '结束 Run', content: '确认结束当前 Run 并写入结果吗？', onConfirm: doFinishRun }) }
function confirmCloseRunLog() { askConfirm({ title: '关闭窗口', content: '确认关闭 Run 日志窗口吗？未填写内容会丢失。', onConfirm: () => { showRunLog.value = false } }) }
function confirmDoAddRunLog() { askConfirm({ title: '追加日志', content: '确认追加这条 Run 日志吗？', onConfirm: doAddRunLog }) }

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
    await createTaskArtifact(props.id, body)
    artifactForm.title = ''; artifactForm.uri = ''
    window.$message?.success('已添加')
    scheduleRefresh('resume')
  } catch (e) { window.$message?.error(e.message) }
}

function onUploadFinish() {
  window.$message?.success('上传成功')
  scheduleRefresh('resume')
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
const stopWatchTaskId = watch(() => props.id, (n, o) => { if (n && n !== o) { expandedKeys.value = []; load() } })
const stopWatchRouteNode = watch(() => route.query.node, (nodeId) => {
  if (typeof nodeId === 'string' && nodeId && nodeId !== selectedNodeId.value) {
    selectNode(nodeId)
  }
})
onUnmounted(() => {
  stopWatchTaskId()
  stopWatchRouteNode()
  if (eventSource) { eventSource.close(); eventSource = null }
})
</script>

<style scoped>
.memory-card :deep(.n-card-header) {
  padding-bottom: 8px;
}

.memory-card :deep(.n-card__content) {
  padding-top: 10px;
  padding-bottom: 10px;
}

.memory-summary {
  font-size: 11px;
  line-height: 1.55;
  white-space: pre-wrap;
  margin-bottom: 6px;
}

.memory-sections {
  margin-bottom: 6px;
}

.memory-section {
  margin-bottom: 6px;
}

.memory-section-title {
  font-size: 10px;
  font-weight: 600;
}

.memory-list {
  margin-top: 3px;
}

.memory-item {
  font-size: 11px;
  line-height: 1.55;
  white-space: pre-wrap;
}

.memory-save-btn {
  margin-top: 6px;
}
</style>
