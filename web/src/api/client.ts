// Minimal typed fetch wrapper. The bearer token is supplied per call from the
// OIDC auth context. Errors surface the gateway/service error envelope.

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

const base = import.meta.env.VITE_API_BASE || '/api'

export async function api<T>(
  token: string | undefined,
  path: string,
  init?: RequestInit,
): Promise<T> {
  const headers = new Headers(init?.headers)
  headers.set('Accept', 'application/json')
  if (init?.body) headers.set('Content-Type', 'application/json')
  if (token) headers.set('Authorization', `Bearer ${token}`)

  const res = await fetch(base + path, { ...init, headers })

  if (!res.ok) {
    let code = 'error'
    let message = res.statusText || `HTTP ${res.status}`
    try {
      const body = (await res.json()) as { error?: { code?: string; message?: string } }
      if (body.error?.code) code = body.error.code
      if (body.error?.message) message = body.error.message
    } catch {
      // non-JSON error body; keep status text
    }
    throw new ApiError(res.status, code, message)
  }

  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}
