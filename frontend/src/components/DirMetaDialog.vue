<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import { api } from '../services/api'
import type { DirMeta } from '../types'
import KeyValueEditor from './common/KeyValueEditor.vue'

const props = defineProps<{ modelValue: boolean; rel: string }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const visible = ref(false)
const meta = ref<DirMeta>({
  id: '',
  name: '',
  headers: [],
  variables: [],
})
const loading = ref(false)

async function load() {
  if (!props.rel) return
  loading.value = true
  try {
    const data = await api.loadDirMeta(props.rel)
    meta.value = {
      id: data.id || '',
      name: data.name || '',
      headers: (data.headers || []).map((h: any) => ({ ...h, enabled: h.enabled !== false })),
      variables: data.variables || [],
    } as DirMeta
  } catch (err) {
    ElMessage.error(`加载目录元数据失败: ${err}`)
  } finally {
    loading.value = false
  }
}

async function save() {
  try {
    await api.saveDirMeta(props.rel, meta.value as any)
    ElMessage.success('目录设置已保存')
    emit('update:modelValue', false)
  } catch (err) {
    ElMessage.error(`保存失败: ${err}`)
  }
}

watch(
  () => props.modelValue,
  (v) => {
    visible.value = v
    if (v) load()
  }
)

watch(visible, (v) => emit('update:modelValue', v))
</script>

<template>
  <el-dialog v-model="visible" title="目录设置" width="640px">
    <div v-if="loading" class="loading-text">加载中...</div>
    <div v-else class="dir-meta-form">
      <el-tabs type="border-card">
        <el-tab-pane label="请求头">
          <KeyValueEditor v-model="meta.headers" />
        </el-tab-pane>
        <el-tab-pane label="变量">
          <KeyValueEditor
            :model-value="meta.variables as any"
            value-label="Value"
            show-description
            @update:model-value="meta.variables = $event as any"
          />
        </el-tab-pane>
      </el-tabs>
    </div>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" @click="save">保存</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.loading-text {
  padding: 20px;
  text-align: center;
  color: var(--app-text-secondary);
}

.dir-meta-form {
  min-height: 240px;
}
</style>
