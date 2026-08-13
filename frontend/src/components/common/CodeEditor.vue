<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted } from 'vue'

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

const highlightedHtml = computed(() => {
  const text = internalValue.value || ''
  const html = highlightJson(text)
  return html + '\n'
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

onMounted(() => {
  nextTick(syncScroll)
})
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
  </div>
</template>

<style scoped>
.code-editor {
  display: flex;
  height: 100%;
  overflow: hidden;
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
</style>
