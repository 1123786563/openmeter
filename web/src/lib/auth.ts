import { UserManager, type User } from 'oidc-client-ts'

/**
 * Casdoor OIDC configuration (authorization code flow + PKCE).
 *
 * Issuer and client id come from Vite env vars so each environment (local,
 * staging, production) can point at its own Casdoor instance. Redirect URIs
 * default to the current origin: the dev server may run on any port (5173 is
 * often taken and Vite silently moves to 5174/5175), and a hardcoded port
 * would send the post-login and post-logout redirects to whatever app owns
 * that port instead of back here.
 */
const issuer = import.meta.env.VITE_CASDOOR_ISSUER ?? ''
const clientId = import.meta.env.VITE_CASDOOR_CLIENT_ID ?? ''
const redirectUri =
  import.meta.env.VITE_CASDOOR_REDIRECT_URI ??
  `${window.location.origin}/auth/callback`
const logoutRedirectUri =
  import.meta.env.VITE_CASDOOR_LOGOUT_REDIRECT_URI ??
  `${window.location.origin}/sign-in`

if (!issuer || !clientId) {
  // Missing config breaks every login attempt; surface it early instead of
  // failing inside the OIDC redirect dance where it is hard to debug.
  // eslint-disable-next-line no-console
  console.warn(
    '[auth] VITE_CASDOOR_ISSUER or VITE_CASDOOR_CLIENT_ID is not set. Copy .env.example to .env and configure Casdoor first.'
  )
}

export const userManager = new UserManager({
  authority: issuer,
  client_id: clientId,
  redirect_uri: redirectUri,
  post_logout_redirect_uri: logoutRedirectUri,
  scope: 'openid profile',
  response_type: 'code',
  automaticSilentRenew: true,
})

/** Minimal user shape consumed by the admin UI. */
export interface AuthUser {
  id: string
  name: string
  email: string
  avatar: string | null
}

/** Project an OIDC user onto the shape the UI renders. */
export function toAuthUser(user: User): AuthUser {
  const profile = user.profile
  return {
    id: profile.sub,
    name:
      profile.name ??
      profile.preferred_username ??
      profile.given_name ??
      profile.sub,
    email: profile.email ?? '',
    avatar: profile.picture ?? null,
  }
}

/**
 * Normalize a user-supplied redirect target to a same-origin app path.
 *
 * Only absolute in-app paths are honored. Protocol-relative (`//host`) and
 * backslash-prefixed (`/\host`) forms are rejected because browsers resolve
 * them cross-origin, which would turn the post-login redirect into an open
 * redirect suitable for phishing.
 */
export function safeRedirectPath(target: string | null | undefined): string {
  if (typeof target !== 'string') return '/'
  if (
    !target.startsWith('/') ||
    target.startsWith('//') ||
    target.startsWith('/\\')
  ) {
    return '/'
  }
  return target
}
