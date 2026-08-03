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
 * Read and parse a JSON test fixture relative to the testdata/ directory.
 */
function readFixture(relativePath) {
  return JSON.parse(
    readFileSync(path.join(pkgDir, 'testdata', relativePath), 'utf-8'),
  )
}

describe('WeKnora Phase 1 contract', () => {
  describe('routes', () => {
    it('exposes all five Phase 1 paths with correct HTTP methods', () => {
      const spec = compileAIP()
      assert.ok(spec.paths['/ai-usage-batches'].post, 'POST /ai-usage-batches')
      assert.ok(
        spec.paths['/ai-usage-batches/{batchId}'].get,
        'GET /ai-usage-batches/{batchId}',
      )
      assert.ok(
        spec.paths['/customers/{customerId}/runtime-authorization'].get,
        'GET runtime-authorization',
      )
      assert.ok(
        spec.paths['/customers/{customerId}/credit-balance'].get,
        'GET credit-balance',
      )
      assert.ok(
        spec.paths['/customers/{customerId}/credit-transactions'].get,
        'GET credit-transactions',
      )
    })

    it('uses AI Usage-specific operation IDs distinct from OpenMeter Credits', () => {
      const spec = compileAIP()
      assert.equal(
        spec.paths['/ai-usage-batches'].post.operationId,
        'create-ai-usage-batch',
      )
      assert.equal(
        spec.paths['/ai-usage-batches/{batchId}'].get.operationId,
        'get-ai-usage-batch',
      )
      assert.equal(
        spec.paths['/customers/{customerId}/runtime-authorization'].get
          .operationId,
        'get-customer-runtime-authorization',
      )
      assert.equal(
        spec.paths['/customers/{customerId}/credit-balance'].get.operationId,
        'get-ai-usage-credit-balance',
      )
      assert.equal(
        spec.paths['/customers/{customerId}/credit-transactions'].get
          .operationId,
        'list-ai-usage-credit-transactions',
      )
    })
  })

  describe('batch schema', () => {
    it('uses int64 for integer Credit and sequence fields', () => {
      const spec = compileAIP()
      const schema = spec.components.schemas.AIUsageUsageBatchCreate
      const props = schema.properties

      assert.equal(props.tenant_seq.format, 'int64')
      assert.equal(props.reservation_ceiling_credits.format, 'int64')
    })

    it('requires at least one line item', () => {
      const spec = compileAIP()
      const schema = spec.components.schemas.AIUsageUsageBatchCreate
      assert.equal(schema.properties.lines.minItems, 1)
    })

    it('exposes component and bundle billing modes', () => {
      const spec = compileAIP()
      assert.deepEqual(spec.components.schemas.AIUsageBillingMode.enum, [
        'component',
        'bundle',
      ])
    })

    it('exposes settled and corrected batch states', () => {
      const spec = compileAIP()
      assert.deepEqual(spec.components.schemas.AIUsageBatchStatus.enum, [
        'settled',
        'corrected',
      ])
    })

    it('documents idempotent replay as HTTP 200 and conflict as 409', () => {
      const spec = compileAIP()
      const responses = spec.paths['/ai-usage-batches'].post.responses
      assert.ok(responses['201'], 'first submit returns 201')
      assert.ok(responses['200'], 'identical replay returns 200')
      assert.ok(responses['409'], 'hash conflict returns 409')
    })
  })

  describe('fixtures', () => {
    it('keeps the usage-batch fixture stable', () => {
      const batch = readFixture('weknora/phase1/usage-batch.json')
      assert.equal(batch.reservation_ceiling_credits, 40)
      assert.deepEqual(
        batch.lines.map((line) => line.resource_code),
        ['chat_input_token', 'rag_retrieval', 'mcp_call'],
      )
      assert.equal(batch.billing_mode, 'component')
      assert.equal(batch.tenant_seq, 42)
    })

    it('keeps the runtime-authorization fixture stable', () => {
      const authz = readFixture('weknora/phase1/runtime-authorization.json')
      assert.equal(authz.authorized, true)
      assert.equal(authz.available_credits, 1250)
      assert.equal(authz.covered_tenant_seq, 41)
    })
  })
})
