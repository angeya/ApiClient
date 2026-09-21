<script setup lang="ts">
import { computed, ref, watch, nextTick, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import { useWorkspaceStore } from '../../stores/workspace'
import { api, responseBodyToString } from '../../services/api'
import KeyValueEditor from '../common/KeyValueEditor.vue'
import BodyEditor from './BodyEditor.vue'
import ScriptEditor from './ScriptEditor.vue'
import LoadTestTab from './LoadTestTab.vue'
import WebSocketTab from './WebSocketTab.vue'
import type { models } from '../../../wailsjs/go/models'
import type { QueryParam, DirMeta, Header } from '../../types'

const workspace = useWorkspaceStore()

const editorTab = ref('params')
const preview = ref<models.RequestPreview | null>(null)
const previewLoading = ref(false)
const dirMeta = ref<DirMeta | null>(null)

// WebSocket 连接状态
const wsConnecting = ref(false)

// URL 与 Query 表格双向同步的标记
let syncSource: 'url' | 'query' | null = null
// 切换标签页时抑制 URL/Query 自动同步，避免初始加载误标记 dirty
let suppressInitialSync = false

const draft = computed({
  get: () => workspace.activeDraft,
  set: (v) => {
    const tab = workspace.activeTab
    if (!tab || !v) return
    ensureArrays(v)
    workspace.updateDraft(tab.rel, v)
  },
})

const wsConnected = computed(() => {
  return draft.value ? workspace.isWsConnected(draft.value.id) : false
})

const httpMethods = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS']
const wsMethods = ['WS', 'WSS']

const methodOptions = computed(() => {
  return draft.value?.type === 'websocket' ? wsMethods : httpMethods
})

function ensureArrays(req: any) {
  if (!req.headers) req.headers = []
  if (!req.query) req.query = []
  if (!req.formData) req.formData = []
}

function update(patch: any) {
  if (!draft.value) return
  ensureArrays(draft.value)
  draft.value = { ...draft.value, ...patch }
}

function toggleInheritedHeader(key: string, enabled: boolean) {
  if (!draft.value) return
  const normalizedKey = key.trim().toLowerCase()
  const current = new Set((draft.value.disabledInheritedHeaderKeys || []).map((k) => k.toLowerCase()))
  if (enabled) {
    current.delete(normalizedKey)
  } else {
    current.add(normalizedKey)
  }
  draft.value = { ...draft.value, disabledInheritedHeaderKeys: Array.from(current) }
}

function currentDirRel(rel: string): string {
  return rel.includes('\\') ? rel.substring(0, rel.lastIndexOf('\\')) : ''
}

async function loadDirMeta() {
  const tab = workspace.activeTab
  if (!tab) {
    dirMeta.value = null
    return
  }
  const dirRel = currentDirRel(tab.rel)
  if (!dirRel) {
    dirMeta.value = null
    return
  }
  try {
    dirMeta.value = (await api.loadDirMeta(dirRel)) as DirMeta
  } catch {
    dirMeta.value = null
  }
}

const inheritedHeaders = computed<Header[]>(() => {
  const disabled = new Set((draft.value?.disabledInheritedHeaderKeys || []).map((k) => k.toLowerCase()))
  const list: Header[] = []
  for (const h of workspace.settings.globalHeaders || []) {
    if (h.key?.trim()) list.push({ ...h, enabled: !disabled.has(h.key.trim().toLowerCase()) })
  }
  for (const h of dirMeta.value?.headers || []) {
    if (h.key?.trim()) list.push({ ...h, enabled: !disabled.has(h.key.trim().toLowerCase()) })
  }
  return list
})

const queryCount = computed(() => {
  return (draft.value?.query || []).filter((q) => q.key.trim() !== '').length
})

const headerCount = computed(() => {
  const custom = (draft.value?.headers || []).filter((h) => h.enabled && h.key.trim() !== '').length
  const inherited = inheritedHeaders.value.filter((h) => h.enabled && h.key.trim() !== '').length
  return custom + inherited
})

const bodyCount = computed(() => {
  if (!draft.value) return 0
  if (draft.value.bodyType === 'json' || draft.value.bodyType === 'text') {
    return draft.value.body?.trim() ? 1 : 0
  }
  if (draft.value.bodyType === 'form-data' || draft.value.bodyType === 'x-www-form-urlencoded') {
    return (draft.value.formData || []).filter((f) => f.key.trim() !== '').length
  }
  return 0
})

const hasPreScript = computed(() => !!draft.value?.preRequestScript?.trim())
const hasPostScript = computed(() => !!draft.value?.postResponseScript?.trim())

function normalizeUrl(url: string, type: string, method: string): string {
  const trimmed = url.trim()
  if (!trimmed) return trimmed
  if (type === 'websocket') {
    if (/^wss?:\/\//i.test(trimmed)) return trimmed
    return (method === 'WSS' ? 'wss://' : 'ws://') + trimmed
  }
  if (/^https?:\/\//i.test(trimmed)) return trimmed
  return 'http://' + trimmed
}

function parseUrlQueries(url: string): QueryParam[] {
  const list: QueryParam[] = []
  try {
    const qIndex = url.indexOf('?')
    if (qIndex === -1) return list
    const params = new URLSearchParams(url.slice(qIndex + 1))
    const map = new Map<string, string>()
    params.forEach((value, key) => {
      map.set(key, value)
    })
    map.forEach((value, key) => {
      list.push({ key, value, enabled: true, description: '' })
    })
  } catch {
    // 忽略非法 URL
  }
  return list
}

function mergeQueries(current: QueryParam[], fromUrl: QueryParam[]): QueryParam[] {
  const urlMap = new Map(fromUrl.map((p) => [p.key, p.value]))
  const merged: QueryParam[] = []
  const seen = new Set<string>()

  for (const row of current) {
    if (row.key === '') {
      merged.push(row)
      continue
    }
    if (urlMap.has(row.key)) {
      merged.push({ ...row, value: urlMap.get(row.key)!, enabled: true })
      seen.add(row.key)
    }
  }

  for (const p of fromUrl) {
    if (!seen.has(p.key)) {
      merged.push(p)
    }
  }

  return merged
}

function buildUrlWithQueries(url: string, queries: QueryParam[]): string {
  try {
    const qIndex = url.indexOf('?')
    const hashIndex = url.indexOf('#')
    const base = qIndex !== -1 ? url.slice(0, qIndex) : hashIndex !== -1 ? url.slice(0, hashIndex) : url
    const hash = hashIndex !== -1 ? url.slice(hashIndex) : ''
    const params = new URLSearchParams()
    for (const q of queries) {
      if (q.enabled && q.key.trim() !== '') {
        params.set(q.key, q.value)
      }
    }
    const search = params.toString()
    return search ? `${base}?${search}${hash}` : `${base}${hash}`
  } catch {
    return url
  }
}

async function send() {
  const tab = workspace.activeTab
  const req = plainDraft()
  if (!tab || !req) return
  req.url = normalizeUrl(req.url, req.type, req.method)
  workspace.startSendingTimer(req.id)
  try {
    const dirRel = currentDirRel(tab.rel)
    const resp = await api.sendHttpRequest(req, dirRel)
    window.dispatchEvent(new CustomEvent('api-response', { detail: resp }))
  } catch (err) {
    ElMessage.error(`请求失败: ${err}`)
  } finally {
    // 无论成功、失败或取消，都必须清除发送中状态，确保遮罩/加载态能正常消失
    workspace.stopSendingTimer()
  }
}

async function connectWebSocket() {
  const tab = workspace.activeTab
  const req = plainDraft()
  if (!tab || !req || req.type !== 'websocket') return
  req.url = normalizeUrl(req.url, req.type, req.method)
  wsConnecting.value = true
  try {
    await api.connectWebSocket(req.id, req, currentDirRel(tab.rel))
    workspace.setWsConnected(req.id, true)
    ElMessage.success('WebSocket 已连接')
  } catch (err) {
    workspace.setWsConnected(req.id, false)
    ElMessage.error(`连接失败: ${err}`)
  } finally {
    wsConnecting.value = false
  }
}

async function disconnectWebSocket() {
  const req = plainDraft()
  if (!req || req.type !== 'websocket') return
  try {
    await api.disconnectWebSocket(req.id)
    workspace.setWsConnected(req.id, false)
    ElMessage.info('WebSocket 已断开')
  } catch (err) {
    ElMessage.error(`断开失败: ${err}`)
  }
}

function toggleWebSocket() {
  if (wsConnected.value) {
    disconnectWebSocket()
  } else {
    connectWebSocket()
  }
}

function onWsDisconnected() {
  const req = draft.value
  if (!req) return
  workspace.setWsConnected(req.id, false)
}

async function save() {
  const tab = workspace.activeTab
  if (!tab) return
  await workspace.saveRequest(tab.rel)
  ElMessage.success('已保存')
}

function onKeyDown(e: KeyboardEvent) {
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 's') {
    e.preventDefault()
    const tab = workspace.activeTab
    if (tab && workspace.dirtyMap.get(tab.rel)) {
      save()
    }
  }
}

onMounted(() => {
  document.addEventListener('keydown', onKeyDown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', onKeyDown)
})

function onMethodChange(method: string) {
  if (!draft.value) return
  update({ method })
}

function plainDraft(): any {
  if (!draft.value) return null
  return JSON.parse(JSON.stringify(draft.value))
}

async function loadPreview() {
  const tab = workspace.activeTab
  const req = plainDraft()
  if (!tab || !req) return
  previewLoading.value = true
  preview.value = null
  try {
    preview.value = await api.previewRequest(req, currentDirRel(tab.rel))
  } catch (err) {
    ElMessage.error(`预览失败: ${err}`)
  } finally {
    previewLoading.value = false
  }
}

watch(editorTab, (tab) => {
  if (tab === 'preview') loadPreview()
})

watch(() => workspace.activeDraft, () => {
  if (editorTab.value === 'preview') loadPreview()
})

watch(() => workspace.activeTab?.rel, () => {
  // 切换标签页时抑制一次 URL/Query 自动同步，避免初始加载误标记 dirty
  suppressInitialSync = true
  nextTick(() => {
    suppressInitialSync = false
  })
  // 每次打开接口时默认展示 Query 标签页
  editorTab.value = 'params'
  loadDirMeta()
}, { immediate: true })

// URL 变化时同步到 Query 表格
watch(
  () => draft.value?.url,
  async (url) => {
    if (suppressInitialSync || !draft.value || !url || syncSource === 'query') return
    syncSource = 'url'
    const fromUrl = parseUrlQueries(url)
    const merged = mergeQueries(draft.value.query || [], fromUrl)
    if (JSON.stringify(merged) !== JSON.stringify(draft.value.query || [])) {
      update({ query: merged })
    }
    await nextTick()
    syncSource = null
  }
)

// Query 表格变化时同步回 URL
watch(
  () => draft.value?.query,
  async (query) => {
    if (suppressInitialSync || !draft.value || syncSource === 'url') return
    syncSource = 'query'
    const newUrl = buildUrlWithQueries(draft.value.url, query || [])
    if (newUrl !== draft.value.url) {
      update({ url: newUrl })
    }
    await nextTick()
    syncSource = null
  },
  { deep: true }
)
</script>

<template>
  <div v-if="!draft" class="empty-editor">
    <el-empty description="从左侧选择一个请求开始" />
  </div>
  <div v-else class="request-editor">
    <Teleport to="#request-toolbar-target">
      <div class="toolbar">
        <el-select
          :model-value="draft.method"
          size="default"
          class="method-select"
          @update:model-value="onMethodChange"
        >
          <el-option
            v-for="m in methodOptions"
            :key="m"
            :label="m"
            :value="m"
          />
        </el-select>
        <el-input
          v-model="draft.url"
          class="url-input"
          :placeholder="draft.type === 'websocket' ? '输入 WebSocket URL，支持 {{变量}}' : '输入 URL，支持 {{变量}}'"
          @input="update({})"
          @keyup.enter="draft.type === 'websocket' ? toggleWebSocket() : send()"
        />
        <template v-if="draft.type === 'websocket'">
          <el-button v-if="!wsConnected" type="primary" :loading="wsConnecting" @click="connectWebSocket">连接</el-button>
          <el-button v-else type="danger" @click="disconnectWebSocket">断开</el-button>
        </template>
        <template v-else>
          <!-- 发送按钮保持“发送”不变，请求期间仅通过 loading/disabled 变为不可点击，取消操作由响应面板遮罩中的取消按钮完成 -->
          <el-button type="primary" :loading="workspace.sending" @click="send">发送</el-button>
        </template>
        <el-button @click="save">保存</el-button>
      </div>
    </Teleport>

    <div v-if="draft.type === 'websocket'" class="editor-body websocket-body">
      <WebSocketTab :connected="wsConnected" @disconnected="onWsDisconnected" />
    </div>
    <div v-else class="editor-body">
      <el-tabs v-model="editorTab" class="editor-tabs">
        <el-tab-pane name="headers">
          <template #label>
            <span>Headers</span>
            <span v-if="headerCount > 0" class="tab-count">{{ headerCount }}</span>
          </template>
          <KeyValueEditor
            :model-value="draft.headers"
            :inherited-rows="inheritedHeaders"
            @update:model-value="update({ headers: $event })"
            @toggle-inherited="toggleInheritedHeader"
          />
        </el-tab-pane>
        <el-tab-pane name="params">
          <template #label>
            <span>Query</span>
            <span v-if="queryCount > 0" class="tab-count">{{ queryCount }}</span>
          </template>
          <KeyValueEditor :model-value="draft.query" @update:model-value="update({ query: $event })" />
        </el-tab-pane>
        <el-tab-pane name="body">
          <template #label>
            <span>Body</span>
            <span v-if="bodyCount > 0" class="tab-count">{{ bodyCount }}</span>
          </template>
          <BodyEditor v-model="draft" />
        </el-tab-pane>
        <el-tab-pane name="pre">
          <template #label>
            <span>Pre-request</span>
            <span v-if="hasPreScript" class="tab-dot" />
          </template>
          <ScriptEditor
            :model-value="draft.preRequestScript"
            @update:model-value="update({ preRequestScript: $event })"
          />
        </el-tab-pane>
        <el-tab-pane name="post">
          <template #label>
            <span>Post-response</span>
            <span v-if="hasPostScript" class="tab-dot" />
          </template>
          <ScriptEditor
            :model-value="draft.postResponseScript"
            @update:model-value="update({ postResponseScript: $event })"
          />
        </el-tab-pane>
        <el-tab-pane label="实际请求" name="preview">
          <div v-if="previewLoading" class="preview-loading">正在解析...</div>
          <div v-else-if="preview" class="preview-panel">
            <div class="preview-section">
              <div class="preview-title">解析后的 URL</div>
              <div class="preview-value">{{ preview.url || '-' }}</div>
            </div>
            <div class="preview-section">
              <div class="preview-title">Headers（含全局/目录继承）</div>
              <div v-for="(h, idx) in preview.headers || []" :key="idx" class="preview-row">
                <span class="preview-key">{{ h.key }}</span>
                <span class="preview-val">{{ h.value }}</span>
              </div>
              <div v-if="!(preview.headers || []).length" class="preview-empty">无</div>
            </div>
            <div class="preview-section">
              <div class="preview-title">Query 参数</div>
              <div v-for="(q, idx) in preview.query || []" :key="idx" class="preview-row">
                <span class="preview-key">{{ q.key }}</span>
                <span class="preview-val">{{ q.value }}</span>
              </div>
              <div v-if="!(preview.query || []).length" class="preview-empty">无</div>
            </div>
            <div class="preview-section">
              <div class="preview-title">Body 内容</div>
              <div v-if="preview.bodyType === 'none' || preview.bodyType === ''" class="preview-empty">无 Body</div>
              <div v-else-if="preview.bodyType === 'form-data'" class="preview-body">
                <div class="preview-meta">Content-Type: {{ preview.contentType || 'multipart/form-data' }}</div>
                <div v-for="(f, idx) in preview.formData || []" :key="idx" class="preview-row">
                  <span class="preview-key">{{ f.key }} ({{ f.type === 'file' ? '文件' : '文本' }})</span>
                  <span class="preview-val">{{ f.value }}</span>
                </div>
                <div v-if="!(preview.formData || []).length" class="preview-empty">无字段</div>
              </div>
              <div v-else-if="preview.bodyType === 'x-www-form-urlencoded'" class="preview-body">
                <div class="preview-meta">Content-Type: {{ preview.contentType || 'application/x-www-form-urlencoded' }}</div>
                <pre class="preview-code">{{ preview.body || '无' }}</pre>
              </div>
              <div v-else class="preview-body">
                <div class="preview-meta">Content-Type: {{ preview.contentType || preview.bodyType }}</div>
                <pre class="preview-code">{{ preview.body || '无' }}</pre>
              </div>
            </div>
          </div>
        </el-tab-pane>
        <el-tab-pane label="压测" name="loadtest">
          <LoadTestTab />
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>

<style scoped>
.empty-editor {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
}

.request-editor {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  height: 100%;
  box-sizing: border-box;
}

.method-select {
  width: 110px;
  flex-shrink: 0;
}

.url-input {
  flex: 1;
  min-width: 120px;
}

.editor-body {
  flex: 1;
  min-height: 0;
  padding: 12px;
  overflow: auto;
}

.editor-tabs {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.editor-tabs :deep(.el-tabs__content) {
  flex: 1;
  overflow: auto;
}

.websocket-body {
  padding: 0;
  overflow: hidden;
}

.preview-loading {
  padding: 12px;
  font-size: 13px;
  color: var(--app-text-secondary);
}

.preview-panel {
  display: flex;
  flex-direction: column;
  gap: 5px;
  padding: 12px 4px 4px;
}

.preview-section {
  display: flex;
  flex-direction: column;
  border: 1px solid var(--app-border);
  border-radius: 4px;
  overflow: hidden;
}

.preview-title {
  font-size: 13px;
  font-weight: 700;
  color: var(--app-text);
  background-color: var(--app-bg-secondary);
  padding: 6px 10px;
  border-bottom: 1px solid var(--app-border);
}

.preview-section .preview-value,
.preview-section .preview-row,
.preview-section .preview-empty,
.preview-section .preview-body {
  padding: 3px 10px;
}

.preview-section .preview-value:last-child,
.preview-section .preview-row:last-child,
.preview-section .preview-empty:last-child,
.preview-section .preview-body:last-child,
.preview-section > .preview-value:only-child,
.preview-section > .preview-empty:only-child {
  padding-bottom: 8px;
}

.preview-value {
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 12px;
  word-break: break-word;
}

.preview-row {
  display: flex;
  gap: 12px;
  font-size: 12px;
}

.preview-key {
  width: 160px;
  color: var(--app-text-secondary);
  flex-shrink: 0;
}

.preview-val {
  font-family: 'Consolas', 'Monaco', monospace;
  word-break: break-word;
}

.preview-empty {
  font-size: 12px;
  color: var(--app-text-secondary);
}

.preview-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.preview-meta {
  font-size: 12px;
  color: var(--app-text-secondary);
}

.preview-code {
  margin: 0;
  padding: 10px;
  border-radius: 4px;
  background-color: var(--app-bg-secondary);
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-word;
  color: var(--app-text);
  max-height: 300px;
  overflow: auto;
}

.tab-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 14px;
  height: 14px;
  padding: 0 3px;
  border-radius: 7px;
  background-color: var(--app-text-secondary);
  color: var(--app-bg);
  font-size: 10px;
  margin-left: 4px;
  vertical-align: middle;
}

.tab-dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background-color: var(--app-accent);
  margin-left: 4px;
  vertical-align: middle;
}
</style>
