<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../services/api'
import { useWorkspaceStore } from '../stores/workspace'

const props = defineProps<{ modelValue: boolean; initialCode?: string }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const workspace = useWorkspaceStore()
const code = ref('')
const loading = ref(false)

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

watch(visible, (v) => {
  if (v) {
    code.value = props.initialCode || ''
  }
})

function guessNameFromUrl(url: string): string {
  try {
    const u = new URL(url)
    const segments = u.pathname.split('/').filter(Boolean)
    if (segments.length > 0) {
      return segments[segments.length - 1]
    }
    return u.hostname || '未命名'
  } catch {
    const segments = url.split('/').filter(Boolean)
    return segments.length > 0 ? segments[segments.length - 1] : '未命名'
  }
}

async function parse() {
  if (!code.value.trim()) return
  loading.value = true
  try {
    const result = await api.parseFetch(code.value)
    const name = guessNameFromUrl(result.url)
    const headers = Object.entries(result.headers || {}).map(([key, value]) => ({
      key,
      value,
      enabled: true,
      description: '',
    }))
    const query = (result.query || []).map((p) => ({
      key: p.key,
      value: p.value,
      enabled: true,
      description: '',
    }))
    await workspace.createRequest('', name, 'http')
    const tab = workspace.activeTab
    if (tab) {
      workspace.updateDraft(tab.rel, {
        name,
        url: result.url,
        method: result.method,
        headers,
        query,
        body: result.body,
        bodyType: guessBodyType(result.body),
      })
    }
    ElMessage.success('导入成功')
    visible.value = false
  } catch (err) {
    ElMessage.error(`解析失败: ${err}`)
  } finally {
    loading.value = false
  }
}

function guessBodyType(body: string): 'none' | 'json' | 'text' {
  if (!body) return 'none'
  try {
    JSON.parse(body)
    return 'json'
  } catch {
    return 'text'
  }
}
</script>

<template>
  <el-dialog v-model="visible" title="从 Fetch API 导入" width="680px">
    <el-input
      v-model="code"
      type="textarea"
      :rows="12"
      placeholder="粘贴 fetch(...) 代码"
    />
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="loading" @click="parse">导入</el-button>
    </template>
  </el-dialog>
</template>

<style scoped></style>
