ALTER TABLE credit_charges
  DROP COLUMN reversal_payload_hash,
  DROP COLUMN reversal_idempotency_key;
