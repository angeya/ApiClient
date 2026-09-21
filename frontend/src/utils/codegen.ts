import type { models } from '../../wailsjs/go/models'

/**
 * 命令行代码生成。
 * 输入为后端 PreviewRequest 解析后的请求（变量已替换、含全局/目录继承头），
 * 因此生成的命令与「实际发送」的请求保持一致。
 */

/** 目标终端类型 */
export type CurlFlavor = 'cmd' | 'shell'

/** 可供选择的终端类型，cmd 为默认项 */
export const CURL_FLAVORS: { label: string; value: CurlFlavor }[] = [
  { label: 'curl (cmd)', value: 'cmd' },
  { label: 'curl (shell)', value: 'shell' },
]

export interface CurlGenerateOptions {
  /** 解析后的请求 */
  preview: models.RequestPreview
  /** 请求方法 */
  method: string
  /** 目标终端类型 */
  flavor: CurlFlavor
  /** 为 true 时输出单行命令，不使用续行符换行 */
  singleLine?: boolean
}

/** shell 单引号字面量：内部单引号需用 '\'' 拼接 */
function quoteShell(value: string): string {
  return `'${value.replace(/'/g, "'\\''")}'`
}

/** CMD 双引号字面量：反斜杠与双引号都要转义，否则 JSON 体会被 CMD 提前截断 */
function quoteCmd(value: string): string {
  return `"${value.replace(/\\/g, '\\\\').replace(/"/g, '\\"')}"`
}

function quote(value: string, flavor: CurlFlavor): string {
  return flavor === 'shell' ? quoteShell(value) : quoteCmd(value)
}

/**
 * 去掉 JSON 字符串之外的所有空白，使请求体压缩为单行。
 * CMD 不支持跨行的引号参数，而 JSON 的空白不影响语义，因此 CMD 模式下使用。
 */
function compactJsonWhitespace(text: string): string {
  const out: string[] = []
  let inString = false
  for (let i = 0; i < text.length; i++) {
    const ch = text[i]
    if (inString) {
      out.push(ch)
      if (ch === '\\') {
        // 转义字符连同后一个字符一起保留，避免误判字符串结束
        if (i + 1 < text.length) out.push(text[i + 1])
        i++
      } else if (ch === '"') {
        inString = false
      }
      continue
    }
    if (ch === '"') {
      inString = true
      out.push(ch)
      continue
    }
    if (ch === ' ' || ch === '\t' || ch === '\n' || ch === '\r') continue
    out.push(ch)
  }
  return out.join('')
}

/** 按终端类型准备请求体文本 */
function prepareBody(body: string, bodyType: string, flavor: CurlFlavor): string {
  if (flavor === 'cmd' && bodyType === 'json') {
    return compactJsonWhitespace(body)
  }
  return body
}

/**
 * 生成 curl 命令。
 * shell 使用单引号 + 反斜杠换行，cmd 使用双引号 + 脱字符换行。
 */
export function generateCurl(options: CurlGenerateOptions): string {
  const { preview, method, flavor, singleLine } = options
  const continuation = flavor === 'shell' ? '\\' : '^'
  const parts: string[] = []
  const bodyType = (preview.bodyType || 'none').toLowerCase()
  // multipart / urlencoded 的 Content-Type 由 curl 自行生成（含 boundary），不能沿用解析结果
  const skipContentType = bodyType === 'form-data' || bodyType === 'x-www-form-urlencoded'

  parts.push(`curl -X ${(method || 'GET').toUpperCase()} ${quote(preview.url || '', flavor)}`)

  const seen = new Set<string>()
  for (const header of preview.headers || []) {
    const key = (header.key || '').trim()
    if (!key) continue
    const lowerKey = key.toLowerCase()
    // curl 会自行计算 Content-Length，重复携带会导致请求异常
    if (lowerKey === 'content-length') continue
    if (skipContentType && lowerKey === 'content-type') continue
    if (seen.has(lowerKey)) continue
    seen.add(lowerKey)
    parts.push(`-H ${quote(`${key}: ${header.value ?? ''}`, flavor)}`)
  }

  // 后端发送时会为 json/text 自动补齐 Content-Type，这里同步补上以保持一致
  if (!skipContentType && bodyType !== 'none' && preview.contentType && !seen.has('content-type')) {
    parts.push(`-H ${quote(`Content-Type: ${preview.contentType}`, flavor)}`)
  }

  const body = preview.body || ''
  if (bodyType === 'form-data') {
    for (const field of preview.formData || []) {
      const key = (field.key || '').trim()
      if (!key) continue
      const value = field.type === 'file' ? `@${field.value}` : field.value
      parts.push(`-F ${quote(`${key}=${value}`, flavor)}`)
    }
  } else if (bodyType !== 'none' && body) {
    parts.push(`--data-raw ${quote(prepareBody(body, bodyType, flavor), flavor)}`)
  }

  // 单行模式下直接用空格拼接，展示区会自动折行，无需续行符
  if (singleLine) {
    return parts.join(' ')
  }
  return parts.join(` ${continuation}\n  `)
}

/**
 * 提示生成结果的已知局限。
 * CMD 无法在引号参数中携带换行，而 shell 的单引号可以，因此仅对 CMD 提示。
 */
export function getCurlWarning(preview: models.RequestPreview, flavor: CurlFlavor): string {
  if (flavor !== 'cmd') return ''
  const bodyType = (preview.bodyType || 'none').toLowerCase()
  // JSON 会被压缩为单行，不存在换行问题
  if (bodyType !== 'text') return ''
  if ((preview.body || '').includes('\n')) {
    return 'CMD 不支持带换行的参数，文本体中的换行会导致命令无法直接执行，建议改用 curl (shell)'
  }
  return ''
}
