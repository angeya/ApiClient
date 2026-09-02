<script setup lang="ts">
import { ref, nextTick, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import type { ApiNode } from '../types'
import { useWorkspaceStore } from '../stores/workspace'
import { api } from '../services/api'
import FetchImportDialog from './FetchImportDialog.vue'
import DirMetaDialog from './DirMetaDialog.vue'

const workspace = useWorkspaceStore()

const fetchDialogVisible = ref(false)
const dirMetaVisible = ref(false)
const dirMetaRel = ref('')
const newName = ref('')
const creatingType = ref<'dir' | 'http' | 'websocket'>('http')
const createParentRel = ref('')
const createDialogVisible = ref(false)
const refreshing = ref(false)
const searchQuery = ref('')
const activeSearchQuery = ref('')

const treeRef = ref()
// 跟踪已展开节点，避免 reloadTree 后展开状态丢失
const expandedKeys = ref<string[]>([])

function onNodeExpand(data: ApiNode) {
  if (!expandedKeys.value.includes(data.id)) {
    expandedKeys.value.push(data.id)
  }
}

function onNodeCollapse(data: ApiNode) {
  const idx = expandedKeys.value.indexOf(data.id)
  if (idx !== -1) {
    expandedKeys.value.splice(idx, 1)
  }
}

function allowDrag(draggingNode: any) {
  return !!draggingNode.data?.path
}

function allowDrop(draggingNode: any, dropNode: any, type: string) {
  // 只允许拖到目录节点内（inner），或根级别（根节点视为目录）
  if (type === 'inner') {
    return dropNode.data?.type === 'dir'
  }
  // before/after 只允许同级排序
  return true
}

async function onNodeDrop(draggingNode: any, dropNode: any, dropType: string, ev: DragEvent) {
  const srcData = draggingNode.data as ApiNode
  const destData = dropNode.data as ApiNode

  if (!srcData?.path) return

  let destDirRel = ''
  if (dropType === 'inner') {
    // 拖入某个目录
    if (destData.type !== 'dir') return
    destDirRel = destData.path
    // 确保目标目录在展开列表中，拖入后保持展开
    if (!expandedKeys.value.includes(destData.id)) {
      expandedKeys.value.push(destData.id)
    }
  } else {
    // before/after：同级排序，目标目录为父级
    const parent = dropNode.parent
    if (parent?.data?.path) {
      destDirRel = parent.data.path
    } else {
      destDirRel = ''
    }
  }

  // 避免拖到自己所在目录（无意义的移动）
  const srcParent = srcData.path.includes('\\')
    ? srcData.path.substring(0, srcData.path.lastIndexOf('\\'))
    : ''
  if (destDirRel === srcParent && dropType !== 'inner') {
    // 同目录内排序，文件系统按字母排序，无需移动
    return
  }

  // 保存当前展开状态，移动后恢复
  const savedExpandedKeys = [...expandedKeys.value]
  try {
    await workspace.moveNode(srcData.path, destDirRel)
    // reloadTree 会重置树数据，需在下一个 tick 恢复展开状态
    await nextTick()
    expandedKeys.value = savedExpandedKeys
    ElMessage.success('移动成功')
  } catch (err) {
    ElMessage.error(`移动失败: ${err}`)
    await workspace.reloadTree()
    await nextTick()
    expandedKeys.value = savedExpandedKeys
  }
}

function handleNodeClick(node: ApiNode) {
  if (node.type === 'request') {
    workspace.loadRequest(node.path)
  }
}

function showCreate(parentRel: string, type: 'dir' | 'http' | 'websocket') {
  if (type === 'dir') {
    createParentRel.value = parentRel
    creatingType.value = type
    newName.value = ''
    createDialogVisible.value = true
  } else {
    createRequestDirect(parentRel, type)
  }
}

async function createRequestDirect(parentRel: string, type: 'http' | 'websocket') {
  try {
    await workspace.createRequest(parentRel, '未命名', type)
  } catch (err) {
    ElMessage.error(`创建失败: ${err}`)
  }
}

async function confirmCreate() {
  if (!newName.value.trim()) return
  try {
    if (creatingType.value === 'dir') {
      await workspace.createDir(createParentRel.value, newName.value)
    }
    createDialogVisible.value = false
  } catch (err) {
    ElMessage.error(`创建失败: ${err}`)
  }
}

async function removeNode(node: ApiNode) {
  // 请求类型且持久化数据中 URL 为空时，直接删除无需二次确认
  if (node.type === 'request' && !node.url) {
    try {
      await workspace.deleteNode(node.path, false)
      ElMessage.success('已删除')
    } catch (err) {
      ElMessage.error(`删除失败: ${err}`)
    }
    return
  }
  try {
    await ElMessageBox.confirm(`确定删除 ${node.name} 吗？`, '确认删除', { type: 'warning' })
    await workspace.deleteNode(node.path, node.type === 'dir')
    ElMessage.success('已删除')
  } catch {
    // cancelled
  }
}

async function renameNode(node: ApiNode) {
  try {
    const { value } = await ElMessageBox.prompt('请输入新名称', '重命名', {
      inputValue: node.name,
      confirmButtonText: '确定',
      cancelButtonText: '取消',
    })
    if (value && value.trim()) {
      await workspace.rename(node.path, value.trim(), node.type === 'dir')
    }
  } catch {
    // cancelled
  }
}

async function cloneNode(node: ApiNode) {
  if (node.type !== 'request') return
  try {
    const newName = node.name + ' 副本'
    await workspace.cloneRequest(node.path, newName)
    ElMessage.success('已克隆')
  } catch (err) {
    ElMessage.error(`克隆失败: ${err}`)
  }
}

function openDirMeta(node: ApiNode) {
  dirMetaRel.value = node.path
  dirMetaVisible.value = true
}

async function openWorkspaceInExplorer() {
  if (!workspace.root) {
    ElMessage.warning('工作区未打开')
    return
  }
  try {
    await api.openInExplorer(workspace.root)
  } catch (err) {
    ElMessage.error(`打开目录失败: ${err}`)
  }
}

async function refreshTree() {
  refreshing.value = true
  try {
    await workspace.reloadTree()
    ElMessage.success('接口列表已刷新')
  } catch (err) {
    ElMessage.error(`刷新失败: ${err}`)
  } finally {
    refreshing.value = false
  }
}

function findNodePath(nodes: ApiNode[], targetPath: string): ApiNode[] | null {
  for (const node of nodes) {
    if (node.path === targetPath) {
      return [node]
    }
    if (node.children?.length) {
      const childPath = findNodePath(node.children, targetPath)
      if (childPath) {
        return [node, ...childPath]
      }
    }
  }
  return null
}

async function locateActiveTab() {
  const rel = workspace.activeTab?.rel
  if (!rel) {
    ElMessage.warning('当前没有打开的接口')
    return
  }
  const path = findNodePath(workspace.tree, rel)
  if (!path || path.length === 0) {
    ElMessage.warning('未在列表中找到当前接口')
    return
  }
  if (!treeRef.value) {
    ElMessage.warning('树组件未就绪')
    return
  }

  const targetNode = path[path.length - 1]

  // 展开路径上的所有目录
  const newExpandedKeys = [...expandedKeys.value]
  for (const node of path) {
    if (node.type === 'dir' && !newExpandedKeys.includes(node.id)) {
      newExpandedKeys.push(node.id)
    }
  }
  expandedKeys.value = newExpandedKeys

  // 等待树节点渲染完成
  await nextTick()
  await new Promise((resolve) => setTimeout(resolve, 150))

  // 高亮目标节点
  treeRef.value.setCurrentKey(targetNode.id)

  // 等待 current 样式渲染
  await nextTick()
  await new Promise((resolve) => setTimeout(resolve, 100))

  let contentEl: Element | null = null
  for (let i = 0; i < 10; i++) {
    contentEl =
      document.querySelector(`.tree .el-tree-node[data-key="${targetNode.id}"] > .el-tree-node__content`) ||
      document.querySelector('.tree .el-tree-node.is-current > .el-tree-node__content')
    if (contentEl) break
    await new Promise((resolve) => setTimeout(resolve, 60))
  }

  if (!contentEl) {
    ElMessage.warning('定位节点未渲染到界面')
    return
  }

  const el = contentEl as HTMLElement
  el.scrollIntoView({ behavior: 'smooth', block: 'center' })
  el.classList.remove('locate-flash')
  void el.offsetWidth
  el.classList.add('locate-flash')
  setTimeout(() => el.classList.remove('locate-flash'), 1500)
}

function triggerSearch() {
  activeSearchQuery.value = searchQuery.value
}

const filteredTree = computed<ApiNode[]>(() => {
  const query = activeSearchQuery.value.trim()
  if (!query) return workspace.tree
  return filterNodes(workspace.tree, query)
})

function filterNodes(nodes: ApiNode[], query: string): ApiNode[] {
  const q = query.toLowerCase()
  const result: ApiNode[] = []
  for (const node of nodes) {
    if (node.type === 'request') {
      const nameMatch = node.name.toLowerCase().includes(q)
      const urlMatch = (node.url || '').toLowerCase().includes(q)
      if (nameMatch || urlMatch) {
        result.push(node)
      }
    } else {
      const filteredChildren = node.children ? filterNodes(node.children, query) : []
      if (filteredChildren.length > 0) {
        result.push({ ...node, children: filteredChildren })
      }
    }
  }
  return result
}

function renderMenu(node: ApiNode) {
  const common = [
    { label: '新建目录', command: () => showCreate(node.path, 'dir') },
    { label: '新建 HTTP 请求', command: () => showCreate(node.path, 'http') },
    { label: '新建 WebSocket', command: () => showCreate(node.path, 'websocket') },
    { label: '克隆', command: () => cloneNode(node) },
    { label: '重命名', command: () => renameNode(node) },
    { label: '删除', command: () => removeNode(node) },
  ]
  if (node.type === 'dir') {
    common.splice(3, 0, { label: '目录设置', command: () => openDirMeta(node) })
  }
  return common
}

</script>

<template>
  <div class="sidebar">
    <div class="tree-toolbar">
      <el-button size="small" text class="toolbar-btn" @click="showCreate('', 'http')">
        <el-icon><Plus /></el-icon>HTTP
      </el-button>
      <el-button size="small" text class="toolbar-btn" @click="showCreate('', 'websocket')">
        <el-icon><Promotion /></el-icon>WS
      </el-button>
      <el-button size="small" text class="toolbar-btn" @click="showCreate('', 'dir')">
        <el-icon><FolderAdd /></el-icon>目录
      </el-button>
      <el-button size="small" text class="toolbar-btn" @click="fetchDialogVisible = true">
        <el-icon><Download /></el-icon>Fetch
      </el-button>
    </div>
    <div class="sidebar-actions">
      <div class="search-wrap">
        <el-input
          v-model="searchQuery"
          size="small"
          placeholder="搜索接口名称或 URL"
          clearable
          @keyup.enter="triggerSearch"
          @clear="triggerSearch"
        >
          <template #append>
            <el-button :icon="Search" @click="triggerSearch" />
          </template>
        </el-input>
      </div>
      <div class="action-btns">
        <el-tooltip content="刷新接口列表" placement="bottom">
          <el-icon :class="['action-icon', { 'is-loading': refreshing }]" @click="refreshTree">
            <Refresh />
          </el-icon>
        </el-tooltip>
        <el-tooltip content="定位" placement="bottom">
          <el-icon class="action-icon" @click="locateActiveTab">
            <Aim />
          </el-icon>
        </el-tooltip>
      </div>
    </div>

    <el-tree
      ref="treeRef"
      :data="filteredTree"
      :props="{ label: 'name', children: 'children' }"
      node-key="id"
      class="tree"
      highlight-current
      draggable
      :allow-drop="allowDrop"
      :allow-drag="allowDrag"
      :default-expanded-keys="expandedKeys"
      @node-click="handleNodeClick"
      @node-drop="onNodeDrop"
      @node-expand="onNodeExpand"
      @node-collapse="onNodeCollapse"
    >
      <template #default="{ node, data }">
        <span class="tree-node">
          <el-icon v-if="data.type === 'dir'"><Folder /></el-icon>
          <template v-else>
            <span v-if="data.requestType === 'websocket'" class="method-badge method-ws">WS</span>
            <span v-else :class="['method-badge', 'method-' + (data.method || 'GET').toLowerCase()]">{{ data.method || 'GET' }}</span>
          </template>
          <span class="node-label" :title="data.name">{{ data.name }}</span>
          <span class="node-more-wrap" @click.stop @mousedown.stop>
            <el-dropdown trigger="click" @command="(cmd: any) => cmd()">
              <el-icon class="node-more"><MoreFilled /></el-icon>
              <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item
                  v-for="(item, idx) in renderMenu(data)"
                  :key="idx"
                  :command="item.command"
                >
                  {{ item.label }}
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
          </span>
        </span>
      </template>
    </el-tree>

    <div class="workspace-info">
      <el-tooltip :content="workspace.root" placement="top">
        <span class="root-path">{{ workspace.root }}</span>
      </el-tooltip>
      <el-icon class="open-explorer-btn" title="在文件管理器中打开" @click="openWorkspaceInExplorer">
        <FolderOpened />
      </el-icon>
    </div>

    <el-dialog v-model="createDialogVisible" title="新建" width="360px">
      <el-input v-model="newName" placeholder="请输入名称" @keyup.enter="confirmCreate" />
      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmCreate">确定</el-button>
      </template>
    </el-dialog>

    <FetchImportDialog v-model="fetchDialogVisible" />
    <DirMetaDialog v-model="dirMetaVisible" :rel="dirMetaRel" />
  </div>
</template>

<style scoped>
.sidebar {
  display: flex;
  flex-direction: column;
}

.sidebar-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  border-bottom: 1px solid var(--app-border);
}

.search-wrap {
  flex: 1;
  min-width: 0;
}

.search-wrap :deep(.el-input__wrapper) {
  padding-left: 8px;
}

.action-btns {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
}

.action-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 4px;
  cursor: pointer;
  color: var(--app-text-secondary);
  transition: all 0.15s;
}

.action-icon:hover {
  color: var(--app-accent);
  background-color: var(--app-bg-tertiary);
}

.action-icon.is-loading {
  animation: rotating 1s linear infinite;
}

@keyframes rotating {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.tree-toolbar {
  display: flex;
  flex-wrap: nowrap;
  gap: 2px;
  padding: 6px 8px;
  border-bottom: 1px solid var(--app-border);
}

.tree-toolbar :deep(.toolbar-btn) {
  flex: 1;
  min-width: 0;
  padding: 0 4px;
  font-size: 12px;
}

.tree-toolbar :deep(.toolbar-btn .el-icon) {
  margin-right: 2px;
  font-size: 12px;
}

.tree {
  flex: 1;
  overflow: auto;
  padding: 4px 0;
}

.tree-node {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  font-size: 13px;
}

.method-badge {
  font-size: 10px;
  font-weight: 700;
  font-family: 'Consolas', 'Monaco', monospace;
  padding: 1px 4px;
  border-radius: 3px;
  flex-shrink: 0;
  min-width: 30px;
  text-align: center;
}

.method-get { color: var(--method-get, #61affe); }
.method-post { color: var(--method-post, #49cc90); }
.method-put { color: var(--method-put, #fca130); }
.method-delete { color: var(--method-delete, #f93e3e); }
.method-patch { color: var(--method-patch, #50e3c2); }
.method-head { color: var(--method-head, #9012fe); }
.method-options { color: var(--method-options, #9012fe); }
.method-ws { color: var(--method-ws, #8b5cf6); }

.node-label {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.node-more-wrap {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  flex-shrink: 0;
  border-radius: 4px;
  transition: background-color 0.15s;
}

.node-more-wrap:hover {
  background-color: var(--app-bg-tertiary);
}

.node-more {
  opacity: 0;
  font-size: 14px;
  cursor: pointer;
  transition: opacity 0.15s;
}

.tree-node:hover .node-more {
  opacity: 1;
}

.tree :deep(.el-tree-node.is-dragging) {
  opacity: 0.5;
}

.tree :deep(.el-tree-node.is-drop-inner > .el-tree-node__content) {
  background-color: var(--app-accent) !important;
  color: #fff;
}

.tree :deep(.el-tree-node__content.locate-flash) {
  animation: locate-flash 1.3s ease;
}

@keyframes locate-flash {
  0%,
  100% {
    background-color: transparent;
    color: var(--app-text);
    box-shadow: none;
  }
  20% {
    background-color: color-mix(in srgb, var(--app-accent) 88%, #000);
    color: #fff;
    box-shadow: inset 0 0 0 1.5px var(--app-accent);
  }
  30% {
    background-color: transparent;
    color: var(--app-text);
    box-shadow: none;
  }
  53% {
    background-color: color-mix(in srgb, var(--app-accent) 88%, #000);
    color: #fff;
    box-shadow: inset 0 0 0 1.5px var(--app-accent);
  }
  63% {
    background-color: transparent;
    color: var(--app-text);
    box-shadow: none;
  }
  86% {
    background-color: color-mix(in srgb, var(--app-accent) 88%, #000);
    color: #fff;
    box-shadow: inset 0 0 0 1.5px var(--app-accent);
  }
}

.workspace-info {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  border-top: 1px solid var(--app-border);
  font-size: 11px;
  color: var(--app-text-secondary);
}

.root-path {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.open-explorer-btn {
  flex-shrink: 0;
  cursor: pointer;
  font-size: 14px;
  color: var(--app-text-secondary);
  transition: color 0.15s;
}

.open-explorer-btn:hover {
  color: var(--app-accent);
}
</style>
