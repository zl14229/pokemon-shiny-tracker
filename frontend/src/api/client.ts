export type ErrorKind = 'http' | 'network' | 'timeout' | 'aborted' | 'invalid_response'

export class ApiError extends Error {
  readonly name = 'ApiError'
  constructor(
    readonly kind: ErrorKind,
    message: string,
    readonly status: number | null = null,
    readonly code: string | null = null,
  ) {
    super(message)
  }
}

interface RequestOptions {
  signal?: AbortSignal
  timeoutMs?: number
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function httpError(status: number, body: unknown): ApiError {
  const messages: Record<string, string> = {
    not_found: '请求的服务接口不存在，请确认应用版本。',
    method_not_allowed: '服务不支持此操作，请确认应用版本。',
    internal_error: '本地服务暂时不可用，请稍后重试。',
  }
  const code = isRecord(body) && isRecord(body.error) && typeof body.error.code === 'string'
    && Object.hasOwn(messages, body.error.code) ? body.error.code : null
  // Unknown server text/HTML must not be rendered as a user-facing error.
  return new ApiError('http', code ? messages[code]! : `服务请求失败（HTTP ${status}），请重试。`, status, code)
}

export class ApiClient {
  constructor(private readonly baseURL = '/api/v1') {}

  async getJSON(path: string, options: RequestOptions = {}): Promise<unknown> {
    const controller = new AbortController()
    const cancel = () => controller.abort()
    if (options.signal?.aborted) cancel()
    options.signal?.addEventListener('abort', cancel, { once: true })
    let timedOut = false
    const timeout = setTimeout(() => {
      timedOut = true
      controller.abort()
    }, options.timeoutMs ?? 5_000)
    try {
      const response = await fetch(`${this.baseURL}${path}`, {
        method: 'GET', signal: controller.signal,
        headers: { Accept: 'application/json' }, cache: 'no-store', redirect: 'error',
      })
      // Reading the body is covered by the same cancellation/timeout deadline.
      const text = await response.text()
      let body: unknown
      try { body = JSON.parse(text) } catch { body = undefined }
      if (!response.ok) throw httpError(response.status, body)
      if (!response.headers.get('Content-Type')?.toLowerCase().startsWith('application/json') || body === undefined) {
        throw new ApiError('invalid_response', '服务返回了无法识别的响应，请稍后重试。')
      }
      return body
    } catch (error) {
      if (error instanceof ApiError) throw error
      if (timedOut) throw new ApiError('timeout', '连接超时，请确认本地服务已启动后重试。')
      if (controller.signal.aborted) throw new ApiError('aborted', '请求已取消。')
      throw new ApiError('network', '无法连接本地服务，请确认服务已启动后重试。')
    } finally {
      clearTimeout(timeout)
      options.signal?.removeEventListener('abort', cancel)
    }
  }

  async getHealth(options: RequestOptions = {}): Promise<{ status: 'ok' }> {
    const body = await this.getJSON('/health', options)
    if (!isRecord(body) || !isRecord(body.data) || body.data.status !== 'ok') {
      throw new ApiError('invalid_response', '服务返回了无法识别的响应，请稍后重试。')
    }
    return { status: 'ok' }
  }
}

export const api = new ApiClient()
