DROP INDEX "paymentfact_namespace_provider_provider_order_id";
DROP INDEX "paymentfact_namespace_provider_provider_event_id";

ALTER TABLE "payment_facts"
  DROP COLUMN "success",
  DROP COLUMN "currency",
  DROP COLUMN "amount_minor",
  DROP COLUMN "application_id",
  DROP COLUMN "merchant_id",
  DROP COLUMN "provider_event_id",
  DROP COLUMN "provider_payment_id",
  DROP COLUMN "provider_order_id";

ALTER TABLE "payment_attempts"
  DROP COLUMN "expected_application_id",
  DROP COLUMN "expected_merchant_id";
