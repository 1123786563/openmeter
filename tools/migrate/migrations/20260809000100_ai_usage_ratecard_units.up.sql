-- Add the unit conversion fields required by the AI usage ratecard Ent schema.
ALTER TABLE "ai_usage_ratecard_entries"
  ADD COLUMN IF NOT EXISTS "credits_per_unit" bigint NOT NULL DEFAULT 1,
  ADD COLUMN IF NOT EXISTS "unit_size" bigint NOT NULL DEFAULT 1;

