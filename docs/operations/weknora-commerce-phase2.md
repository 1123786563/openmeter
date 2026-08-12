# OpenMeter Commerce Phase 2 — Operations Guide

## Overview

Phase 2 adds commercial capabilities to OpenMeter: wallet top-ups, subscription
purchases/renewals, WeChat/Alipay payments, enterprise monthly receivables,
original-route refunds, and reconciliation. This document covers deployment,
configuration, monitoring, and incident response.

## Architecture

```
Client → API (v3) → Commerce Handler
                      ↓
    ┌────────────────┼─────────────────┐
    │                │                 │
  Wallet         Orders           Payment
  (read-only)    (state machine)  (facts)
                     │                │
              Fulfillment         Refund
              (credits)           (fence)
                     │
              Enterprise
              (receivables)
```

Background workers handle recovery, fulfillment, refund polling, receivable
closing, and reconciliation.

## Configuration

Commerce requires the following configuration:

| Key | Description |
|-----|-------------|
| `COMMERCE_ENABLED` | Enable/disable commerce routes (default: false) |
| `COMMERCE_WECHAT_MCH_ID` | WeChat Pay merchant ID |
| `COMMERCE_WECHAT_APP_ID` | WeChat Pay app ID |
| `COMMERCE_WECHAT_API_KEY` | WeChat Pay API key (from secret manager) |
| `COMMERCE_ALIPAY_APP_ID` | Alipay app ID |
| `COMMERCE_ALIPAY_PRIVATE_KEY` | Alipay private key (from secret manager) |
| `COMMERCE_FULFILLMENT_INTERVAL` | Worker poll interval (default: 10s) |
| `COMMERCE_RECONCILIATION_INTERVAL` | Reconciliation interval (default: 5m) |

## Worker Lifecycle

Five lifecycle-managed workers start on server boot:

1. **payment-query-recovery**: Polls providers for orders where the callback was
   lost. Runs every 30s.
2. **fulfillment**: Processes pending fulfillment records toward `fulfilled`.
   Runs every 10s. Recovers expired leases on startup (60s timeout).
3. **refund-query**: Drives new `pending_fence` refunds, polls provider status
   in `provider_processing`, and resumes `ledger_reversing` after a crash. Runs
   every 15s and processes at most 100 records per tick in stable
   `updated_at,id` order.
4. **receivable-close**: Closes enterprise receivable periods that have ended.
   Runs hourly.
5. **reconciliation**: Runs invariant checks. Runs every 5m. Reports only —
   never silently rewrites data.

### Shutdown Order

Workers stop in reverse registration order to respect dependencies:
reconciliation → receivable-close → refund-query → fulfillment → payment-query.
Each worker drains its current tick before stopping.

## Monitoring

### Health Checks

- `GET /api/v3/health` — server health (includes worker status)
- Worker runner names visible in startup logs

### Key Metrics

| Metric | Description |
|--------|-------------|
| `commerce_orders_total` | Orders created by kind/status |
| `commerce_fulfillment_duration_seconds` | Time from paid to fulfilled |
| `commerce_refund_fenced_total` | Refunds that entered fencing |
| `commerce_reconciliation_findings_total` | Findings by check and severity |

### Reconciliation Alerts

The reconciliation worker emits findings for these invariants:

1. **paid_order_without_fulfillment** — paid order stuck > threshold (error).
2. **fulfilled_without_ledger_grant** — fulfilled order missing its grant (error).
3. **provider_success_without_payment_fact** — provider says success, no fact (error).
4. **refund_fact_without_fence_reversal** — refund fact without fence (error).
5. **wallet_differs_from_ledger** — Wallet total ≠ Ledger total (error).
6. **closed_receivable_range_changed** — frozen period amount changed (error).
7. **unknown_event_types** — outbox published unapproved event type (error).
8. **event_id_outbox_id_mismatch** — event ID ≠ outbox row ID (error).

## Incident Response

### Stuck Fulfillment

1. Check the fulfillment record status and `claimed_at`.
2. If `claimed_at` is older than 60s, the lease expired — the worker will re-claim.
3. If the order is in `paid` but no fulfillment record exists, run
   `RequestFulfillment` manually.

### Refund Fence Stuck

1. Check the refund status. If `provider_processing`, the fence is held.
2. Query the provider for the refund status.
3. If the provider returned failure, the refund transitions to `failed` and
   releases the fence.
4. If the provider returned success, the reversal proceeds.

### Wallet Discrepancy

Reconciliation flags wallet-vs-ledger mismatches. This is a report-only check:
investigate the root cause before manually correcting. The Wallet is always
derived from the Ledger, so a discrepancy indicates a Ledger data issue.

## Backup & Recovery

- The commerce tables (orders, products, attempts, facts, fulfillments, refunds,
  receivables) are in the same PostgreSQL database as Phase 1.
- Standard PostgreSQL backups include commerce data.
- The Credit Ledger is the source of truth for balances — Wallet is derived.

## Security

- Provider API keys come from the secret manager, never from env files or Ent.
- Callback signature verification is mandatory — no domain effect without it.
- No secret fields appear in API responses (api_key, token, secret, app_secret).
- Cross-customer reads return 404, not 403, to prevent existence leakage.
