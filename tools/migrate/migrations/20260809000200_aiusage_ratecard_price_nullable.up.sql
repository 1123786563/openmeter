-- The AI usage ratecard bootstrap is credit-based and does not provide the
-- legacy fiat price field. Keep the Ent schema and config seed compatible.
ALTER TABLE "ai_usage_ratecard_entries"
  ALTER COLUMN "price_per_unit_cny" DROP NOT NULL;

