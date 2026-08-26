import { OpenMeter } from '@openmeter/client'
import { useAuthStore } from '@/stores/auth-store'
import { useNamespaceStore } from '@/stores/namespace-store'

const API_BASE = import.meta.env.VITE_API_BASE ?? '/api'

/**
 * Shared request hooks for every v3 SDK call.
 *
 * - `Authorization` comes from the OIDC access token (read per request so
 *   silent renew is picked up without rebuilding the client).
 * - `X-Namespace` mirrors the namespace switcher store (fork addition; the
 *   backend validates it against the namespace allowlist).
 * - A 401 clears the local session; the query cache steers back to sign-in.
 */
const requestHooks = {
  beforeRequest: [
    async ({ request }: { request: Request }) => {
      const { currentNamespace } = useNamespaceStore.getState()
      if (currentNamespace) {
        request.headers.set('X-Namespace', currentNamespace)
      }
    },
  ],
  afterResponse: [
    async ({ response }: { response: Response }) => {
      if (response.status === 401) {
        useAuthStore.getState().reset()
      }
    },
  ],
}

/**
 * Singleton v3 API client backed by the repo-generated SDK
 * (`api/spec/packages/aip-client-javascript`), served under `/api/v3`.
 */
export const api = new OpenMeter({
  baseUrl: `${API_BASE}/v3`,
  apiKey: () => useAuthStore.getState().accessToken || '',
  hooks: requestHooks,
})
