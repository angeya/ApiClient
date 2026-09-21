<script setup lang="ts">
import { computed } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import type { Request } from '../../types'
import FormDataEditor from './FormDataEditor.vue'
import CodeEditor from '../common/CodeEditor.vue'

const props = defineProps<{ modelValue: Request }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: Request): void }>()

const req = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const bodyTypes = [
  { label: 'none', value: 'none' },
  { label: 'json', value: 'json' },
  { label: 'text', value: 'text' },
  { label: 'form-data', value: 'form-data' },
  { label: 'x-www-form-urlencoded', value: 'x-www-form-urlencoded' },
]

const editorLanguage = computed(() => req.value.bodyType === 'json' ? 'json' : 'text')

/** 从 JSON.parse 的错误信息里提取行列，给出更友好的提示 */
function describeJsonError(err: unknown): string {
  const msg = err instanceof Error ? err.message : String(err)
  const posMatch = /position (\d+)/.exec(msg)
  const lineMatch = /line (\d+) column (\d+)/.exec(msg)
  if (lineMatch) return `第 ${lineMatch[1]} 行第 ${lineMatch[2]} 列语法错误`
  if (posMatch) {
    const text = props.modelValue.body ?? ''
    const pos = Math.min(Number(posMatch[1]), text.length)
    const line = text.slice(0, pos).split('\n').length
    const column = pos - (text.lastIndexOf('\n', pos - 1) + 1) + 1
    return `第 ${line} 行第 ${column} 列语法错误`
  }
  return msg
}

function formatJson() {
  const text = props.modelValue.body ?? ''
  if (!text.trim()) {
    ElMessage.warning('内容为空，无需格式化')
    return
  }
  try {
    const formatted = JSON.stringify(JSON.parse(text), null, 2)
    if (formatted === text) {
      ElMessage.info('JSON 已是格式化状态')
      return
    }
    req.value = { ...props.modelValue, body: formatted }
    ElMessage.success('JSON 已格式化')
  } catch (err) {
    ElMessage.error({
      message: `JSON 格式错误：${describeJsonError(err)}`,
      duration: 4000,
      showClose: true,
    })
  }
}
</script>

<template>
  <div class="body-editor">
    <div class="body-type-bar">
      <el-radio-group v-model="req.bodyType" size="default">
        <el-radio-button v-for="t in bodyTypes" :key="t.value" :label="t.value">
          {{ t.label }}
        </el-radio-button>
      </el-radio-group>

      <div class="bar-actions">
        <el-button
          v-if="req.bodyType === 'json'"
          class="format-btn"
          size="small"
          title="格式化 JSON（缩进 2 个空格）"
          @click="formatJson"
        >
          <el-icon><MagicStick /></el-icon>
          <span>格式化</span>
        </el-button>
      </div>
    </div>

    <div v-if="req.bodyType === 'json' || req.bodyType === 'text'" class="body-text">
      <CodeEditor
        v-model="req.body"
        :language="editorLanguage"
        :placeholder="req.bodyType === 'json' ? '请输入 JSON' : '请输入文本'"
      />
    </div>

    <div v-else-if="req.bodyType === 'form-data'" class="form-data">
      <FormDataEditor v-model="req.formData" show-type />
    </div>

    <div v-else-if="req.bodyType === 'x-www-form-urlencoded'" class="form-data">
      <FormDataEditor v-model="req.formData" />
    </div>
  </div>
</template>

<style scoped>
.body-editor {
  display: flex;
  flex-direction: column;
  height: 100%;
  gap: 10px;
}

.body-type-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  flex-shrink: 0;
}

.bar-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
  flex-shrink: 0;
}

.format-btn {
  gap: 4px;
}

.body-type-bar :deep(.el-radio-button__inner) {
  padding: 8px 16px;
  font-size: 13px;
  font-weight: 500;
}

.body-text {
  flex: 1;
  min-height: 0;
}

.form-data {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
</style>
