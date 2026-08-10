CREATE TYPE credit_charge_state AS ENUM ('SETTLED', 'REVERSED');

CREATE TABLE credit_reservations (
  id char(26) NOT NULL,
  namespace varchar NOT NULL,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL,
  deleted_at timestamptz NULL,
  customer_id varchar NOT NULL,
  subject_id varchar NOT NULL,
  client_call_id varchar NOT NULL,
  operation varchar NOT NULL,
  idempotency_key varchar NOT NULL,
  payload_hash varchar NOT NULL,
  currency jsonb NOT NULL,
  custom_currency_id varchar NULL,
  estimated_lines jsonb NOT NULL,
  rated_lines jsonb NOT NULL,
  actual_lines jsonb NULL,
  ceiling_credits bigint NOT NULL DEFAULT 0,
  prepaid_hold bigint NOT NULL DEFAULT 0,
  enterprise_hold bigint NOT NULL DEFAULT 0,
  settled_credits bigint NOT NULL DEFAULT 0,
  rate_version varchar NOT NULL DEFAULT '',
  state varchar NOT NULL,
  provider varchar NOT NULL DEFAULT '',
  model varchar NOT NULL DEFAULT '',
  request_id varchar NOT NULL DEFAULT '',
  authorization_expires_at timestamptz NULL,
  execution_deadline timestamptz NULL,
  hold_ledger_group_id varchar NOT NULL DEFAULT '',
  settlement_ledger_group_id varchar NOT NULL DEFAULT '',
  release_ledger_group_id varchar NOT NULL DEFAULT '',
  usage_event_id varchar NOT NULL DEFAULT '',
  PRIMARY KEY (id)
);
CREATE UNIQUE INDEX creditreservation_id ON credit_reservations (id);
CREATE INDEX creditreservation_namespace ON credit_reservations (namespace);
CREATE UNIQUE INDEX credit_reservation_idempotency ON credit_reservations (namespace, idempotency_key);
CREATE UNIQUE INDEX credit_reservation_call ON credit_reservations (namespace, client_call_id);
CREATE INDEX credit_reservation_active_holds
  ON credit_reservations (namespace, customer_id, custom_currency_id, state)
  WHERE state IN ('ACTIVE', 'EXECUTING', 'UNKNOWN', 'MANUAL_REVIEW');

CREATE TABLE credit_charges (
  id char(26) NOT NULL,
  namespace varchar NOT NULL,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL,
  deleted_at timestamptz NULL,
  reservation_id varchar NULL,
  customer_id varchar NOT NULL,
  subject_id varchar NOT NULL,
  operation varchar NOT NULL,
  idempotency_key varchar NOT NULL,
  payload_hash varchar NOT NULL,
  currency jsonb NOT NULL,
  custom_currency_id varchar NULL,
  rated_lines jsonb NOT NULL,
  amount bigint NOT NULL,
  state credit_charge_state NOT NULL,
  settlement_ledger_group_id varchar NOT NULL DEFAULT '',
  reversal_ledger_group_id varchar NOT NULL DEFAULT '',
  usage_event_id varchar NOT NULL DEFAULT '',
  PRIMARY KEY (id)
);
CREATE UNIQUE INDEX creditcharge_id ON credit_charges (id);
CREATE INDEX creditcharge_namespace ON credit_charges (namespace);
CREATE UNIQUE INDEX creditcharge_namespace_idempotency_key ON credit_charges (namespace, idempotency_key);
CREATE INDEX creditcharge_namespace_reservation_id ON credit_charges (namespace, reservation_id);

CREATE TABLE credit_reservation_outboxes (
  id char(26) NOT NULL,
  namespace varchar NOT NULL,
  created_at timestamptz NOT NULL,
  event_id varchar NOT NULL,
  aggregate_type varchar NOT NULL,
  aggregate_id varchar NOT NULL,
  event_type varchar NOT NULL,
  payload jsonb NOT NULL,
  published boolean NOT NULL DEFAULT false,
  published_at timestamptz NULL,
  owner varchar NOT NULL DEFAULT '',
  claim_count integer NOT NULL DEFAULT 0,
  leased_until timestamptz NULL,
  dead_lettered boolean NOT NULL DEFAULT false,
  dead_letter_reason varchar NOT NULL DEFAULT '',
  PRIMARY KEY (id)
);
CREATE UNIQUE INDEX creditreservationoutbox_id ON credit_reservation_outboxes (id);
CREATE INDEX creditreservationoutbox_namespace ON credit_reservation_outboxes (namespace);
CREATE UNIQUE INDEX creditreservationoutbox_namespace_event_id ON credit_reservation_outboxes (namespace, event_id);
CREATE INDEX creditreservationoutbox_namespace_published_dead_lettered_leased_until
  ON credit_reservation_outboxes (namespace, published, dead_lettered, leased_until);
CREATE INDEX creditreservationoutbox_namespace_aggregate_id ON credit_reservation_outboxes (namespace, aggregate_id);
