<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { debounce } from '../../utils/helpers'

const props = withDefaults(defineProps<{
  modelValue?: string
  readOnly?: boolean
  language?: 'json' | 'text'
  placeholder?: string
}>(), {
  modelValue: '',
  readOnly: false,
  language: 'text',
  placeholder: '',
})

const emit = defineEmits<{ (e: 'update:modelValue', v: string): void }>()

/** 单行高度（px），需与 .code-readonly 中 pre 的 font-size 13px × line-height 1.6 保持一致 */
const LINE_HEIGHT = 20.8
/** 内容区上下内边距（px），需与样式中的 padding 保持一致 */
const CONTENT_PADDING = 12
/** 行数超过该值时才启用虚拟滚动，小文本继续保留自动换行的阅读体验 */
const VIRTUAL_THRESHOLD_LINES = 1000
/** 视口上下额外渲染的行数，避免快速滚动时出现空白 */
const OVERSCAN_LINES = 30
/** 搜索命中的采集上限，避免超大文本下匹配列表本身占用过多内存 */
const MAX_SEARCH_MATCHES = 1000
/** 搜索重建的防抖时间（毫秒），避免大文本下逐字输入造成卡顿 */
const SEARCH_DEBOUNCE_MS = 200

const textareaRef = ref<HTMLTextAreaElement | null>(null)
const gutterRef = ref<HTMLDivElement | null>(null)
const highlightRef = ref<HTMLPreElement | null>(null)
const scrollRef = ref<HTMLDivElement | null>(null)
const internalValue = ref(props.modelValue)

// 搜索状态
const searchTerm = ref('')
const matchCount = ref(0)
const currentMatch = ref(0)
const searchVisible = ref(false)
const searchInputRef = ref<HTMLInputElement | null>(null)
/** 行号 -> 该行命中的全局序号列表，仅包含被采集到的命中 */
const lineMatches = ref<Map<number, number[]>>(new Map())
/** 全局序号 -> 行号，用于命中项跳转定位 */
const matchLines = ref<number[]>([])
/** 命中数量是否已达到采集上限 */
const matchesTruncated = ref(false)

// 虚拟滚动状态
const scrollTop = ref(0)
const viewportHeight = ref(0)
let resizeObserver: ResizeObserver | null = null

watch(() => props.modelValue, (v) => {
  if (v === internalValue.value) return
  internalValue.value = v
  // 整块内容被替换（例如切换请求标签）时重置搜索与滚动位置
  if (props.readOnly) {
    closeSearch()
    scrollTop.value = 0
    nextTick(() => {
      if (scrollRef.value) scrollRef.value.scrollTop = 0
      if (gutterRef.value) gutterRef.value.scrollTop = 0
    })
  }
})

/** 按行拆分内容，避免每次访问都重新 split 大字符串 */
const lines = computed(() => {
  const text = internalValue.value || ''
  return text ? text.split('\n') : ['']
})

const lineCount = computed(() => lines.value.length)

/** 大文本只读场景启用虚拟滚动 */
const virtualEnabled = computed(() => props.readOnly && lineCount.value > VIRTUAL_THRESHOLD_LINES)

/** 行号栏宽度随位数自适应 */
const gutterWidth = computed(() => Math.max(42, String(lineCount.value).length * 8 + 14))

/** 当前渲染窗口的起始行（含上下缓冲） */
const startLine = computed(() => {
  if (!virtualEnabled.value) return 0
  return Math.max(0, Math.floor(scrollTop.value / LINE_HEIGHT) - OVERSCAN_LINES)
})

/** 当前渲染窗口的结束行（不含） */
const endLine = computed(() => {
  if (!virtualEnabled.value) return lineCount.value
  const height = viewportHeight.value || 480
  const visibleCount = Math.ceil(height / LINE_HEIGHT) + OVERSCAN_LINES * 2
  return Math.min(lineCount.value, startLine.value + visibleCount)
})

const visibleLines = computed(() => lines.value.slice(startLine.value, endLine.value))

/** 渲染窗口的垂直偏移，保证窗口内容与滚动位置严格对齐 */
const offsetY = computed(() => (virtualEnabled.value ? startLine.value * LINE_HEIGHT : 0))

/** 虚拟滚动撑起的总高度，使滚动条长度与全文一致 */
const virtualTotalHeight = computed(() =>
  virtualEnabled.value ? lineCount.value * LINE_HEIGHT + CONTENT_PADDING * 2 : 0
)

const visibleLineNumbers = computed(() => {
  const start = startLine.value
  const nums: string[] = []
  for (let i = 0; i < visibleLines.value.length; i++) {
    nums.push(String(start + i + 1))
  }
  return nums.join('\n')
})

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
}

function escapeRegExp(text: string): string {
  return text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function highlightJson(text: string): string {
  const escaped = escapeHtml(text)
  if (props.language !== 'json') return escaped
  return escaped.replace(
    /("(?:\\.|[^"\\])*"\s*:?)|(\b(?:true|false|null)\b)|(-?\d+\.?\d*(?:[eE][+-]?\d+)?)/g,
    (match, str, bool, num) => {
      if (str) {
        if (str.endsWith(':')) {
          return `<span class="tok-key">${str.slice(0, -1)}</span>:`
        }
        return `<span class="tok-string">${str}</span>`
      }
      if (bool) return `<span class="tok-bool">${bool}</span>`
      if (num) return `<span class="tok-num">${num}</span>`
      return match
    }
  )
}

/**
 * 重新扫描全文收集搜索命中位置。
 * 只记录命中所在行与行内序号，不保存原文片段，因此大文本下内存可控；
 * 命中数量达到上限后停止扫描。
 */
function rebuildMatches() {
  const term = searchTerm.value
  const map = new Map<number, number[]>()
  const globalLines: number[] = []
  if (!term) {
    lineMatches.value = map
    matchLines.value = globalLines
    matchCount.value = 0
    currentMatch.value = 0
    matchesTruncated.value = false
    return
  }
  const re = new RegExp(escapeRegExp(term), 'gi')
  const all = lines.value
  let count = 0
  let truncated = false
  for (let i = 0; i < all.length; i++) {
    re.lastIndex = 0
    let m: RegExpExecArray | null
    let ids: number[] | null = null
    while ((m = re.exec(all[i])) !== null) {
      // 空匹配（例如搜索词为纯正则元字符）时强制前进，避免死循环
      if (m[0] === '') {
        re.lastIndex++
        continue
      }
      if (!ids) ids = []
      ids.push(count)
      globalLines.push(i)
      count++
      if (count >= MAX_SEARCH_MATCHES) {
        truncated = true
        break
      }
    }
    if (ids) map.set(i, ids)
    if (truncated) break
  }
  lineMatches.value = map
  matchLines.value = globalLines
  matchCount.value = count
  matchesTruncated.value = truncated
  if (currentMatch.value >= count) {
    currentMatch.value = count > 0 ? count - 1 : 0
  }
}

/** 渲染单行：先做 JSON 高亮，再按全局序号标记搜索命中位置 */
function renderLine(rawLine: string, lineIndex: number): string {
  const html = highlightJson(rawLine)
  const ids = searchTerm.value ? lineMatches.value.get(lineIndex) : undefined
  if (!ids || !ids.length) return html
  // 标签整体跳过，避免搜索词命中高亮标签自身的属性导致 HTML 被破坏
  const re = new RegExp(`(<[^>]*>)|${escapeRegExp(escapeHtml(searchTerm.value))}`, 'gi')
  let k = 0
  return html.replace(re, (matched, tag?: string) => {
    if (tag) return tag
    const id = ids[k]
    k++
    // 只标记采集到的命中，避免出现计数之外的高亮
    if (id === undefined) return matched
    const cls = id === currentMatch.value ? 'search-current' : 'search-match'
    return `<mark class="${cls}">${matched}</mark>`
  })
}

/** 仅渲染可视窗口内的行，避免超大响应生成海量 DOM 节点 */
const renderedHtml = computed(() => {
  const start = startLine.value
  const list = visibleLines.value
  const parts: string[] = []
  for (let i = 0; i < list.length; i++) {
    parts.push(renderLine(list[i], start + i))
  }
  return parts.join('\n') + '\n'
})

function onInput(e: Event) {
  const target = e.target as HTMLTextAreaElement
  internalValue.value = target.value
  emit('update:modelValue', target.value)
  syncScroll()
}

function syncScroll() {
  if (!textareaRef.value) return
  const ta = textareaRef.value
  if (gutterRef.value) gutterRef.value.scrollTop = ta.scrollTop
  if (highlightRef.value) {
    highlightRef.value.scrollTop = ta.scrollTop
    highlightRef.value.scrollLeft = ta.scrollLeft
  }
}

function onScroll() {
  syncScroll()
}

function onReadOnlyScroll() {
  if (!scrollRef.value) return
  const el = scrollRef.value
  scrollTop.value = el.scrollTop
  if (gutterRef.value) gutterRef.value.scrollTop = el.scrollTop
}

/** 记录视口高度，用于推算虚拟滚动的渲染窗口 */
function measureViewport() {
  if (!scrollRef.value) return
  viewportHeight.value = scrollRef.value.clientHeight
}

/** 滚动到指定行（虚拟滚动模式下按行高换算滚动位置） */
function scrollToLine(line: number) {
  const container = scrollRef.value
  if (!container) return
  const elTop = CONTENT_PADDING + line * LINE_HEIGHT
  const elBottom = elTop + LINE_HEIGHT
  if (elTop < container.scrollTop) {
    container.scrollTop = elTop
  } else if (elBottom > container.scrollTop + container.clientHeight) {
    container.scrollTop = elBottom - container.clientHeight
  }
}

/** 滚动到当前命中项 */
function scrollToCurrentMatch() {
  const line = matchLines.value[currentMatch.value]
  if (line === undefined) return
  if (virtualEnabled.value) {
    scrollToLine(line)
    return
  }
  nextTick(() => {
    if (!scrollRef.value || !highlightRef.value) return
    const el = highlightRef.value.querySelector('.search-current') as HTMLElement | null
    if (!el) return
    const container = scrollRef.value
    const elTop = el.offsetTop
    const elBottom = elTop + el.offsetHeight
    if (elTop < container.scrollTop) {
      container.scrollTop = elTop
    } else if (elBottom > container.scrollTop + container.clientHeight) {
      container.scrollTop = elBottom - container.clientHeight
    }
  })
}

/** 大文本下重建命中列表较耗时，输入时做防抖处理 */
const scheduleRebuildMatches = debounce(() => {
  rebuildMatches()
  nextTick(scrollToCurrentMatch)
}, SEARCH_DEBOUNCE_MS)

function onSearchInput() {
  currentMatch.value = 0
  if (!searchTerm.value) {
    rebuildMatches()
    return
  }
  scheduleRebuildMatches()
}

function findNext() {
  if (matchCount.value === 0) return
  currentMatch.value = (currentMatch.value + 1) % matchCount.value
  nextTick(scrollToCurrentMatch)
}

function findPrev() {
  if (matchCount.value === 0) return
  currentMatch.value = (currentMatch.value - 1 + matchCount.value) % matchCount.value
  nextTick(scrollToCurrentMatch)
}

function onSearchKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter') {
    e.preventDefault()
    if (e.shiftKey) findPrev()
    else findNext()
  } else if (e.key === 'Escape') {
    e.preventDefault()
    closeSearch()
  }
}

function openSearch() {
  searchVisible.value = true
  nextTick(() => {
    searchInputRef.value?.focus()
    searchInputRef.value?.select()
  })
}

function closeSearch() {
  searchVisible.value = false
  searchTerm.value = ''
  matchCount.value = 0
  currentMatch.value = 0
  lineMatches.value = new Map()
  matchLines.value = []
  matchesTruncated.value = false
}

onMounted(() => {
  nextTick(() => {
    syncScroll()
    measureViewport()
  })
  // 视口尺寸变化时刷新渲染窗口，保证虚拟滚动内容始终铺满可视区域
  if (scrollRef.value && typeof ResizeObserver !== 'undefined') {
    resizeObserver = new ResizeObserver(measureViewport)
    resizeObserver.observe(scrollRef.value)
  }
})

onBeforeUnmount(() => {
  if (resizeObserver) {
    resizeObserver.disconnect()
    resizeObserver = null
  }
  closeSearch()
})

defineExpose({ openSearch, closeSearch, findNext, findPrev })
</script>

<template>
  <div class="code-editor">
    <div ref="gutterRef" class="code-gutter" :style="{ width: gutterWidth + 'px' }">
      <template v-if="virtualEnabled">
        <div class="gutter-spacer" :style="{ height: virtualTotalHeight + 'px' }">
          <pre class="gutter-lines" :style="{ transform: `translateY(${offsetY}px)` }">{{ visibleLineNumbers }}</pre>
        </div>
      </template>
      <pre v-else>{{ visibleLineNumbers }}</pre>
    </div>

    <template v-if="readOnly">
      <div ref="scrollRef" class="code-readonly" @scroll="onReadOnlyScroll">
        <div v-if="virtualEnabled" class="code-virtual" :style="{ height: virtualTotalHeight + 'px' }">
          <pre
            ref="highlightRef"
            class="code-highlight-readonly code-virtual-lines"
            :style="{ transform: `translateY(${offsetY}px)` }"
            v-html="renderedHtml"
          />
        </div>
        <pre v-else ref="highlightRef" class="code-highlight-readonly" v-html="renderedHtml" />
        <div v-if="!internalValue" class="code-placeholder">{{ placeholder }}</div>
      </div>
    </template>

    <template v-else>
      <div class="code-edit-area">
        <pre ref="highlightRef" class="code-highlight-edit" v-html="renderedHtml" />
        <textarea
          ref="textareaRef"
          :value="internalValue"
          class="code-textarea"
          :placeholder="placeholder"
          spellcheck="false"
          @input="onInput"
          @scroll="onScroll"
        />
      </div>
    </template>

    <!-- 搜索浮层（仅只读模式可用） -->
    <div v-if="searchVisible && readOnly" class="search-bar">
      <input
        ref="searchInputRef"
        v-model="searchTerm"
        class="search-input"
        placeholder="查找"
        spellcheck="false"
        @input="onSearchInput"
        @keydown="onSearchKeydown"
      />
      <span class="search-count">
        <template v-if="searchTerm">
          {{ matchCount > 0 ? `${currentMatch + 1}/${matchCount}${matchesTruncated ? '+' : ''}` : '0/0' }}
        </template>
      </span>
      <button
        class="search-nav-btn"
        title="上一个 (Shift+Enter)"
        :disabled="matchCount === 0"
        @click="findPrev"
      >
        <el-icon><ArrowUp /></el-icon>
      </button>
      <button
        class="search-nav-btn"
        title="下一个 (Enter)"
        :disabled="matchCount === 0"
        @click="findNext"
      >
        <el-icon><ArrowDown /></el-icon>
      </button>
      <button class="search-close-btn" title="关闭 (Esc)" @click="closeSearch">
        <el-icon><Close /></el-icon>
      </button>
    </div>
  </div>
</template>

<style scoped>
.code-editor {
  display: flex;
  height: 100%;
  overflow: hidden;
  position: relative;
  border: 1px solid var(--app-border);
  border-radius: 4px;
  background-color: var(--app-bg-primary);
}

.code-gutter {
  flex-shrink: 0;
  width: 42px;
  overflow: hidden;
  background-color: var(--app-bg-secondary);
  border-right: 1px solid var(--app-border);
  user-select: none;
}

.code-gutter pre {
  margin: 0;
  padding: 12px 8px 12px 0;
  text-align: right;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  color: var(--app-text-secondary);
  white-space: pre;
}

/* 虚拟滚动模式下的行号栏：外层撑起总高度，内层按窗口偏移 */
.gutter-spacer {
  position: relative;
}

.gutter-lines {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
}

/* Read-only mode: scrollable + selectable */
.code-readonly {
  flex: 1;
  overflow: auto;
  position: relative;
}

.code-highlight-readonly {
  margin: 0;
  padding: 12px;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  color: var(--app-text);
  user-select: text;
}

/* 虚拟滚动容器：撑起总高度，只渲染可视窗口内的行 */
.code-virtual {
  position: relative;
}

/* 虚拟滚动要求行高固定，长行改为横向滚动，避免换行导致行高不一致 */
.code-highlight-readonly.code-virtual-lines {
  position: absolute;
  top: 0;
  left: 0;
  white-space: pre;
  word-break: normal;
}

/* Edit mode: textarea overlay */
.code-edit-area {
  flex: 1;
  position: relative;
  overflow: hidden;
}

.code-highlight-edit {
  margin: 0;
  padding: 12px;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  color: var(--app-text);
  pointer-events: none;
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  overflow: hidden;
}

.code-textarea {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  width: 100%;
  height: 100%;
  padding: 12px;
  border: none;
  outline: none;
  resize: none;
  background: transparent;
  color: transparent;
  caret-color: var(--app-accent);
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  z-index: 1;
  overflow: auto;
}

.code-textarea::placeholder {
  color: var(--app-text-secondary);
}

.code-placeholder {
  position: absolute;
  top: 12px;
  left: 12px;
  color: var(--app-text-secondary);
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  pointer-events: none;
}

/* Token colors from theme variables */
:deep(.tok-key) { color: var(--tok-key, #c792ea); }
:deep(.tok-string) { color: var(--tok-string, #c3e88d); }
:deep(.tok-num) { color: var(--tok-num, #f78c6c); }
:deep(.tok-bool) { color: var(--tok-bool, #ff9cac); }

/* 搜索命中高亮 */
:deep(.search-match) {
  background-color: rgba(255, 213, 79, 0.35);
  color: inherit;
  border-radius: 2px;
  padding: 0;
}

:deep(.search-current) {
  background-color: rgba(255, 165, 0, 0.65);
  color: #000;
  border-radius: 2px;
  padding: 0;
}

/* 搜索浮层 */
.search-bar {
  position: absolute;
  top: 8px;
  right: 12px;
  z-index: 20;
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 6px;
  background-color: var(--app-bg-secondary);
  border: 1px solid var(--app-border);
  border-radius: 6px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.25);
}

.search-input {
  width: 160px;
  height: 26px;
  padding: 0 8px;
  border: 1px solid var(--app-border);
  border-radius: 4px;
  background-color: var(--app-bg-primary);
  color: var(--app-text);
  font-size: 12px;
  outline: none;
}

.search-input:focus {
  border-color: var(--app-accent);
}

.search-count {
  min-width: 48px;
  text-align: center;
  font-size: 11px;
  color: var(--app-text-secondary);
  font-family: 'Consolas', 'Monaco', monospace;
}

.search-nav-btn,
.search-close-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  padding: 0;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--app-text-secondary);
  cursor: pointer;
  transition: all 0.15s ease;
}

.search-nav-btn:hover:not(:disabled),
.search-close-btn:hover {
  background-color: var(--app-bg-tertiary);
  color: var(--app-accent);
}

.search-nav-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
</style>
