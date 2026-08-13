<script setup lang="ts">
import { computed, watch, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useResizableColumns } from '../../composables/useResizableColumns'
import { api } from '../../services/api'
import type { FormField } from '../../types'

const props = defineProps<{
  modelValue: FormField[]
  showType?: boolean
}>()

const emit = defineEmits<{ (e: 'update:modelValue', v: FormField[]): void }>()

const tableRef = ref<HTMLElement | null>(null)
const showType = props.showType
const defaultWidths = showType
  ? [36, 200, 100, 200, 200, 32]
  : [36, 200, 200, 200, 32]
const minWidths = showType
  ? [36, 60, 80, 60, 60, 32]
  : [36, 60, 60, 60, 32]
const flexible = showType
  ? [false, true, true, true, true, false]
  : [false, true, true, true, false]
const { gridTemplateColumns, startResize } = useResizableColumns({
  storageKey: showType ? 'formdata-editor-widths' : 'urlencoded-editor-widths',
  defaultWidths,
  minWidths,
  flexible,
  containerRef: () => tableRef.value,
})

const rows = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const isIndeterminate = computed(() => {
  const enabledCount = rows.value.filter((r) => r.enabled).length
  return enabledCount > 0 && enabledCount < rows.value.length
})

const allEnabled = computed({
  get: () => rows.value.length > 0 && rows.value.every((r) => r.enabled),
  set: (v) => {
    rows.value.forEach((r) => (r.enabled = v))
  },
})

function emptyRow(): FormField {
  return { key: '', value: '', type: 'text', enabled: false, description: '' }
}

function ensureEmptyRow() {
  const list = rows.value
  if (list.length === 0) {
    list.push(emptyRow())
    return
  }
  const last = list[list.length - 1]
  if (last.key.trim() !== '') {
    list.push(emptyRow())
  }
}

function onKeyInput(idx: number) {
  const row = rows.value[idx]
  if (!row) return
  if (row.key.trim() && !row.enabled) {
    row.enabled = true
  }
  ensureEmptyRow()
}

async function pickFile(idx: number) {
  try {
    const path = await api.pickFile()
    if (!path) return
    rows.value[idx].value = path
    if (!rows.value[idx].enabled) {
      rows.value[idx].enabled = true
    }
  } catch (err) {
    ElMessage.error(`选择文件失败: ${err instanceof Error ? err.message : String(err)}`)
  }
}

function removeRow(idx: number) {
  rows.value.splice(idx, 1)
  if (rows.value.length === 0) {
    rows.value.push(emptyRow())
  }
}

watch(
  () => props.modelValue,
  (list) => {
    // 直接向数组追加空行，避免通过 emit 触发 update:modelValue 导致误标记 dirty
    if (list && list.length === 0) {
      list.push(emptyRow())
    }
  },
  { immediate: true }
)
</script>

<template>
  <div class="form-data-editor">
    <div ref="tableRef" class="form-table" :style="{ '--form-cols': gridTemplateColumns }">
      <div class="form-header">
        <div class="form-cell cell-check">
          <el-checkbox v-model="allEnabled" :indeterminate="isIndeterminate" size="small" />
          <div class="col-resizer" @mousedown.stop="startResize(0, $event)"></div>
        </div>
        <div class="form-cell cell-key">
          参数名
          <div class="col-resizer" @mousedown.stop="startResize(1, $event)"></div>
        </div>
        <div v-if="showType" class="form-cell cell-type">
          类型
          <div class="col-resizer" @mousedown.stop="startResize(2, $event)"></div>
        </div>
        <div class="form-cell cell-value">
          参数值
          <div class="col-resizer" @mousedown.stop="startResize(showType ? 3 : 2, $event)"></div>
        </div>
        <div class="form-cell cell-desc">
          描述
          <div class="col-resizer" @mousedown.stop="startResize(showType ? 4 : 3, $event)"></div>
        </div>
        <div class="form-cell cell-action"></div>
      </div>
      <div
        v-for="(row, idx) in rows"
        :key="idx"
        class="form-row"
        :class="{ disabled: !row.enabled }"
      >
        <div class="form-cell cell-check">
          <el-checkbox v-model="row.enabled" size="small" />
        </div>
        <div class="form-cell cell-key">
          <el-input v-model="row.key" size="small" placeholder="参数名" @input="onKeyInput(idx)" />
        </div>
        <div v-if="showType" class="form-cell cell-type">
          <el-select v-model="row.type" size="small" class="type-select">
            <el-option label="文本" value="text" />
            <el-option label="文件" value="file" />
          </el-select>
        </div>
        <div class="form-cell cell-value">
          <el-input
            v-if="!showType || row.type === 'text'"
            v-model="row.value"
            size="small"
            placeholder="参数值"
          />
          <div v-else class="file-cell">
            <el-input v-model="row.value" size="small" placeholder="点击右侧按钮选择文件" readonly />
            <el-button size="small" class="file-btn" @click="pickFile(idx)">
              <el-icon><FolderOpened /></el-icon>
            </el-button>
          </div>
        </div>
        <div class="form-cell cell-desc">
          <el-input v-model="row.description" size="small" placeholder="描述" />
        </div>
        <div class="form-cell cell-action">
          <el-icon class="del-icon" @click="removeRow(idx)"><Delete /></el-icon>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.form-data-editor {
  width: 100%;
}

.form-table {
  display: flex;
  flex-direction: column;
  border: 1px solid var(--app-border);
  border-radius: 4px;
  overflow: hidden;
}

.form-header,
.form-row {
  display: grid;
  grid-template-columns: var(--form-cols);
  align-items: center;
  font-size: 12px;
}

.form-header {
  background-color: var(--app-bg-secondary);
  color: var(--app-text-secondary);
  font-weight: 600;
  border-bottom: 1px solid var(--app-border);
}

.form-row {
  border-bottom: 1px solid var(--app-border);
  transition: background-color 0.15s;
}

.form-row:last-child {
  border-bottom: none;
}

.form-row:hover {
  background-color: var(--app-bg-tertiary);
}

.form-row:focus-within {
  background-color: var(--app-bg-secondary);
}

.form-row.disabled {
  opacity: 0.55;
}

.form-cell {
  position: relative;
  padding: 4px 6px;
  overflow: hidden;
}

.col-resizer {
  position: absolute;
  right: 0;
  top: 0;
  bottom: 0;
  width: 6px;
  cursor: col-resize;
  z-index: 1;
  background-color: transparent;
  transition: background-color 0.15s;
}

.col-resizer:hover {
  background-color: var(--app-accent);
}

.cell-check,
.cell-action {
  display: flex;
  align-items: center;
  justify-content: center;
}

.cell-type {
  min-width: 80px;
}

.type-select {
  width: 100%;
}

.file-cell {
  display: flex;
  align-items: center;
  gap: 6px;
}

.file-btn {
  flex-shrink: 0;
  padding: 0 8px;
}

.file-cell :deep(.el-input) {
  flex: 1;
  min-width: 0;
}

.del-icon {
  cursor: pointer;
  color: var(--app-text-secondary);
  transition: color 0.15s;
  font-size: 14px;
}

.del-icon:hover {
  color: var(--app-danger);
}

.form-cell :deep(.el-input__wrapper),
.form-cell :deep(.el-select .el-input__wrapper) {
  box-shadow: 0 0 0 1px var(--app-border) inset !important;
  padding: 0 6px;
  background-color: var(--app-bg) !important;
  border-radius: 3px;
}

.form-cell :deep(.el-input__wrapper:focus-within) {
  box-shadow: 0 0 0 1px var(--app-accent) inset !important;
}

.form-cell :deep(.el-input__inner) {
  background-color: transparent !important;
}
</style>
