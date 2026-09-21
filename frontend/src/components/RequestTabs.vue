<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import { ElMessageBox } from 'element-plus/es/components/message-box/index'
import { useWorkspaceStore } from '../stores/workspace'
import type { RequestTab } from '../types'

const workspace = useWorkspaceStore()

const tabs = computed(() => workspace.tabs)

function activate(id: string) {
  workspace.setActiveTab(id)
}

async function close(id: string) {
  const ok = await workspace.closeTab(id)
  if (!ok) {
    try {
      await ElMessageBox.confirm('当前请求有未保存的更改，是否保存？', '未保存', {
        confirmButtonText: '保存',
        cancelButtonText: '放弃',
        distinguishCancelAndClose: true,
        type: 'warning',
      })
      const tab = workspace.tabs.find((t) => t.id === id)
      if (tab) await workspace.saveRequest(tab.rel)
      await workspace.forceCloseTab(id)
    } catch (action: any) {
      if (action === 'cancel') {
        await workspace.forceCloseTab(id)
      }
    }
  }
}

async function createNewRequest(type: 'http' | 'websocket') {
  try {
    await workspace.createRequest('', '未命名', type)
  } catch (err) {
    ElMessage.error(`创建失败: ${err}`)
  }
}

function onCommand(cmd: string) {
  if (cmd === 'http' || cmd === 'websocket') {
    createNewRequest(cmd)
  }
}

async function renameTab(tab: RequestTab) {
  try {
    const { value } = await ElMessageBox.prompt('请输入新名称', '重命名', {
      inputValue: tab.name,
      confirmButtonText: '确定',
      cancelButtonText: '取消',
    })
    if (value && value.trim()) {
      await workspace.rename(tab.rel, value.trim(), false)
    }
  } catch {
    // cancelled
  }
}

async function cloneTab(tab: RequestTab) {
  try {
    const newName = tab.name + ' 副本'
    await workspace.cloneRequest(tab.rel, newName)
    ElMessage.success('已克隆')
  } catch (err) {
    ElMessage.error(`克隆失败: ${err}`)
  }
}

const contextMenuVisible = ref(false)
const contextMenuX = ref(0)
const contextMenuY = ref(0)
const contextMenuTab = ref<RequestTab | null>(null)

function showContextMenu(e: MouseEvent, tab: RequestTab) {
  e.preventDefault()
  contextMenuTab.value = tab
  contextMenuX.value = e.clientX
  contextMenuY.value = e.clientY
  contextMenuVisible.value = true
}

function hideContextMenu() {
  contextMenuVisible.value = false
  contextMenuTab.value = null
}

function onMenuRename() {
  if (contextMenuTab.value) {
    renameTab(contextMenuTab.value)
  }
  hideContextMenu()
}

function onMenuClone() {
  if (contextMenuTab.value) {
    cloneTab(contextMenuTab.value)
  }
  hideContextMenu()
}

function onMenuClose() {
  if (contextMenuTab.value) {
    close(contextMenuTab.value.id)
  }
  hideContextMenu()
}

function onDocumentClick() {
  hideContextMenu()
}

onMounted(() => {
  document.addEventListener('click', onDocumentClick)
})

onUnmounted(() => {
  document.removeEventListener('click', onDocumentClick)
})
</script>

<template>
  <div class="request-tabs">
    <div
      v-for="tab in tabs"
      :key="tab.id"
      :class="['tab-item', { active: tab.id === workspace.activeTabId }]"
      @click="activate(tab.id)"
      @mousedown.middle.prevent.stop="close(tab.id)"
      @contextmenu.prevent.stop="showContextMenu($event, tab)"
    >
      <span :class="{ 'dirty-dot': tab.dirty }">{{ tab.name }}</span>
      <el-icon class="close-icon" @click.stop="close(tab.id)"><Close /></el-icon>
    </div>
    <el-dropdown trigger="click" @command="onCommand" placement="bottom-start">
      <div class="tab-add-btn" title="新建请求">
        <el-icon><Plus /></el-icon>
      </div>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item command="http">HTTP</el-dropdown-item>
          <el-dropdown-item command="websocket">WebSocket</el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>
    <div v-if="tabs.length === 0" class="tab-empty">未打开任何请求</div>
  </div>

  <div
    v-if="contextMenuVisible"
    class="tab-context-menu"
    :style="{ left: contextMenuX + 'px', top: contextMenuY + 'px' }"
    @click.stop
  >
    <div class="tab-context-item" @click="onMenuRename">重命名</div>
    <div class="tab-context-item" @click="onMenuClone">克隆</div>
    <div class="tab-context-divider" />
    <div class="tab-context-item" @click="onMenuClose">关闭</div>
  </div>
</template>

<style scoped>
.request-tabs {
  display: flex;
  align-items: center;
  height: 38px;
  min-height: 38px;
  background-color: var(--app-bg-secondary);
  border-bottom: 1px solid var(--app-border);
  overflow-x: auto;
  overflow-y: hidden;
}

.tab-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 12px;
  height: 38px;
  font-size: 12px;
  cursor: pointer;
  border-right: 1px solid var(--app-border);
  white-space: nowrap;
  user-select: none;
  position: relative;
  transition: background-color 0.15s;
}

.tab-item:hover {
  background-color: var(--app-bg-tertiary);
}

.tab-item.active {
  background-color: var(--app-bg);
}

.tab-item::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 2px;
  background-color: var(--app-accent);
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.2s;
}

.tab-item.active::after {
  opacity: 1;
}

.close-icon {
  font-size: 12px;
  opacity: 0.6;
}

.close-icon:hover {
  opacity: 1;
  color: var(--app-danger);
}

.tab-add-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 100%;
  cursor: pointer;
  color: var(--app-text-secondary);
  flex-shrink: 0;
  transition: all 0.15s;
}

.tab-add-btn:hover {
  color: var(--app-accent);
  background-color: var(--app-bg-tertiary);
}

.tab-empty {
  padding: 0 16px;
  font-size: 12px;
  color: var(--app-text-secondary);
}

.tab-context-menu {
  position: fixed;
  z-index: 2000;
  min-width: 120px;
  padding: 4px 0;
  background-color: var(--app-bg);
  border: 1px solid var(--app-border);
  border-radius: 4px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.2);
  font-size: 13px;
}

.tab-context-item {
  padding: 8px 16px;
  cursor: pointer;
  color: var(--app-text);
  transition: background-color 0.15s, color 0.15s;
}

.tab-context-item:hover {
  background-color: var(--app-bg-tertiary);
  color: var(--app-accent);
}

.tab-context-divider {
  height: 1px;
  margin: 4px 0;
  background-color: var(--app-border);
}
</style>
