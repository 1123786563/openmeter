# WeKnora AI Billing — Phase 1 Operations Guide

> **CERTIFICATION GATE**: This feature has NOT been certified for production use.
> Live E2E tests against real PostgreSQL, Kafka, and ClickHouse engines are
> PENDING CERTIFICATION (see `docs/test-reports/openmeter-ai-billing-phase1.md`).
> The `ai_usage.enabled` flag MUST NOT be set to `true` in any production or
> staging deployment until the live E2E gate passes and this warning is removed.

Operational runbook for the AI Usage billing subsystem shipped in Phase 1.
Covers signing key rotation, outbox replay, migration rollback, production
bootstrap, and metrics/alerts.

> FROZEN: `ai_usage_*`、`/ai-usage-batches`、runtime authorization 和旧 rate-entry API 不再接受功能扩展。它们只在 Reservation v2 切换窗口内提供回滚路径，并在删除门通过后整体移除。

## Feature gate

AI Usage is disabled by default. Enable it in `.env` / config:

```yaml
ai_usage:
  enabled: true
  signing:
    current_key_id: "wkc-p1-key-001"
    current_seed: "<64-char hex Ed25519 seed>"
  authorization_ttl: "5m"
  worker:
    lease_duration: "30s"
    batch_size: 50
```

When disabled, all `/ai-usage-batches` and
`/customers/{customerId}/runtime-authorization` routes return HTTP 404.

## Signing key rotation

Runtime authorization packages are signed with Ed25519. The signer supports a
**previous key** overlap window so consumers can refresh their cached public
key without a hard cutover.

### Rotation procedure

1. **Generate a new key pair.**

   ```bash
   # 32-byte seed -> 64-char hex
   openssl rand -hex 32
   ```

2. **Set the new key as `current` and the old key as `previous`.**

   ```yaml
   ai_usage:
     signing:
       current_key_id: "wkc-p1-key-002"
       current_seed: "<new 64-char hex>"
       previous_key_id: "wkc-p1-key-001"
       previous_seed: "<old 64-char hex>"
   ```

3. **Rolling-restart all OpenMeter instances.** During the overlap window,
   new packages are signed with `current_key_id`; consumers holding packages
   signed by the previous key can still verify until their TTL expires.

4. **After one full TTL cycle (default 5 min), remove the previous key.**

   ```yaml
   ai_usage:
     signing:
       current_key_id: "wkc-p1-key-002"
       current_seed: "<64-char hex>"
   ```

5. **Restart again to finalize.**

### Constraints

- `current_key_id` must differ from `previous_key_id`.
- Seeds must be 32 bytes (64 hex chars).
- TTL must be between 0 and 15 minutes (default 5 min).

## Outbox replay

The transactional outbox (`ai_usage_outbox`) is the bridge between synchronous
PostgreSQL settlement and asynchronous Kafka/ClickHouse projection. The worker
claims unpublished rows, publishes to all projections, and marks them
published only after every projection acknowledges.

### Manual replay

If the worker falls behind or a projection was temporarily unavailable:

```bash
# Count unpublished rows
psql -c "SELECT count(*) FROM ai_usage_outboxes WHERE published_at IS NULL;"

# The worker resumes automatically from the oldest unpublished row on restart.
# To force an immediate sweep, restart the openmeter process.
```

### Dead-letter recovery

A row is dead-lettered after the second lease expiry. To replay a dead-lettered
row:

```sql
-- Reset the row so the worker reclaims it on the next poll.
UPDATE ai_usage_outboxes
   SET published_at = NULL,
       claim_count = 0,
       dead_letter_reason = NULL,
       owner = NULL,
       leased_until = NULL
 WHERE id = '<row-id>';
```

The worker republishes the same Event ID (the outbox row ID), so downstream
consumers deduplicate — no second ledger effect.

### Exactly-once guarantee

The outbox row ID equals the Kafka event ID. Downstream consumers must key
their deduplication table on this ID.

## Migration rollback

Migration `20260803000100` creates the AI billing core tables. The down
migration drops them in dependency order (children first).

### Roll forward

```bash
make migrate-up
```

### Roll back

```bash
make migrate-down
```

Or force a specific version:

```bash
make migrate-force version=20260803000099
```

**Warning:** rolling back drops all AI billing data. Take a backup first
(`pg_dump`). After rollback, set `ai_usage.enabled=false` to avoid the server
refusing to start (it validates that the schema exists when the feature is on).

## No-production-data bootstrap

For a fresh environment with no production data:

1. **Run migrations.**

   ```bash
   make migrate-up
   ```

2. **Generate the signing seed.**

   ```bash
   openssl rand -hex 32
   ```

3. **Set config.**

   ```yaml
   ai_usage:
     enabled: true
     signing:
       current_key_id: "wkc-p1-initial"
       current_seed: "<seed from step 2>"
     authorization_ttl: "5m"
     worker:
       lease_duration: "30s"
       batch_size: 50
   ```

4. **Start the server.**

   ```bash
   make dev-app   # or docker compose up
   ```

5. **Verify the feature is live.**

   ```bash
   curl -sf http://localhost:8888/api/v3/ai-usage-batches -X POST \
     -H 'Content-Type: application/json' \
     -d '{"idempotency_key":"probe","payload_hash":"probe","billing_customer_id":"0000000000000000000000000","subject_key":"probe","tenant_seq":1,"occurred_at":"2026-01-01T00:00:00Z","billing_mode":"component","provider_managed":true,"lines":[{"resource_code":"rag_queries","quantity":1,"canonical_line_index":0}]}'
   ```

   A 422 (invalid customer ID) confirms the route is live. A 404 means the
   feature is disabled.

## Metrics and alerts

### Recommended alerts

| Alert | Condition | Severity |
|---|---|---|
| Outbox backlog growing | `ai_usage_outbox_unpublished_count > 1000` for 5 min | Warning |
| Outbox dead-letter rate | `rate(ai_usage_outbox_dead_letter_total[5m]) > 0` | Critical |
| Settlement latency | `histogram_quantile(0.99, ai_usage_settlement_duration_seconds) > 2` | Warning |
| Signing key expiring | `time() - ai_usage_signing_key_created_timestamp > 7776000` (90 days) | Warning |
| Feature disabled unexpectedly | `ai_usage_enabled == 0` | Info |

### Key metrics

| Metric | Type | Description |
|---|---|---|
| `ai_usage_batches_settled_total` | counter | Batches settled (by status, billing mode) |
| `ai_usage_credits_charged_total` | counter | Total integer credits charged |
| `ai_usage_settlement_duration_seconds` | histogram | End-to-end settlement latency |
| `ai_usage_outbox_unpublished_count` | gauge | Unpublished outbox rows |
| `ai_usage_outbox_dead_letter_total` | counter | Rows dead-lettered |
| `ai_usage_authorization_signed_total` | counter | Runtime authorization packages signed |
| `ai_usage_authorization_denied_total` | counter | Authorization denials (by reason) |
| `ai_usage_watermark_gap` | gauge | Current gap between latest seq and covered seq |
