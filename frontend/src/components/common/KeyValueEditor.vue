<script setup lang="ts">
import { computed, watch, ref } from 'vue'
import { useResizableColumns } from '../../composables/useResizableColumns'

interface Row {
  key: string
  value: string
  enabled: boolean
  description: string
}

const props = defineProps<{
  modelValue: Row[]
  inheritedRows?: Row[]
  valueLabel?: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: Row[]): void
  (e: 'toggleInherited', key: string, enabled: boolean): void
}>()

const tableRef = ref<HTMLElement | null>(null)
const { gridTemplateColumns, startResize } = useResizableColumns({
  storageKey: 'kv-editor-widths',
  defaultWidths: [36, 200, 200, 200, 32],
  minWidths: [36, 60, 60, 60, 32],
  flexible: [false, true, true, true, false],
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

function ensureEmptyRow() {
  const list = rows.value
  if (list.length === 0) {
    list.push({ key: '', value: '', enabled: false, description: '' })
    return
  }
  const last = list[list.length - 1]
  if (last.key.trim() !== '') {
    list.push({ key: '', value: '', enabled: false, description: '' })
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

function removeRow(idx: number) {
  rows.value.splice(idx, 1)
  if (rows.value.length === 0) {
    rows.value.push({ key: '', value: '', enabled: false, description: '' })
  }
}

function onInheritedToggle(row: Row, enabled: boolean | string | number) {
  const bool = typeof enabled === 'boolean' ? enabled : !!enabled
  emit('toggleInherited', row.key, bool)
}

watch(
  () => props.modelValue,
  (list) => {
    // 直接向数组追加空行，避免通过 emit 触发 update:modelValue 导致误标记 dirty
    if (list && list.length === 0) {
      list.push({ key: '', value: '', enabled: false, description: '' })
    }
  },
  { immediate: true }
)
</script>

<template>
  <div class="kv-editor">
    <div ref="tableRef" class="kv-table" :style="{ '--kv-cols': gridTemplateColumns }">
      <div class="kv-header">
        <div class="kv-cell cell-check">
          <el-checkbox v-model="allEnabled" :indeterminate="isIndeterminate" size="small" />
          <div class="col-resizer" @mousedown.stop="startResize(0, $event)"></div>
        </div>
        <div class="kv-cell cell-key">
          参数名
          <div class="col-resizer" @mousedown.stop="startResize(1, $event)"></div>
        </div>
        <div class="kv-cell cell-value">
          {{ valueLabel || '参数值' }}
          <div class="col-resizer" @mousedown.stop="startResize(2, $event)"></div>
        </div>
        <div class="kv-cell cell-desc">
          描述
          <div class="col-resizer" @mousedown.stop="startResize(3, $event)"></div>
        </div>
        <div class="kv-cell cell-action"></div>
      </div>
      <div
        v-for="(row, idx) in inheritedRows"
        :key="'inherited-' + idx"
        class="kv-row inherited"
      >
        <div class="kv-cell cell-check">
          <el-checkbox :model-value="row.enabled" size="small" @change="onInheritedToggle(row, $event)" />
        </div>
        <div class="kv-cell cell-key">
          <el-input :model-value="row.key" size="small" placeholder="参数名" disabled />
        </div>
        <div class="kv-cell cell-value">
          <el-input :model-value="row.value" size="small" placeholder="参数值" disabled />
        </div>
        <div class="kv-cell cell-desc">
          <el-input :model-value="row.description" size="small" placeholder="描述" disabled />
        </div>
        <div class="kv-cell cell-action">
          <span class="inherited-badge">继承</span>
        </div>
      </div>
      <div
        v-for="(row, idx) in rows"
        :key="idx"
        class="kv-row"
        :class="{ disabled: !row.enabled }"
      >
        <div class="kv-cell cell-check">
          <el-checkbox v-model="row.enabled" size="small" />
        </div>
        <div class="kv-cell cell-key">
          <el-input v-model="row.key" size="small" placeholder="参数名" @input="onKeyInput(idx)" />
        </div>
        <div class="kv-cell cell-value">
          <el-input v-model="row.value" size="small" placeholder="参数值" />
        </div>
        <div class="kv-cell cell-desc">
          <el-input v-model="row.description" size="small" placeholder="描述" />
        </div>
        <div class="kv-cell cell-action">
          <el-icon class="del-icon" @click="removeRow(idx)"><Delete /></el-icon>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.kv-editor {
  width: 100%;
}

.kv-table {
  display: flex;
  flex-direction: column;
  border: 1px solid var(--app-border);
  border-radius: 4px;
  overflow: hidden;
}

.kv-header,
.kv-row {
  display: grid;
  grid-template-columns: var(--kv-cols);
  align-items: center;
  font-size: 12px;
}

.kv-header {
  background-color: var(--app-bg-secondary);
  color: var(--app-text-secondary);
  font-weight: 600;
  border-bottom: 1px solid var(--app-border);
}

.kv-row {
  border-bottom: 1px solid var(--app-border);
  transition: background-color 0.15s;
}

.kv-row:last-child {
  border-bottom: none;
}

.kv-row:hover {
  background-color: var(--app-bg-tertiary);
}

.kv-row:focus-within {
  background-color: var(--app-bg-secondary);
}

.kv-row.disabled {
  opacity: 0.55;
}

.kv-row.inherited {
  background-color: var(--app-bg-secondary);
}

.kv-row.inherited :deep(.el-input__wrapper) {
  background-color: var(--app-bg-tertiary) !important;
  cursor: not-allowed;
}

.kv-row.inherited :deep(.el-input__inner) {
  cursor: not-allowed;
  color: var(--app-text-secondary);
}

.inherited-badge {
  font-size: 11px;
  color: var(--app-text-secondary);
  padding: 2px 6px;
  border-radius: 3px;
  background-color: var(--app-bg-tertiary);
}

.kv-cell {
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

.cell-check {
  display: flex;
  align-items: center;
  justify-content: center;
}

.cell-action {
  display: flex;
  align-items: center;
  justify-content: center;
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

.kv-cell :deep(.el-input__wrapper),
.kv-cell :deep(.el-select .el-input__wrapper) {
  box-shadow: 0 0 0 1px var(--app-border) inset !important;
  padding: 0 6px;
  background-color: var(--app-bg) !important;
  border-radius: 3px;
}

.kv-cell :deep(.el-input__wrapper:focus-within) {
  box-shadow: 0 0 0 1px var(--app-accent) inset !important;
}

.kv-cell :deep(.el-input__inner) {
  background-color: transparent !important;
}
</style>
