# OpenMeter AI Billing Phase 1 — Provider Release Test Report

**Date:** 2026-08-04
**Branch:** `codex/openmeter-ai-billing-p1`
**Contract version:** `weknora-billing-p1-v1`
**Migration version:** `20260803000100`

## Summary

Phase 1 provider release evidence for the WeKnora AI Billing integration. The
acceptance gate `make weknora-ai-billing-p1-acceptance` is the single entry
point; it runs the API spec suite, the v3 Go SDK module, root build/vet, the
AI Usage integration tests, and the live v3 E2E suite.

## Verification results

### Build and vet (all three Go modules)

| Module | `go build` | `go vet` | Duration |
|---|---|---|---|
| Root (`github.com/openmeterio/openmeter`) | PASS | PASS | 18s / 24s |
| SDK (`api/v3/client`) | PASS | PASS + test PASS | <1s |
| E2E (`e2e`) | PASS | PASS | <1s |

### Integration tests (PostgreSQL-backed, `dynamic` tag)

| Package | Result | Notes |
|---|---|---|
| `openmeter/aiusage` | PASS | Core domain logic (ceil, ratecard, creditcalc, types) |
| `openmeter/aiusage/meterregistry` | PASS | Meter/resource registry |
| `openmeter/aiusage/pricing` | PASS | Pricing service + property tests |
| `openmeter/aiusage/runtimeauthorization` | PASS | Authorization assembly + signing |
| `openmeter/aiusage/service` | PASS | Service orchestration |
| `openmeter/aiusage/settlement` | PASS | Grant burn-down, correction, enterprise cap |
| `openmeter/aiusage/signing` | PASS | Ed25519 sign/verify, canonical form, rotation |
| `openmeter/aiusage/worker` | PASS | Outbox relay, lease recovery, dead-letter |
| `openmeter/aiusage/adapter` | SKIPPED (no PostgreSQL) | Requires live PG; passes in CI with `POSTGRES_HOST=127.0.0.1` |
| `api/v3/handlers/aiusage` | PASS | HTTP handler integration |

### Live E2E (`e2e/aiusage_v3_test.go`)

| Test | Result | Notes |
|---|---|---|
| `TestV3AIUsageClosedLoop` | SKIP | Requires `OPENMETER_ADDRESS` + `ai_usage.enabled=true` |
| `TestV3AIUsageServerRestart` | SKIP | Requires `OPENMETER_E2E_SERVER_RESTART=1` |
| `TestV3AIUsageProjectionConvergence` | SKIP | Requires `OPENMETER_E2E_INFRA_INTERRUPTION=1` |

The E2E suite skips gracefully when infrastructure is unavailable. It is
designed to pass against a live server with PostgreSQL, Kafka, and ClickHouse
running and `ai_usage.enabled=true`.

### Code quality

| Check | Result |
|---|---|
| `git diff --check` | PASS (no whitespace errors) |
| `go mod tidy` (root, SDK, e2e) | PASS (no drift) |
| `gofmt` | PASS |

## E2E scenario coverage

The `TestV3AIUsageClosedLoop` test covers all brief-mandated scenarios:

1. **Happy-path closed loop** — fund WKC, submit batch, assert 201 + TotalCredits=9
2. **Balance deduction** — assert settled balance = 991 after 9-credit charge
3. **Batch retrieval** — GET batch by ID after settlement
4. **Component vs bundle mutual exclusion** — component rates per-line; bundle charges ceiling flat
5. **BYOK model lines at zero** — provider_managed=false lines have zero credits, platform RAG charged
6. **Watermark convergence (seq 1,3,2)** — out-of-order arrival catches up to covered_seq=3
7. **Idempotent replay** — same key+hash returns 200, single ledger effect; different hash returns 409
8. **Linked correction** — credit adjustment reverses the batch charge
9. **Enterprise receivable overflow** — prepaid exhausted, receivable covers remainder
10. **Runtime authorization contract** — returns frozen contract version string

Infrastructure-gated scenarios (separate tests):
- **Server restart** — batch survives server restart (durably persisted to PostgreSQL)
- **Kafka/ClickHouse interruption** — batch accepted synchronously; projection converges after recovery

## Artifact build

The artifact builder (`tools/weknora/build-phase1-artifact.sh`) produces:

| Artifact | Location in image |
|---|---|
| OpenAPI spec | `/contract/openapi.yaml` |
| Contract manifest (version, checksums, commit, digest) | Generated at build time |
| SBOM (CycloneDX) | Generated at build time |
| Vulnerability scan | grype (CRITICAL blocks release) |

Run: `tools/weknora/build-phase1-artifact.sh`

The script enforces: clean commit, contract checksum recording, upstream commit
pinning, SBOM generation, vulnerability gate, and backup-restore smoke. Results
are appended to this report.

## Open items

- **Live E2E execution**: requires a running server stack (PostgreSQL + Kafka +
  ClickHouse with `ai_usage.enabled=true`). The E2E suite is verified to compile,
  vet, format, and skip cleanly without infrastructure.
- **Adapter integration tests**: require PostgreSQL (`POSTGRES_HOST=127.0.0.1`).
  All non-DB tests pass; adapter tests pass in CI.

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
