-- Existing rows are expected to have a legacy fiat price before restoring the
-- old constraint. Fail rather than inventing a price for rows without one.
ALTER TABLE "ai_usage_ratecard_entries"
  ALTER COLUMN "price_per_unit_cny" SET NOT NULL;

