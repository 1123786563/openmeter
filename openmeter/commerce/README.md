# Commerce Domain (Phase 2)

## Overview

The commerce domain implements the Phase 2 commercialization layer for OpenMeter:
wallet top-ups, subscription purchases/renewals, enterprise monthly receivables,
original-route refunds, and reconciliation. It sits on top of the Phase 1 Credit
Ledger and settlement infrastructure without altering existing BYOK/free-model
behavior.

## Ownership Boundaries

| Package | Responsibility |
|---------|---------------|
| `commerce` | Root types (Wallet, Product, Order), repository interfaces, approved event names |
| `commerce/wallet` | Read-only Wallet aggregation over the Credit Ledger |
| `commerce/catalog` | Product catalog CRUD (SKU, price, credits) |
| `commerce/order` | Order lifecycle + state machine |
| `commerce/payment` | Payment fact verification, provider adapters (WeChat, Alipay, offline) |
| `commerce/fulfillment` | Exactly-once fulfillment of paid orders |
| `commerce/refund` | Refundable Credit fence + original-route refunds |
| `commerce/enterprise` | Enterprise monthly receivables + offline authorizations |
| `commerce/worker` | Lifecycle-managed background runners |
| `commerce/reconciliation` | Scheduled invariant checks (report, never rewrite) |

## Wallet Derivation

The Wallet is a **read-only projection** over the immutable Credit Ledger. It
never holds a mutable second balance. `GetWallet` recomputes from grants on every
call:

1. Reads non-expired allocation grants (plan, gift, recharge).
2. Reads the enterprise receivable line (if any).
3. Aggregates into ordered buckets by consumption priority:
   `plan(10) < gift(20) < recharge(30) < enterprise_receivable(40)`.
4. Returns the contract version `commerce.phase2.v1`.

## Entity State Machines

### Order

```
created → awaiting_payment → paid → fulfilled
                              ↓
                    refund_pending → partially_refunded → refunded
                    cancelled / expired
```

`paid` and `fulfilled` are **separate states**: paid means a verified provider
fact exists; fulfilled means the commercial invoice is marked paid and credits
have been granted.

### Fulfillment

```
pending → processing → fulfilled
                  ↘ failed (retry)
```

A partial unique index `(order, status='fulfilled')` guarantees exactly one
successful fulfillment per order. Expired processing leases are re-claimed after
`ProcessingLeaseTimeout` (60s).

### Refund

```
pending_fence → provider_processing → ledger_reversing → fulfilled
                                             ↘ failed
```

The refund holds a whole-customer Credit fence during `provider_processing` and
`ledger_reversing`. The fence is released only on fulfilled or failed.

## Sequences

### Payment → Invoice → Fulfillment

1. Client creates an order (`created`).
2. Client initiates checkout → provider QR code.
3. Provider callback (or confirmation query) verifies signature → Payment Fact.
4. Payment Fact moves order to `paid` + creates fulfillment request (one tx).
5. Fulfillment worker: marks invoice paid → grants credits → marks `fulfilled`.

### Refund Fence

1. Refund created in `pending_fence`.
2. Establish whole-customer fence (stops new reservations, drains in-flight).
3. Under fence: lock source allocation, recompute unused credit, reserve exact
   quantum (10 Credit : 1 fen), submit to provider → `provider_processing`.
4. Provider success → reverse fenced credit, transition to `fulfilled`, release the fence.
5. Definitive failure → `failed`, release fence.

## Transaction & Idempotency Rules

- **Credits are int64** — no floats. Money uses minor currency units + ISO 4217.
- **Order creation** is idempotent on `idempotency_key` — same key + same payload
  returns the stored order; different payload for same key returns 409.
- **Payment facts** deduplicate on `RawHash` (SHA-256 of raw callback body) and
  `ProviderEventID`.
- **Fulfillment** uses a partial unique index to enforce exactly-once.
- **Refund credits** are reserved atomically under the fence.
- **Customer-locked transactions** via `WithCustomerLock` (Postgres advisory lock).

## Approved Events

Successful state transitions publish only these event names:

| Event | When |
|-------|------|
| `order.updated` | Order status transition |
| `payment.settled` | Payment fact verified, order moved to paid |
| `payment.failed` | Payment definitively failed |
| `refund.updated` | Refund status transition |
| `invoice.updated` | Commercial invoice marked paid |
| `subscription.updated` | Subscription renewed |

Event IDs always equal Outbox row IDs. Retries do not create a second domain
effect — the ledger effect was committed in the original transaction.

## Provider Secret Handling

- Provider API keys come from a `SecretProvider` (Vault / cloud secret manager).
- Keys **never** appear in API responses, logs, or Ent metadata.
- Callbacks verify signatures before any domain effect.
- Raw callback bodies are never persisted — only their SHA-256 hash.

## Dependencies

- Phase 1 Credit Ledger (grants, settlement, collector)
- Ent (PostgreSQL) for persistence
- `pkg/clock` for all time operations

## Operations

### Production payment boundary

- Provider callbacks terminate at OpenMeter only:
  `/api/v3/payment-providers/wechat/callback` and
  `/api/v3/payment-providers/alipay/callback`.
- OpenMeter owns provider credentials, signature verification, immutable
  `PaymentFact` records, paid transitions, query recovery, and fulfillment.
- WeKnora calls Commerce with an OpenMeter API key and may verify signed
  OpenMeter responses with the OpenMeter public-key keyring. It does not hold
  WeChat or Alipay merchant credentials and does not receive provider callbacks.
- The repository's protocol-compatible provider test stand is loopback-only;
  its success is not an official sandbox or live-provider acceptance result.

### Workers
See [README-payment-production.md](README-payment-production.md) for the
production payment operations manual covering callback endpoints, credential
ownership, payment recovery, certificate rotation, emergency shutdown,
core alerting rules, and rollout/rollback procedures.

Registered runners (lifecycle-managed via `worker.Manager`):

1. `payment-query-recovery` — polls providers for callback-lost attempts.
2. `fulfillment` — processes pending fulfillment records.
3. `refund-query` — polls providers for refund status.
4. `receivable-close` — closes ended enterprise receivable periods.
5. `reconciliation` — runs scheduled invariant checks.

Startup recovers expired leases; shutdown stops intake, drains, closes in order.

### Reconciliation

Checks (report only, never rewrite):

1. Paid order without fulfillment beyond threshold.
2. Fulfilled order without exactly one Ledger grant.
3. Provider success without Payment Fact.
4. Refund Fact without matching fence/reversal.
5. Wallet aggregate differing from Ledger-derived value.
6. Closed receivable differing from frozen settlement range.
7. Unknown event types in the outbox.
8. Event ID ≠ Outbox row ID.

## Test Commands

```bash
# Commerce unit tests (race)
go test ./openmeter/commerce/... ./api/v3/handlers/commerce/... -race

# Single package
go test -run TestName ./openmeter/commerce/order/...

# Format
gofmt -w openmeter/commerce/... api/v3/handlers/commerce/...
```
