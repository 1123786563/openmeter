import { execFileSync } from 'node:child_process'
import { readFileSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import * as assert from 'node:assert'
import { describe, it } from 'node:test'
import YAML from 'yaml'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const pkgDir = path.resolve(__dirname, '..')

let cachedSpec = null

/**
 * Compile the AIP TypeSpec spec and return the parsed OpenAPI document for the
 * OpenMeter service. The result is cached for the duration of the test run.
 */
function compileAIP() {
  if (cachedSpec) return cachedSpec

  execFileSync(
    path.join(pkgDir, 'node_modules', '.bin', 'tsp'),
    [
      'compile',
      '--config',
      'tspconfig.yaml',
      './src',
      '--emit',
      '@typespec/openapi3',
    ],
    {
      cwd: pkgDir,
      stdio: 'pipe',
    },
  )

  const outputPath = path.join(
    pkgDir,
    'output/definitions/metering-and-billing/v3/openapi.OpenMeter.yaml',
  )
  cachedSpec = YAML.parse(readFileSync(outputPath, 'utf-8'))
  return cachedSpec
}

/**
 * Expected Phase 2 commerce routes under /api/v3.
 * Each entry is [method, path, operationId].
 */
const expectedRoutes = [
  ['get', '/customers/{customerId}/wallet', 'get-customer-wallet'],
  ['get', '/recharge-products', 'list-recharge-products'],
  ['post', '/orders', 'create-order'],
  ['get', '/orders/{orderId}', 'get-order'],
  ['post', '/orders/{orderId}/checkout-sessions', 'create-checkout-session'],
  ['get', '/checkout-sessions/{sessionId}', 'get-checkout-session'],
  ['post', '/refunds', 'create-refund'],
  ['get', '/refunds/{refundId}', 'get-refund'],
  ['post', '/payment-providers/wechat/callback', 'wechat-payment-callback'],
  [
    'post',
    '/payment-providers/wechat/refund-callback',
    'wechat-refund-callback',
  ],
  ['post', '/payment-providers/alipay/callback', 'alipay-payment-callback'],
  [
    'get',
    '/customers/{customerId}/receivable-periods',
    'list-receivable-periods',
  ],
  [
    'post',
    '/customers/{customerId}/offline-payments',
    'create-offline-payment',
  ],
  [
    'put',
    '/customers/{customerId}/receivable-periods/{periodId}/external-invoice',
    'update-external-invoice',
  ],
]

describe('WeKnora Phase 2 commerce contract', () => {
  describe('routes', () => {
    it('exposes all 14 Phase 2 commerce paths with correct methods and operation IDs', () => {
      const spec = compileAIP()
      for (const [method, routePath, opId] of expectedRoutes) {
        const pathItem = spec.paths[routePath]
        assert.ok(pathItem, `Path ${routePath} must exist`)
        const op = pathItem[method]
        assert.ok(
          op,
          `Method ${method.toUpperCase()} on ${routePath} must exist`,
        )
        assert.equal(
          op.operationId,
          opId,
          `Operation ID for ${method.toUpperCase()} ${routePath}`,
        )
      }
    })

    it('declares provider-native callback request media types', () => {
      const spec = compileAIP()
      const callbackMediaTypes = [
        ['/payment-providers/wechat/callback', ['application/json']],
        ['/payment-providers/wechat/refund-callback', ['application/json']],
        [
          '/payment-providers/alipay/callback',
          ['application/x-www-form-urlencoded'],
        ],
      ]

      for (const [routePath, wantMediaTypes] of callbackMediaTypes) {
        const requestContent = spec.paths[routePath]?.post?.requestBody?.content
        assert.ok(requestContent, `${routePath} must declare a request body`)
        assert.deepEqual(
          Object.keys(requestContent),
          wantMediaTypes,
          `${routePath} request media types`,
        )
      }
    })
  })

  describe('order plan references', () => {
    it('allows create-order plan_id omission while response plan_id remains required', () => {
      const spec = compileAIP()
      const requestRef = spec.components.schemas.CommerceOrderCreatePlanRef
      const responseRef = spec.components.schemas.CommercePlanRef

      assert.ok(requestRef.properties.plan_id, 'request plan_id property')
      assert.ok(
        !requestRef.required.includes('plan_id'),
        'request plan_id is optional',
      )
      assert.ok(
        responseRef.required.includes('plan_id'),
        'response plan_id is required',
      )
    })
  })

  describe('enums', () => {
    it('WalletBucketSource has the four required sources', () => {
      const spec = compileAIP()
      assert.deepEqual(
        spec.components.schemas.CommerceWalletBucketSource.enum,
        ['plan', 'gift', 'recharge', 'enterprise_receivable'],
      )
    })

    it('WalletTransactionKind has the five required kinds', () => {
      const spec = compileAIP()
      assert.deepEqual(
        spec.components.schemas.CommerceWalletTransactionKind.enum,
        ['funded', 'consumed', 'expired', 'refunded', 'adjusted'],
      )
    })

    it('OrderKind has the three required kinds', () => {
      const spec = compileAIP()
      assert.deepEqual(spec.components.schemas.CommerceOrderKind.enum, [
        'plan_purchase',
        'subscription_renewal',
        'wallet_top_up',
      ])
    })

    it('OrderStatus has all nine states', () => {
      const spec = compileAIP()
      assert.deepEqual(spec.components.schemas.CommerceOrderStatus.enum, [
        'created',
        'awaiting_payment',
        'paid',
        'fulfilled',
        'cancelled',
        'expired',
        'refund_pending',
        'partially_refunded',
        'refunded',
      ])
    })

    it('PaymentAttemptStatus has all five states', () => {
      const spec = compileAIP()
      assert.deepEqual(
        spec.components.schemas.CommercePaymentAttemptStatus.enum,
        ['created', 'pending', 'succeeded', 'failed', 'closed'],
      )
    })

    it('PaymentProvider is wechat, alipay, offline', () => {
      const spec = compileAIP()
      assert.deepEqual(spec.components.schemas.CommercePaymentProvider.enum, [
        'wechat',
        'alipay',
        'offline',
      ])
    })

    it('RefundStatus has all five states including fence', () => {
      const spec = compileAIP()
      assert.deepEqual(spec.components.schemas.CommerceRefundStatus.enum, [
        'pending_fence',
        'provider_processing',
        'ledger_reversing',
        'fulfilled',
        'failed',
      ])
    })

    it('ReceivablePeriodStatus has all five states', () => {
      const spec = compileAIP()
      assert.deepEqual(
        spec.components.schemas.CommerceReceivablePeriodStatus.enum,
        ['open', 'closed', 'partially_paid', 'paid', 'overdue'],
      )
    })
  })

  describe('wallet model', () => {
    it('WalletBucket carries available_credits, expires_at, refundable_credits, and source', () => {
      const spec = compileAIP()
      const props = spec.components.schemas.CommerceWalletBucket.properties
      assert.ok(props.source, 'source field')
      assert.ok(props.available_credits, 'available_credits field')
      assert.ok(props.expires_at, 'expires_at field')
      assert.ok(props.refundable_credits, 'refundable_credits field')
    })

    it('WalletTransaction carries immutable Ledger provenance and occurred_at', () => {
      const spec = compileAIP()
      const props = spec.components.schemas.CommerceWalletTransaction.properties
      assert.ok(props.kind, 'kind field')
      assert.ok(props.amount, 'amount field')
      assert.ok(props.provenance, 'provenance field')
      assert.ok(props.occurred_at, 'occurred_at field')
    })
  })

  describe('idempotency and security', () => {
    it('every mutation request accepts Idempotency-Key', () => {
      const spec = compileAIP()
      const mutationRoutes = [
        ['post', '/orders'],
        ['post', '/orders/{orderId}/checkout-sessions'],
        ['post', '/refunds'],
        ['post', '/customers/{customerId}/offline-payments'],
        [
          'put',
          '/customers/{customerId}/receivable-periods/{periodId}/external-invoice',
        ],
      ]

      for (const [method, routePath] of mutationRoutes) {
        const op = spec.paths[routePath][method]
        const bodySchemaName =
          op.requestBody?.content?.['application/json']?.schema?.$ref
        assert.ok(
          bodySchemaName,
          `${method.toUpperCase()} ${routePath} should have a JSON body`,
        )

        // Resolve the schema ref to check for idempotency_key
        const schemaName = bodySchemaName.split('/').pop()
        const schema = spec.components.schemas[schemaName]
        const props = schema.properties || schema.allOf?.[0]?.properties || {}
        assert.ok(
          props.idempotency_key,
          `${schemaName} must have idempotency_key`,
        )
      }
    })

    it('no response schema contains secret or raw credential fields', () => {
      const spec = compileAIP()
      const secretPatterns = [
        'secret',
        'password',
        'api_key',
        'apikey',
        'credential',
        'private_key',
      ]

      for (const [name, schema] of Object.entries(spec.components.schemas)) {
        if (!name.startsWith('Commerce')) continue
        const props = schema.properties || {}
        for (const fieldName of Object.keys(props)) {
          for (const pattern of secretPatterns) {
            assert.ok(
              !fieldName.toLowerCase().includes(pattern),
              `Field ${name}.${fieldName} looks like a secret`,
            )
          }
        }
      }
    })
  })
})
