DROP INDEX IF EXISTS creditreservationoutbox_namespace_aggregate_id;
DROP INDEX IF EXISTS creditreservationoutbox_namespace_published_dead_lettered_leased_until;
DROP INDEX IF EXISTS creditreservationoutbox_namespace_event_id;
DROP INDEX IF EXISTS creditreservationoutbox_namespace;
DROP INDEX IF EXISTS creditreservationoutbox_id;
DROP TABLE IF EXISTS credit_reservation_outboxes;

DROP INDEX IF EXISTS creditcharge_namespace_reservation_id;
DROP INDEX IF EXISTS creditcharge_namespace_idempotency_key;
DROP INDEX IF EXISTS creditcharge_namespace;
DROP INDEX IF EXISTS creditcharge_id;
DROP TABLE IF EXISTS credit_charges;

DROP INDEX IF EXISTS credit_reservation_active_holds;
DROP INDEX IF EXISTS credit_reservation_call;
DROP INDEX IF EXISTS credit_reservation_idempotency;
DROP INDEX IF EXISTS creditreservation_namespace;
DROP INDEX IF EXISTS creditreservation_id;
DROP TABLE IF EXISTS credit_reservations;

DROP TYPE IF EXISTS credit_charge_state;
