# Remove AIUsage API Contract and SDKs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the retired AIUsage API surface from OpenMeter TypeSpec, OpenAPI, generated Go/JavaScript SDKs, and the remaining HTTP compatibility stubs.

**Architecture:** TypeSpec remains the only API source of truth. Delete the complete `AIUsage` namespace and its OpenMeter service bindings, regenerate all v3 contract artifacts, then delete hand-written 404 compatibility methods that are no longer required by the generated server interface. Preserve Reservation v2 and every generic use of the `ai_usage` feature key.

**Tech Stack:** TypeSpec 1.14, custom OpenMeter TypeScript/Go emitters, OpenAPI 3, oapi-codegen v2, Go 1.26, Vitest/pnpm.

## Global Constraints

- Follow `docs/superpowers/specs/2026-08-09-openmeter-authoritative-credit-flow-design.md`, especially sections 2, 4, 9, 13, and 14.
- Delete the entire retired API group: usage batches, runtime authorization, AI-specific credit balance, and AI-specific credit transactions.
- Do not retain a compatibility API or 404 route stubs.
- Preserve generic Credit Reservation v2 APIs and the `feature=ai_usage` routing value.
- Preserve historical database migrations byte-for-byte and retire their schema only through a newer forward migration.
- Do not modify the WeKnora repository in this task.
- Generated OpenAPI and SDK files must be produced by the repository generators, not edited manually.

---

### Task 1: Add an executable contract-removal regression gate

**Files:**
- Create: `api/v3/aiusage_removed_test.go`

**Interfaces:**
- Consumes: `api.GetSpec() (*openapi3.T, error)` from generated `api/v3/api.gen.go`.
- Produces: a permanent test proving retired paths, schemas, operations, and tags cannot re-enter the v3 OpenAPI document.

- [ ] **Step 1: Write the failing test**

```go
package v3_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
)

func TestAIUsageContractIsRemoved(t *testing.T) {
	spec, err := api.GetSpec()
	require.NoError(t, err)

	retiredPaths := []string{
		"/ai-usage-batches",
		"/ai-usage-batches/{batchId}",
		"/customers/{customerId}/runtime-authorization",
		"/customers/{customerId}/credit-balance",
		"/customers/{customerId}/credit-transactions",
	}
	for _, path := range retiredPaths {
		require.Nilf(t, spec.Paths.Find(path), "retired AIUsage path %s must not be published", path)
	}

	for name := range spec.Components.Schemas {
		require.Falsef(t, strings.HasPrefix(name, "AIUsage"), "retired AIUsage schema %s must not be published", name)
	}

	for _, tag := range spec.Tags {
		require.NotEqual(t, "AI Usage", tag.Name)
	}
}
```

- [ ] **Step 2: Run the test to verify RED**

Run: `go test ./api/v3 -run '^TestAIUsageContractIsRemoved$' -count=1`

Expected: FAIL because `/ai-usage-batches` and the `AI Usage` schemas/tag still exist in the generated OpenAPI document.

- [ ] **Step 3: Keep the failing test unchanged for the source-first deletion**

The production change that makes this test pass is deletion of the TypeSpec service bindings and namespace in Task 2. Do not weaken the assertions or special-case generated output.

---

### Task 2: Delete the AIUsage TypeSpec source and regenerate all contract artifacts

**Files:**
- Delete: `api/spec/packages/aip/src/aiusage/index.tsp`
- Delete: `api/spec/packages/aip/src/aiusage/batch.tsp`
- Delete: `api/spec/packages/aip/src/aiusage/pricing.tsp`
- Delete: `api/spec/packages/aip/src/aiusage/runtime_authorization.tsp`
- Delete: `api/spec/packages/aip/src/aiusage/operations.tsp`
- Delete: `api/spec/packages/aip/test/weknora-phase1-contract.test.js`
- Delete: `api/spec/packages/aip/testdata/weknora/phase1/runtime-authorization.json`
- Delete: `api/spec/packages/aip/testdata/weknora/phase1/usage-batch.json`
- Modify: `api/spec/packages/aip/test/weknora-phase2-contract.test.js`
- Modify: `api/spec/packages/aip/src/openmeter.tsp`
- Modify: `api/spec/packages/aip/src/shared/consts.tsp`
- Regenerate: `api/spec/packages/aip/output/definitions/metering-and-billing/v3/*.yaml`
- Regenerate: `api/v3/openapi.yaml`
- Regenerate: `api/v3/api.gen.go`
- Regenerate: `api/v3/client/**`
- Regenerate: `api/spec/packages/aip-client-javascript/**`

**Interfaces:**
- Consumes: the approved contract boundary in the authoritative-credit-flow design.
- Produces: a v3 OpenAPI service and generated SDKs with no AIUsage endpoints or AIUsage-specific models/services.

- [ ] **Step 1: Remove AIUsage from the TypeSpec service graph**

In `api/spec/packages/aip/src/openmeter.tsp` delete:

```typespec
import "./aiusage/index.tsp";
```

Delete the `@tagMetadata(Shared.AIUsageTag, ...)` declaration and these four service interfaces:

```typespec
AIUsageBatchEndpoints
CustomerRuntimeAuthorizationEndpoints
AIUsageCustomerCreditBalanceEndpoints
AIUsageCustomerCreditTransactionEndpoints
```

- [ ] **Step 2: Remove AIUsage-only shared constants**

From `api/spec/packages/aip/src/shared/consts.tsp`, delete only:

```typespec
AIUsageTag
AIUsageDescription
AIUsageContractVersion
```

Do not remove the generic `ai_usage` feature value from Reservation examples, quickstart configuration, or tests.

- [ ] **Step 3: Delete the AIUsage TypeSpec namespace directory**

Delete the five tracked files under `api/spec/packages/aip/src/aiusage/`. Do not manually edit any emitter output or SDK file.

- [ ] **Step 4: Regenerate TypeSpec, OpenAPI, Go SDK, JavaScript SDK, and Go server bindings**

Run: `make update-openapi`

Expected: exit 0. The TypeSpec emitters remove AIUsage files and exports from `api/v3/client` and `api/spec/packages/aip-client-javascript`; `go generate ./api/...` removes AIUsage paths, models, and server methods from `api/v3/api.gen.go`.

- [ ] **Step 5: Run the regression test to verify GREEN**

Run: `go test ./api/v3 -run '^TestAIUsageContractIsRemoved$' -count=1`

Expected: PASS.

- [ ] **Step 6: Inspect generated changes before proceeding**

Run:

```bash
git status --short
git diff --stat
git diff -- api/spec/packages/aip/src api/v3/openapi.yaml api/v3/client api/spec/packages/aip-client-javascript
```

Expected: changes are limited to AIUsage source removal and deterministic generated fallout. No Reservation v2 endpoint or generic `ai_usage` feature value is removed.

---

### Task 3: Remove the hand-written HTTP compatibility layer and stale build/config references

**Files:**
- Modify: `api/v3/server/server.go`
- Modify: `api/v3/server/creditreservations.go`
- Modify: `api/v3/server/creditreservations_test.go`
- Modify: `config.example.yaml`
- Modify: `Makefile`

**Interfaces:**
- Consumes: the regenerated `api.ServerInterface`, which no longer declares AIUsage operations.
- Produces: a server with no AIUsage route implementation or configuration/build affordance.

- [ ] **Step 1: Delete the five AIUsage 404 methods**

From `api/v3/server/server.go`, delete the `AI Usage (removed subsystem — returns Not Found)` block containing:

```go
CreateAiUsageBatch
GetAiUsageBatch
GetAiUsageCreditBalance
ListAiUsageCreditTransactions
GetCustomerRuntimeAuthorization
```

Remove the `apierrors` import only if no other server code still uses it; let `goimports`/`gofmt` determine the final import set.

- [ ] **Step 2: Wire the generated Reservation v2 server interface**

Wire the existing Credit Reservation handler through the newly generated eight-method `api.ServerInterface` surface, remove `registerCreditReservationRoutes`, and preserve HTTP 404 when the Credit Reservation feature is disabled. This is the required adapter for keeping the authoritative Reservation v2 API after regenerating the server binding; do not add new reservation business logic.

- [ ] **Step 3: Delete the stale example configuration**

Remove the commented `ai_usage:` configuration block and its description from `config.example.yaml`. Preserve `quickstart/config.yaml` values such as `defaultFeatureKey: "ai_usage"`, which belong to Reservation v2 pricing.

- [ ] **Step 4: Delete the stale AIUsage test target**

Remove the Makefile target/commands that run `./openmeter/aiusage/...`, `./api/v3/handlers/aiusage/...`, or `TestV3AIUsage`. Do not change unrelated generation, lint, or Reservation v2 targets.

- [ ] **Step 5: Format hand-written Go changes**

Run: `gofmt -w api/v3/aiusage_removed_test.go api/v3/server/server.go`

- [ ] **Step 6: Compile the affected server and SDK packages**

Run:

```bash
go test ./api/v3 ./api/v3/server -count=1
go test -C api/v3/client ./... -count=1
```

Expected: PASS with no missing generated interface methods or stale AIUsage types.

---

### Task 4: Verify source-of-truth, SDK, and runtime removal

**Files:**
- Verify only; do not add new files unless a failing gate identifies a scoped defect.

**Interfaces:**
- Consumes: regenerated contract artifacts and hand-written server cleanup from Tasks 1-3.
- Produces: evidence that the retired API cannot be generated, imported, documented, or routed.

- [ ] **Step 1: Run TypeSpec and generated SDK tests**

Run: `make -C api/spec test`

Expected: TypeSpec emitter tests, JavaScript SDK tests/coverage, and TypeScript checks all pass.

- [ ] **Step 2: Run API lint**

Run: `make lint-api-spec`

Expected: exit 0 with no OpenAPI or generated SDK lint errors.

- [ ] **Step 3: Run focused Go tests**

Run:

```bash
go test ./api/v3/... ./openmeter/creditreservation/... -count=1
go test -C api/v3/client ./... -count=1
```

Expected: PASS. Reservation v2 remains compilable and its tests still use `feature_key="ai_usage"` where appropriate.

### Task 5: Preserve migration history and remove the legacy schema forward

**Files:**
- Restore: `tools/migrate/migrations/20260803000100_ai_billing_core.{up,down}.sql`
- Restore: `tools/migrate/migrations/20260809000100_ai_usage_ratecard_units.{up,down}.sql`
- Restore: `tools/migrate/migrations/20260809000200_aiusage_ratecard_price_nullable.{up,down}.sql`
- Restore: `tools/migrate/migrations/20260809000300_aiusage_ratecard_price_default.{up,down}.sql`
- Create: `tools/migrate/migrations/20260811000100_drop_legacy_ai_usage.{up,down}.sql`
- Regenerate: `tools/migrate/migrations/atlas.sum`

- [ ] **Step 1: Restore the original migrations from the pre-removal revision**

Expected: existing databases retain their migration history and clean databases can still replay every historical version.

- [ ] **Step 2: Add the forward drop migration**

Drop foreign-key children before parent tables and drop the two legacy enum types last. Do not touch native Ledger, Credit Grant, Purchase, Customer, payment, Reservation, or Charge tables.

- [ ] **Step 3: Validate migration hashes and a disposable database replay**

Run: `atlas migrate hash --dir file://tools/migrate/migrations` and `make migrate-check-validate`; run the repository migration test path against disposable PostgreSQL when available.

Expected: checksum validation and clean replay pass, ending without legacy AIUsage tables.

- [ ] **Step 4: Prove generated artifacts are reproducible**

Run:

```bash
git diff --exit-code -- api/spec/packages/aip/output api/v3/openapi.yaml api/v3/api.gen.go api/v3/client api/spec/packages/aip-client-javascript
```

Run this only after recording the intended diff, regenerating once more with `make update-openapi`, and comparing against that recorded state. The second generation must add no new diff.

- [ ] **Step 5: Scan for active AIUsage contract residue**

Run:

```bash
rg -n "AIUsage|AiUsage|aiUsage|ai-usage-batches|runtime-authorization|AI Usage" \
  api/spec/packages/aip/src \
  api/spec/packages/aip-client-javascript \
  api/v3 \
  app config.example.yaml Makefile
```

Expected: no active contract, SDK, route, handler, config, or build-target references. Historical design/runbook/test-report documents are outside this scan. Generic `ai_usage` feature-key occurrences under Credit Reservation v2 remain valid.

- [ ] **Step 6: Review repository scope**

Run:

```bash
git status --short
git diff --check
git diff --stat
```

Expected: only OpenMeter files in this plan changed; no WeKnora files, and every restored historical migration matches its pre-removal revision byte-for-byte.
