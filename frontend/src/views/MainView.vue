<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { ElMessage, ElMessageBox, ElNotification } from 'element-plus'
import { EventsOn, Quit } from '../../wailsjs/runtime/runtime'
import TitleBar from '../components/TitleBar.vue'
import Sidebar from '../components/Sidebar.vue'
import RightToolbar from '../components/RightToolbar.vue'
import RequestTabs from '../components/RequestTabs.vue'
import RequestEditor from '../components/request/RequestEditor.vue'
import ResponsePanel from '../components/response/ResponsePanel.vue'
import FetchImportDialog from '../components/FetchImportDialog.vue'
import { api } from '../services/api'
import { useWorkspaceStore } from '../stores/workspace'
import { useThemeStore } from '../stores/theme'

const workspace = useWorkspaceStore()
const theme = useThemeStore()

const loading = ref(true)
const fetchVisible = ref(false)
const fetchCode = ref('')
const closing = ref(false)

// 可拖拽调节宽度的面板状态
const sidebarWidth = ref(260)
const lowerPaneWidth = ref(50) // 下部左侧请求区宽度百分比

// WebSocket 请求使用单栏布局，不显示右侧响应面板
const isWebSocket = computed(() => workspace.activeDraft?.type === 'websocket')

// 拖拽状态
let dragSide: 'sidebar' | 'lower' | null = null
let startX = 0
let startWidth = 0

function startDrag(side: 'sidebar' | 'lower', e: MouseEvent) {
  e.preventDefault()
  e.stopPropagation()
  dragSide = side
  startX = e.clientX
  if (side === 'sidebar') {
    startWidth = sidebarWidth.value
  } else {
    startWidth = lowerPaneWidth.value
  }
  document.addEventListener('mousemove', onDrag)
  document.addEventListener('mouseup', stopDrag)
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
}

function onDrag(e: MouseEvent) {
  if (!dragSide) return
  const delta = e.clientX - startX
  if (dragSide === 'sidebar') {
    // 向右拖增加侧栏宽度
    const newWidth = startWidth + delta
    sidebarWidth.value = Math.max(180, Math.min(500, newWidth))
  } else if (dragSide === 'lower') {
    // 根据主区域宽度计算新的左侧百分比
    const mainWidth = getMainAreaWidth()
    if (mainWidth <= 0) return
    const newPercent = startWidth + (delta / mainWidth) * 100
    lowerPaneWidth.value = Math.max(20, Math.min(80, newPercent))
  }
}

function getMainAreaWidth(): number {
  const el = document.querySelector('.main-area')
  return el ? el.getBoundingClientRect().width : 0
}

function stopDrag() {
  dragSide = null
  document.removeEventListener('mousemove', onDrag)
  document.removeEventListener('mouseup', stopDrag)
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
}

onUnmounted(() => {
  document.removeEventListener('mousemove', onDrag)
  document.removeEventListener('mouseup', stopDrag)
})

async function doClose(saveDirty: boolean) {
  if (saveDirty) {
    for (const tab of workspace.tabs) {
      if (tab.dirty) {
        await workspace.saveRequest(tab.rel)
      }
    }
  }
  await api.allowClose()
  Quit()
}

onMounted(async () => {
  try {
    const defaultPath = await api.getDefaultWorkspaceRoot()
    await workspace.openWorkspace(defaultPath)
    theme.initTheme(workspace.settings.theme)
    EventsOn('clipboard:fetch', (code: string) => {
      fetchCode.value = code
      ElNotification({
        title: '检测到 Fetch 代码',
        message: '剪贴板中包含 fetch 调用，是否导入？',
        type: 'info',
        duration: 8000,
        onClick: () => {
          fetchVisible.value = true
        },
      })
    })
    EventsOn('app:close-request', async () => {
      if (closing.value) return
      closing.value = true
      if (!workspace.hasDirty) {
        await doClose(false)
        return
      }
      try {
        await ElMessageBox.confirm('当前有未保存的更改，关闭前是否保存？', '未保存的更改', {
          confirmButtonText: '保存后退出',
          cancelButtonText: '放弃更改',
          distinguishCancelAndClose: true,
          type: 'warning',
        })
        await doClose(true)
      } catch (action: any) {
        if (action === 'cancel') {
          await doClose(false)
        } else {
          closing.value = false
        }
      }
    })
  } catch (err) {
    ElMessage.error(`打开工作区失败: ${err}`)
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div v-if="loading" class="loading-screen">
    <span>正在加载 ApiClient...</span>
  </div>
  <div v-else class="app-layout">
    <TitleBar />
    <div class="app-body">
      <Sidebar class="sidebar" :style="{ width: sidebarWidth + 'px', flexShrink: 0 }" />
      <div class="resize-divider" @mousedown="startDrag('sidebar', $event)" />
      <div class="main-area">
        <RequestTabs />
        <div id="request-toolbar-target" class="request-toolbar-area" />
        <div v-if="workspace.tabs.length" class="lower-area">
          <RequestEditor
            class="request-pane"
            :class="{ 'websocket-pane': isWebSocket }"
            :style="isWebSocket ? { width: '100%', flexShrink: 0 } : { width: lowerPaneWidth + '%', flexShrink: 0 }"
          />
          <template v-if="!isWebSocket">
            <div class="resize-divider" @mousedown="startDrag('lower', $event)" />
            <ResponsePanel
              class="response-pane"
              :style="{ width: (100 - lowerPaneWidth) + '%', flexShrink: 0 }"
            />
          </template>
        </div>
        <div v-else class="lower-empty">
          <el-empty description="从左侧选择一个请求开始" />
        </div>
      </div>
      <RightToolbar class="right-toolbar" />
    </div>
    <FetchImportDialog v-model="fetchVisible" :initial-code="fetchCode" />
  </div>
</template>

<style scoped>
.loading-screen {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: var(--app-bg);
  color: var(--app-text);
}

.app-layout {
  display: flex;
  flex-direction: column;
  height: 100%;
  position: relative;
}

.app-body {
  flex: 1;
  display: flex;
  min-height: 0;
  overflow: hidden;
}

.main-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  background-color: var(--app-bg);
}

.request-toolbar-area {
  min-height: 46px;
  border-bottom: 1px solid var(--app-border);
  background-color: var(--app-bg);
}

.lower-area {
  flex: 1;
  display: flex;
  min-height: 0;
  overflow: hidden;
}

.lower-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.request-pane,
.response-pane {
  overflow: hidden;
}

.resize-divider {
  width: 4px;
  flex-shrink: 0;
  cursor: col-resize;
  background-color: var(--app-border);
  position: relative;
  z-index: 10;
  transition: background-color 0.15s;
}

.resize-divider:hover,
.resize-divider:active {
  background-color: var(--app-accent);
}
</style>
