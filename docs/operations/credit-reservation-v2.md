# Credit Reservation Operations Guide

> **Certification gate:** Credit Reservation v2 is disabled by default. Do not
> enable it in staging or production until the cutover checks in
> [credit-reservation-v2-cutover.md](credit-reservation-v2-cutover.md) are
> approved and the reservation service plus outbox worker are registered in
> the application container.

## Feature gate and limits

The lifecycle has three independent safety gates:

1. `credits.enabled` and `credits.reservationsEnabled` enable the CREDIT
   capability and the unstable reservation API family.
2. `creditReservation.enabled` is the deployment-level opt-in gate.
3. The process must supply a reservation HTTP handler. A missing handler keeps
   the routes unavailable, even when both configuration gates are enabled.

The application currently carries the second gate through the HTTP server, but
does not construct the reservation service or run the outbox worker. Therefore
setting this configuration alone intentionally leaves the routes at HTTP 404;
it is not a production enablement procedure.

```yaml
credits:
  enabled: true
  reservationsEnabled: true

creditReservation:
  enabled: false
  authorizationTTL: 5m
  executionDeadline: 10m
  unknownManualReviewAfter: 1h
  worker:
    pollInterval: 5s
    leaseDuration: 30s
    batchSize: 50
    maxClaimCount: 3
```

When enabled, configuration validation requires authorization and execution
durations between 1 second and 24 hours, a manual-review threshold from 1
minute to 7 days that is not shorter than the execution deadline, batches from
1 to 500, and claim counts from 1 to 10. These bounds limit the effect of a
bad rollout value.

## Safe rollout and rollback

1. Run the database migrations and confirm the cutover reconciliation checks.
2. Register the service, HTTP handler, and outbox worker in the application
   container; smoke-test with an invalid request and expect HTTP 422, not 404.
3. Enable both `credits.reservationsEnabled` and `creditReservation.enabled`
   in one rolling deployment.
4. Before disabling either gate, drain or explicitly reconcile ACTIVE,
   EXECUTING, and UNKNOWN reservations. Disabling a route must not be used to
   infer that an in-flight provider call was not executed.

The lifecycle deadline values are deployment policy. Callers still provide the
reservation expiry and execution deadlines to the service; do not change the
configured values to silently rewrite existing reservations.

## Required prechecks and evidence

Before enabling the gate, record the migration version, a production database
backup location, the deployed OpenMeter revision, and the approving operator.
Confirm that the runtime bundle has a non-empty worker owner ID and that the
startup check would reject an enabled configuration without the injected HTTP
handler. Keep provider request IDs, idempotency keys, payload hashes, and the
provider's execution or non-execution evidence with the incident record;
UNKNOWN is not permission to release a hold.

```sql
-- No aged ACTIVE/EXECUTING/UNKNOWN rows may be ignored during cutover.
SELECT state, count(*), min(created_at)
  FROM credit_reservations
 GROUP BY state;

-- Projection delivery must be understood before route enablement.
SELECT count(*) AS unpublished, count(*) FILTER (WHERE dead_lettered) AS dead_lettered
  FROM credit_reservation_outboxes
 WHERE published = false;
```

Dead-letter rows are not deleted as a recovery action. Preserve the row and
its lease/claim history, correct the collector dependency, then reset it only
under an incident ticket so the original outbox ID remains the downstream
deduplication key.

## Deletion gate

Do not delete the legacy path, reservation tables, or outbox tables until a
signed reconciliation demonstrates zero unresolved ACTIVE, EXECUTING, UNKNOWN,
and MANUAL_REVIEW reservations; zero unexplained unpublished/dead-lettered
outbox rows; and a retained export of the command/evidence records. The
cutover runbook remains the authoritative detailed migration and rollback
procedure.

## Metrics and alerts

The outbox worker emits
`openmeter.credit_reservation.outbox.processed` when it is constructed with an
OpenTelemetry meter. Its only attribute is the fixed-cardinality `outcome`
enum: `published`, `retry`, `release_failed`, `ack_failed`, `dead_lettered`,
or `dead_letter_failed`. It deliberately never labels by reservation,
customer, namespace, event, or provider.

The lifecycle service emits bounded command (`operation`, `outcome`) and
transition (`state`) counters plus ceiling and enterprise-hold credit
histograms. Backlog is intentionally not emitted by the service because the
repository does not expose an aggregate-count contract; use the precheck query
above until that read model is added.

Alert on a non-zero rate of `dead_lettered`, `dead_letter_failed`, or
`release_failed` outcomes, and investigate a sustained `retry` rate. Pair
those alerts with the outbox backlog and reservation-state reconciliation
queries in the cutover runbook; a delivery failure may delay projection but
must never cause the ledger settlement to run a second time.
