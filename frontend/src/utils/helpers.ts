import type { Header, QueryParam, Variable, Request } from '../types'

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
