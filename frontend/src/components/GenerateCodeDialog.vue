<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import { useWorkspaceStore } from '../stores/workspace'
import { api } from '../services/api'
import { CURL_FLAVORS, generateCurl, getCurlWarning, type CurlFlavor } from '../utils/codegen'
import { normalizeUrl } from '../utils/helpers'
import type { models } from '../../wailsjs/go/models'

const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const workspace = useWorkspaceStore()

// 默认展示 curl (cmd)，默认多行展示
const flavor = ref<CurlFlavor>('cmd')
const singleLine = ref(false)
const preview = ref<models.RequestPreview | null>(null)
const loading = ref(false)
const errorMessage = ref('')

const isWebSocket = computed(() => workspace.activeDraft?.type === 'websocket')

const generatedCode = computed(() => {
  if (!preview.value || isWebSocket.value) return ''
  return generateCurl({
    preview: preview.value,
    method: workspace.activeDraft?.method || 'GET',
    flavor: flavor.value,
    singleLine: singleLine.value,
  })
})

/** 生成结果的已知局限提示（如 CMD 下的多行文本体） */
const warning = computed(() => {
  if (!preview.value || isWebSocket.value) return ''
  return getCurlWarning(preview.value, flavor.value)
})

/** 取请求所在目录，用于继承目录级请求头与变量 */
function currentDirRel(rel: string): string {
  return rel.includes('\\') ? rel.substring(0, rel.lastIndexOf('\\')) : ''
}

/** 打开弹窗时按当前草稿解析一次实际请求 */
async function loadPreview() {
  const tab = workspace.activeTab
  const draft = workspace.activeDraft
  preview.value = null
  errorMessage.value = ''
  if (!tab || !draft) {
    errorMessage.value = '请先在左侧选择一个请求'
    return
  }
  if (draft.type === 'websocket') {
    errorMessage.value = 'WebSocket 请求暂不支持生成命令行代码'
    return
  }
  loading.value = true
  try {
    // 深拷贝后再交给后端，避免解析过程影响编辑中的草稿
    const req = JSON.parse(JSON.stringify(draft))
    // 与实际发送保持一致，未写协议时补上，保证生成的命令地址可用
    req.url = normalizeUrl(req.url, req.type, req.method)
    preview.value = await api.previewRequest(req, currentDirRel(tab.rel))
  } catch (err) {
    errorMessage.value = `解析请求失败: ${err}`
  } finally {
    loading.value = false
  }
}

async function copyCode() {
  if (!generatedCode.value) return
  try {
    await navigator.clipboard.writeText(generatedCode.value)
    ElMessage.success('代码已复制到剪贴板')
  } catch (err) {
    ElMessage.error(`复制失败: ${err}`)
  }
}

function close() {
  emit('update:modelValue', false)
}

watch(
  () => props.modelValue,
  (visible) => {
    if (visible) {
      flavor.value = 'cmd'
      loadPreview()
    }
  }
)
</script>

<template>
  <el-dialog
    :model-value="modelValue"
    title="生成代码"
    width="720px"
    top="8vh"
    class="gen-code-dialog"
    :close-on-click-modal="false"
    @update:model-value="emit('update:modelValue', $event)"
    @close="close"
  >
    <div class="code-toolbar">
      <div class="toolbar-left">
        <el-radio-group v-model="flavor" size="small">
          <el-radio-button v-for="f in CURL_FLAVORS" :key="f.value" :label="f.value">
            {{ f.label }}
          </el-radio-button>
        </el-radio-group>
        <el-switch v-model="singleLine" class="single-line-switch" size="small" />
        <span class="switch-label">单行</span>
      </div>
      <el-button size="small" text :disabled="!generatedCode" @click="copyCode">
        <el-icon><CopyDocument /></el-icon>
        <span class="copy-text">复制</span>
      </el-button>
    </div>

    <div class="code-area">
      <pre v-if="generatedCode" class="code-block">{{ generatedCode }}</pre>
      <div v-else-if="loading" class="code-empty">正在解析请求...</div>
      <div v-else class="code-empty">{{ errorMessage || '暂无可生成的代码' }}</div>
    </div>

    <div v-if="warning" class="code-warning">
      <el-icon><WarningFilled /></el-icon>
      <span>{{ warning }}</span>
    </div>

    <template #footer>
      <el-button @click="close">关闭</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.code-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 10px;
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.single-line-switch {
  margin-left: 4px;
}

.switch-label {
  font-size: 12px;
  color: var(--app-text-secondary);
}

.copy-text {
  margin-left: 4px;
}

/* 让弹窗整体占满可用高度，代码区再吃掉剩余空间 */
.gen-code-dialog {
  display: flex;
  flex-direction: column;
  height: 78vh;
}

.gen-code-dialog :deep(.el-dialog__header),
.gen-code-dialog :deep(.el-dialog__footer) {
  flex-shrink: 0;
}

.gen-code-dialog :deep(.el-dialog__body) {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding-top: 10px;
}

.code-area {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.code-block {
  flex: 1;
  min-height: 0;
  margin: 0;
  padding: 12px;
  overflow: auto;
  border: 1px solid var(--app-border);
  border-radius: 4px;
  background-color: var(--app-bg-secondary);
  color: var(--app-text);
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
  user-select: text;
}

.code-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  color: var(--app-text-secondary);
}

.code-warning {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 10px;
  padding: 6px 10px;
  border-radius: 4px;
  background-color: rgba(230, 162, 60, 0.12);
  color: var(--app-warning, #e6a23c);
  font-size: 12px;
}
</style>
