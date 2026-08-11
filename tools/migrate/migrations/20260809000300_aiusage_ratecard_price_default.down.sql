-- Return to the nullable compatibility state used before this backfill.
ALTER TABLE "ai_usage_ratecard_entries"
  ALTER COLUMN "price_per_unit_cny" DROP NOT NULL,
  ALTER COLUMN "price_per_unit_cny" DROP DEFAULT;

