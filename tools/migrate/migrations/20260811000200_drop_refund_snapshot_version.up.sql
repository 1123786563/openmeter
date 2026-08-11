-- Runtime Authorization snapshot versions are retired with the AIUsage flow.
ALTER TABLE "refund_requests" DROP COLUMN IF EXISTS "snapshot_version";
