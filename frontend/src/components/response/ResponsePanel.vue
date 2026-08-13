<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { useWorkspaceStore } from '../../stores/workspace'
import type { HttpResponse } from '../../services/api'
import { api, responseBodyToString } from '../../services/api'
import {
  tryFormatJson,
  formatBytes,
  getExtensionFromContentType,
  parseFilenameFromHeaders,
  generateRandomFilename,
} from '../../utils/helpers'
import CodeEditor from '../common/CodeEditor.vue'

interface ScriptResult {
  success: boolean
  error?: string
  durationMs: number
  logs: string[]
}

interface HttpResponseWithScripts extends HttpResponse {
  preScript?: ScriptResult
  postScript?: ScriptResult
}

const workspace = useWorkspaceStore()

const activeTab = ref('body')
const currentRequestId = computed(() => workspace.activeDraft?.id || '')

const responseMap = ref<Map<string, HttpResponseWithScripts>>(new Map())
const responseTimeMap = ref<Map<string, string>>(new Map())

const response = computed(() => {
  if (!currentRequestId.value) return null
  return responseMap.value.get(currentRequestId.value) || null
})

const responseTime = computed(() => {
  if (!currentRequestId.value) return ''
  return responseTimeMap.value.get(currentRequestId.value) || ''
})

const isSendingCurrent = computed(() => {
  return workspace.sending && workspace.sendingRequestId === currentRequestId.value
})

async function cancelCurrentRequest() {
  if (!workspace.sendingRequestId) return
  try {
    await api.cancelRequest(workspace.sendingRequestId)
    ElMessage.info('请求已取消')
  } catch (err) {
    ElMessage.error(`取消失败: ${err}`)
  }
}

const totalLogCount = computed(() => {
  return (response.value?.preScript?.logs?.length || 0) + (response.value?.postScript?.logs?.length || 0)
})

const hasScriptError = computed(() => {
  return response.value?.preScript?.success === false || response.value?.postScript?.success === false
})

const allLogs = computed(() => {
  const logs: { source: string; text: string; timestamp: string }[] = []
  const now = new Date()
  const ts = `${now.getHours().toString().padStart(2, '0')}:${now.getMinutes().toString().padStart(2, '0')}:${now.getSeconds().toString().padStart(2, '0')}`
  if (response.value?.preScript) {
    for (const l of response.value.preScript.logs || []) {
      logs.push({ source: 'Pre-request', text: l, timestamp: ts })
    }
  }
  if (response.value?.postScript) {
    for (const l of response.value.postScript.logs || []) {
      logs.push({ source: 'Post-response', text: l, timestamp: ts })
    }
  }
  return logs
})

function formatResponseTime(): string {
  const now = new Date()
  const y = now.getFullYear()
  const M = (now.getMonth() + 1).toString().padStart(2, '0')
  const d = now.getDate().toString().padStart(2, '0')
  const h = now.getHours().toString().padStart(2, '0')
  const m = now.getMinutes().toString().padStart(2, '0')
  const s = now.getSeconds().toString().padStart(2, '0')
  return `${y}-${M}-${d} ${h}:${m}:${s}`
}

function onResponse(ev: Event) {
  const detail = (ev as CustomEvent).detail as HttpResponseWithScripts
  if (detail && currentRequestId.value) {
    responseMap.value.set(currentRequestId.value, detail)
    responseTimeMap.value.set(currentRequestId.value, formatResponseTime())
  }
  // 保持用户当前选中的响应标签页，不自动切换
}

const scriptFilter = ref<'all' | 'pre' | 'post'>('all')
const filteredLogs = computed(() => {
  if (scriptFilter.value === 'all') return allLogs.value
  return allLogs.value.filter((l) =>
    scriptFilter.value === 'pre' ? l.source === 'Pre-request' : l.source === 'Post-response'
  )
})

function copyLogs() {
  const text = allLogs.value.map((l) => `[${l.timestamp}] [${l.source}] ${l.text}`).join('\n')
  navigator.clipboard.writeText(text)
}

function clearResponse() {
  if (currentRequestId.value) {
    responseMap.value.delete(currentRequestId.value)
    responseTimeMap.value.delete(currentRequestId.value)
  }
}

onMounted(() => {
  window.addEventListener('api-response', onResponse)
})

onUnmounted(() => {
  window.removeEventListener('api-response', onResponse)
})

const formattedBody = computed(() => {
  if (!response.value) return ''
  const text = responseBodyToString(response.value.body)
  if (response.value.contentType?.includes('application/json')) {
    return tryFormatJson(text)
  }
  return text
})

const bodyLanguage = computed(() => {
  if (!response.value) return 'text'
  return response.value.contentType?.includes('application/json') ? 'json' : 'text'
})

function headerEntries(): [string, string][] {
  if (!response.value || !response.value.headers) return []
  const out: [string, string][] = []
  for (const [k, vals] of Object.entries(response.value.headers)) {
    out.push([k, (vals || []).join('; ')])
  }
  return out
}

async function copyResponseBody() {
  if (!formattedBody.value) return
  try {
    await navigator.clipboard.writeText(formattedBody.value)
    ElMessage.success('响应内容已复制到剪贴板')
  } catch (err) {
    ElMessage.error(`复制失败: ${err}`)
  }
}

async function downloadResponseBody() {
  if (!response.value || !formattedBody.value) {
    ElMessage.warning('没有可下载的响应内容')
    return
  }
  const headers = response.value.headers || {}
  let filename = parseFilenameFromHeaders(headers)
  if (!filename) {
    const ext = getExtensionFromContentType(response.value.contentType || '')
    filename = generateRandomFilename(ext)
  }
  try {
    const savedPath = await api.saveResponseBody(formattedBody.value, filename)
    if (savedPath) {
      ElMessage.success(`已保存到 ${savedPath}`)
    }
  } catch (err) {
    ElMessage.error(`保存失败: ${err}`)
  }
}
</script>

<template>
  <div class="response-panel">
    <div class="panel-header">
      <span>响应</span>
      <div v-if="response" class="response-meta">
        <span :class="['meta-tag', response.statusCode >= 200 && response.statusCode < 300 ? 'meta-success' : 'meta-danger']">
          {{ response.statusCode }} {{ response.status }}
        </span>
        <el-tooltip placement="top" popper-class="timing-tooltip-popper" :show-arrow="false">
          <template #content>
            <div class="timing-tooltip">
              <div class="timing-row">
                <span class="timing-label">服务器响应</span>
                <span class="timing-value">{{ response.serverTimeMs ?? response.timeMs }} ms</span>
              </div>
              <div class="timing-row">
                <span class="timing-label">网络传输</span>
                <span class="timing-value">{{ response.transferTimeMs ?? 0 }} ms</span>
              </div>
              <div class="timing-divider" />
              <div class="timing-row">
                <span class="timing-label">总耗时</span>
                <span class="timing-value">{{ response.timeMs }} ms</span>
              </div>
            </div>
          </template>
          <span class="meta-tag meta-accent timing-tag">{{ response.timeMs }} ms</span>
        </el-tooltip>
        <span class="meta-tag">{{ formatBytes(response.sizeBytes) }}</span>
        <span v-if="responseTime" class="response-time">{{ responseTime }}</span>
        <el-button size="small" text @click="clearResponse">清除</el-button>
      </div>
    </div>
    <div class="panel-body">
      <div v-if="isSendingCurrent" class="request-overlay">
        <div class="overlay-card">
          <el-icon class="overlay-spinner"><Loading /></el-icon>
          <div class="overlay-timer">{{ workspace.requestTimer }}</div>
          <div class="overlay-status">请求发送中...</div>
          <el-button type="danger" size="small" @click="cancelCurrentRequest">取消</el-button>
        </div>
      </div>
      <el-tabs v-model="activeTab" class="response-tabs">
        <el-tab-pane label="响应内容" name="body">
          <div v-if="response && formattedBody" class="response-body-wrap">
            <div class="response-body-toolbar">
              <el-tooltip content="复制响应内容" placement="top">
                <el-button size="small" text @click="copyResponseBody">
                  <el-icon><CopyDocument /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="下载响应内容" placement="top">
                <el-button size="small" text @click="downloadResponseBody">
                  <el-icon><Download /></el-icon>
                </el-button>
              </el-tooltip>
            </div>
            <div class="response-body-content">
              <CodeEditor
                :model-value="formattedBody"
                read-only
                :language="bodyLanguage"
                placeholder="响应内容为空"
              />
            </div>
          </div>
          <el-empty v-else-if="!response" description="发送请求后查看响应" />
          <el-empty v-else description="响应内容为空" />
        </el-tab-pane>
        <el-tab-pane label="响应头" name="headers">
          <table v-if="response" class="kv-table">
            <tr v-for="([k, v], idx) in headerEntries()" :key="idx">
              <td class="key">{{ k }}</td>
              <td class="value">{{ v }}</td>
            </tr>
          </table>
        </el-tab-pane>
        <el-tab-pane label="Cookies" name="cookies">
          <div v-if="response && response.cookies?.length">
            <div v-for="(c, idx) in response.cookies" :key="idx" class="cookie-item">{{ c }}</div>
          </div>
          <el-empty v-else description="无 Cookies" />
        </el-tab-pane>
        <el-tab-pane name="scripts">
          <template #label>
            <span>脚本终端
              <el-badge
                v-if="totalLogCount > 0"
                :type="hasScriptError ? 'danger' : 'success'"
                :value="totalLogCount"
                class="script-badge"
              />
            </span>
          </template>
          <div class="script-panel">
            <!-- Pre-request result -->
            <div v-if="response?.preScript" class="script-section">
              <div class="script-section-header">
                <span :class="['script-status', response.preScript.success ? 'success' : 'error']">
                  {{ response.preScript.success ? '✓' : '✗' }} Pre-request
                </span>
                <span class="script-duration">{{ response.preScript.durationMs }} ms</span>
              </div>
              <div v-if="response.preScript.error" class="script-error">{{ response.preScript.error }}</div>
            </div>
            <!-- Post-response result -->
            <div v-if="response?.postScript" class="script-section">
              <div class="script-section-header">
                <span :class="['script-status', response.postScript.success ? 'success' : 'error']">
                  {{ response.postScript.success ? '✓' : '✗' }} Post-response
                </span>
                <span class="script-duration">{{ response.postScript.durationMs }} ms</span>
              </div>
              <div v-if="response.postScript.error" class="script-error">{{ response.postScript.error }}</div>
            </div>
            <!-- Logs -->
            <div class="script-logs-header">
              <span>日志输出</span>
              <div class="log-actions">
                <el-radio-group v-model="scriptFilter" size="small">
                  <el-radio-button label="all">全部</el-radio-button>
                  <el-radio-button label="pre">Pre</el-radio-button>
                  <el-radio-button label="post">Post</el-radio-button>
                </el-radio-group>
                <el-button size="small" text @click="copyLogs">复制</el-button>
              </div>
            </div>
            <div v-if="filteredLogs.length" class="log-list">
              <div v-for="(log, idx) in filteredLogs" :key="idx" class="log-item">
                <span class="log-ts">{{ log.timestamp }}</span>
                <span :class="['log-source', log.source === 'Pre-request' ? 'pre' : 'post']">{{ log.source }}</span>
                <span class="log-text">{{ log.text }}</span>
              </div>
            </div>
            <el-empty v-else description="暂无脚本日志" :image-size="40" />
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>

<style scoped>
.response-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 8px;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 4px 8px;
  font-size: 13px;
  font-weight: 600;
}

.response-meta {
  display: flex;
  gap: 6px;
  align-items: center;
}

.response-time {
  font-size: 11px;
  color: var(--app-text-secondary);
  font-weight: 400;
  font-family: 'Consolas', 'Monaco', monospace;
}

.panel-body {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  position: relative;
}

.response-tabs {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.response-tabs :deep(.el-tabs__content) {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.response-body-wrap {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.response-body-toolbar {
  display: flex;
  justify-content: flex-end;
  gap: 4px;
  padding: 4px 0;
  flex-shrink: 0;
}

.response-body-content {
  flex: 1;
  min-height: 0;
}

.kv-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}

.kv-table td {
  padding: 6px 8px;
  border-bottom: 1px solid var(--app-border);
  vertical-align: top;
}

.kv-table .key {
  width: 140px;
  color: var(--app-text-secondary);
  font-weight: 600;
}

.cookie-item {
  padding: 6px 0;
  border-bottom: 1px solid var(--app-border);
  font-size: 12px;
}

.script-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 4px;
}

.script-section {
  padding: 8px 10px;
  border-radius: 4px;
  background-color: var(--app-bg-tertiary);
}

.script-section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.script-status {
  font-size: 13px;
  font-weight: 600;
}

.script-status.success {
  color: var(--app-success);
}

.script-status.error {
  color: var(--app-danger);
}

.script-duration {
  font-size: 11px;
  color: var(--app-text-secondary);
}

.script-error {
  margin-top: 6px;
  padding: 6px 8px;
  border-radius: 3px;
  background-color: rgba(244, 67, 54, 0.1);
  color: var(--app-danger);
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-word;
}

.script-logs-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 12px;
  font-weight: 600;
  padding: 4px 0;
}

.log-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.log-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 300px;
  overflow-y: auto;
}

.log-item {
  display: flex;
  gap: 8px;
  padding: 4px 8px;
  border-radius: 3px;
  background-color: var(--app-bg-tertiary);
  font-size: 12px;
  font-family: 'Consolas', 'Monaco', monospace;
}

.log-ts {
  color: var(--app-text-secondary);
  flex-shrink: 0;
}

.log-source {
  flex-shrink: 0;
  font-weight: 600;
}

.log-source.pre {
  color: var(--app-accent);
}

.log-source.post {
  color: var(--app-success);
}

.log-text {
  word-break: break-word;
}

.script-badge :deep(.el-badge__content) {
  font-size: 10px;
  height: 16px;
  line-height: 16px;
  padding: 0 4px;
}

.request-overlay {
  position: absolute;
  inset: 0;
  background-color: var(--app-bg);
  opacity: 0.65;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10;
}

.overlay-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 24px 36px;
  border-radius: 8px;
  background-color: var(--app-bg-secondary);
  border: 1px solid var(--app-border);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.2);
}

.overlay-spinner {
  font-size: 28px;
  color: var(--app-accent);
  animation: rotating 1s linear infinite;
}

.overlay-timer {
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 18px;
  color: var(--app-accent);
}

.overlay-status {
  font-size: 12px;
  color: var(--app-text-secondary);
}

@keyframes rotating {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.timing-tag {
  cursor: help;
}

/* 响应元数据标签：高对比度，符合 WCAG 标准 */
.meta-tag {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 8px;
  border-radius: 3px;
  font-size: 12px;
  font-weight: 600;
  line-height: 1;
  background-color: var(--app-bg-tertiary);
  color: var(--app-text);
  border: 1px solid var(--app-border);
}

.meta-success {
  background-color: rgba(102, 221, 106, 0.18);
  color: #8aff8e;
  border-color: rgba(102, 221, 106, 0.4);
}

.meta-danger {
  background-color: rgba(255, 102, 89, 0.18);
  color: #ff8a80;
  border-color: rgba(255, 102, 89, 0.4);
}

.meta-accent {
  background-color: rgba(94, 231, 203, 0.15);
  color: var(--app-accent);
  border-color: rgba(94, 231, 203, 0.35);
}

:global([data-theme='dark-2']) .meta-success {
  background-color: rgba(86, 211, 100, 0.2);
  color: #7ee787;
  border-color: rgba(86, 211, 100, 0.4);
}

:global([data-theme='dark-2']) .meta-danger {
  background-color: rgba(255, 123, 114, 0.2);
  color: #ffa198;
  border-color: rgba(255, 123, 114, 0.4);
}

:global([data-theme='dark-2']) .meta-accent {
  background-color: rgba(122, 184, 255, 0.18);
  color: #9ecaff;
  border-color: rgba(122, 184, 255, 0.4);
}

/* 彻底清除 Element Plus tooltip 默认 padding，由内部 .timing-tooltip 自行控制 */
:global(.timing-tooltip-popper) {
  padding: 0 !important;
  background: transparent !important;
  border: none !important;
  box-shadow: none !important;
  max-height: none !important;
  overflow: visible !important;
}

:global(.timing-tooltip-popper .el-tooltip__popper) {
  padding: 0 !important;
  background: transparent !important;
  border: none !important;
  box-shadow: none !important;
  max-height: none !important;
  overflow: visible !important;
}

:global(.timing-tooltip) {
  min-width: 160px;
  background-color: var(--app-bg-secondary);
  color: var(--app-text);
  border: 1px solid var(--app-border);
  border-radius: 4px;
  padding: 6px 10px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
}

:global(.timing-tooltip) .timing-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  font-size: 12px;
  line-height: 1.6;
}

:global(.timing-tooltip) .timing-label {
  color: var(--app-text-secondary);
}

:global(.timing-tooltip) .timing-value {
  font-family: 'Consolas', 'Monaco', monospace;
  color: var(--app-text);
  font-weight: 600;
}

:global(.timing-tooltip) .timing-divider {
  height: 1px;
  background-color: var(--app-border);
  margin: 4px 0;
}
</style>
