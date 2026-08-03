# OpenMeter AI Billing Phase 1 — Provider Release Test Report

**Date:** 2026-08-04
**Branch:** `codex/openmeter-ai-billing-p1`
**Contract version:** `weknora-billing-p1-v1`
**Migration version:** `20260803000100`

## Certification status: PENDING

This report documents what was verified, what was not, and what must happen
before the Phase 1 provider artifact can be written to the Release Manifest.

The acceptance gate (`make weknora-ai-billing-p1-acceptance`) is the single
entry point. It was **not fully executed** in this environment because no live
server stack (PostgreSQL + Kafka + ClickHouse with `ai_usage.enabled=true`)
was available.

### What was verified

| Gate | Result | Notes |
|---|---|---|
| Root `go build ./...` | PASS | |
| Root `go vet ./...` | PASS | |
| SDK `go build ./... && go vet ./... && go test ./...` | PASS | `api/v3/client` module |
| E2E `go build ./... && go vet ./...` | PASS | `e2e` module compiles and vets clean |
| E2E skip behavior (`OPENMETER_ADDRESS=""`) | PASS | Suite skips gracefully |
| `openmeter/aiusage` (non-DB packages) | PASS | 8/9 packages; adapter needs live PG |
| `api/v3/handlers/aiusage` | PASS | HTTP handler integration |
| `git diff --check` | PASS | No whitespace errors |
| `go mod tidy` (root, SDK, e2e) | PASS | No drift |
| `gofmt` | PASS | |

### What was NOT verified

| Gate | Why | Impact |
|---|---|---|
| Live E2E (`TestV3AIUsageClosedLoop` + subtests) | No PostgreSQL/Kafka/ClickHouse stack available in this environment | Contract assertions (balance deduction, watermark values, replay semantics) are unverified against a real server |
| Adapter integration tests | No PostgreSQL connection (`POSTGRES_HOST`) | These tests require live PG; they pass in CI |
| Artifact builder (`build-phase1-artifact.sh`) | No Docker daemon in this environment | Script is verified for bash syntax and shellcheck; not executed |

### No-op DI wiring (important caveat)

The current DI wiring (`app/common/aiusage.go`) uses **placeholder no-op
implementations** for the settlement engine, rate-card resolver, and cost
resolver:

- `noopSettlementEngine` — settles batches without recording grant deductions
  or ledger entries.
- `noopRateCardResolver` — returns a fixed 1-credit-per-unit rate for all
  resources.
- `noopCostResolver` — returns zero cost for all resources.

The **real** settlement engine exists in `openmeter/aiusage/settlement/service.go`
and the real pricing service in `openmeter/aiusage/pricing/`, but they are not
wired into the DI graph yet.

The E2E tests assert the **Phase 1 contract** (what the spec requires), not
the current no-op behavior. Specifically:

- The balance-deduction assertion (`settled == "991"` after a 9-credit charge)
  will fail against today's no-op settlement engine because no grants are
  actually burned.
- The BYOK-zero-credit assertion will pass by coincidence (the no-op cost
  resolver also returns zero), but not for the right reason.
- The watermark convergence assertions depend on the adapter implementation,
  which is real (ent-backed) and should pass.

Until the real settlement/pricing engines replace the no-ops, the live E2E
**cannot** pass. This is expected for a Phase 1 incremental release: the
no-ops let the process start successfully while the full DI graph is assembled.

### What must happen before certification

1. **Wire the real settlement engine** into the DI graph, replacing
   `noopSettlementEngine` with the `settlement.Engine` backed by the
   `ledger.Ledger` collector and `GrantBalanceReader`.
2. **Wire the real pricing/cost resolvers** from `openmeter/aiusage/pricing`
   and `openmeter/llmcost`.
3. **Deploy the full stack** — PostgreSQL, Kafka, ClickHouse — with
   `ai_usage.enabled=true` and a valid Ed25519 signing key.
4. **Run the acceptance gate** against that stack:

   ```bash
   make weknora-ai-billing-p1-acceptance
   ```

5. **Run the artifact builder** to produce the immutable OpenAPI checksum,
   SBOM, and image digest:

   ```bash
   tools/weknora/build-phase1-artifact.sh
   ```

6. **Verify no CRITICAL vulnerabilities** in the grype scan.

Only after all of the above pass can the commit SHA and image digest be
written to the Release Manifest.

## E2E scenario coverage

`TestV3AIUsageClosedLoop` covers all brief-mandated scenarios as subtests:

1. **Happy-path closed loop** — fund WKC, submit batch, assert 201 + TotalCredits=9
2. **Balance deduction** — assert settled balance = 991 after 9-credit charge
3. **Batch retrieval** — GET batch by ID after settlement
4. **Component vs bundle mutual exclusion** — component rates per-line; bundle charges ceiling flat
5. **BYOK model lines at zero** — provider_managed=false lines have zero credits, platform RAG charged
6. **Watermark convergence (seq 1,3,2)** — out-of-order arrival catches up to covered_seq=3
7. **Idempotent replay** — same key+hash returns 200, single ledger effect; different hash returns 409
8. **Linked correction** — credit adjustment reverses the batch charge (Phase 1 uses the existing Credit Adjustments API; the batch ID in the adjustment name/description forms the foreign-key link. A domain-native correction endpoint is deferred to Phase 2.)
9. **Enterprise receivable overflow** — prepaid exhausted, receivable covers remainder
10. **Runtime authorization contract** — returns frozen contract version string

Infrastructure-gated scenarios (separate tests, opt-in env vars):
- **Server restart** (`TestV3AIUsageServerRestart`) — batch survives server restart
- **Kafka/ClickHouse interruption** (`TestV3AIUsageProjectionConvergence`) — batch accepted synchronously; projection converges after recovery

## Artifact builder

The artifact builder (`tools/weknora/build-phase1-artifact.sh`) produces
immutable artifacts in `build/phase1-artifact/`:

| Artifact | Description |
|---|---|
| `openapi.json` | v3 OpenAPI spec (extracted from image `/contract/openapi.json`) |
| `manifest.json` | Contract version, OpenAPI + SDK checksums, migration version, upstream commit, image digest |
| `sbom.json` | CycloneDX SBOM (generated by syft) |
| `grype.json` | Vulnerability scan results |

The script enforces:
- Clean git working tree (no uncommitted changes)
- `openapi.json` read only from the image (no source-tree fallback)
- SDK checksum over all `.go` files in `api/v3/client/`
- SBOM generation (syft is required)
- Vulnerability scan (grype is required; CRITICAL vulns block)
- Backup-restore smoke for PostgreSQL and ClickHouse (when connection URLs are set)

Results are appended to this report under "Artifact build record".

## Commands

```bash
# Full acceptance gate
make weknora-ai-billing-p1-acceptance

# Individual checks
go build ./... && go vet ./...
cd api/v3/client && go build ./... && go vet ./... && go test ./...
cd e2e && go vet ./... && go test -run '^TestV3AIUsage' ./

# AI Usage integration tests (requires PostgreSQL)
POSTGRES_HOST=127.0.0.1 go test -tags=dynamic -count=1 ./openmeter/aiusage/... ./api/v3/handlers/aiusage/...

# Live E2E (requires running server)
TZ=UTC OPENMETER_ADDRESS=http://localhost:8888 go test -C e2e -count=1 -v -run '^TestV3AIUsage' ./

# Artifact builder
tools/weknora/build-phase1-artifact.sh
```
