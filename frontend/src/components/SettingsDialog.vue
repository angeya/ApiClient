<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import { ElMessageBox } from 'element-plus/es/components/message-box/index'
import { useWorkspaceStore } from '../stores/workspace'
import { useThemeStore, type Theme } from '../stores/theme'
import { api } from '../services/api'
import type { Settings, WorkspaceMigrationResult } from '../types'

const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const workspace = useWorkspaceStore()
const theme = useThemeStore()

const form = ref<Settings>({ ...workspace.settings })
const workspacePath = ref('')
const saving = ref(false)
const themes: { label: string; value: Theme }[] = [
  { label: '深色主题 1', value: 'dark-1' },
  { label: '深色主题 2', value: 'dark-2' },
  { label: '暖色主题', value: 'warm' },
]

// 迁移结果弹窗状态
const migrationVisible = ref(false)
const migrationResult = ref<WorkspaceMigrationResult | null>(null)
let migrationResolver: (() => void) | null = null

watch(
  () => props.modelValue,
  (visible) => {
    if (visible) {
      form.value = { ...workspace.settings }
      workspacePath.value = workspace.root
    }
  },
  { immediate: true }
)

function timeoutSeconds(ms: number): number {
  return Math.round(ms / 1000)
}

function toMs(seconds: number): number {
  return Math.max(1, Math.round(seconds)) * 1000
}

const BYTES_PER_MB = 1024 * 1024

/**
 * 响应渲染上限在界面上以 MB 为单位编辑，底层设置仍以字节存储。
 */
const renderLimitMb = computed({
  get: () => Math.max(1, Math.round((form.value.responseRenderLimitBytes || 0) / BYTES_PER_MB)),
  set: (mb: number) => {
    const value = Number.isFinite(mb) ? mb : 1
    form.value.responseRenderLimitBytes = Math.max(1, Math.round(value)) * BYTES_PER_MB
  },
})

async function chooseWorkspace() {
  try {
    const selected = await api.pickWorkspaceDir()
    if (selected) {
      workspacePath.value = selected
    }
  } catch (err) {
    ElMessage.error(`选择工作区失败: ${err}`)
  }
}

/**
 * 以自定义弹窗展示迁移结果，返回 Promise 以便调用方等待用户确认
 */
function showMigrationResult(result: WorkspaceMigrationResult): Promise<void> {
  migrationResult.value = result
  migrationVisible.value = true
  return new Promise((resolve) => {
    migrationResolver = resolve
  })
}

function closeMigrationResult() {
  migrationVisible.value = false
  migrationResult.value = null
  if (migrationResolver) {
    migrationResolver()
    migrationResolver = null
  }
}

async function save() {
  if (saving.value) return
  const next: Settings = {
    ...form.value,
    requestTimeoutMs: toMs(timeoutSeconds(form.value.requestTimeoutMs)),
  }
  const oldSettings = { ...workspace.settings }
  const settingsChanged = JSON.stringify(next) !== JSON.stringify(oldSettings)
  const workspaceChanged = !!(workspacePath.value && workspacePath.value !== workspace.root)

  saving.value = true
  theme.setTheme(next.theme as Theme)
  try {
    if (workspaceChanged) {
      // 二次确认：修改工作目录将触发数据迁移
      try {
        await ElMessageBox.confirm(
          '修改工作目录将会进行数据迁移，是否继续？',
          '确认修改工作目录',
          { type: 'warning', confirmButtonText: '继续', cancelButtonText: '取消' }
        )
      } catch {
        // 用户取消迁移：不修改工作目录，其他设置照常保存
        workspacePath.value = workspace.root
        if (settingsChanged) {
          workspace.settings = next
          await workspace.saveSettings()
          ElMessage.success('设置已保存')
        } else {
          theme.setTheme(oldSettings.theme as Theme)
          ElMessage.info('工作目录未修改')
        }
        emit('update:modelValue', false)
        return
      }
      // 用户确认：执行数据迁移
      const result = await workspace.changeWorkspace(workspacePath.value) as WorkspaceMigrationResult
      await showMigrationResult(result)
    }
    if (settingsChanged) {
      workspace.settings = next
      await workspace.saveSettings()
    } else {
      theme.setTheme(workspace.settings.theme as Theme)
    }
    ElMessage.success('设置已保存')
    emit('update:modelValue', false)
  } catch (err) {
    workspace.settings = oldSettings
    theme.setTheme(oldSettings.theme as Theme)
    ElMessage.error(`保存失败: ${err}`)
  } finally {
    saving.value = false
  }
}

function close() {
  emit('update:modelValue', false)
}
</script>

<template>
  <el-dialog
    :model-value="modelValue"
    title="设置"
    width="560px"
    class="app-dialog-center"
    :close-on-click-modal="false"
    @update:model-value="emit('update:modelValue', $event)"
    @close="close"
  >
    <el-form label-position="left" label-width="100px">
      <el-form-item label="主题">
        <el-select v-model="form.theme" placeholder="选择主题">
          <el-option
            v-for="t in themes"
            :key="t.value"
            :label="t.label"
            :value="t.value"
          />
        </el-select>
      </el-form-item>

      <el-form-item label="请求超时">
        <el-input-number
          v-model="form.requestTimeoutMs"
          :min="1000"
          :max="300000"
          :step="1000"
          controls-position="right"
        />
        <span class="unit">毫秒</span>
      </el-form-item>

      <el-form-item label="渲染上限">
        <el-input-number
          v-model="renderLimitMb"
          :min="1"
          :max="1024"
          :step="1"
          controls-position="right"
        />
        <span class="unit">MB</span>
        <el-tooltip
          content="响应内容超过该大小时不再渲染正文，需下载响应内容后浏览"
          placement="top"
        >
          <el-icon class="hint-icon"><QuestionFilled /></el-icon>
        </el-tooltip>
      </el-form-item>

      <el-form-item label="自动命名">
        <el-switch v-model="form.autoRenameOnSave" />
        <el-tooltip
          content="开启后，保存接口时若名称包含“未命名”（含“未命名 副本”），将按接口 URL 路径最后一段自动重命名"
          placement="top"
        >
          <el-icon class="hint-icon"><QuestionFilled /></el-icon>
        </el-tooltip>
      </el-form-item>

      <el-form-item label="工作区">
        <div class="workspace-row">
          <el-input v-model="workspacePath" readonly />
          <el-button @click="chooseWorkspace">选择</el-button>
        </div>
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="close">取消</el-button>
      <el-button type="primary" :loading="saving" @click="save">保存</el-button>
    </template>
  </el-dialog>

  <!-- 迁移结果弹窗 -->
  <el-dialog
    v-model="migrationVisible"
    width="540px"
    :close-on-click-modal="false"
    :show-close="false"
    class="migration-dialog app-dialog-center"
  >
    <div v-if="migrationResult" class="migration-result">
      <div :class="['migration-header', migrationResult.failedFiles > 0 ? 'is-warning' : 'is-success']">
        <el-icon class="migration-icon">
          <WarningFilled v-if="migrationResult.failedFiles > 0" />
          <SuccessFilled v-else />
        </el-icon>
        <div class="migration-title-block">
          <div class="migration-title">工作区迁移结果</div>
          <div class="migration-subtitle">{{ migrationResult.message }}</div>
        </div>
      </div>

      <div class="migration-body">
        <div class="migration-paths">
          <div class="path-item">
            <span class="path-label">源目录</span>
            <span class="path-value" :title="migrationResult.sourcePath">{{ migrationResult.sourcePath || '-' }}</span>
          </div>
          <el-icon class="path-arrow"><ArrowDown /></el-icon>
          <div class="path-item">
            <span class="path-label">目标目录</span>
            <span class="path-value" :title="migrationResult.targetPath">{{ migrationResult.targetPath || '-' }}</span>
          </div>
        </div>

        <div class="migration-stats">
          <div class="stat-card">
            <div class="stat-num success">{{ migrationResult.copiedFiles }}</div>
            <div class="stat-label">复制文件</div>
          </div>
          <div class="stat-card">
            <div class="stat-num">{{ migrationResult.skippedFiles }}</div>
            <div class="stat-label">跳过同名</div>
          </div>
          <div class="stat-card">
            <div class="stat-num">{{ migrationResult.createdDirs }}</div>
            <div class="stat-label">创建目录</div>
          </div>
          <div class="stat-card">
            <div class="stat-num" :class="{ danger: migrationResult.failedFiles > 0 }">{{ migrationResult.failedFiles }}</div>
            <div class="stat-label">失败文件</div>
          </div>
        </div>

        <div v-if="migrationResult.failureSamples?.length" class="migration-failures">
          <div class="failures-title">失败示例</div>
          <div class="failures-list">
            <div v-for="(s, i) in migrationResult.failureSamples" :key="i" class="failure-item">{{ s }}</div>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <el-button type="primary" @click="closeMigrationResult">确定</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.unit {
  margin-left: 8px;
  font-size: 13px;
  color: var(--app-text-secondary);
}

.hint-icon {
  margin-left: 6px;
  font-size: 14px;
  color: var(--app-text-secondary);
  cursor: help;
}

.workspace-row {
  display: flex;
  width: 100%;
  gap: 8px;
}

.migration-dialog :deep(.el-dialog__body) {
  padding: 0;
}

.migration-result {
  display: flex;
  flex-direction: column;
}

/* 标题区域 */
.migration-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 20px 24px;
  border-bottom: 1px solid var(--app-border);
}

.migration-header.is-success {
  background-color: var(--app-success-bg, rgba(82, 196, 26, 0.08));
}

.migration-header.is-warning {
  background-color: var(--app-warning-bg, rgba(230, 162, 60, 0.08));
}

.migration-icon {
  font-size: 36px;
  flex-shrink: 0;
}

.migration-header.is-success .migration-icon {
  color: var(--app-success, #67c23a);
}

.migration-header.is-warning .migration-icon {
  color: var(--app-warning, #e6a23c);
}

.migration-title-block {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.migration-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--app-text);
}

.migration-subtitle {
  font-size: 12px;
  color: var(--app-text-secondary);
  word-break: break-all;
}

/* 内容区域 */
.migration-body {
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 20px 24px;
}

.migration-paths {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 16px;
  background-color: var(--app-bg-secondary);
  border-radius: 6px;
}

.path-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.path-label {
  font-size: 12px;
  color: var(--app-text-secondary);
  flex-shrink: 0;
  width: 56px;
}

.path-value {
  font-size: 12px;
  font-family: 'Consolas', 'Monaco', monospace;
  color: var(--app-text);
  word-break: break-all;
  min-width: 0;
}

.path-arrow {
  align-self: center;
  font-size: 14px;
  color: var(--app-text-secondary);
}

.migration-stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
}

.stat-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 12px 8px;
  background-color: var(--app-bg-secondary);
  border-radius: 6px;
}

.stat-num {
  font-size: 22px;
  font-weight: 700;
  font-family: 'Consolas', 'Monaco', monospace;
  color: var(--app-text);
}

.stat-num.success {
  color: var(--app-success, #67c23a);
}

.stat-num.danger {
  color: var(--app-danger, #f56c6c);
}

.stat-label {
  font-size: 11px;
  color: var(--app-text-secondary);
}

.migration-failures {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.failures-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--app-text-secondary);
}

.failures-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 120px;
  overflow-y: auto;
}

.failure-item {
  font-size: 11px;
  font-family: 'Consolas', 'Monaco', monospace;
  color: var(--app-danger, #f56c6c);
  padding: 4px 8px;
  background-color: var(--app-bg-secondary);
  border-radius: 4px;
  word-break: break-all;
}
</style>
