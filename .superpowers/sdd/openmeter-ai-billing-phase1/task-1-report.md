# Task 1 Report: Freeze the Phase 1 TypeSpec Contract for AI Usage Billing

## Status: DONE

## What was implemented

### TypeSpec contract files (5 new files)

All files live under `api/spec/packages/aip/src/aiusage/`:

1. **`index.tsp`** — barrel import of all sub-modules.
2. **`batch.tsp`** — `BillingMode` enum (`component|bundle`), `BatchStatus` enum (`settled|corrected`), `UsageLineCreate`, `UsageLine`, `UsageBatchCreate`, and `UsageBatch` models. All integer Credit and sequence fields use `int64`; `lines` has `@minItems(1)`.
3. **`pricing.tsp`** — `AIUsageErrorCode` enum (`idempotency_conflict|rate_missing|resource_unknown|credit_insufficient|credit_limit_exceeded|reservation_ceiling_exceeded`), `CostSnapshot`, `SalesSnapshot`, `RatingSnapshot`, `LedgerEntryRef`, and `BatchSettlementResult` models.
4. **`runtime_authorization.tsp`** — `RuntimeAuthorization`, `RuntimeAuthorizationQuery`, plus AI Usage-specific integer-credit models: `AICreditBalance`, `AICreditTransaction`, and `AICreditTransactionType`.
5. **`operations.tsp`** — four operation interfaces:
   - `AIUsageBatchOperations`: `create` (POST, 201/200/409), `get` (GET by batchId)
   - `RuntimeAuthorizationOperations`: `get` (GET runtime authorization)
   - `CreditBalanceOperations`: `get` (GET integer credit balance)
   - `CreditTransactionOperations`: `list` (GET cursor-paginated integer credit transactions)

### Modified files

- **`api/spec/packages/aip/src/openmeter.tsp`** — renamed two interface declarations from `...EndpointsAIUsage` to `AIUsage...Endpoints` so they satisfy the `friendly-name` linter (interface names must end with `Endpoints` or `Operations`).
- **`api/spec/packages/typespec-go/src/go-types.tsx`** — added `['ai', 'AI']` to the Go initialisms map so `goExportedName("AIUsage")` produces `AIUsage` (not `AiUsage`), fixing a service-type naming mismatch between `client.go` (constructor) and `ai_usage.go` (struct definition).

### Test fixtures (2 new files)

- **`testdata/weknora/phase1/usage-batch.json`** — canonical batch with `reservation_ceiling_credits: 40`, `billing_mode: component`, and three resource codes: `chat_input_token`, `rag_retrieval`, `mcp_call`.
- **`testdata/weknora/phase1/runtime-authorization.json`** — authorization response with `available_credits: 1250`, `covered_tenant_seq: 41`.

### Contract test (1 new file)

- **`test/weknora-phase1-contract.test.js`** — compiles the spec via `tsp compile --emit @typespec/openapi3`, parses the OpenAPI YAML, and asserts:
  - All five Phase 1 paths exist with correct HTTP methods
  - AI Usage-specific operation IDs are distinct from OpenMeter Credits
  - `int64` format on `tenant_seq` and `reservation_ceiling_credits`
  - `minItems: 1` on `lines`
  - `BillingMode` enum values (`component`, `bundle`)
  - `BatchStatus` enum values (`settled`, `corrected`)
  - HTTP 201/200/409 response codes documented on batch create
  - Fixture stability (resource codes, ceiling, authorization values)

### Generated SDK output

- Go SDK: `api/v3/client/ai_usage.go` (service + operations), `api/v3/client/models_ai_usage.go` (types)
- TypeScript SDK: `api/spec/packages/aip-client-javascript/src/funcs/aiUsage.ts`, `src/models/operations/aiUsage.ts`, `src/sdk/aiUsage.ts`

## Test results

| Suite | Result |
|-------|--------|
| TypeSpec contract tests (`node --test`) | 14/14 pass |
| TypeScript SDK tests (vitest) | 450/450 pass |
| Go SDK build (`go build ./...`) | OK |
| Go SDK tests (`go test ./...`) | OK |
| Go vet (`go vet ./...`) | OK |
| TS SDK typecheck (`tsc --noEmit`) | OK |

## Key design decisions

1. **Integer-credit credit models**: Instead of reusing the decimal `Customers.CreditBalances`/`CreditTransaction` models (which use `Shared.Numeric`), the AIUsage credit endpoints return distinct integer-credit models (`AICreditBalance`, `AICreditTransaction`). This matches the Go domain's `int64` credit representation and avoids Go/TS SDK type-name collisions that occur when two operations return the same response model type.

2. **`{batchId}` and `{customerId}` path params**: TypeSpec preserves the literal `@route` path segments. The OpenAPI output uses camelCase path params (`{batchId}`, `{customerId}`), consistent with every other route in the codebase.

3. **Resource codes**: The fixture uses the Go implementation's canonical codes (`chat_input_token`, `rag_retrieval`, `mcp_call`), which match the approved Phase 1 spec. The contract allows any string for `resource_code` with a doc listing all 16 valid codes.

4. **Go initialism fix**: Added `ai` to the `goExportedName` initialisms map so `AIUsage` is preserved as `AIUsage` in Go type names, matching the field name in `client.go`.

## Concerns

1. **`openmeter/aiusage/testdata/fuzz/`**: There are untracked fuzz corpus files (`openmeter/aiusage/testdata/fuzz/FuzzCeilCredits/`) from prior Go work. These are unrelated to the TypeSpec contract and were left untouched.

2. **Broad SDK regeneration diff**: The `tsp compile` regeneration touches many existing TS SDK files (formatting/ordering changes from the TypeScript emitter). This is expected — the emitter regenerates all files on each compile. The diff is purely mechanical.
