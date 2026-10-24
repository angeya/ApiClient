import type { Header, QueryParam, Variable, Request } from '../types'

/** 请求默认超时时间（毫秒） */
export const DEFAULT_REQUEST_TIMEOUT_MS = 30000

/**
 * 补全请求 URL 的协议前缀。
 * HTTP 请求缺省补 http://，WebSocket 依据方法补 ws:// 或 wss://；
 * 已带协议的 URL 原样返回，便于调用方判断是否需要回填。
 */
export function normalizeUrl(url: string, type: string, method: string): string {
  const trimmed = url.trim()
  if (!trimmed) return trimmed
  if (type === 'websocket') {
    if (/^wss?:\/\//i.test(trimmed)) return trimmed
    return (method === 'WSS' ? 'wss://' : 'ws://') + trimmed
  }
  if (/^https?:\/\//i.test(trimmed)) return trimmed
  return 'http://' + trimmed
}

/** 响应内容默认渲染上限（字节），默认 10MB */
export const DEFAULT_RESPONSE_RENDER_LIMIT_BYTES = 10 * 1024 * 1024

export function emptyHeader(): Header {
  return { key: '', value: '', enabled: true, description: '' }
}

export function emptyQuery(): QueryParam {
  return { key: '', value: '', enabled: true, description: '' }
}

export function emptyVariable(): Variable {
  return { key: '', value: '', enabled: false, description: '' }
}

export function debounce<T extends (...args: unknown[]) => void>(fn: T, ms: number) {
  let timer: number
  return (...args: Parameters<T>) => {
    window.clearTimeout(timer)
    timer = window.setTimeout(() => fn(...args), ms)
  }
}

export function formatBytes(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(2) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(2) + ' MB'
}

export function tryFormatJson(input: string): string {
  try {
    return JSON.stringify(JSON.parse(input), null, 2)
  } catch {
    return input
  }
}

/**
 * 以字符流扫描的方式为 JSON 文本添加缩进，不构建对象图。
 * JSON.parse + JSON.stringify 在处理数 MB 的响应时会生成庞大的中间对象，
 * 是本项目响应区卡死/OOM 的主要原因，因此大响应改用该实现。
 * 遇到明显不合法的结构（字符串未闭合、括号不匹配）时返回 null，由调用方回退为原文。
 */
export function prettyPrintJson(input: string, indentSize = 2): string | null {
  const indentUnit = ' '.repeat(Math.max(0, indentSize))
  const out: string[] = []
  const n = input.length
  let depth = 0
  let i = 0

  while (i < n) {
    const ch = input[i]

    if (ch === ' ' || ch === '\t' || ch === '\n' || ch === '\r') {
      i++
      continue
    }

    // 字符串：整段拷贝，避免破坏其中的转义与空格
    if (ch === '"') {
      const start = i
      i++
      let closed = false
      while (i < n) {
        const c = input[i]
        if (c === '\\') {
          i += 2
          continue
        }
        if (c === '"') {
          closed = true
          i++
          break
        }
        i++
      }
      if (!closed) return null
      out.push(input.slice(start, i))
      continue
    }

    if (ch === '{' || ch === '[') {
      const close = ch === '{' ? '}' : ']'
      let j = i + 1
      while (j < n && (input[j] === ' ' || input[j] === '\t' || input[j] === '\n' || input[j] === '\r')) {
        j++
      }
      // 空对象/空数组保持紧凑形式，避免出现只有换行的空块
      if (input[j] === close) {
        out.push(ch, close)
        i = j + 1
        continue
      }
      depth++
      out.push(ch, '\n', indentUnit.repeat(depth))
      i++
      continue
    }

    if (ch === '}' || ch === ']') {
      depth--
      if (depth < 0) return null
      out.push('\n', indentUnit.repeat(depth), ch)
      i++
      continue
    }

    if (ch === ',') {
      out.push(',', '\n', indentUnit.repeat(depth))
      i++
      continue
    }

    if (ch === ':') {
      out.push(': ')
      i++
      continue
    }

    // 数字、true、false、null 等字面量
    const start = i
    while (i < n) {
      const c = input[i]
      if (c === ',' || c === '}' || c === ']' || c === ' ' || c === '\t' || c === '\n' || c === '\r') break
      i++
    }
    out.push(input.slice(start, i))
  }

  return depth === 0 ? out.join('') : null
}

/** 安全格式化 JSON：失败时回退为原文 */
export function safeFormatJson(input: string, indentSize = 2): string {
  return prettyPrintJson(input, indentSize) ?? input
}

/**
 * 从 URL 推断接口名称：取路径最后一段，无路径段时取主机名。
 * 与 Fetch 导入的命名逻辑保持一致；无法推断时返回空字符串，由调用方决定回退值。
 */
export function guessNameFromUrl(url: string): string {
  const trimmed = (url || '').trim()
  if (!trimmed) return ''
  try {
    const u = new URL(trimmed)
    const segments = u.pathname.split('/').filter(Boolean)
    if (segments.length > 0) {
      return segments[segments.length - 1]
    }
    return u.hostname || ''
  } catch {
    const segments = trimmed.split('/').filter(Boolean)
    return segments.length > 0 ? segments[segments.length - 1] : ''
  }
}

/** 判断一行参数是否全空：参数名、参数值、描述均为空 */
function isRowEmpty(row: { key?: string; value?: string; description?: string }): boolean {
  return (
    !(row.key || '').trim() &&
    !(row.value || '').trim() &&
    !(row.description || '').trim()
  )
}

/**
 * 保存前的请求清理：
 * 1. 自动命名：名称包含“未命名”（含“未命名 副本”）时按 URL 路径最后一段重命名；
 * 2. 删除参数名、参数值、描述三项全空的 Headers / Query / Form 行。
 */
export function cleanupRequestForSave(req: Request, autoRename: boolean): Request {
  const next = { ...req }
  if (autoRename && (next.name || '').includes('未命名')) {
    const derived = guessNameFromUrl(next.url)
    if (derived) {
      next.name = derived
    }
  }
  if (Array.isArray(next.headers)) {
    next.headers = next.headers.filter((r) => !isRowEmpty(r))
  }
  if (Array.isArray(next.query)) {
    next.query = next.query.filter((r) => !isRowEmpty(r))
  }
  if (Array.isArray(next.formData)) {
    next.formData = next.formData.filter((r) => !isRowEmpty(r))
  }
  return next
}

export function normalizeRequest(req: any): Request {
  if (!req) return req
  const normalizeRows = (rows: any[]) => (rows || []).map((r: any) => ({
    ...r,
    enabled: r.enabled !== undefined ? r.enabled : true,
  }))
  return {
    ...req,
    headers: normalizeRows(Array.isArray(req.headers) ? req.headers : []),
    query: normalizeRows(Array.isArray(req.query) ? req.query : []),
    formData: normalizeRows(Array.isArray(req.formData) ? req.formData : []),
    disabledInheritedHeaderKeys: Array.isArray(req.disabledInheritedHeaderKeys)
      ? req.disabledInheritedHeaderKeys
      : [],
  }
}

const CONTENT_TYPE_EXTENSIONS: Record<string, string> = {
  'application/json': 'json',
  'text/html': 'html',
  'text/plain': 'txt',
  'text/xml': 'xml',
  'application/xml': 'xml',
  'application/javascript': 'js',
  'text/javascript': 'js',
  'text/css': 'css',
  'text/markdown': 'md',
  'application/octet-stream': 'bin',
  'image/png': 'png',
  'image/jpeg': 'jpg',
  'image/jpg': 'jpg',
  'image/gif': 'gif',
  'image/svg+xml': 'svg',
  'application/pdf': 'pdf',
}

export function getExtensionFromContentType(contentType: string): string {
  if (!contentType) return 'txt'
  const base = contentType.split(';')[0].trim().toLowerCase()
  return CONTENT_TYPE_EXTENSIONS[base] || 'txt'
}

export function parseFilenameFromHeaders(headers: Record<string, string[]>): string | undefined {
  const cdEntry = Object.entries(headers).find(([k]) => k.toLowerCase() === 'content-disposition')
  if (!cdEntry) return undefined
  const value = cdEntry[1].join('; ')
  const starMatch = value.match(/filename\*=["']?(?:UTF-8['"]*)?([^;"'\n]+)/i)
  if (starMatch) {
    try {
      return decodeURIComponent(starMatch[1].trim())
    } catch {
      return starMatch[1].trim()
    }
  }
  const plainMatch = value.match(/filename=["']?([^;"'\n]+)/i)
  if (plainMatch) {
    const raw = plainMatch[1].trim()
    try {
      return decodeURIComponent(raw)
    } catch {
      return raw
    }
  }
  return undefined
}

export function generateRandomFilename(extension: string): string {
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-')
  const random = Math.random().toString(36).substring(2, 8)
  return `response-${timestamp}-${random}.${extension}`
}
