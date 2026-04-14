<template>
  <n-card size="small">
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
          <n-button type="primary" size="small" @click="emit('create-artifact')">添加链接</n-button>
        </n-form>
        <n-divider />
        <n-upload :action="uploadAction" :data="{ node_id: selectedNodeId || '' }" name="file" @finish="value => emit('upload-finish', value)">
          <n-button size="small">上传文件</n-button>
        </n-upload>
      </n-collapse-item>
    </n-collapse>
    <n-empty v-if="selectedArtifacts.length === 0" description="暂无产物" />
    <n-list v-else size="small">
      <n-list-item v-for="art in selectedArtifacts" :key="art.id">
        <n-thing :title="art.title || art.id" :description="art.uri" content-style="font-size:12px;">
          <template #header-extra>
            <n-space :size="4">
              <n-tag size="small">{{ art.kind || 'link' }}</n-tag>
              <n-tag size="small">{{ shortTime(art.created_at) }}</n-tag>
              <n-button
                v-if="(art.uri || '').startsWith('local://')"
                size="tiny"
                type="primary"
                quaternary
                tag="a"
                :href="'/v1/artifacts/' + art.id + '/download'"
                target="_blank"
              >
                下载
              </n-button>
            </n-space>
          </template>
        </n-thing>
      </n-list-item>
    </n-list>
  </n-card>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  selectedArtifacts: { type: Array, default: () => [] },
  artifactForm: { type: Object, required: true },
  taskId: { type: String, required: true },
  selectedNodeId: { type: String, default: '' },
  shortTime: { type: Function, required: true },
})

const emit = defineEmits(['create-artifact', 'upload-finish'])

const uploadAction = computed(() => `/v1/tasks/${props.taskId}/artifacts/upload`)
</script>
