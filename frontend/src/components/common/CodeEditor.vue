<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onBeforeUnmount } from 'vue'

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

watch(() => props.modelValue, (v) => {
  if (v !== internalValue.value) {
    internalValue.value = v
  }
})

const lineCount = computed(() => {
  const lines = internalValue.value.split('\n')
  return Math.max(lines.length, 1)
})

const lineNumbers = computed(() => {
  const count = lineCount.value
  return Array.from({ length: count }, (_, i) => i + 1).join('\n')
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

/** 将搜索词转义后高亮命中位置（当前命中项额外标记） */
function applySearchHighlight(html: string): string {
  const term = searchTerm.value
  if (!term) {
    matchCount.value = 0
    currentMatch.value = 0
    return html
  }
  // 在已转义的 HTML 上匹配，因 escapeHtml 仅替换 &< >，搜索词也需做同样转义
  const termEscaped = escapeHtml(term)
  const termForRegex = escapeRegExp(termEscaped)
  const re = new RegExp(termForRegex, 'gi')
  let idx = 0
  const matches: number[] = []
  const result = html.replace(re, (m) => {
    const i = idx++
    matches.push(i)
    const cls = i === currentMatch.value ? 'search-current' : 'search-match'
    return `<mark class="${cls}">${m}</mark>`
  })
  matchCount.value = idx
  if (currentMatch.value >= matchCount.value) {
    currentMatch.value = matchCount.value > 0 ? matchCount.value - 1 : 0
  }
  return result
}

const highlightedHtml = computed(() => {
  const text = internalValue.value || ''
  const html = highlightJson(text)
  const withSearch = applySearchHighlight(html)
  return withSearch + '\n'
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
  if (!scrollRef.value || !gutterRef.value) return
  gutterRef.value.scrollTop = scrollRef.value.scrollTop
}

/** 滚动到当前命中项 */
function scrollToCurrentMatch() {
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

function onSearchInput() {
  currentMatch.value = 0
  nextTick(scrollToCurrentMatch)
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
}

onMounted(() => {
  nextTick(syncScroll)
})

onBeforeUnmount(() => {
  closeSearch()
})

defineExpose({ openSearch, closeSearch, findNext, findPrev })
</script>

<template>
  <div class="code-editor">
    <div ref="gutterRef" class="code-gutter">
      <pre>{{ lineNumbers }}</pre>
    </div>

    <template v-if="readOnly">
      <div ref="scrollRef" class="code-readonly" @scroll="onReadOnlyScroll">
        <pre ref="highlightRef" class="code-highlight-readonly" v-html="highlightedHtml" />
        <div v-if="!internalValue" class="code-placeholder">{{ placeholder }}</div>
      </div>
    </template>

    <template v-else>
      <div class="code-edit-area">
        <pre ref="highlightRef" class="code-highlight-edit" v-html="highlightedHtml" />
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
          {{ matchCount > 0 ? `${currentMatch + 1}/${matchCount}` : '0/0' }}
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
