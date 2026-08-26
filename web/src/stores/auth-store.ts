import { type User } from 'oidc-client-ts'
import { create } from 'zustand'
import {
  safeRedirectPath,
  toAuthUser,
  userManager,
  type AuthUser,
} from '@/lib/auth'

interface AuthState {
  /** Null until `initialize` resolves and no session exists. */
  user: AuthUser | null
  /** In-memory access token; refreshed automatically via silent renew. */
  accessToken: string
  initialized: boolean
  /** Idempotent: restores the session from the OIDC store and wires events. */
  initialize: () => Promise<void>
  /** Mirror an OIDC user (or clear on null) into the store. */
  setOidcUser: (user: User | null) => void
  /** Clear the local session without contacting Casdoor. */
  reset: () => void
  /** Redirect to Casdoor to sign in. */
  signin: (redirectTo?: string) => Promise<void>
  /** Redirect to Casdoor to sign out (clears local state first). */
  signout: () => Promise<void>
}

export const useAuthStore = create<AuthState>()((set, get) => {
  // The OIDC session lives in the UserManager store; this zustand slice is
  // the in-memory projection the UI reads. `initialize` is guarded so the
  // router guard and React tree converge on a single bootstrap.
  let bootstrap: Promise<void> | null = null

  return {
    user: null,
    accessToken: '',
    initialized: false,

    initialize: () => {
      if (!bootstrap) {
        bootstrap = (async () => {
          userManager.events.addUserLoaded((user) => get().setOidcUser(user))
          userManager.events.addUserUnloaded(() => get().setOidcUser(null))
          const user = await userManager.getUser()
          get().setOidcUser(user)
          set({ initialized: true })
        })()
      }
      return bootstrap
    },

    setOidcUser: (user) => {
      if (user) {
        set({
          user: toAuthUser(user),
          accessToken: user.access_token,
        })
      } else {
        set({ user: null, accessToken: '' })
      }
    },

    reset: () => {
      void userManager.removeUser().catch(() => undefined)
      set({ user: null, accessToken: '' })
    },

    signin: async (redirectTo) => {
      await userManager.signinRedirect({
        // Sanitize before the value enters the OIDC state: it round-trips
        // through Casdoor and drives the post-login navigation.
        state: redirectTo
          ? { redirect: safeRedirectPath(redirectTo) }
          : undefined,
      })
    },

    signout: async () => {
      get().reset()
      await userManager.signoutRedirect()
    },
  }
})
