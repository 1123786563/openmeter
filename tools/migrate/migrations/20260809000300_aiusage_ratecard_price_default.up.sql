-- The current credit-based seed omits the legacy fiat price field. Backfill
-- existing rows and provide a stable zero default so Ent list/create calls do
-- not scan nullable values or fail inserts.
UPDATE "ai_usage_ratecard_entries"
SET "price_per_unit_cny" = 0
WHERE "price_per_unit_cny" IS NULL;

ALTER TABLE "ai_usage_ratecard_entries"
  ALTER COLUMN "price_per_unit_cny" SET DEFAULT 0,
  ALTER COLUMN "price_per_unit_cny" SET NOT NULL;

