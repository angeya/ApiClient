import { ref, computed, onMounted, onUnmounted } from 'vue'

interface UseResizableColumnsOptions {
  storageKey: string
  defaultWidths: number[]
  minWidths?: number[]
  flexible?: boolean[]
  containerRef: () => HTMLElement | null
}

/**
 * Provides draggable column resizing for CSS-Grid based tables.
 *
 * Column widths are persisted in sessionStorage only after the user manually
 * drags a resizer. Auto-fitted widths are never persisted, which prevents
 * collapsed widths from being saved when a table is initialised inside a
 * hidden/animating container (drawer, inactive tab, etc.).
 */
export function useResizableColumns(options: UseResizableColumnsOptions) {
  const widths = ref<number[]>([...options.defaultWidths])
  const minWidths = options.minWidths || options.defaultWidths.map(() => 40)
  const flexible = options.flexible || options.defaultWidths.map(() => true)
  const startX = ref(0)
  const startWidths = ref<number[]>([])
  const resizingIndex = ref(-1)

  let hasUserResized = false
  let resizeObserver: ResizeObserver | null = null
  let lastContainerWidth = 0
  let resizeTimer: number | null = null

  const gridTemplateColumns = computed(() => widths.value.map((w) => `${w}px`).join(' '))

  /**
   * Try to restore user-resized widths from sessionStorage.
   * If the stored widths look collapsed (all flexible columns are at their
   * minimum width) or if storage is empty/invalid, fall back to auto-fitting.
   */
  function loadWidths() {
    try {
      const raw = sessionStorage.getItem(options.storageKey)
      if (raw) {
        const parsed = JSON.parse(raw)
        if (Array.isArray(parsed) && parsed.length === options.defaultWidths.length && !isCollapsed(parsed)) {
          widths.value = parsed
          hasUserResized = true
          return
        }
      }
    } catch {
      // ignore invalid storage data
    }
    applyDefaultOrFit()
  }

  /**
   * Returns true when every flexible column is at (or very close to) its
   * minimum width. This is the typical signature of a previous initialisation
   * that measured a zero-width container.
   */
  function isCollapsed(saved: number[]) {
    const flexibleIndices = flexible.map((f, idx) => (f ? idx : -1)).filter((idx) => idx >= 0)
    if (flexibleIndices.length === 0) return false
    return flexibleIndices.every((idx) => saved[idx] <= minWidths[idx] + 1)
  }

  /**
   * Use the configured default widths when they fit the container, otherwise
   * distribute the available space across flexible columns. When the container
   * is not yet rendered (width <= 0) keep the defaults and start observing.
   */
  function applyDefaultOrFit() {
    const el = options.containerRef()
    if (!el) {
      widths.value = [...options.defaultWidths]
      return
    }
    const width = el.getBoundingClientRect().width
    lastContainerWidth = width
    if (width <= 0) {
      widths.value = [...options.defaultWidths]
    } else {
      fitToWidth(width)
    }
    // 始终注册 ResizeObserver，使容器宽度变化时能重新适配列宽
    observeContainer()
  }

  /**
   * Distribute container width across columns. Fixed columns keep their default
   * width; flexible columns share the remaining space but never drop below their
   * minimum width. When the defaults are narrower than the container, flexible
   * columns are expanded so the table fills the available width.
   */
  function fitToWidth(containerWidth: number) {
    const defaultTotal = options.defaultWidths.reduce((sum, w) => sum + w, 0)
    const flexibleCount = flexible.filter((f) => f).length
    if (flexibleCount === 0) {
      widths.value = [...options.defaultWidths]
      return
    }

    if (defaultTotal <= containerWidth) {
      // Defaults fit: expand flexible columns to fill the container.
      const extra = containerWidth - defaultTotal
      const extraPerColumn = extra / flexibleCount
      widths.value = options.defaultWidths.map((w, idx) => {
        if (!flexible[idx]) return w
        return w + extraPerColumn
      })
      return
    }

    // Defaults overflow: shrink flexible columns to fit.
    const fixedTotal = options.defaultWidths.reduce((sum, w, idx) => {
      return flexible[idx] ? sum : sum + w
    }, 0)
    const available = Math.max(0, containerWidth - fixedTotal)
    const unit = available / flexibleCount
    widths.value = options.defaultWidths.map((w, idx) => {
      if (!flexible[idx]) return w
      return Math.max(minWidths[idx], unit)
    })
  }

  /**
   * Watch the container size so tables inside drawers or inactive tabs can
   * re-fit once they become visible. A short debounce avoids fitting while an
   * animation is still running. User-resized widths are never overwritten.
   */
  function observeContainer() {
    if (resizeObserver || typeof ResizeObserver === 'undefined') return
    const el = options.containerRef()
    if (!el) return

    resizeObserver = new ResizeObserver((entries) => {
      if (hasUserResized) return
      const width = entries[0].contentRect.width
      if (width > 0 && width !== lastContainerWidth) {
        if (resizeTimer) window.clearTimeout(resizeTimer)
        resizeTimer = window.setTimeout(() => {
          fitToWidth(width)
        }, 100)
      }
      lastContainerWidth = width
    })
    resizeObserver.observe(el)
  }

  function saveWidths() {
    try {
      sessionStorage.setItem(options.storageKey, JSON.stringify(widths.value))
    } catch {
      // ignore storage errors
    }
  }

  function startResize(idx: number, e: MouseEvent) {
    e.preventDefault()
    e.stopPropagation()
    hasUserResized = true
    resizingIndex.value = idx
    startX.value = e.clientX
    startWidths.value = [...widths.value]
    document.addEventListener('mousemove', onMouseMove)
    document.addEventListener('mouseup', onMouseUp)
    document.body.style.cursor = 'col-resize'
    document.body.style.userSelect = 'none'
  }

  function onMouseMove(e: MouseEvent) {
    if (resizingIndex.value < 0) return
    const delta = e.clientX - startX.value
    const idx = resizingIndex.value
    const next = idx + 1
    if (next >= widths.value.length) return

    const newCurrent = startWidths.value[idx] + delta
    const newNext = startWidths.value[next] - delta
    if (newCurrent < minWidths[idx] || newNext < minWidths[next]) return

    widths.value[idx] = newCurrent
    widths.value[next] = newNext
  }

  function onMouseUp() {
    resizingIndex.value = -1
    document.removeEventListener('mousemove', onMouseMove)
    document.removeEventListener('mouseup', onMouseUp)
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
    saveWidths()
  }

  onMounted(() => {
    loadWidths()
  })

  onUnmounted(() => {
    if (resizeTimer) window.clearTimeout(resizeTimer)
    if (resizeObserver) {
      resizeObserver.disconnect()
      resizeObserver = null
    }
  })

  return {
    widths,
    gridTemplateColumns,
    startResize,
    initFromContainer: applyDefaultOrFit,
  }
}
