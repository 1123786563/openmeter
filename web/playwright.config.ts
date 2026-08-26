import { defineConfig, devices } from '@playwright/test'

const MOCK_IDP_URL = 'http://127.0.0.1:9999'
const APP_HOST = 'http://127.0.0.1:4173'

/**
 * E2E smoke setup: a mock OIDC IdP plus the built SPA served by
 * `vite preview`.
 *
 * VITE_* variables are baked in at build time, so the preview webServer
 * rebuilds with the mock IdP issuer/client wired in (its `env` replaces
 * the whole environment, hence spreading process.env to keep PATH etc).
 * The dev-time redirect URI default points at port 5173; the e2e build
 * must use the preview port 4173 instead.
 */
export default defineConfig({
  testDir: './e2e',
  timeout: 60_000,
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI
    ? [
        ['github'],
        ['html', { open: 'never' }],
      ]
    : [['list']],
  use: {
    baseURL: APP_HOST,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: [
    {
      command: 'node e2e/mock-idp.mjs',
      url: `${MOCK_IDP_URL}/.well-known/openid-configuration`,
      reuseExistingServer: !process.env.CI,
    },
    {
      command: 'pnpm build && pnpm preview --host 127.0.0.1 --port 4173 --strictPort',
      url: APP_HOST,
      timeout: 240_000,
      reuseExistingServer: !process.env.CI,
      env: {
        ...process.env,
        VITE_CASDOOR_ISSUER: MOCK_IDP_URL,
        VITE_CASDOOR_CLIENT_ID: 'openmeter-admin-e2e',
        VITE_CASDOOR_REDIRECT_URI: `${APP_HOST}/auth/callback`,
        VITE_CASDOOR_LOGOUT_REDIRECT_URI: `${APP_HOST}/sign-in`,
      } as Record<string, string>,
    },
  ],
})
