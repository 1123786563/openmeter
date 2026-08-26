import { useAuthStore } from '@/stores/auth-store'
import { useNamespaceStore } from '@/stores/namespace-store'

const API_BASE = import.meta.env.VITE_API_BASE ?? '/api'

/** Error thrown for any non-2xx API response. */
export class ApiError extends Error {
  readonly status: number
  readonly body: unknown

  constructor(status: number, message: string, body?: unknown) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.body = body
  }
}

/**
 * Thin fetch wrapper for the OpenMeter API.
 *
 * Requests go through the `/api` base path which the Vite dev server proxies
 * to the Go backend. Once the generated `@openmeter/client` SDK lands, this
 * wrapper (and its token injection) will be replaced by the SDK; keep the
 * surface minimal on purpose.
 */
export async function apiFetch<T = unknown>(
  path: string,
  init: RequestInit = {}
): Promise<T> {
  const { accessToken, reset } = useAuthStore.getState()
  const { currentNamespace } = useNamespaceStore.getState()

  const headers = new Headers(init.headers)
  if (init.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }
  if (accessToken) {
    headers.set('Authorization', `Bearer ${accessToken}`)
  }
  if (currentNamespace) {
    headers.set('X-Namespace', currentNamespace)
  }

  const res = await fetch(`${API_BASE}${path}`, { ...init, headers })

  if (!res.ok) {
    if (res.status === 401) {
      // Drop the local session; global handlers (query cache / guards) steer
      // the user back to the sign-in page.
      reset()
    }

    let body: unknown
    try {
      body = await res.json()
    } catch {
      body = undefined
    }
    const message =
      (body as { message?: string; title?: string } | null)?.message ??
      (body as { title?: string } | null)?.title ??
      `HTTP ${res.status} ${res.statusText}`.trim()

    throw new ApiError(res.status, message, body)
  }

  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}
