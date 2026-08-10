# Production Payment Operations Manual

This document covers the operational procedures for running the OpenMeter
commerce payment system in production. It is intended for on-call engineers,
SREs, and payment operations staff.

For the domain model, state machines, and architecture overview, see
[README.md](README.md). This document focuses exclusively on production
operations: callbacks, credentials, recovery, rotation, shutdown, alerting,
and rollout/rollback.

## 1. Architecture Overview

OpenMeter is the **single payment authority** in the WeKnora + OpenMeter
integration. All provider secrets, callback verification, and payment-to-order
state transitions live exclusively inside OpenMeter. WeKnora never holds
merchant keys, never receives provider webhooks, and never independently
decides that a payment succeeded.

```
  WeChat Pay ----- callback -----+
                                 +--> OpenMeter (verify -> PaymentFact -> order.paid -> fulfillment)
  Alipay ------- callback ------+            ^
                                             | API Key + public-key keyring
                                   WeKnora --+ (BFF / frontend, no provider secrets)
```

The paid-to-fulfilled pipeline inside OpenMeter:

1. Provider callback arrives at the callback endpoint.
2. The adapter verifies the signature and decrypts the payload (WeChat) or
   checks the RSA2 sign (Alipay) before any domain effect.
3. A `PaymentFact` is deduplicated on `RawHash` (SHA-256 of the raw body) and
   `ProviderEventID`.
4. The paid transaction moves the order to `paid` and creates a fulfillment
   request in the same database transaction.
5. The fulfillment worker grants credits and marks the order `fulfilled`.

If the callback is lost, the `payment-query-recovery` worker polls the provider
directly and drives the same pipeline.

## 2. Callback Endpoints

Both endpoints are public (no RBAC). Signature verification is mandatory and
happens inside the payment service before any domain effect.

| Provider | Endpoint | Method | Success Response |
|----------|----------|--------|------------------|
| WeChat Pay v3 | `/api/v3/payment-providers/wechat/callback` | POST | `204 No Content` (empty body) |
| Alipay | `/api/v3/payment-providers/alipay/callback` | POST | `200 OK` with plain-text body `success` |

### WeChat Pay v3 Callback

The adapter reads the `Wechatpay-Serial` header to select the matching platform
public key from the configured keyring. It then verifies the RSA-SHA256
signature over `timestamp\nnonce\nbody\n`, checks the callback freshness
window (`callback_max_age`, default 5 minutes), and decrypts the
`resource` ciphertext via AES-256-GCM using the API v3 key. Only after all
checks pass are merchant ID, application ID, amount, and currency validated
against the attempt.

- Request body limit: 1 MiB (`MaxBytesReader`).
- Duplicate events and already-fulfilled orders return `204` so WeChat stops
  retrying.
- Signature failures, protocol errors, and fact mismatches return `400` and are
  **not** acknowledged.
- Database, transaction, and timeout errors return `500` so WeChat retries.

### Alipay Callback

The adapter parses the form-encoded body, extracts the `sign` field, and
verifies the RSA2 (SHA256withRSA) signature over the canonicalized
notification content. It enforces `sign_type=RSA2`, validates `app_id` and
`seller_id`, and checks `trade_status`, amount, and `out_trade_no` before
returning a `PaymentFact`.

- On success the handler writes the literal string `success` as the response
  body (required by Alipay to stop retries).
- Duplicate events and already-fulfilled orders also return `success`.
- Signature and protocol failures return `400` and are not acknowledged.

## 3. Credential Ownership

All payment provider secrets are owned by OpenMeter and sourced from a
`SecretProvider` (file-mounted secrets in the current implementation, Vault or
cloud secret manager in production). Secrets **never** appear in API responses,
logs, Ent metadata, or configuration files in plaintext.

| Secret | Owner | WeKnora Access |
|--------|-------|----------------|
| WeChat merchant private key | OpenMeter | None |
| WeChat API v3 key (32 bytes) | OpenMeter | None |
| WeChat platform public key(s) | OpenMeter | None |
| Alipay app private key | OpenMeter | None |
| Alipay public key | OpenMeter | None |
| Alipay gateway credentials | OpenMeter | None |

**WeKnora receives only:**

1. An **OpenMeter API Key** for authenticated calls to the commerce API
   (create order, create checkout, query order status, create refund).
2. The **OpenMeter public-key keyring** for verifying signed responses from
   OpenMeter (if response signing is enabled).

WeKnora must **never** contain any WeChat or Alipay merchant key, certificate,
or API credential in its environment variables, configuration JSON, or container
image. The release gate is: WeKnora container environment and config JSON must
not contain any `wechat_*` or `alipay_*` secret material.

## 4. Payment Recovery

When a provider callback is lost or delayed, the `payment-query-recovery`
worker restores convergence without manual intervention.

- **Stale threshold**: An attempt is considered stale when it has been
  `pending` for longer than `commerce.payment.pendingStaleAfter`
  (default **30 seconds**).
- **Worker interval**: The `payment-query-recovery` runner ticks every
  **30 seconds** (configured in the worker registration).
- **Recovery batch size**: 100 attempts per tick (ordered by `updated_at`
  then ID ascending).
- **Recovery action**: For each stale attempt, the worker calls the provider's
  `QueryPayment` to get a verified `PaymentFact`. If the provider reports
  `SUCCESS`, the same paid-transaction pipeline runs. Non-success terminal
  states do not trigger a transition.

The refund side has an analogous `refund-query` worker (15-second interval)
that polls refunds in `provider_processing`. It does **not** infer a
`PROCESSING` refund status as success.

## 5. Certificate Rotation Procedure

This procedure applies primarily to WeChat platform certificate rotation,
which is the most common rotation event. Alipay key rotation follows the same
principle: add the new key first, then cut over, then remove the old one.

### WeChat Platform Public Key Rotation

The adapter supports **multiple platform public key serials simultaneously**.
Each serial maps to a secret via `PlatformPublicKeySecret(serial)`, and the
`Wechatpay-Serial` header in each callback selects which key to use for
verification. This enables zero-downtime rotation.

1. **Mount the new serial public key.** Add the new platform public key PEM
   to the secret provider under `wechat_platform_public_key/{NEW_SERIAL}`.
   Do not change any configuration yet.

2. **Update configuration and roll.** Add the new serial to
   `commerce.payment.wechat.platformPublicKeyFiles` alongside the existing
   serial(s). Run the configuration validator (`--validate`) to confirm
   all configured serials resolve to valid public keys. Then perform a
   rolling restart of OpenMeter instances.

3. **Verify convergence.** Confirm that callbacks signed with both the old and
   new serial are verifying successfully. Monitor the `signature failure` alert
   for any spikes.

4. **Remove the old public key.** Once WeChat has fully switched to signing
   with the new serial (no callbacks referencing the old serial for a full
   observation window), remove the old serial from
   `platformPublicKeyFiles` and remove the old key from the secret provider.
   Roll again.

### Alipay Public Key Rotation

1. Mount the new Alipay public key PEM alongside the existing one.
2. Update `commerce.payment.alipay.alipayPublicKeyFile` to the new path and
   roll.
3. Verify callbacks are verifying successfully.
4. Remove the old key file.

### Merchant Private Key Rotation (WeChat / Alipay)

Coordinate with the provider portal: generate the new key pair, upload the new
public key or certificate to the provider, then mount the new private key,
validate, and roll. Until the provider recognizes the new key, outbound
requests (QR code creation, queries, refunds) may fail; schedule this during
a maintenance window.

## 6. Emergency Shutdown Procedure

When a payment channel must be taken offline urgently (provider outage,
security incident, integration defect), use the per-channel `enabled` flag.

### Per-Channel Disable

Set the channel's `enabled` to `false` in configuration and roll:

```yaml
commerce:
  payment:
    wechat:
      enabled: false   # disables WeChat checkout and callbacks
    alipay:
      enabled: false   # disables Alipay checkout and callbacks
```

Effects of `enabled=false`:

- New checkout sessions for that channel are rejected.
- The callback handler returns `501 Not Implemented` for that channel.
- **Order queries remain available** -- existing orders can still be looked up.
- **The fulfillment worker continues running** -- already-paid orders still
  receive their credit grants.
- **The payment-query-recovery worker continues running** -- pending attempts
  for existing orders still converge if the provider reports success.

### What Must NOT Be Done During Shutdown

- **Do not delete PaymentFact rows.** They are the immutable audit record of
  verified payments. Deleting them breaks reconciliation and refund matching.
- **Do not run destructive migrations** that drop payment columns or tables.
  PaymentFact and migrations are never rolled back via deletion.
- **Do not disable the fulfillment or recovery workers** unless the shutdown
  is specifically for a fulfillment defect. Paid orders must still converge to
  fulfilled.

### Full Commerce Disable

To disable all commerce payment processing (both channels):

```yaml
commerce:
  enabled: false
```

This disables the entire payment subsystem. Use only when both providers must
be taken down simultaneously.

## 7. Core Alerting Rules

The following alerts are the minimum set for production payment operations.
Each should be wired to the on-call rotation with appropriate severity.

| Alert | Condition | Severity | Action |
|-------|-----------|----------|--------|
| **Callback 5xx** | Callback endpoint returns 5xx rate > 1% over 5 min | Critical | Investigate database health, transaction failures, or adapter errors. Provider is retrying; prolonged 5xx causes callback storms. |
| **Signature failure** | Callback signature verification failure rate > 0.1% over 5 min | High | Check for certificate rotation in progress, keyring misconfiguration, or a man-in-the-middle. Do not bypass verification. |
| **Pending age** | Payment attempt pending beyond `2 x pendingStaleAfter` (60s) | High | Check `payment-query-recovery` worker health. If the worker is down, attempts will not converge without callbacks. |
| **Paid-not-fulfilled** | Order in `paid` state without fulfillment beyond threshold (e.g., 2 min) | Critical | Check fulfillment worker health. Credits are not being granted to customers. |
| **Duplicate event** | Duplicate provider event rate > 5% over 5 min | Medium | Investigate provider retry behavior. Deduplication is working (events are not double-counted), but high duplicate rates indicate provider instability or our 5xx responses triggering retries. |
| **Provider query failure** | `payment-query-recovery` worker error rate > 10% over 5 min | Medium | Check provider API health, network connectivity, and credential validity. Recovery worker cannot converge lost callbacks. |

Additional monitoring signals (informational, not page-worthy):

- Reconciliation findings count (reconciliation runner logs findings every 5
  minutes; any non-zero count indicates an invariant violation that needs
  investigation).
- Checkout creation rate per channel (sudden drop may indicate upstream
  issues).
- Fulfillment processing latency (time from `paid` to `fulfilled`).

## 8. Rollout and Rollback Procedures

### Rollout Order

1. **Schema migration.** Deploy the read-only-compatible schema migration and
   the complete `PaymentFact` field set. Migration is forward-only.

2. **Provider code (disabled).** Deploy the OpenMeter provider code with both
   WeChat and Alipay `enabled=false`. No payment traffic flows yet.

3. **Mount secrets and validate.** Mount all provider secrets and platform
   public keys into the secret provider. Run the configuration validator
   (`--validate`) to confirm every configured key resolves and parses.

4. **Internal test merchant.** Enable one channel for a single internal test
   merchant. Complete a 1-fen recharge, a duplicate notification, a lost
   notification (recovery), and a refund query.

5. **WeKnora internal tenant canary.** Enable WeKnora's internal tenant
   grayscale. Observe callback 5xx, pending age, and paid-not-fulfilled
   metrics for at least one full payment window.

6. **Expand gradually.** Increase tenant coverage one step at a time. Enable
   the second channel only after the first is stable, or run them in sequence
   but change only one variable per step.

7. **Cleanup.** After stable operation, remove WeKnora's local payment
   configuration and any legacy MD5 simulator paths. PaymentFact rows and
   migrations are never deleted.

### Rollback Rules

| Scenario | Rollback Action |
|----------|----------------|
| **Provider checkout errors** | Disable the affected channel (`enabled=false`). Keep order queries, callbacks, and fulfillment running so existing orders converge. |
| **Callback signature errors** | Restore the previous platform public key mapping. Never bypass verification or accept unsigned/`resource`-only callbacks. |
| **Paid transaction errors** | Stop new checkout (disable channel). Let the provider retry callbacks. Fix and deploy, then let the callback or recovery worker converge pending orders. |
| **WeKnora display errors** | Roll back WeKnora BFF/frontend only. OpenMeter PaymentFact and order states are not rolled back. |
| **Migration errors** | Apply a forward-fix migration validated by Atlas only. Never delete verified payment facts or run destructive rollback migrations. |

### Invariants During Rollback

- PaymentFact rows are **never** deleted, regardless of rollback scenario.
- Order state transitions are **never** reversed (e.g., `paid` is not moved
  back to `awaiting_payment`).
- Credit grants already issued are **never** silently reversed; corrections go
  through the refund flow.
- Migrations are **always** forward-only.
