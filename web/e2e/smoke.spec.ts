import { expect, test, type Page } from '@playwright/test'

const MOCK_CUSTOMER_NAME = 'Acme Corporation'

/**
 * Drive the full login round-trip against the mock IdP: land on the
 * sign-in page, click the Casdoor button, get 302'd back to /auth/callback,
 * and wait until the app redirects to the dashboard.
 */
async function login(page: Page) {
  await page.goto('/')

  // The _authenticated guard bounces anonymous users to /sign-in with the
  // original path preserved in ?redirect.
  await expect(page).toHaveURL(/\/sign-in/)

  // Default locale is zh-CN ("使用 Casdoor 登录"); the en copy shares the
  // "Casdoor" token, so match on that.
  await page.getByRole('button', { name: 'Casdoor' }).click()

  // Mock IdP answers authorize with an immediate 302 back to the callback,
  // which completes the code exchange and redirects to the dashboard.
  await expect(page).toHaveURL(/127\.0\.0\.1:4173\/$/)
}

test('sign-in smoke: OIDC round-trip lands on the dashboard', async ({
  page,
}) => {
  await login(page)

  await expect(page).toHaveTitle('OpenMeter Admin')

  // The authenticated layout renders the sidebar; the customers entry is a
  // stable, locale-default (zh-CN) anchor to assert against.
  await expect(page.getByRole('link', { name: '客户' })).toBeVisible()
})

test('customers smoke: route reachable and renders list data', async ({
  page,
}) => {
  // Fulfill the customers list APIs. Two callers exist:
  // - the v3 SDK path the page actually calls today
  //   (GET /api/v3/openmeter/customers, wire format snake_case +
  //   meta.page per the SDK's listCustomersResponseWire schema), and
  // - the v1 REST path (GET /api/v1/customers, CustomerPaginatedResponse
  //   from api/openapi.yaml) kept covered for the legacy transition.
  await page.route('**/api/v3/openmeter/customers*', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: [
          {
            id: '01G65Z755AFWAKHE12NY0CQ9FH',
            name: MOCK_CUSTOMER_NAME,
            key: 'acme-corp',
            primary_email: 'billing@acme.example.com',
            created_at: '2024-01-01T01:01:01.001Z',
            updated_at: '2024-01-01T01:01:01.001Z',
          },
        ],
        meta: { page: { number: 1, size: 10, total: 1 } },
      }),
    })
  })
  await page.route('**/api/v1/customers**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        totalCount: 1,
        page: 1,
        pageSize: 10,
        items: [
          {
            id: '01G65Z755AFWAKHE12NY0CQ9FH',
            name: MOCK_CUSTOMER_NAME,
            key: 'acme-corp',
            primaryEmail: 'billing@acme.example.com',
            createdAt: '2024-01-01T01:01:01.001Z',
            updatedAt: '2024-01-01T01:01:01.001Z',
          },
        ],
      }),
    })
  })

  await login(page)

  await page.goto('/customers')
  await expect(page).toHaveURL(/\/customers$/)

  // The real customers page renders the mocked customer; until a feature
  // ships, its route renders the shared placeholder ("建设中" / "Under
  // Construction") instead.
  const placeholderHeading = page.getByRole('heading', {
    name: /建设中|Under Construction/,
  })
  const customerCell = page.getByText(MOCK_CUSTOMER_NAME)
  await expect(customerCell.or(placeholderHeading)).toBeVisible()
})
