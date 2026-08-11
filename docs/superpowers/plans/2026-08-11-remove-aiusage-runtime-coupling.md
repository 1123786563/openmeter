# Remove OpenMeter AIUsage Runtime Authorization Coupling Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the remaining OpenMeter refund-domain coupling to the retired AIUsage Runtime Authorization Snapshot flow while preserving generic Credit Reservation v2, commerce refund accounting, and the existing AIUsage schema retirement migration.

**Architecture:** Refund completion remains a local commerce lifecycle: after verified provider success, reverse the fenced Credit, transition the refund to fulfilled, and release the reservation fence. It no longer publishes or waits for a WeKnora Runtime Authorization Snapshot. The Ent schema and generated database layer will express only durable refund facts still owned by commerce; a forward migration removes any legacy snapshot column from existing databases.

**Tech Stack:** Go 1.25.6, Ent code generation, PostgreSQL golang-migrate/Atlas SQL migrations, OpenMeter v3 Go API and Credit Reservation v2.

## Global Constraints

- Follow `docs/superpowers/specs/2026-08-09-openmeter-authoritative-credit-flow-design.md`, especially sections 2, 4, 9, 13, and 14.
- Remove retired Runtime Authorization Snapshot, signing, and old AIUsage coupling; do not add a compatibility API or replacement snapshot route.
- Preserve OpenMeter native Ledger, Credit Grant, Purchase, Customer, payment, Charge, and Credit Reservation v2 behavior.
- Preserve the generic `feature_key="ai_usage"` routing value used by Credit Reservation v2 pricing and tests.
- Historical AIUsage migrations remain byte-for-byte unchanged; schema cleanup is forward-only.
- Do not modify the WeKnora repository in this task.
- Ent generated files must be regenerated from `openmeter/ent/schema`; do not hand-edit generated database code.
- New behavior changes follow red-green-refactor: add a failing regression test, observe the expected failure, then implement the minimal change.

---

### Task 1: Add a failing regression gate for the retired snapshot surface

**Files:**
- Create: `openmeter/commerce/refund/legacy_aiusage_removed_test.go`

**Interfaces:**
- Consumes: `refund.Config` and `refund.RefundRequest`.
- Produces: permanent tests proving the refund domain no longer exposes the retired `Snapshots` or `SnapshotVersion` fields.

- [ ] **Step 1: Write the failing test**

Create this test:

```go
package refund

import (
    "reflect"
    "testing"

    "github.com/stretchr/testify/require"
)

func TestRetiredAIUsageSnapshotSurfaceIsRemoved(t *testing.T) {
    _, hasSnapshots := reflect.TypeOf(Config{}).FieldByName("Snapshots")
    require.False(t, hasSnapshots, "refund Config must not expose Runtime Authorization snapshots")

    _, hasSnapshotVersion := reflect.TypeOf(RefundRequest{}).FieldByName("SnapshotVersion")
    require.False(t, hasSnapshotVersion, "refund requests must not persist Runtime Authorization snapshot versions")
}
```

- [ ] **Step 2: Run the test to verify RED**

Run: `go test ./openmeter/commerce/refund -run '^TestRetiredAIUsageSnapshotSurfaceIsRemoved$' -count=1`

Expected: FAIL because the current `Config` and `RefundRequest` still expose `Snapshots` and `SnapshotVersion`.

- [ ] **Step 3: Commit the failing test**

Do not commit a test that passes before the production removal. The implementer will keep this test unchanged while removing the retired surface in Task 2.

---

### Task 2: Remove Runtime Authorization Snapshot publication and confirmation

**Files:**
- Modify: `openmeter/commerce/refund/service.go`
- Modify: `openmeter/commerce/refund/openmeter_fence.go`
- Modify: `openmeter/commerce/refund/service_test.go`
- Modify: `openmeter/commerce/ent_adapter_commerce.go`
- Modify: `openmeter/ent/schema/refund_request.go`
- Regenerate: `openmeter/ent/db/**`

**Interfaces:**
- Consumes: existing refund provider, Credit Reverser, and durable reservation fence interfaces.
- Produces: refund completion that does not depend on Runtime Authorization snapshots.

- [ ] **Step 1: Remove the retired domain fields and interfaces**

In `service.go`:
- Remove `RefundRequest.SnapshotVersion`.
- Remove `Repository.SetSnapshot`.
- Remove `FenceClient.ConfirmSnapshotApplied`.
- Remove `SnapshotPublisher`, `PublishSnapshotInput`, `Config.Snapshots`, and the service's `snapshots` field.
- Remove the snapshot publication and confirmation block from `processLedgerReversing`.
- Keep the existing order: reverse Credit, transition to `RefundStatusFulfilled`, then release the fence.
- Update comments so the lifecycle describes Credit reversal and fence release without Runtime Authorization.

In `openmeter_fence.go`, remove `ConfirmSnapshotApplied` and keep the compile-time `FenceClient` assertion.

In `service_test.go`, delete the snapshot publisher mock, harness field, repository setter, fence confirmation mock method, and `Snapshots` config entries. Keep refund success, provider failure, unknown-result, and fence-release assertions.

- [ ] **Step 2: Run the regression test and focused refund tests**

Run:

```bash
go test ./openmeter/commerce/refund -run '^TestRetiredAIUsageSnapshotSurfaceIsRemoved$' -count=1
go test ./openmeter/commerce/refund -count=1
```

Expected: both commands PASS, and the refund tests prove provider success still reverses Credit, fulfills the refund, and releases the fence.

- [ ] **Step 3: Remove the field from the Ent schema**

In `openmeter/ent/schema/refund_request.go`, remove the `snapshot_version` field and its obsolete comment. Do not change unrelated refund fields or status transitions.

- [ ] **Step 4: Regenerate Ent output**

Run: `go generate ./openmeter/ent/...`

Expected: generated files remove only the `snapshot_version` field, constants, predicates, mutation helpers, builders, and runtime descriptors that derive from the schema change.

- [ ] **Step 5: Re-run the refund package and schema compilation**

Run:

```bash
go test ./openmeter/commerce/refund ./openmeter/commerce -count=1
go test ./openmeter/ent/... -count=1
```

Expected: PASS with no generated-code references to `SnapshotVersion`, `SetSnapshot`, or `ConfirmSnapshotApplied`.

---

### Task 3: Remove the retired snapshot column forward and test rollback safety

**Files:**
- Create: `tools/migrate/migrations/20260811000200_drop_refund_snapshot_version.up.sql`
- Create: `tools/migrate/migrations/20260811000200_drop_refund_snapshot_version.down.sql`
- Create: `tools/migrate/refund_snapshot_version_migration_test.go`
- Modify: `tools/migrate/migrations/atlas.sum`

**Interfaces:**
- Consumes: the `refund_requests` table created by the commerce migration.
- Produces: forward-only cleanup of a potentially existing `snapshot_version` column without touching native Ledger or Reservation tables.

- [ ] **Step 1: Write the failing migration test**

Create a PostgreSQL migration test using the existing `runner`, `stops`, `directionUp`, and `directionDown` helpers. At version `20260811000100` on the upward path, add the legacy column explicitly:

```go
func TestDropRefundSnapshotVersionMigration(t *testing.T) {
    runner{
        stops: stops{
            {
                version:   20260811000100,
                direction: directionUp,
                action: func(t *testing.T, db *sql.DB) {
                    _, err := db.Exec(`ALTER TABLE refund_requests ADD COLUMN snapshot_version character varying`)
                    require.NoError(t, err)
                },
            },
            {
                version:   20260811000200,
                direction: directionUp,
                action: func(t *testing.T, db *sql.DB) {
                    var count int
                    err := db.QueryRow(`
                        SELECT count(*)
                        FROM information_schema.columns
                        WHERE table_schema = current_schema()
                          AND table_name = 'refund_requests'
                          AND column_name = 'snapshot_version'
                    `).Scan(&count)
                    require.NoError(t, err)
                    require.Zero(t, count)
                },
            },
            {
                version:   20260811000200,
                direction: directionDown,
                action: func(t *testing.T, db *sql.DB) {
                    var count int
                    err := db.QueryRow(`
                        SELECT count(*)
                        FROM information_schema.columns
                        WHERE table_schema = current_schema()
                          AND table_name = 'refund_requests'
                          AND column_name = 'snapshot_version'
                    `).Scan(&count)
                    require.NoError(t, err)
                    require.Equal(t, 1, count)
                },
            },
        },
    }.Test(t)
}
```

- [ ] **Step 2: Run the migration test to verify RED**

Run: `go test ./tools/migrate -run '^TestDropRefundSnapshotVersionMigration$' -count=1`

Expected: FAIL because migration `20260811000200` does not yet exist and the manually added column remains.

- [ ] **Step 3: Add the forward and reverse SQL**

`20260811000200_drop_refund_snapshot_version.up.sql`:

```sql
-- Runtime Authorization snapshot versions are retired with the AIUsage flow.
ALTER TABLE "refund_requests" DROP COLUMN IF EXISTS "snapshot_version";
```

`20260811000200_drop_refund_snapshot_version.down.sql`:

```sql
-- Restore the removed legacy column only when rolling back the cleanup migration.
ALTER TABLE "refund_requests"
  ADD COLUMN IF NOT EXISTS "snapshot_version" character varying;
```

- [ ] **Step 4: Refresh the migration checksum**

Run: `atlas migrate hash --dir file://tools/migrate/migrations`

Expected: `tools/migrate/migrations/atlas.sum` includes the new migration and all historical migration hashes remain unchanged.

- [ ] **Step 5: Run the migration regression test to verify GREEN**

Run: `go test ./tools/migrate -run '^TestDropRefundSnapshotVersionMigration$' -count=1`

Expected: PASS when PostgreSQL is available; if PostgreSQL is unavailable, report the infrastructure skip/failure separately rather than treating it as a code pass.

---

### Task 4: Regenerate API outputs, clean stale ignored output, and verify the removal

**Files:**
- Regenerate: `api/spec/packages/aip/output/definitions/metering-and-billing/v3/**`
- Regenerate and inspect: `api/v3/openapi.yaml`, `api/v3/api.gen.go`, `api/v3/client/**`, and `api/spec/packages/aip-client-javascript/**`
- Modify only if a focused gate identifies a source residue; do not modify WeKnora.

**Interfaces:**
- Consumes: the already-removed AIUsage TypeSpec source and the cleaned refund domain.
- Produces: reproducible generated output without retired AIUsage paths, models, or stale local bundled OpenAPI files.

- [ ] **Step 1: Regenerate the source-derived API artifacts**

Run: `make update-openapi`

Expected: exit 0, no AIUsage source is reintroduced, Reservation v2 remains present, and generated files remain consistent with TypeSpec.

- [ ] **Step 2: Verify the stale ignored bundle is gone**

Run:

```bash
rg -n "ai-usage-batches|runtime-authorization|AI Usage|AIUsage"   api/spec/packages/aip/output/definitions/metering-and-billing/v3   api/v3/openapi.yaml api/v3/api.gen.go api/v3/client   api/spec/packages/aip-client-javascript/src
```

Expected: no output. Generic `feature_key: ai_usage` matches in Credit Reservation v2 tests remain allowed.

- [ ] **Step 3: Verify no active runtime residue remains**

Run:

```bash
rg -n "SnapshotPublisher|PublishSnapshot|ConfirmSnapshotApplied|SnapshotVersion|snapshot_version|Runtime Authorization|runtime_authorization|AIUsage|AiUsage|ai-usage-batches"   openmeter api app cmd test e2e   --glob '!**/ent/db/runtime.go'   --glob '!docs/**'
```

Expected: no active AIUsage/runtime-authorization business references. Generic invoice/quantity snapshot terminology and the TypeSpec generator's isolated test model are outside this business-residue scan and may remain where semantically unrelated.

- [ ] **Step 4: Run focused OpenMeter gates**

Run:

```bash
go test ./api/v3 ./api/v3/server ./openmeter/commerce/... ./openmeter/creditreservation/... -count=1
go test -C api/v3/client ./... -count=1
go vet ./...
```

Expected: exit 0 for all commands.

- [ ] **Step 5: Run migration and quality gates**

Run in order:

```bash
make migrate-check
make lint-go
make fmt
git diff --check
```

Expected: migration checks, Go lint, formatting, and whitespace validation pass. If a gate requires unavailable PostgreSQL, Node, or another external service, report the exact blocker and the successful focused evidence separately.

- [ ] **Step 6: Review final scope**

Run:

```bash
git status --short --branch
git diff --stat
git diff --name-only
```

Expected: only OpenMeter files are changed; no WeKnora files, unrelated billing behavior, native Ledger tables, or generic Credit Reservation v2 routes are removed.
