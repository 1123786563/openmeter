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

## Metrics and alerts

The outbox worker emits
`openmeter.credit_reservation.outbox.processed` when it is constructed with an
OpenTelemetry meter. Its only attribute is the fixed-cardinality `outcome`
enum: `published`, `retry`, `release_failed`, `ack_failed`, `dead_lettered`,
or `dead_letter_failed`. It deliberately never labels by reservation,
customer, namespace, event, or provider.

Alert on a non-zero rate of `dead_lettered`, `dead_letter_failed`, or
`release_failed` outcomes, and investigate a sustained `retry` rate. Pair
those alerts with the outbox backlog and reservation-state reconciliation
queries in the cutover runbook; a delivery failure may delay projection but
must never cause the ledger settlement to run a second time.
