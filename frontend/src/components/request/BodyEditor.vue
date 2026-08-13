<script setup lang="ts">
import { computed } from 'vue'
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
</script>

<template>
  <div class="body-editor">
    <div class="body-type-bar">
      <el-radio-group v-model="req.bodyType" size="default">
        <el-radio-button v-for="t in bodyTypes" :key="t.value" :label="t.value">
          {{ t.label }}
        </el-radio-button>
      </el-radio-group>
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
  gap: 8px;
  flex-shrink: 0;
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
