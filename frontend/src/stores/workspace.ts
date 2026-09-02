import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '../services/api'
import type { WorkspaceInitData } from '../services/api'
import { normalizeRequest } from '../utils/helpers'
import type { ApiNode, Request, Settings } from '../types'
import type { RequestTab } from '../types'

function normalizeSettings(s: Settings): Settings {
  return {
    ...s,
    requestTimeoutMs: s.requestTimeoutMs > 0 ? s.requestTimeoutMs : 30000,
    globalHeaders: s.globalHeaders || [],
    globalVariables: s.globalVariables || [],
  }
}

export const useWorkspaceStore = defineStore('workspace', () => {
  const root = ref('')
  const tree = ref<ApiNode[]>([])
  const settings = ref<Settings>({
    version: '1.0.0',
    theme: 'dark-1',
    requestTimeoutMs: 30000,
    globalHeaders: [],
    globalVariables: [],
  })
  const tabs = ref<RequestTab[]>([])
  const activeTabId = ref('')
  const drafts = ref<Map<string, Request>>(new Map())
  const dirtyMap = ref<Map<string, boolean>>(new Map())
  const sending = ref(false)
  const sendingRequestId = ref('')
  const requestTimer = ref('0.0s')
  let timerInterval: ReturnType<typeof setInterval> | null = null
  const wsConnected = ref<Map<string, boolean>>(new Map())

  const hasDirty = computed(() => {
    for (const v of dirtyMap.value.values()) {
      if (v) return true
    }
    return false
  })

  // 让 activeTab 依赖 dirtyMap，确保 dirty 状态变化时能触发响应式更新
  const activeTab = computed(() => {
    const tab = tabs.value.find((t) => t.id === activeTabId.value)
    if (!tab) return undefined
    // 读取 dirtyMap 以建立依赖关系
    const isDirty = dirtyMap.value.get(tab.rel) || false
    return { ...tab, dirty: isDirty }
  })
  const activeDraft = computed(() => {
    if (!activeTab.value) return null
    return drafts.value.get(activeTab.value.rel) || null
  })

  async function openWorkspace(path: string) {
    await api.openWorkspace(path)
    root.value = await api.getWorkspaceRoot()
    const s = await api.loadSettings()
    settings.value = normalizeSettings(s as Settings)
    await reloadTree()
  }

  /** 启动时一次性加载工作区、设置和树，减少 IPC 往返 */
  async function initApp() {
    const data = await api.initApp() as WorkspaceInitData
    root.value = data.root
    settings.value = normalizeSettings(data.settings as unknown as Settings)
    tree.value = (data.tree as ApiNode[]) || []
  }

  async function changeWorkspace(path: string) {
    const result = await api.changeWorkspace(path)
    root.value = await api.getWorkspaceRoot()
    const s = await api.loadSettings()
    settings.value = normalizeSettings(s as Settings)
    tabs.value = []
    activeTabId.value = ''
    drafts.value.clear()
    dirtyMap.value.clear()
    await reloadTree()
    return result
  }

  async function reloadTree() {
    tree.value = (await api.loadTree() as ApiNode[]) || []
  }

  async function saveSettings() {
    await api.saveSettings(settings.value as any)
  }

  async function loadSettings() {
    const s = await api.loadSettings()
    settings.value = normalizeSettings(s as Settings)
  }

  async function loadRequest(rel: string, activateTab = true) {
    const existing = tabs.value.find((t) => t.rel === rel)
    if (existing) {
      if (activateTab) {
        activeTabId.value = existing.id
      }
      return
    }
    const req = normalizeRequest(await api.loadRequest(rel)) as Request
    drafts.value.set(rel, req)
    dirtyMap.value.set(rel, false)
    const tab: RequestTab = {
      id: req.id,
      rel,
      name: req.name || rel,
      dirty: false,
    }
    tabs.value.push(tab)
    if (activateTab) {
      activeTabId.value = tab.id
    }
  }

  async function cloneRequest(rel: string, newName: string) {
    const newRel = await api.cloneRequest(rel, newName)
    await reloadTree()
    await loadRequest(newRel, false)
  }

  function updateDraft(rel: string, patch: Partial<Request>) {
    const current = drafts.value.get(rel)
    if (!current) return
    const next = { ...current, ...patch }
    // 数据未发生变化时不标记为脏，避免组件初始化时的默认值同步触发未保存提示
    if (JSON.stringify(current) === JSON.stringify(next)) {
      return
    }
    drafts.value.set(rel, next)
    dirtyMap.value.set(rel, true)
    const idx = tabs.value.findIndex((t) => t.rel === rel)
    if (idx !== -1) {
      // 使用 splice 确保 Vue 响应式系统正确追踪数组变化
      tabs.value.splice(idx, 1, { ...tabs.value[idx], dirty: true })
    }
  }

  async function saveRequest(rel: string) {
    const req = drafts.value.get(rel)
    if (!req) return
    await api.saveRequest(rel, req as any)
    dirtyMap.value.set(rel, false)
    const idx = tabs.value.findIndex((t) => t.rel === rel)
    if (idx !== -1) {
      tabs.value[idx] = { ...tabs.value[idx], dirty: false }
    }
    await reloadTree()
  }

  async function closeTab(id: string) {
    const idx = tabs.value.findIndex((t) => t.id === id)
    if (idx === -1) return true
    const tab = tabs.value[idx]
    if (tab.dirty) {
      return false
    }
    if (wsConnected.value.get(tab.id)) {
      await disconnectWebSocket(tab.id)
    }
    drafts.value.delete(tab.rel)
    dirtyMap.value.delete(tab.rel)
    wsConnected.value.delete(tab.id)
    tabs.value.splice(idx, 1)
    if (activeTabId.value === id) {
      activeTabId.value = tabs.value[idx > 0 ? idx - 1 : 0]?.id || ''
    }
    return true
  }

  async function forceCloseTab(id: string) {
    const idx = tabs.value.findIndex((t) => t.id === id)
    if (idx === -1) return
    const tab = tabs.value[idx]
    if (wsConnected.value.get(tab.id)) {
      await disconnectWebSocket(tab.id)
    }
    drafts.value.delete(tab.rel)
    dirtyMap.value.delete(tab.rel)
    wsConnected.value.delete(tab.id)
    tabs.value.splice(idx, 1)
    if (activeTabId.value === id) {
      activeTabId.value = tabs.value[idx > 0 ? idx - 1 : 0]?.id || ''
    }
  }

  function setWsConnected(id: string, connected: boolean) {
    wsConnected.value.set(id, connected)
  }

  function isWsConnected(id: string): boolean {
    return wsConnected.value.get(id) || false
  }

  async function disconnectWebSocket(id: string) {
    try {
      await api.disconnectWebSocket(id)
    } catch {
      // ignore disconnect errors
    }
    wsConnected.value.delete(id)
  }

  async function createDir(parentRel: string, name: string) {
    await api.createDir(parentRel, name)
    await reloadTree()
  }

  async function createRequest(parentRel: string, name: string, type_: 'http' | 'websocket') {
    const rel = await api.createRequest(parentRel, name, type_)
    await reloadTree()
    await loadRequest(rel)
  }

  async function deleteNode(rel: string, isDir: boolean) {
    await api.deleteNode(rel, isDir)
    const affectedTabs = tabs.value.filter((t) => t.rel === rel || t.rel.startsWith(rel + '\\'))
    for (const t of affectedTabs) {
      await forceCloseTab(t.id)
    }
    await reloadTree()
  }

  async function rename(rel: string, newName: string, isDir: boolean) {
    const newRel = await api.rename(rel, newName, isDir)
    for (let i = 0; i < tabs.value.length; i++) {
      const t = tabs.value[i]
      if (t.rel === rel) {
        tabs.value.splice(i, 1, { ...t, rel: newRel, name: newName })
      } else if (isDir && t.rel.startsWith(rel + '\\')) {
        tabs.value.splice(i, 1, { ...t, rel: t.rel.replace(rel, newRel) })
      }
    }
    // 请求重命名时同步更新对应草稿的名称，否则保存时会把旧的“未命名”写回文件，导致左侧树名被还原
    if (!isDir) {
      const draft = drafts.value.get(rel)
      if (draft) {
        drafts.value.set(rel, { ...draft, name: newName })
      }
    }
    updateTreeNodeAfterRename(rel, newRel, newName, isDir)
    await reloadTree()
  }

  function updateTreeNodeAfterRename(
    oldRel: string,
    newRel: string,
    newName: string,
    isDir: boolean
  ) {
    const walk = (nodes: ApiNode[]) => {
      for (const node of nodes) {
        if (node.path === oldRel) {
          node.name = newName
          if (isDir) {
            node.path = newRel
            if (node.children) {
              updateChildrenPath(node.children, oldRel, newRel)
            }
          }
          return
        }
        if (node.children) {
          walk(node.children)
        }
      }
    }
    walk(tree.value)
  }

  function updateChildrenPath(nodes: ApiNode[], oldPrefix: string, newPrefix: string) {
    for (const node of nodes) {
      if (node.path.startsWith(oldPrefix + '\\')) {
        node.path = newPrefix + node.path.substring(oldPrefix.length)
      }
      if (node.children) {
        updateChildrenPath(node.children, oldPrefix, newPrefix)
      }
    }
  }

  async function moveNode(srcRel: string, destDirRel: string) {
    await api.moveNode(srcRel, destDirRel)
    // 更新受移动影响的已打开标签页及其草稿数据
    const srcBase = srcRel.split('\\').pop() || srcRel
    const newPrefix = destDirRel ? destDirRel + '\\' + srcBase : srcBase
    const relMapping = new Map<string, string>()
    for (const t of tabs.value) {
      let newRel = ''
      if (t.rel === srcRel) {
        newRel = newPrefix
      } else if (t.rel.startsWith(srcRel + '\\')) {
        newRel = newPrefix + t.rel.substring(srcRel.length)
      }
      if (newRel) {
        relMapping.set(t.rel, newRel)
        t.rel = newRel
      }
    }
    for (const [oldRel, newRel] of relMapping) {
      const draft = drafts.value.get(oldRel)
      if (draft) {
        drafts.value.delete(oldRel)
        drafts.value.set(newRel, draft)
      }
      const dirty = dirtyMap.value.get(oldRel)
      if (dirty !== undefined) {
        dirtyMap.value.delete(oldRel)
        dirtyMap.value.set(newRel, dirty)
      }
    }
    await reloadTree()
  }

  function setActiveTab(id: string) {
    activeTabId.value = id
  }

  function startSendingTimer(requestId: string) {
    sendingRequestId.value = requestId
    sending.value = true
    requestTimer.value = '0.0s'
    const start = Date.now()
    timerInterval = window.setInterval(() => {
      requestTimer.value = ((Date.now() - start) / 1000).toFixed(1) + 's'
    }, 100)
  }

  function stopSendingTimer() {
    if (timerInterval !== null) {
      clearInterval(timerInterval)
      timerInterval = null
    }
    sending.value = false
    sendingRequestId.value = ''
  }

  return {
    root,
    tree,
    settings,
    tabs,
    activeTabId,
    drafts,
    dirtyMap,
    sending,
    sendingRequestId,
    requestTimer,
    hasDirty,
    activeTab,
    activeDraft,
    openWorkspace,
    initApp,
    changeWorkspace,
    reloadTree,
    saveSettings,
    loadSettings,
    loadRequest,
    cloneRequest,
    updateDraft,
    saveRequest,
    closeTab,
    forceCloseTab,
    createDir,
    createRequest,
    deleteNode,
    rename,
    moveNode,
    setActiveTab,
    startSendingTimer,
    stopSendingTimer,
    setWsConnected,
    isWsConnected,
    disconnectWebSocket,
  }
})
