import { expect, test, type Page } from '@playwright/test'
import { mkdirSync, writeFileSync } from 'node:fs'

/**
 * Issue #29 acceptance walkthrough (scratch artifact — NOT committed).
 *
 * Full-chain write-operation walkthrough over the four config blocks with
 * STATEFUL route mocks: POSTs mutate an in-memory store, GETs read it back,
 * so each create completes the genuine UI round trip (form -> submit ->
 * invalidate -> refetch -> row visible). Any /api call without an explicit
 * mock is aborted and recorded — unknown endpoints surface as failures
 * instead of silently hitting whatever listens behind the preview proxy.
 */

const SHOTS = '/tmp/issue29-shots'
mkdirSync(SHOTS, { recursive: true })

const NOW = '2026-08-29T12:00:00.000Z'
const ULID = {
  feature: '01JDVF0R4T2Z3Y8W7X6V5U4T3R',
  channel: '01JDVF0R4T2Z3Y8W7X6V5U4T4S',
  taxCode: '01JDVF0R4T2Z3Y8W7X6V5U4T5T',
  app: '01JDVF0R4T2Z3Y8W7X6V5U4T6U',
}

const evidence: Record<string, unknown>[] = []

test.use({ viewport: { width: 1440, height: 900 } })

function record(page: Page) {
  page.on('request', (req) => {
    if (req.url().includes('/api/')) {
      evidence.push({ kind: 'request', method: req.method(), url: req.url() })
    }
  })
}

function json(route: { fulfill: (a: unknown) => Promise<void> }, status: number, body: unknown) {
  return route.fulfill({
    status,
    contentType: 'application/json',
    body: JSON.stringify(body),
  })
}

async function login(page: Page) {
  await page.goto('/')
  await expect(page).toHaveURL(/\/sign-in/)
  await page.getByRole('button', { name: 'Casdoor' }).click()
  await expect(page).toHaveURL(/127\.0\.0\.1:4173\/$/)
}

async function mockNamespaces(page: Page) {
  await page.route('**/api/v3/openmeter/namespaces*', (route) =>
    json(route, 200, { default: 'default', namespaces: ['default'] })
  )
}

/* ------------------------------------------------------------------ */
/* Block 1: /config/features — create feature                          */
/* ------------------------------------------------------------------ */
test('walkthrough 1: features create', async ({ page }) => {
  record(page)
  // Catch-all FIRST (lowest precedence): unknown /api endpoints abort.
  const aborted: string[] = []
  await page.route('**/api/**', (route) => {
    aborted.push(`${route.request().method()} ${route.request().url()}`)
    return route.abort()
  })

  const features: Record<string, unknown>[] = []
  await mockNamespaces(page)
  await page.route('**/api/v3/openmeter/features*', async (route) => {
    const req = route.request()
    if (req.method() === 'GET') {
      return json(route, 200, {
        data: features,
        meta: { page: { number: 1, size: 50, total: features.length } },
      })
    }
    const body = req.postDataJSON() as { name: string; key: string }
    const created = {
      id: ULID.feature,
      name: body.name,
      key: body.key,
      created_at: NOW,
      updated_at: NOW,
    }
    features.push(created)
    evidence.push({ kind: 'feature-created', stored: created })
    return json(route, 201, created)
  })

  await login(page)
  await page.goto('/config/features')
  // Sidebar config group visible alongside the result (screenshot requirement).
  await expect(page.getByRole('link', { name: '功能', exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: '新建功能' })).toBeVisible()

  await page.getByRole('button', { name: '新建功能' }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByLabel('名称').fill('验收-功能')
  await dialog.getByLabel('标识（key）').fill('acceptance_feature')
  await dialog.getByRole('button', { name: '确定', exact: true }).click()

  await expect(page.getByText('功能已创建')).toBeVisible()
  await expect(page.getByText('验收-功能')).toBeVisible()
  await expect(page.getByText('acceptance_feature')).toBeVisible()
  await page.screenshot({ path: `${SHOTS}/01-config-features-created.png` })
  evidence.push({ kind: 'assert', block: 1, passed: ['toast 功能已创建', 'row 验收-功能', 'row acceptance_feature'] })
})

/* ------------------------------------------------------------------ */
/* Block 2: /config/notification/channels — create webhook channel     */
/* ------------------------------------------------------------------ */
test('walkthrough 2: channels create', async ({ page }) => {
  record(page)
  await page.route('**/api/**', (route) => route.abort())
  const channels: Record<string, unknown>[] = []
  await mockNamespaces(page)
  await page.route('**/api/v1/notification/channels**', async (route) => {
    const req = route.request()
    if (req.method() === 'GET') {
      return json(route, 200, {
        totalCount: channels.length,
        page: 1,
        pageSize: 50,
        items: channels,
      })
    }
    const body = req.postDataJSON() as { name: string; url: string }
    const created = {
      id: ULID.channel,
      type: 'WEBHOOK',
      name: body.name,
      url: body.url,
      disabled: false,
      createdAt: NOW,
      updatedAt: NOW,
    }
    channels.push(created)
    evidence.push({ kind: 'channel-created', stored: created })
    return json(route, 201, created)
  })

  await login(page)
  await page.goto('/config/notification/channels')
  await expect(page.getByRole('link', { name: '通知', exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: '新建渠道' })).toBeVisible()

  await page.getByRole('button', { name: '新建渠道' }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByLabel('名称').fill('验收-Webhook')
  await dialog.getByLabel('Webhook URL').fill('https://example.com/webhook')
  await dialog.getByRole('button', { name: '确定', exact: true }).click()

  await expect(page.getByText('通知渠道已创建。')).toBeVisible()
  const row = page.getByRole('row').filter({ hasText: '验收-Webhook' })
  await expect(row).toBeVisible()
  await expect(row).not.toContainText('已禁用')
  await page.screenshot({ path: `${SHOTS}/02-config-channel-created.png` })
  evidence.push({ kind: 'assert', block: 2, passed: ['toast 通知渠道已创建。', 'row 验收-Webhook visible', 'row not 已禁用'] })
})

/* ------------------------------------------------------------------ */
/* Block 3: /config/tax-codes — create tax code                        */
/* ------------------------------------------------------------------ */
test('walkthrough 3: tax codes create', async ({ page }) => {
  record(page)
  await page.route('**/api/**', (route) => route.abort())
  const taxCodes: Record<string, unknown>[] = []
  await mockNamespaces(page)
  await page.route('**/api/v3/openmeter/tax-codes*', async (route) => {
    const req = route.request()
    if (req.method() === 'GET') {
      return json(route, 200, {
        data: taxCodes,
        meta: { page: { number: 1, size: 100, total: taxCodes.length } },
      })
    }
    const body = req.postDataJSON() as { name: string; key: string }
    const created = {
      id: ULID.taxCode,
      name: body.name,
      key: body.key,
      app_mappings: [],
      created_at: NOW,
      updated_at: NOW,
    }
    taxCodes.push(created)
    evidence.push({ kind: 'tax-code-created', stored: created })
    return json(route, 201, created)
  })
  await page.route('**/api/v3/openmeter/defaults/tax-codes*', (route) =>
    json(route, 200, {
      invoicing_tax_code: { id: ULID.taxCode },
      credit_grant_tax_code: { id: ULID.taxCode },
      created_at: NOW,
      updated_at: NOW,
    })
  )

  await login(page)
  await page.goto('/config/tax-codes')
  await expect(page.getByRole('link', { name: '税码', exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: '新建税码' })).toBeVisible()

  await page.getByRole('button', { name: '新建税码' }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByLabel('名称').fill('验收税码')
  await dialog.getByLabel('标识（key）').fill('acceptance_tax_code')
  await dialog.getByRole('button', { name: '确定', exact: true }).click()

  await expect(page.getByText('税码已创建')).toBeVisible()
  await expect(page.getByRole('cell', { name: '验收税码', exact: true })).toBeVisible()
  await page.screenshot({ path: `${SHOTS}/03-config-tax-code-created.png` })
  evidence.push({ kind: 'assert', block: 3, passed: ['toast 税码已创建', 'row 验收税码'] })
})

/* ------------------------------------------------------------------ */
/* Block 4: /config/apps — install sandbox app from catalog            */
/* ------------------------------------------------------------------ */
test('walkthrough 4: apps install sandbox', async ({ page }) => {
  record(page)
  await page.route('**/api/**', (route) => route.abort())
  const apps: Record<string, unknown>[] = []
  await mockNamespaces(page)

  const sandboxCatalogItem = {
    type: 'sandbox',
    name: 'Sandbox',
    description: 'Sandbox billing app for acceptance walkthrough.',
    capabilities: [
      {
        type: 'report_usage',
        key: 'sandbox_usage',
        name: 'Sandbox usage',
        description: 'Report usage into the sandbox app.',
      },
    ],
    install_methods: ['no_credentials_required'],
  }

  await page.route('**/api/v3/openmeter/apps*', (route) =>
    json(route, 200, {
      data: apps,
      meta: { page: { number: 1, size: 100, total: apps.length } },
    })
  )
  await page.route('**/api/v3/openmeter/app-catalog**', async (route) => {
    const req = route.request()
    try {
      if (req.method() === 'GET' && !req.url().includes('install')) {
        return json(route, 200, {
          data: [sandboxCatalogItem],
          meta: { page: { number: 1, size: 10, total: 1 } },
        })
      }
      evidence.push({ kind: 'install-handler-enter', method: req.method(), url: req.url(), body: req.postDataJSON() })
      const body = route.request().postDataJSON() as { name: string }
    const installed = {
      id: ULID.app,
      name: body.name,
      type: 'sandbox',
      definition: sandboxCatalogItem,
      status: 'ready',
      created_at: NOW,
      updated_at: NOW,
    }
    apps.push(installed)
    evidence.push({ kind: 'app-installed', stored: { id: installed.id, name: installed.name, type: 'sandbox', status: 'ready' } })
    // Install response wraps the app (billingInstallAppResponseWire).
    return json(route, 201, { app: installed, default_for_capability_types: ['report_usage'] })
    } catch (err) {
      evidence.push({ kind: 'install-handler-error', error: String(err) })
      return json(route, 500, { error: 'walkthrough mock handler failure' })
    }
  })

  await login(page)
  await page.goto('/config/apps')
  await expect(page.getByRole('link', { name: '应用', exact: true })).toBeVisible()
  await expect(page.getByText('Sandbox').first()).toBeVisible()

  page.on('requestfailed', (req) => {
    evidence.push({ kind: 'requestfailed', url: req.url(), failure: req.failure()?.errorText })
  })
  page.on('response', (res) => {
    if (res.url().includes('install')) {
      evidence.push({ kind: 'install-response', status: res.status(), url: res.url() })
    }
  })

  // Catalog card's install button opens the install dialog.
  await page.getByRole('button', { name: '安装' }).first().click()
  const dialog = page.getByRole('dialog')
  await dialog.getByLabel('名称').fill('验收-Sandbox')
  await dialog.getByRole('button', { name: '安装', exact: true }).click()

  await expect(page.getByText('应用已安装', { exact: true })).toBeVisible()
  await expect(page.getByText('验收-Sandbox').first()).toBeVisible()
  await page.screenshot({ path: `${SHOTS}/04-config-app-installed.png` })
  evidence.push({ kind: 'assert', block: 4, passed: ['toast 应用已安装', 'installed row 验收-Sandbox'] })
})

test.afterAll(async () => {
  writeFileSync('/tmp/issue29-walkthrough-evidence.json', JSON.stringify(evidence, null, 2))
})
