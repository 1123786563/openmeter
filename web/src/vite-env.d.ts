/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE?: string
  readonly VITE_CASDOOR_ISSUER?: string
  readonly VITE_CASDOOR_CLIENT_ID?: string
  readonly VITE_CASDOOR_REDIRECT_URI?: string
  readonly VITE_CASDOOR_LOGOUT_REDIRECT_URI?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
