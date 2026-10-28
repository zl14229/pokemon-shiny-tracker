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

export interface RequestOptions {
  signal?: AbortSignal
  timeoutMs?: number
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function httpError(status: number, body: unknown): ApiError {
  const messages: Record<string, string> = {
    reference_unavailable: '本地刷闪资料来源或映射不完整，请核对资料；不会补入未知目标。',
    insufficient_candidates: '已核实候选数量不足，请查看资料覆盖与候选数量。',
    not_found: '记录不存在、已删除或接口不可用，请刷新后确认。',
    method_not_allowed: '服务不支持此操作，请确认应用版本。',
    internal_error: '本地服务暂时不可用，请稍后重试。',
    invalid_request: '输入或引用资料无效，请检查表单后重试。',
    conflict: '记录存在关联或状态冲突，请刷新后重试。',
    request_too_large: '内容过长，请缩短备注后重试。',
    unsupported_media_type: '提交格式不受支持，请确认应用版本。',
  }
  const code =
    isRecord(body) &&
    isRecord(body.error) &&
    typeof body.error.code === 'string' &&
    Object.hasOwn(messages, body.error.code)
      ? body.error.code
      : null
  // Unknown server text/HTML must not be rendered as a user-facing error.
  return new ApiError(
    'http',
    code ? messages[code]! : `服务请求失败（HTTP ${status}），请重试。`,
    status,
    code,
  )
}

export class ApiClient {
  constructor(private readonly baseURL = '/api/v1') {}

  getJSON(path: string, options: RequestOptions = {}): Promise<unknown> {
    return this.requestJSON('GET', path, undefined, options)
  }

  postJSON(path: string, data: unknown, options: RequestOptions = {}): Promise<unknown> {
    return this.requestJSON('POST', path, data, options)
  }

  postForm(path: string, data: FormData, options: RequestOptions = {}): Promise<unknown> {
    return this.requestJSON('POST', path, data, options)
  }

  deleteJSON(path: string, options: RequestOptions = {}): Promise<unknown> {
    return this.requestJSON('DELETE', path, undefined, options)
  }

  putJSON(path: string, data: unknown, options: RequestOptions = {}): Promise<unknown> {
    return this.requestJSON('PUT', path, data, options)
  }

  private async requestJSON(
    method: string,
    path: string,
    data: unknown,
    options: RequestOptions,
  ): Promise<unknown> {
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
        method,
        signal: controller.signal,
        headers:
          data === undefined || data instanceof FormData
            ? { Accept: 'application/json' }
            : {
                Accept: 'application/json',
                'Content-Type': 'application/json',
              },
        ...(data === undefined
          ? {}
          : { body: data instanceof FormData ? data : JSON.stringify(data) }),
        cache: 'no-store',
        redirect: 'error',
      })
      // Reading the body is covered by the same cancellation/timeout deadline.
      const text = await response.text()
      let body: unknown
      try {
        body = JSON.parse(text)
      } catch {
        body = undefined
      }
      if (!response.ok) throw httpError(response.status, body)
      if (
        !response.headers.get('Content-Type')?.toLowerCase().startsWith('application/json') ||
        body === undefined
      ) {
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

  async download(path: string, expectedType: string, options: RequestOptions = {}): Promise<Blob> {
    const controller = new AbortController()
    const cancel = () => controller.abort()
    if (options.signal?.aborted) cancel()
    options.signal?.addEventListener('abort', cancel, { once: true })
    let timedOut = false
    const timer = setTimeout(() => {
      timedOut = true
      controller.abort()
    }, options.timeoutMs ?? 120000)
    try {
      const response = await fetch(`${this.baseURL}${path}`, {
        signal: controller.signal,
        headers: { Accept: expectedType },
        cache: 'no-store',
        redirect: 'error',
      })
      if (!response.ok) {
        const body = await response.json().catch(() => undefined)
        throw httpError(response.status, body)
      }
      const type = response.headers.get('Content-Type')?.split(';')[0]?.toLowerCase()
      const length = Number(response.headers.get('Content-Length'))
      if (type !== expectedType || length > 2 * 1024 * 1024 * 1024)
        throw new ApiError('invalid_response', '导出文件类型或大小不符合要求。')
      const blob = await response.blob()
      if (blob.size === 0 || blob.size > 2 * 1024 * 1024 * 1024)
        throw new ApiError('invalid_response', '导出文件为空或大小超限。')
      return blob
    } catch (e) {
      if (e instanceof ApiError) throw e
      if (timedOut)
        throw new ApiError('timeout', '生成或下载超时；可使用本地命令重试，已有记录不会被覆盖。')
      if (controller.signal.aborted) throw new ApiError('aborted', '下载已取消。')
      throw new ApiError('network', '下载失败，请确认本地服务可用后重试。')
    } finally {
      clearTimeout(timer)
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
