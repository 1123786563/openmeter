# Task 7 Report: Wire API, workers, reconciliation and OpenMeter acceptance

## Status: DONE_WITH_CONCERNS

## Commits

| SHA | Subject |
|-----|---------|
| 1132fc1 | feat(commerce): prepare OpenMeter commercialization release |

## Test Summary

`go test ./openmeter/commerce/... ./api/v3/handlers/commerce/... -race -count=1` — **all 13 packages PASS**.

## Deliverables

1. **HTTP handlers** (`api/v3/handlers/commerce/handler.go`): 13 route handlers covering wallet, catalog, orders, checkout, payment callbacks (WeChat/Alipay), refunds, offline payments, receivable periods, external invoices. RBAC enforced: customer-scoped reads, admin-only mutations, public callbacks with mandatory signature verification (inside payment service), no secret fields in responses.

2. **Handler RBAC tests** (`api/v3/handlers/commerce/handler_test.go`): 14 test scenarios covering customer-scoped reads, cross-customer 404s, missing-param 400s, idempotent replay 200s, creation 201s, payment callback namespace checks, no-secret-field assertions, currency filtering.

3. **Lifecycle-managed workers** (`openmeter/commerce/worker/runner.go`): `Runner` type with Start/Stop/idempotency/context-cancellation + `Manager` for ordered startup/shutdown. Five registered runners: payment-query-recovery, fulfillment, refund-query, receivable-close, reconciliation.

4. **Reconciliation checker** (`openmeter/commerce/reconciliation/checker.go`): All 8 approved invariants implemented as report-only checks with configurable `ProbePort` interface. Tests cover no-findings, each invariant, event-type filtering.

5. **V3 server wiring** (`api/v3/server/commerce.go` + `server.go`): Replaced stub implementations with real handler delegation. `CommerceHandler` is optional in Config — nil yields 501.

6. **Domain README** (`openmeter/commerce/README.md`): Full domain document covering ownership boundaries, wallet derivation, state machines, sequences, transaction/idempotency rules, approved events, secret handling, dependencies, operations, and test commands.

7. **Build artifact script** (`tools/weknora/build-phase2-artifact.sh`): Dirty-tree check, P1 regression, P2 acceptance, OpenAPI checksums, Docker build, digest, SBOM (Syft), vulnerability gate (Grype), evidence report generation.

8. **Operations guide** (`docs/operations/weknora-commerce-phase2.md`): Architecture diagram, configuration, worker lifecycle, monitoring metrics, reconciliation alerts, incident response, backup/recovery, security.

9. **Evidence report** (`docs/test-reports/openmeter-commerce-phase2.md`): Deliverables table, test results, RBAC coverage matrix, reconciliation invariants, approved events.

## Concerns

1. **Enterprise handler stubs**: `CreateOfflinePayment`, `ListReceivablePeriods`, `UpdateExternalInvoice` return placeholder responses. Full wiring needs enterprise service Ent adapter integration (receivable account + period queries). Does not block core commerce flow.

2. **DI wiring (wire.go) not modified**: Commerce handler is accepted as optional in v3 server Config. Production wiring (`cmd/server/wire.go`) needs full service construction (EntAdapter + provider adapters + secret provider). Nil handler → 501.

3. **OpenAPI regen / make targets not run**: `make gen-api`, `make generate`, `make etoe`, `make lint-go` not run (require codegen tooling, ClickHouse, testcontainers infrastructure). The test gate `go test ./openmeter/commerce/... ./api/v3/handlers/commerce/... -race` passes.

4. **Build artifact script**: Written and executable but requires Docker/Syft/Grype for full run.

## Report File

`/Users/wuyongjun/trea/openmeter/.superpowers/sdd/2026-08-03-openmeter-ai-billing-phase2-openmeter/task-7-report.md`
