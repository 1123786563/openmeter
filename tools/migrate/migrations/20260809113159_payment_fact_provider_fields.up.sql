-- Extend payment attempts with the merchant/application identity expected by
-- callback verification. Both columns are nullable for historical attempts.
ALTER TABLE "payment_attempts"
  ADD COLUMN "expected_merchant_id" character varying NULL,
  ADD COLUMN "expected_application_id" character varying NULL;

-- Add fact fields as nullable first so the migration is safe when historical
-- payment facts already exist.
ALTER TABLE "payment_facts"
  ADD COLUMN "provider_order_id" character varying NULL,
  ADD COLUMN "provider_payment_id" character varying NULL,
  ADD COLUMN "provider_event_id" character varying NULL,
  ADD COLUMN "merchant_id" character varying NULL,
  ADD COLUMN "application_id" character varying NULL,
  ADD COLUMN "amount_minor" bigint NULL,
  ADD COLUMN "currency" character varying NULL,
  ADD COLUMN "success" boolean NULL;

-- Recover historical verified fields from signed_payload where possible and
-- fall back to the linked attempt. Empty provider order IDs and conservative
-- success=false preserve facts whose older payload did not retain the field.
UPDATE "payment_facts" AS pf
SET
  "provider_order_id" = COALESCE(
    NULLIF(pa."provider_order_id", ''),
    NULLIF(pf."signed_payload" ->> 'provider_order_id', ''),
    NULLIF(pf."signed_payload" ->> 'out_trade_no', ''),
    NULLIF(pf."signed_payload" ->> 'out_biz_no', ''),
    ''
  ),
  "provider_payment_id" = COALESCE(
    NULLIF(pa."provider_payment_id", ''),
    NULLIF(pf."signed_payload" ->> 'provider_payment_id', ''),
    NULLIF(pf."signed_payload" ->> 'transaction_id', ''),
    NULLIF(pf."signed_payload" ->> 'trade_no', '')
  ),
  "merchant_id" = COALESCE(
    NULLIF(pf."signed_payload" ->> 'merchant_id', ''),
    NULLIF(pf."signed_payload" ->> 'mchid', ''),
    NULLIF(pf."signed_payload" ->> 'seller_id', '')
  ),
  "application_id" = COALESCE(
    NULLIF(pf."signed_payload" ->> 'application_id', ''),
    NULLIF(pf."signed_payload" ->> 'appid', ''),
    NULLIF(pf."signed_payload" ->> 'app_id', '')
  ),
  "amount_minor" = CASE
    WHEN pf."signed_payload" ->> 'amount_minor' ~ '^[0-9]+$'
      THEN (pf."signed_payload" ->> 'amount_minor')::bigint
    WHEN pf."signed_payload" #>> '{amount,total}' ~ '^[0-9]+$'
      THEN (pf."signed_payload" #>> '{amount,total}')::bigint
    WHEN pf."signed_payload" ->> 'total_amount' ~ '^[0-9]+([.][0-9]{1,2})?$'
      THEN round((pf."signed_payload" ->> 'total_amount')::numeric * 100)::bigint
    ELSE pa."amount_cents"
  END,
  "currency" = COALESCE(
    NULLIF(pf."signed_payload" ->> 'currency', ''),
    NULLIF(pf."signed_payload" #>> '{amount,currency}', ''),
    NULLIF(pa."currency", ''),
    'CNY'
  ),
  "success" = CASE
    WHEN lower(COALESCE(
      pf."signed_payload" ->> 'success',
      pf."signed_payload" ->> 'trade_state',
      pf."signed_payload" ->> 'trade_status',
      pf."signed_payload" ->> 'status',
      ''
    )) IN ('true', 'success', 'succeeded', 'paid', 'trade_success', 'trade_finished')
      THEN true
    ELSE false
  END
FROM "payment_attempts" AS pa
WHERE pa."id" = pf."payment_attempt_id";

-- Provider event IDs are recovered only when unique for the provider and
-- namespace, so enabling the partial unique index cannot reject old data.
WITH event_candidates AS (
  SELECT
    pf."id",
    pf."namespace",
    pf."provider",
    COALESCE(
      NULLIF(pf."signed_payload" ->> 'provider_event_id', ''),
      NULLIF(pf."signed_payload" ->> 'event_id', ''),
      NULLIF(pf."signed_payload" ->> 'notify_id', '')
    ) AS event_id
  FROM "payment_facts" AS pf
), unique_event_candidates AS (
  SELECT
    "id",
    event_id,
    count(*) OVER (PARTITION BY "namespace", "provider", event_id) AS occurrences
  FROM event_candidates
  WHERE event_id IS NOT NULL
)
UPDATE "payment_facts" AS pf
SET "provider_event_id" = candidates.event_id
FROM unique_event_candidates AS candidates
WHERE candidates."id" = pf."id"
  AND candidates.occurrences = 1;

-- Historical attempts predate explicit callback identity expectations. Use
-- their already-verified facts when available; attempts without a fact remain
-- nullable and continue to use provider-specific verification defaults.
UPDATE "payment_attempts" AS pa
SET
  "expected_merchant_id" = recovered.merchant_id,
  "expected_application_id" = recovered.application_id
FROM (
  SELECT
    "payment_attempt_id",
    max("merchant_id") AS merchant_id,
    max("application_id") AS application_id
  FROM "payment_facts"
  GROUP BY "payment_attempt_id"
  HAVING count(DISTINCT "merchant_id") <= 1
    AND count(DISTINCT "application_id") <= 1
) AS recovered
WHERE recovered."payment_attempt_id" = pa."id";

ALTER TABLE "payment_facts"
  ALTER COLUMN "provider_order_id" SET NOT NULL,
  ALTER COLUMN "amount_minor" SET NOT NULL,
  ALTER COLUMN "currency" SET NOT NULL,
  ALTER COLUMN "success" SET NOT NULL;

CREATE UNIQUE INDEX "paymentfact_namespace_provider_provider_event_id"
  ON "payment_facts" ("namespace", "provider", "provider_event_id")
  WHERE (provider_event_id IS NOT NULL);

CREATE INDEX "paymentfact_namespace_provider_provider_order_id"
  ON "payment_facts" ("namespace", "provider", "provider_order_id");
