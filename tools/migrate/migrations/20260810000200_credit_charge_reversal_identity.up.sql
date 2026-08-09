ALTER TABLE credit_charges
  ADD COLUMN reversal_idempotency_key varchar NOT NULL DEFAULT '',
  ADD COLUMN reversal_payload_hash varchar NOT NULL DEFAULT '';
