ALTER TABLE credit_charges
  DROP COLUMN settlement_allocations,
  DROP COLUMN rate_version;

ALTER TABLE credit_reservations
  DROP COLUMN settlement_payload_hash,
  DROP COLUMN settlement_idempotency_key;
