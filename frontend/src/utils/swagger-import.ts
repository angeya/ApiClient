import { api } from '../services/api'
import type { Header, QueryParam, Request, FormField } from '../types'

export interface SwaggerImportOptions {
  onProgress?: (current: number, total: number) => void
  onCancelled?: () => boolean
  findExistingFolder?: (name: string) => string | undefined
}

export interface SwaggerImportResult {
  success: number
  failed: number
  errors: string[]
  cancelled: boolean
}

interface OpenApiOperation {
  summary?: string
  description?: string
  tags?: string[]
  parameters?: OpenApiParameter[]
  requestBody?: OpenApiRequestBody
  [key: string]: unknown
}

interface OpenApiParameter {
  name: string
  in: 'query' | 'header' | 'path' | 'cookie'
  description?: string
  required?: boolean
  example?: unknown
  schema?: { type?: string; example?: unknown; default?: unknown }
}

interface OpenApiRequestBody {
  description?: string
  content?: Record<string, OpenApiMediaType>
}

interface OpenApiMediaType {
  schema?: unknown
  example?: unknown
  examples?: Record<string, { value?: unknown }>
}

export interface ImportItem {
  method: string
  url: string
  name: string
  folder: string
  request: Request
}

/**
 * Parse an OpenAPI 3.0 JSON string into a structured import result.
 */
export function parseSwagger30(jsonText: string): { items: ImportItem[]; errors: string[] } {
  const errors: string[] = []
  let doc: Record<string, unknown>
  try {
    doc = JSON.parse(jsonText)
  } catch {
    throw new Error('文件不是有效的 JSON 格式')
  }

  if (typeof doc.openapi !== 'string' || !doc.openapi.startsWith('3.')) {
    throw new Error('仅支持 OpenAPI 3.x 规范文件')
  }

  const paths = doc.paths as Record<string, Record<string, OpenApiOperation>> | undefined
  if (!paths || typeof paths !== 'object') {
    return { items: [], errors: [] }
  }

  const servers = doc.servers as Array<{ url?: string }> | undefined
  const baseUrl = servers?.[0]?.url || ''

  const items: ImportItem[] = []
  for (const [pathKey, pathItem] of Object.entries(paths)) {
    if (!pathItem || typeof pathItem !== 'object') continue
    const url = buildUrl(baseUrl, pathKey)

    for (const [method, operation] of Object.entries(pathItem)) {
      if (!operation || typeof operation !== 'object') continue
      if (!isHttpMethod(method)) continue

      try {
        const item = buildImportItem(method.toUpperCase(), url, operation)
        if (item) items.push(item)
      } catch (err) {
        errors.push(`${method.toUpperCase()} ${url}: ${err instanceof Error ? err.message : String(err)}`)
      }
    }
  }

  return { items, errors }
}

function buildUrl(baseUrl: string, path: string): string {
  if (/^https?:\/\//i.test(path)) {
    return path
  }
  // Some OpenAPI exports store paths as host:port/path without a scheme
  if (/^[\w.-]+:\d+(\/|$)/.test(path)) {
    return 'http://' + path
  }
  if (!baseUrl) return path.startsWith('/') ? path : '/' + path
  const normalizedBase = baseUrl.replace(/\/$/, '')
  const normalizedPath = path.startsWith('/') ? path : '/' + path
  return normalizedBase + normalizedPath
}

function isHttpMethod(method: string): boolean {
  return ['get', 'post', 'put', 'delete', 'patch', 'head', 'options', 'trace'].includes(method.toLowerCase())
}

function buildImportItem(method: string, url: string, operation: OpenApiOperation): ImportItem {
  const name = operation.summary || guessNameFromUrl(url) || '未命名接口'
  const folder = operation.tags?.[0] || '未分类'
  const headers: Header[] = []
  const query: QueryParam[] = []

  for (const param of operation.parameters || []) {
    if (!param || typeof param !== 'object') continue
    const value = String(param.example ?? param.schema?.example ?? param.schema?.default ?? '')
    if (param.in === 'query') {
      query.push({ key: param.name, value, enabled: true, description: param.description || '' })
    } else if (param.in === 'header') {
      headers.push({ key: param.name, value, enabled: true, description: param.description || '' })
    }
  }

  const bodyResult = extractBody(operation.requestBody)
  const request: Request = {
    id: '',
    type: 'http',
    name,
    method,
    url,
    headers,
    query,
    bodyType: bodyResult.bodyType,
    body: bodyResult.body,
    formData: bodyResult.formData,
    preRequestScript: '',
    postResponseScript: '',
    description: operation.description || '',
  }

  return { method, url, name, folder, request }
}

function guessNameFromUrl(url: string): string {
  try {
    const u = new URL(url)
    const segments = u.pathname.split('/').filter(Boolean)
    return segments.length > 0 ? segments[segments.length - 1] : u.hostname || ''
  } catch {
    const segments = url.split('/').filter(Boolean)
    return segments.length > 0 ? segments[segments.length - 1] : ''
  }
}

function extractBody(requestBody?: OpenApiRequestBody): {
  bodyType: Request['bodyType']
  body: string
  formData: FormField[]
} {
  if (!requestBody?.content || typeof requestBody.content !== 'object') {
    return { bodyType: 'none', body: '', formData: [] }
  }

  const jsonMedia = requestBody.content['application/json']
  if (jsonMedia) {
    const body = extractJsonBody(jsonMedia)
    return { bodyType: 'json', body, formData: [] }
  }

  const formMedia = requestBody.content['multipart/form-data']
  if (formMedia?.schema) {
    const formData = extractFormFields(formMedia.schema, true)
    return { bodyType: 'form-data', body: '', formData }
  }

  const urlencodedMedia = requestBody.content['application/x-www-form-urlencoded']
  if (urlencodedMedia?.schema) {
    const formData = extractFormFields(urlencodedMedia.schema, false)
    return { bodyType: 'x-www-form-urlencoded', body: '', formData }
  }

  const textMedia = requestBody.content['text/plain']
  if (textMedia?.example !== undefined) {
    return { bodyType: 'text', body: String(textMedia.example), formData: [] }
  }

  return { bodyType: 'none', body: '', formData: [] }
}

function extractJsonBody(media: OpenApiMediaType): string {
  const example = media.example ?? media.examples?.['default']?.value
  if (example === undefined) {
    if (media.schema) {
      return JSON.stringify(extractExampleFromSchema(media.schema), null, 2)
    }
    return ''
  }

  // 很多 OpenAPI 导出工具会把 JSON example 序列化成字符串存储
  if (typeof example === 'string') {
    const trimmed = example.trim()
    if (trimmed) {
      try {
        return JSON.stringify(JSON.parse(trimmed), null, 2)
      } catch {
        return trimmed
      }
    }
    return ''
  }

  return JSON.stringify(example, null, 2)
}

function extractFormFields(schema: unknown, supportFile: boolean): FormField[] {
  if (!schema || typeof schema !== 'object') return []
  const s = schema as Record<string, unknown>
  if (s.type !== 'object') return []

  const properties = s.properties as Record<string, { type?: string; example?: unknown; format?: string; description?: string }> | undefined
  if (!properties) return []

  return Object.entries(properties).map(([key, value]) => {
    const isFile = supportFile && value.format === 'binary'
    const val = value.example ?? ''
    return {
      key,
      value: typeof val === 'string' ? val : JSON.stringify(val),
      type: isFile ? ('file' as const) : ('text' as const),
      enabled: true,
      description: value.description || '',
    }
  })
}

function extractExampleFromSchema(schema: unknown): unknown {
  if (!schema || typeof schema !== 'object') return {}
  const s = schema as Record<string, unknown>
  if (s.example !== undefined) return s.example
  if (s.type === 'object') {
    const props = s.properties as Record<string, unknown> | undefined
    const obj: Record<string, unknown> = {}
    if (props) {
      for (const [key, value] of Object.entries(props)) {
        obj[key] = extractExampleFromSchema(value)
      }
    }
    return obj
  }
  if (s.type === 'array') {
    const item = extractExampleFromSchema(s.items)
    return Array.isArray(item) ? item : [item]
  }
  if (s.type === 'string') return s.default ?? ''
  if (s.type === 'number' || s.type === 'integer') return s.default ?? 0
  if (s.type === 'boolean') return s.default ?? false
  return {}
}

/**
 * Import parsed Swagger items into the workspace.
 */
export async function importSwaggerItems(
  items: ImportItem[],
  options: SwaggerImportOptions = {}
): Promise<SwaggerImportResult> {
  const result: SwaggerImportResult = { success: 0, failed: 0, errors: [], cancelled: false }
  const folderMap = new Map<string, string>()

  for (let i = 0; i < items.length; i++) {
    if (options.onCancelled?.()) {
      result.cancelled = true
      break
    }

    const item = items[i]
    options.onProgress?.(i, items.length)

    try {
      const parentRel = await ensureFolder(item.folder, folderMap, options.findExistingFolder)
      const data = new TextEncoder().encode(JSON.stringify(item.request))
      await api.importRequest(parentRel, Array.from(data))
      result.success++
    } catch (err) {
      result.failed++
      const message = err instanceof Error ? err.message : String(err)
      result.errors.push(`${item.method} ${item.name}: ${message}`)
    }
  }

  options.onProgress?.(items.length, items.length)
  return result
}

async function ensureFolder(
  folderPath: string,
  folderMap: Map<string, string>,
  findExistingFolder?: (name: string) => string | undefined
): Promise<string> {
  const cached = folderMap.get(folderPath)
  if (cached !== undefined) return cached

  const segments = folderPath.split('/').filter(Boolean)
  if (segments.length === 0) {
    return ''
  }

  // Try to reuse an existing folder hierarchy (e.g. cus/新疆晶诺)
  const existingRel = findExistingFolder?.(folderPath)
  if (existingRel !== undefined) {
    folderMap.set(folderPath, existingRel)
    return existingRel
  }

  // Try to reuse an existing top-level folder and create missing sub-folders
  let currentRel = findExistingFolder?.(segments[0])
  if (currentRel === undefined) {
    currentRel = await api.createDir('', segments[0])
  }

  for (let i = 1; i < segments.length; i++) {
    const subPath = segments.slice(0, i + 1).join('/')
    const cachedSub = folderMap.get(subPath)
    if (cachedSub !== undefined) {
      currentRel = cachedSub
      continue
    }
    const existingSub = findExistingFolder?.(subPath)
    if (existingSub !== undefined) {
      currentRel = existingSub
      folderMap.set(subPath, currentRel)
      continue
    }
    currentRel = await api.createDir(currentRel, segments[i])
    folderMap.set(subPath, currentRel)
  }

  folderMap.set(folderPath, currentRel)
  return currentRel
}
