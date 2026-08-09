# Credit Reservation v2 Acceptance Evidence

**Date:** 2026-08-10
**Status:** NOT CERTIFIED

The acceptance command is `make credit-reservation-v2-acceptance`. Its E2E
test uses the `credit_reservation_acceptance` build tag and fails, rather than
skips, when `OPENMETER_ADDRESS` or the live reservation handler is absent.
It also requires `OPENMETER_CR_CUSTOMER_ID`, `OPENMETER_CR_SUBJECT_ID`,
`OPENMETER_CR_FEATURE_KEY`, `OPENMETER_CR_RESOURCE_CODE`, and crash/outbox
evidence URLs; fixture setup is an explicit release prerequisite.

## Evidence collected

| Gate | Result | Evidence |
|---|---|---|
| Focused reservation service/runtime/HTTP tests | PASS | `go test ./openmeter/creditreservation/... ./api/v3/server -count=1` |
| PostgreSQL acceptance database | BLOCKED | `127.0.0.1:5436` refused connection; no live reservation migration/ledger evidence exists |
| Atlas migration baseline | BLOCKED | Atlas checksum baseline requires the live migration database; do not repair or rewrite checksums without the baseline |
| TypeSpec/OpenAPI generation | BLOCKED | AIP generator fails on existing `runtime_authorization.tsp` `Record<string, unknown>` template diagnostic |
| Application server | BLOCKED | `cmd/server` has existing missing `seedCatalog` and stale generated Wire argument lists |

## Required live evidence

Before certification, attach the command output and identifiers (redacted where
needed) for each scenario: reserve→execute→settle; same-key replay and
different-hash conflict; insufficient prepaid and bounded enterprise
receivable; restart between execute and settle leading to UNKNOWN then
MANUAL_REVIEW; and outbox retry with the same event ID. Include pre/post credit
balances, ledger group IDs, reservation IDs, provider evidence references,
outbox IDs, and the SQL state/outbox counts from the operations runbook.

Do not mark this report certified until PostgreSQL, Kafka, ClickHouse, the
runtime bundle, and generated API surface are all available and the acceptance
target completes with no skipped scenarios.
