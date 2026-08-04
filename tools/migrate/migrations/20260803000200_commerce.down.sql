-- reverse: drop tables, indexes, and enum types (child tables first)

-- drop refund sum trigger and function
DROP TRIGGER IF EXISTS commerce_refund_sum_check ON refund_requests;
DROP FUNCTION IF EXISTS commerce_check_refund_sum();

-- drop "external_invoice_refs"
DROP INDEX IF EXISTS "externalinvoiceref_namespace_receivable_period_id";
DROP INDEX IF EXISTS "externalinvoiceref_namespace_invoice_number";
DROP INDEX IF EXISTS "externalinvoiceref_namespace";
DROP INDEX IF EXISTS "externalinvoiceref_id";
DROP TABLE IF EXISTS "external_invoice_refs";

-- drop "offline_payments"
DROP INDEX IF EXISTS "offlinepayment_namespace_confirmed_at";
DROP INDEX IF EXISTS "offlinepayment_namespace_receivable_account_id";
DROP INDEX IF EXISTS "offlinepayment_namespace";
DROP INDEX IF EXISTS "offlinepayment_id";
DROP TABLE IF EXISTS "offline_payments";

-- drop "receivable_periods"
DROP INDEX IF EXISTS "receivableperiod_namespace_status";
DROP INDEX IF EXISTS "receivableperiod_namespace_receivable_account_id_period_start";
DROP INDEX IF EXISTS "receivableperiod_namespace";
DROP INDEX IF EXISTS "receivableperiod_id";
DROP TABLE IF EXISTS "receivable_periods";

-- drop "receivable_accounts"
DROP INDEX IF EXISTS "receivableaccount_namespace_customer_id";
DROP INDEX IF EXISTS "receivableaccount_annotations";
DROP INDEX IF EXISTS "receivableaccount_namespace";
DROP INDEX IF EXISTS "receivableaccount_id";
DROP TABLE IF EXISTS "receivable_accounts";

-- drop "refund_facts"
DROP INDEX IF EXISTS "refundfact_namespace";
DROP INDEX IF EXISTS "refundfact_id";
DROP TABLE IF EXISTS "refund_facts";

-- drop "refund_requests"
DROP INDEX IF EXISTS "refundrequest_namespace_customer_id_status";
DROP INDEX IF EXISTS "refundrequest_namespace_commerce_order_id";
DROP INDEX IF EXISTS "refundrequest_namespace_customer_id_idempotency_key";
DROP INDEX IF EXISTS "refundrequest_namespace";
DROP INDEX IF EXISTS "refundrequest_id";
DROP TABLE IF EXISTS "refund_requests";

-- drop "fulfillments"
DROP INDEX IF EXISTS "fulfillment_namespace_customer_id_status";
DROP INDEX IF EXISTS "fulfillment_namespace_commerce_order_id";
DROP INDEX IF EXISTS "fulfillment_namespace";
DROP INDEX IF EXISTS "fulfillment_id";
DROP TABLE IF EXISTS "fulfillments";

-- drop "payment_facts"
DROP INDEX IF EXISTS "paymentfact_namespace";
DROP INDEX IF EXISTS "paymentfact_id";
DROP TABLE IF EXISTS "payment_facts";

-- drop "payment_attempts"
DROP INDEX IF EXISTS "paymentattempt_namespace_customer_id_status";
DROP INDEX IF EXISTS "paymentattempt_namespace_commerce_order_id";
DROP INDEX IF EXISTS "paymentattempt_namespace_provider_provider_payment_id";
DROP INDEX IF EXISTS "paymentattempt_namespace_provider_provider_order_id";
DROP INDEX IF EXISTS "paymentattempt_namespace_customer_id_idempotency_key";
DROP INDEX IF EXISTS "paymentattempt_namespace";
DROP INDEX IF EXISTS "paymentattempt_id";
DROP TABLE IF EXISTS "payment_attempts";

-- drop "commerce_order_lines"
DROP INDEX IF EXISTS "commerceorderline_namespace_product_id";
DROP INDEX IF EXISTS "commerceorderline_namespace";
DROP INDEX IF EXISTS "commerceorderline_id";
DROP TABLE IF EXISTS "commerce_order_lines";

-- drop "commerce_orders"
DROP INDEX IF EXISTS "commerceorder_namespace_customer_id_status";
DROP INDEX IF EXISTS "commerceorder_namespace_customer_id_idempotency_key";
DROP INDEX IF EXISTS "commerceorder_public_id";
DROP INDEX IF EXISTS "commerceorder_namespace";
DROP INDEX IF EXISTS "commerceorder_id";
DROP TABLE IF EXISTS "commerce_orders";

-- drop "commerce_products"
DROP INDEX IF EXISTS "commerceproduct_namespace_kind";
DROP INDEX IF EXISTS "commerceproduct_namespace_sku";
DROP INDEX IF EXISTS "commerceproduct_annotations";
DROP INDEX IF EXISTS "commerceproduct_namespace";
DROP INDEX IF EXISTS "commerceproduct_id";
DROP TABLE IF EXISTS "commerce_products";

-- drop enum types
DROP TYPE IF EXISTS "external_invoice_ref_status";
DROP TYPE IF EXISTS "receivable_period_status";
DROP TYPE IF EXISTS "refund_fact_provider";
DROP TYPE IF EXISTS "refund_request_status";
DROP TYPE IF EXISTS "fulfillment_status";
DROP TYPE IF EXISTS "payment_fact_provider";
DROP TYPE IF EXISTS "payment_attempt_status";
DROP TYPE IF EXISTS "payment_attempt_provider";
DROP TYPE IF EXISTS "commerce_order_status";
DROP TYPE IF EXISTS "commerce_order_kind";
DROP TYPE IF EXISTS "commerce_product_kind";
