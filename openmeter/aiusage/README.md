# AI Usage (`openmeter/aiusage`)

The AI Usage subsystem settles WeKnora's canonical AI usage batches — chat
turns, agent runs, retrieval queries — against an integer-credit ledger inside
OpenMeter. It is the Phase 1 contract surface that the WeKnora provider
integrates against: every billable AI action flows through this package as an
atomic, idempotent batch.

## Contract version

`weknora-billing-p1-v1` — frozen for Phase 1. The runtime authorization
endpoint (`GET /customers/{customerId}/runtime-authorization`) returns this
string so clients can detect contract changes.

> **CERTIFICATION GATE**: This subsystem has NOT been certified for production
> use. Live E2E tests against real PostgreSQL, Kafka, and ClickHouse engines
> are PENDING CERTIFICATION. The `ai_usage.enabled` flag MUST NOT be set to
> `true` in any production or staging deployment until the live E2E gate passes.

## Ownership boundaries

| Concern | Owner |
|---|---|
| Batch ingestion, rating, ceiling, settlement, watermark | `openmeter/aiusage/service` + `openmeter/aiusage` (types, interfaces) |
| Ent persistence (batches, allocations, outbox, watermarks) | `openmeter/aiusage/adapter` + `openmeter/ent/db` |
| Pricing / rate resolution | `openmeter/aiusage/pricing` |
| Credit ledger settlement via collector | `openmeter/aiusage/settlement` |
| Ed25519 runtime authorization signing | `openmeter/aiusage/signing` |
| Runtime authorization assembly | `openmeter/aiusage/runtimeauthorization` |
| Outbox relay / Kafka projection | `openmeter/aiusage/worker` |
| HTTP handlers | `api/v3/handlers/aiusage` |
| Route registration + feature gate | `api/v3/server` |
| DI wiring (app, config, providers) | `app/common/aiusage.go`, `app/common/aiusage_providers.go`, `app/common/aiusage_noop.go` |

## Entities

| Entity | Table | Role |
|---|---|---|
| **AIUsageBatch** | `ai_usage_batches` | A settled usage batch (one business action). Immutable once settled. |
| **AIUsageLineItem** | `ai_usage_line_items` | One resource consumption entry within a batch. |
| **AIUsageRatingSnapshot** | `ai_usage_rating_snapshots` | The resolved cost/sales/credit snapshot for a line at settlement time. |
| **AIUsageAllocation** | `ai_usage_allocations` | The credit-grant burn-down provenance for a batch (which grants were consumed, in what order). |
| **AIUsageOutboxEvent** | `ai_usage_outbox` | Transactional outbox row: emitted atomically with the batch, drained by the worker. |
| **AIUsageWatermark** | `ai_usage_watermarks` | Continuous watermark per `(namespace, subject_id)` for gap detection. |
| **AIUsageRateCardEntry** | `ai_usage_rate_card_entries` | Rate-card override for a resource/provider/model/customer combination. |

## Transaction sequence

```
Client --> POST /ai-usage-batches --> Handler --> Service.IngestBatch
                                                  |
                    +-----------------------------+
                    v
         1. Validate input (422 on bad fields)
         2. Idempotency check (409 on hash conflict, 200 on identical replay)
         3. Rate line items (component mode) / use ceiling (bundle mode)
         4. Apply reservation ceiling
         5. Settlement engine burns grants in priority order
         6. Advance continuous watermark
         7. Persist batch + allocations + outbox row atomically (single tx)
                    |
                    v
              201 Created (first) / 200 OK (replay)
                    |
                    v
         Worker drains outbox --> Kafka --> ClickHouse projection (async)
```

The HTTP 201/200 response fires **after** the PostgreSQL transaction commits.
Kafka/ClickHouse projection is eventually consistent via the outbox worker.

## State and idempotency invariants

1. **Exactly-once settlement.** The `(idempotency_key, payload_hash)` pair is
   unique. An identical replay returns the stored result (200); a mutated
   replay returns 409. No duplicate ledger effect is ever produced.

2. **Batch immutability.** Once a batch is `settled`, its lines, rating
   snapshots, and allocations are never mutated. Corrections create new
   reversing allocations, not in-place edits.

3. **Continuous watermark.** `covered_tenant_seq` tracks the highest contiguous
   sequence with no gaps. Out-of-order arrivals (e.g. seq 1, 3, 2) are stored
   but the watermark only advances when the gap fills.

4. **Priority burn order.** Grants are consumed in ascending priority order:
   plan grants first, then promotional, then paid top-ups, then enterprise
   receivable (which can go negative). Non-enterprise customers fail-closed
   when prepaid is exhausted.

5. **Ceiling allocation.** When the batch total exceeds the reservation
   ceiling, credits are allocated to lines by canonical line index (stable
   zero-based position), and the platform absorbs the excess.

## Approved events

| Event | Direction | Trigger |
|---|---|---|
| `ai_usage.batch.settled` | Emitted | Batch committed to PostgreSQL |
| `ai_usage.batch.corrected` | Emitted | Batch reversed via correction |
| `credit.balance.changed` | Emitted | Grant burned or topped up |
| `runtime_authorization.updated` | Emitted | New signed authorization package generated |
| `credit.grant.expired` | Consumed | Prepaid grant expiry triggers re-authorize |
| `subscription.updated` | Consumed | Plan/entitlement change triggers re-authorize |

## Dependencies

- **PostgreSQL** — source of truth for batches, allocations, watermarks, outbox.
- **Kafka** — async projection transport (via outbox worker).
- **ClickHouse** — meter aggregation read model.
- **Credit / Ledger** — `openmeter/credit`, `openmeter/ledger` (grant balances, ledger entries).
- **LLM Cost** — `openmeter/llmcost` (provider cost resolution for managed models).

## Security

- **Signing keys.** Runtime authorization packages are signed with Ed25519.
  The current key ID and seed are configured via `ai_usage.signing.current_key_id`
  and `ai_usage.signing.current_seed`. See the operations guide for rotation.

- **RBAC.** AI Usage routes are protected by the same namespace-aware RBAC
  middleware as the rest of the v3 API. The `billing_customer_id` in each
  batch must resolve to a customer in the caller's namespace.

- **Credential isolation.** No API key, token, or secret flows through AI
  Usage request or response bodies.

## Operations

See [`docs/operations/weknora-ai-billing-phase1.md`](../../docs/operations/weknora-ai-billing-phase1.md)
for signing key rotation, outbox replay, migration rollback, no-production-data
bootstrap, and metrics/alerts.

## Test commands

```bash
# Unit + integration tests (requires PostgreSQL at POSTGRES_HOST)
POSTGRES_HOST=127.0.0.1 go test -tags=dynamic -count=1 ./openmeter/aiusage/...

# Handler integration tests
POSTGRES_HOST=127.0.0.1 go test -tags=dynamic -count=1 ./api/v3/handlers/aiusage/...

# Live E2E (requires running server with ai_usage.enabled)
TZ=UTC OPENMETER_ADDRESS=http://localhost:8888 \
  go test -C e2e -count=1 -v -run '^TestV3AIUsage' ./

# Phase 1 acceptance gate (all three modules + spec + E2E)
make weknora-ai-billing-p1-acceptance
```
