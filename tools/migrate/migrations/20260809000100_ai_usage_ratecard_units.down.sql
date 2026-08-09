-- Remove only the fields introduced by 20260809000100_ai_usage_ratecard_units.
ALTER TABLE "ai_usage_ratecard_entries"
  DROP COLUMN IF EXISTS "unit_size",
  DROP COLUMN IF EXISTS "credits_per_unit";

