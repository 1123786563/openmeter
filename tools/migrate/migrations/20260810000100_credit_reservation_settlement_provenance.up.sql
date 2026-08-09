ALTER TABLE credit_reservations
  ADD COLUMN settlement_idempotency_key varchar NOT NULL DEFAULT '',
  ADD COLUMN settlement_payload_hash varchar NOT NULL DEFAULT '';

ALTER TABLE credit_charges
  ADD COLUMN rate_version varchar NOT NULL DEFAULT '',
  ADD COLUMN settlement_allocations jsonb NOT NULL DEFAULT '[]'::jsonb;
