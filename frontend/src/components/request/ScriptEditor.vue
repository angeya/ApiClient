<script setup lang="ts">
import { computed } from 'vue'
import CodeEditor from '../common/CodeEditor.vue'

const props = defineProps<{ modelValue: string; language?: string }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: string): void }>()

const value = computed({
  get: () => props.modelValue || '',
  set: (v) => emit('update:modelValue', v),
})
</script>

<template>
  <div class="script-editor">
    <CodeEditor
      v-model="value"
      language="text"
      placeholder="// 可用 pm.log(), pm.setGlobal(), pm.setEnv(), pm.request, pm.response"
    />
  </div>
</template>

<style scoped>
.script-editor {
  width: 100%;
  height: 100%;
  min-height: 180px;
}
</style>
