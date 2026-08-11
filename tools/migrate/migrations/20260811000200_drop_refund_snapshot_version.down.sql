-- Restore the removed legacy column only when rolling back the cleanup migration.
ALTER TABLE "refund_requests"
  ADD COLUMN IF NOT EXISTS "snapshot_version" character varying;
