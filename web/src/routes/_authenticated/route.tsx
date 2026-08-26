import { createFileRoute, redirect } from '@tanstack/react-router'
import { useAuthStore } from '@/stores/auth-store'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'

export const Route = createFileRoute('/_authenticated')({
  beforeLoad: async ({ location }) => {
    // Restores the OIDC session (idempotent) before deciding access.
    await useAuthStore.getState().initialize()

    if (!useAuthStore.getState().user) {
      throw redirect({
        to: '/sign-in',
        // Pathname only: the value round-trips through the OIDC state and
        // must stay a same-origin path (see safeRedirectPath).
        search: { redirect: location.pathname },
      })
    }
  },
  component: AuthenticatedLayout,
})
