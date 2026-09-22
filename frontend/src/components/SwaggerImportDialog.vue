<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import { useWorkspaceStore } from '../stores/workspace'
import { parseSwagger30, importSwaggerItems } from '../utils/swagger-import'
import type { SwaggerImportResult, ImportItem } from '../utils/swagger-import'

const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const workspace = useWorkspaceStore()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const step = ref<'select' | 'importing' | 'done'>('select')
const selectedMethod = ref('swagger30')
const progress = ref({ current: 0, total: 0, percent: 0 })
const result = ref<SwaggerImportResult>({ success: 0, failed: 0, errors: [], cancelled: false })
const cancelled = ref(false)
const importFileName = ref('')

function reset() {
  step.value = 'select'
  selectedMethod.value = 'swagger30'
  progress.value = { current: 0, total: 0, percent: 0 }
  result.value = { success: 0, failed: 0, errors: [], cancelled: false }
  cancelled.value = false
  importFileName.value = ''
}

function handleClose() {
  if (step.value === 'importing') return
  visible.value = false
  setTimeout(reset, 200)
}

function triggerFileSelect() {
  if (selectedMethod.value !== 'swagger30') {
    ElMessage.warning('请选择导入方式')
    return
  }
  const input = document.createElement('input')
  input.type = 'file'
  input.accept = '.json,application/json'
  input.onchange = async () => {
    const file = input.files?.[0]
    if (!file) return
    importFileName.value = file.name
    const text = await file.text()
    await startImport(text)
  }
  input.click()
}

async function startImport(jsonText: string) {
  let items: ImportItem[] = []
  try {
    const parseResult = parseSwagger30(jsonText)
    items = parseResult.items
    if (parseResult.errors.length > 0) {
      // Log parse-level errors but continue if we have items
      console.warn('Swagger parse warnings:', parseResult.errors)
    }
    if (items.length === 0) {
      ElMessage.warning('未检测到可导入的接口定义')
      return
    }
  } catch (err) {
    ElMessage.error(`解析失败: ${err instanceof Error ? err.message : String(err)}`)
    return
  }

  step.value = 'importing'
  progress.value = { current: 0, total: items.length, percent: 0 }
  cancelled.value = false

  try {
    result.value = await importSwaggerItems(items, {
      onProgress: (current, total) => {
        progress.value.current = current
        progress.value.total = total
        progress.value.percent = total > 0 ? Math.round((current / total) * 100) : 0
      },
      onCancelled: () => cancelled.value,
      findExistingFolder: (name) => {
        const segments = name.split('/').filter(Boolean)
        if (segments.length === 0) return undefined
        let nodes = workspace.tree || []
        let found: typeof nodes[0] | undefined
        for (const segment of segments) {
          found = nodes.find((n) => n.type === 'dir' && n.name === segment)
          if (!found) return undefined
          nodes = found.children || []
        }
        return found?.path
      },
    })
    await workspace.reloadTree()
    step.value = 'done'
  } catch (err) {
    result.value.errors.push(err instanceof Error ? err.message : String(err))
    step.value = 'done'
  }
}

function cancelImport() {
  cancelled.value = true
}

function closeResult() {
  visible.value = false
  setTimeout(reset, 200)
}
</script>

<template>
  <div>
    <!-- Full-screen overlay during import -->
    <div v-if="step === 'importing'" class="import-overlay">
      <div class="import-overlay-content">
        <div class="importing-title">
          <el-icon :size="28" class="spin-icon"><Loading /></el-icon>
          <span>正在导入...</span>
        </div>
        <div class="file-name">{{ importFileName }}</div>
        <el-progress
          :percentage="progress.percent"
          :stroke-width="12"
          :show-text="true"
          class="import-progress"
        />
        <div class="progress-detail">
          {{ progress.current }} / {{ progress.total }}
        </div>
        <div class="cancel-action">
          <el-button size="default" @click="cancelImport">取消导入</el-button>
        </div>
      </div>
    </div>

    <el-dialog
      v-model="visible"
      title="导入接口"
      width="480px"
      class="app-dialog-center"
      :close-on-click-modal="step !== 'importing'"
      :close-on-press-escape="step !== 'importing'"
      :show-close="step !== 'importing'"
      @close="handleClose"
    >
      <!-- Step: Select method -->
      <div v-if="step === 'select'" class="select-step">
        <div class="method-list">
          <div
            class="method-card"
            :class="{ active: selectedMethod === 'swagger30' }"
            @click="selectedMethod = 'swagger30'"
          >
            <el-icon :size="28" class="method-icon"><Document /></el-icon>
            <div class="method-info">
              <div class="method-name">Swagger 3.0</div>
              <div class="method-desc">从 OpenAPI 3.0 JSON 文件导入接口定义</div>
            </div>
            <el-icon v-if="selectedMethod === 'swagger30'" class="check-icon"><Check /></el-icon>
          </div>
        </div>

        <div class="select-actions">
          <el-button type="primary" @click="triggerFileSelect">
            <el-icon class="btn-icon"><Upload /></el-icon>
            选择文件并导入
          </el-button>
        </div>
      </div>

      <!-- Step: Done -->
    <div v-else-if="step === 'done'" class="done-step">
      <div class="result-header">
        <el-icon :size="36" class="result-icon" :class="result.failed > 0 ? 'warning' : 'success'">
          <CircleCheck v-if="result.failed === 0 && !result.cancelled" />
          <WarningFilled v-else />
        </el-icon>
        <div class="result-title">
          {{ result.cancelled ? '导入已取消' : result.failed > 0 ? '导入完成（存在失败）' : '导入成功' }}
        </div>
      </div>

      <div class="result-stats">
        <div class="stat-item success">
          <span class="stat-label">成功</span>
          <span class="stat-value">{{ result.success }}</span>
        </div>
        <div class="stat-item failed">
          <span class="stat-label">失败</span>
          <span class="stat-value">{{ result.failed }}</span>
        </div>
      </div>

      <div v-if="result.errors.length > 0" class="error-list">
        <div class="error-title">失败详情（{{ result.errors.length }} 条）</div>
        <div class="error-scroll">
          <div v-for="(err, idx) in result.errors" :key="idx" class="error-item">
            {{ err }}
          </div>
        </div>
      </div>

      <div class="done-actions">
        <el-button type="primary" @click="closeResult">关闭</el-button>
      </div>
    </div>
  </el-dialog>
</div>
</template>

<style scoped>
.select-step {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.method-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.method-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  border-radius: 8px;
  border: 1px solid var(--app-border);
  background-color: var(--app-bg-secondary);
  cursor: pointer;
  transition: all 0.15s ease;
}

.method-card:hover {
  border-color: var(--app-accent);
}

.method-card.active {
  border-color: var(--app-accent);
  background-color: var(--app-bg-tertiary);
}

.method-icon {
  color: var(--app-accent);
  flex-shrink: 0;
}

.method-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.method-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--app-text);
}

.method-desc {
  font-size: 12px;
  color: var(--app-text-secondary);
}

.check-icon {
  color: var(--app-accent);
  flex-shrink: 0;
}

.select-actions {
  display: flex;
  justify-content: flex-end;
  padding-top: 4px;
}

.btn-icon {
  margin-right: 4px;
}

/* Full-screen import overlay */
.import-overlay {
  position: fixed;
  inset: 0;
  z-index: 3000;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: var(--app-bg-overlay);
  backdrop-filter: blur(2px);
}

.import-overlay-content {
  width: 420px;
  max-width: 90vw;
  padding: 28px 32px;
  border-radius: 10px;
  background-color: var(--app-bg);
  border: 1px solid var(--app-border);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
}

.importing-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: var(--app-text);
}

.spin-icon {
  animation: spin 1s linear infinite;
  color: var(--app-accent);
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.file-name {
  font-size: 12px;
  color: var(--app-text-secondary);
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.import-progress {
  width: 100%;
}

.progress-detail {
  font-size: 12px;
  color: var(--app-text-secondary);
}

.cancel-action {
  padding-top: 8px;
}

/* Done step */
.done-step {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 4px 0;
}

.result-header {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.result-icon.success {
  color: var(--app-success);
}

.result-icon.warning {
  color: var(--app-warning);
}

.result-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--app-text);
}

.result-stats {
  display: flex;
  justify-content: center;
  gap: 32px;
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.stat-item.success .stat-value {
  color: var(--app-success);
}

.stat-item.failed .stat-value {
  color: var(--app-danger);
}

.stat-label {
  font-size: 12px;
  color: var(--app-text-secondary);
}

.stat-value {
  font-size: 22px;
  font-weight: 700;
}

.error-list {
  border: 1px solid var(--app-border);
  border-radius: 6px;
  background-color: var(--app-bg-secondary);
  overflow: hidden;
}

.error-title {
  padding: 8px 12px;
  font-size: 12px;
  font-weight: 600;
  color: var(--app-text-secondary);
  border-bottom: 1px solid var(--app-border);
  background-color: var(--app-bg-tertiary);
}

.error-scroll {
  max-height: 140px;
  overflow-y: auto;
  padding: 8px 12px;
}

.error-item {
  font-size: 12px;
  color: var(--app-danger);
  line-height: 1.6;
  word-break: break-word;
}

.error-item + .error-item {
  margin-top: 4px;
}

.done-actions {
  display: flex;
  justify-content: center;
  padding-top: 4px;
}
</style>
