CREATE TABLE credit_reservation_commands (
  id char(26) NOT NULL,
  namespace varchar NOT NULL,
  created_at timestamptz NOT NULL,
  reservation_id char(26) NOT NULL,
  command_kind varchar NOT NULL,
  idempotency_key varchar NOT NULL,
  payload_hash varchar NOT NULL,
  PRIMARY KEY (id),
  FOREIGN KEY (reservation_id) REFERENCES credit_reservations(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX credit_reservation_command_identity
  ON credit_reservation_commands(namespace, reservation_id, command_kind, idempotency_key);
