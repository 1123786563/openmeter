-- Commerce Phase 2 tables: 12 entities for orders, payments, refunds, fulfillment,
-- and enterprise receivable accounts.

-- Enum types must be created before tables that reference them.
CREATE TYPE commerce_product_kind AS ENUM ('plan_purchase', 'subscription_renewal', 'wallet_top_up');
CREATE TYPE commerce_order_kind AS ENUM ('plan_purchase', 'subscription_renewal', 'wallet_top_up');
CREATE TYPE commerce_order_status AS ENUM ('created', 'awaiting_payment', 'paid', 'fulfilled', 'cancelled', 'expired', 'refund_pending', 'partially_refunded', 'refunded');
CREATE TYPE payment_attempt_provider AS ENUM ('wechat', 'alipay', 'offline');
CREATE TYPE payment_attempt_status AS ENUM ('created', 'pending', 'succeeded', 'failed', 'closed');
CREATE TYPE payment_fact_provider AS ENUM ('wechat', 'alipay', 'offline');
CREATE TYPE fulfillment_status AS ENUM ('pending', 'processing', 'fulfilled', 'failed');
CREATE TYPE refund_request_status AS ENUM ('pending_fence', 'provider_processing', 'ledger_reversing', 'fulfilled', 'failed');
CREATE TYPE refund_fact_provider AS ENUM ('wechat', 'alipay', 'offline');
CREATE TYPE receivable_period_status AS ENUM ('open', 'closed', 'partially_paid', 'paid', 'overdue');
CREATE TYPE external_invoice_ref_status AS ENUM ('draft', 'issued', 'void', 'paid');

-- create "commerce_products" table
CREATE TABLE "commerce_products" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "annotations" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "sku" character varying NOT NULL,
  "name" character varying NOT NULL,
  "kind" commerce_product_kind NOT NULL,
  "price_cents" bigint NOT NULL,
  "currency" character varying DEFAULT 'CNY',
  "description" character varying NULL,
  "metadata" jsonb NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "commerceproduct_price_cents_check" CHECK ("price_cents" >= 0)
);
CREATE UNIQUE INDEX "commerceproduct_id" ON "commerce_products" ("id");
CREATE INDEX "commerceproduct_namespace" ON "commerce_products" ("namespace");
CREATE INDEX "commerceproduct_annotations" ON "commerce_products" USING GIN ("annotations");
CREATE UNIQUE INDEX "commerceproduct_namespace_sku" ON "commerce_products" ("namespace", "sku");
CREATE INDEX "commerceproduct_namespace_kind" ON "commerce_products" ("namespace", "kind");

-- create "commerce_orders" table
CREATE TABLE "commerce_orders" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "public_id" character(26) NOT NULL,
  "customer_id" character varying NOT NULL,
  "kind" commerce_order_kind NOT NULL,
  "status" commerce_order_status DEFAULT 'created',
  "total_cents" bigint NOT NULL,
  "currency" character varying DEFAULT 'CNY',
  "idempotency_key" character varying NOT NULL,
  "description" character varying NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "commerceorder_total_cents_check" CHECK ("total_cents" >= 0)
);
CREATE UNIQUE INDEX "commerceorder_id" ON "commerce_orders" ("id");
CREATE INDEX "commerceorder_namespace" ON "commerce_orders" ("namespace");
CREATE UNIQUE INDEX "commerceorder_public_id" ON "commerce_orders" ("public_id");
CREATE UNIQUE INDEX "commerceorder_namespace_customer_id_idempotency_key" ON "commerce_orders" ("namespace", "customer_id", "idempotency_key");
CREATE INDEX "commerceorder_namespace_customer_id_status" ON "commerce_orders" ("namespace", "customer_id", "status");

-- create "commerce_order_lines" table (immutable)
CREATE TABLE "commerce_order_lines" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "product_id" character(26) NOT NULL,
  "product_sku" character varying DEFAULT '',
  "product_name" character varying DEFAULT '',
  "quantity" integer NOT NULL,
  "unit_price_cents" bigint NOT NULL,
  "subtotal_cents" bigint NOT NULL,
  "snapshot_data" jsonb,
  "commerce_order_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "commerceorderline_quantity_check" CHECK ("quantity" >= 0),
  CONSTRAINT "commerceorderline_unit_price_cents_check" CHECK ("unit_price_cents" >= 0),
  CONSTRAINT "commerceorderline_subtotal_cents_check" CHECK ("subtotal_cents" >= 0)
);
CREATE UNIQUE INDEX "commerceorderline_id" ON "commerce_order_lines" ("id");
CREATE INDEX "commerceorderline_namespace" ON "commerce_order_lines" ("namespace");
CREATE INDEX "commerceorderline_namespace_product_id" ON "commerce_order_lines" ("namespace", "product_id");
ALTER TABLE "commerce_order_lines" ADD CONSTRAINT "commerce_order_lines_commerce_orders_lines" FOREIGN KEY ("commerce_order_id") REFERENCES "commerce_orders" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;

-- create "payment_attempts" table
CREATE TABLE "payment_attempts" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "customer_id" character varying NOT NULL,
  "provider" payment_attempt_provider NOT NULL,
  "provider_order_id" character varying NULL,
  "provider_payment_id" character varying NULL,
  "status" payment_attempt_status DEFAULT 'created',
  "provider_session_id" character varying NULL,
  "idempotency_key" character varying NOT NULL,
  "amount_cents" bigint NOT NULL,
  "currency" character varying DEFAULT 'CNY',
  "commerce_order_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "paymentattempt_amount_cents_check" CHECK ("amount_cents" >= 0)
);
CREATE UNIQUE INDEX "paymentattempt_id" ON "payment_attempts" ("id");
CREATE INDEX "paymentattempt_namespace" ON "payment_attempts" ("namespace");
CREATE UNIQUE INDEX "paymentattempt_namespace_customer_id_idempotency_key" ON "payment_attempts" ("namespace", "customer_id", "idempotency_key");
CREATE UNIQUE INDEX "paymentattempt_namespace_provider_provider_order_id" ON "payment_attempts" ("namespace", "provider", "provider_order_id") WHERE "provider_order_id" IS NOT NULL;
CREATE UNIQUE INDEX "paymentattempt_namespace_provider_provider_payment_id" ON "payment_attempts" ("namespace", "provider", "provider_payment_id") WHERE "provider_payment_id" IS NOT NULL;
CREATE INDEX "paymentattempt_namespace_commerce_order_id" ON "payment_attempts" ("namespace", "commerce_order_id");
CREATE INDEX "paymentattempt_namespace_customer_id_status" ON "payment_attempts" ("namespace", "customer_id", "status");
ALTER TABLE "payment_attempts" ADD CONSTRAINT "payment_attempts_commerce_orders_payment_attempts" FOREIGN KEY ("commerce_order_id") REFERENCES "commerce_orders" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;

-- create "payment_facts" table (immutable append-only)
CREATE TABLE "payment_facts" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "raw_hash" character varying NOT NULL,
  "provider" payment_fact_provider NOT NULL,
  "signed_payload" jsonb NOT NULL,
  "timestamp" timestamptz NOT NULL,
  "payment_attempt_id" character(26) NOT NULL,
  PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX "paymentfact_id" ON "payment_facts" ("id");
CREATE INDEX "paymentfact_namespace" ON "payment_facts" ("namespace");
ALTER TABLE "payment_facts" ADD CONSTRAINT "payment_facts_payment_attempts_facts" FOREIGN KEY ("payment_attempt_id") REFERENCES "payment_attempts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;

-- create "fulfillments" table
CREATE TABLE "fulfillments" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "customer_id" character varying NOT NULL,
  "status" fulfillment_status DEFAULT 'pending',
  "grant_id" character(26) NULL,
  "credits_granted" bigint DEFAULT 0,
  "fulfilled_at" timestamptz NULL,
  "failure_reason" character varying NULL,
  "commerce_order_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fulfillment_credits_granted_check" CHECK ("credits_granted" >= 0)
);
CREATE UNIQUE INDEX "fulfillment_id" ON "fulfillments" ("id");
CREATE INDEX "fulfillment_namespace" ON "fulfillments" ("namespace");
CREATE UNIQUE INDEX "fulfillment_namespace_commerce_order_id" ON "fulfillments" ("namespace", "commerce_order_id") WHERE "status" = 'fulfilled';
CREATE INDEX "fulfillment_namespace_customer_id_status" ON "fulfillments" ("namespace", "customer_id", "status");
ALTER TABLE "fulfillments" ADD CONSTRAINT "fulfillments_commerce_orders_fulfillments" FOREIGN KEY ("commerce_order_id") REFERENCES "commerce_orders" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;

-- create "refund_requests" table
CREATE TABLE "refund_requests" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "customer_id" character varying NOT NULL,
  "amount_cents" bigint NOT NULL,
  "currency" character varying DEFAULT 'CNY',
  "status" refund_request_status DEFAULT 'pending_fence',
  "reason" character varying NULL,
  "idempotency_key" character varying NOT NULL,
  "commerce_order_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "refundrequest_amount_cents_check" CHECK ("amount_cents" >= 0)
);
CREATE UNIQUE INDEX "refundrequest_id" ON "refund_requests" ("id");
CREATE INDEX "refundrequest_namespace" ON "refund_requests" ("namespace");
CREATE UNIQUE INDEX "refundrequest_namespace_customer_id_idempotency_key" ON "refund_requests" ("namespace", "customer_id", "idempotency_key");
CREATE INDEX "refundrequest_namespace_commerce_order_id" ON "refund_requests" ("namespace", "commerce_order_id");
CREATE INDEX "refundrequest_namespace_customer_id_status" ON "refund_requests" ("namespace", "customer_id", "status");
ALTER TABLE "refund_requests" ADD CONSTRAINT "refund_requests_commerce_orders_refund_requests" FOREIGN KEY ("commerce_order_id") REFERENCES "commerce_orders" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;

-- create "refund_facts" table (immutable append-only)
CREATE TABLE "refund_facts" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "raw_hash" character varying NOT NULL,
  "provider" refund_fact_provider NOT NULL,
  "signed_payload" jsonb NOT NULL,
  "timestamp" timestamptz NOT NULL,
  "refund_request_id" character(26) NOT NULL,
  PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX "refundfact_id" ON "refund_facts" ("id");
CREATE INDEX "refundfact_namespace" ON "refund_facts" ("namespace");
ALTER TABLE "refund_facts" ADD CONSTRAINT "refund_facts_refund_requests_facts" FOREIGN KEY ("refund_request_id") REFERENCES "refund_requests" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;

-- create "receivable_accounts" table
CREATE TABLE "receivable_accounts" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "annotations" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "customer_id" character varying NOT NULL,
  "credit_limit_cents" bigint DEFAULT 0,
  "current_balance_cents" bigint DEFAULT 0,
  "currency" character varying DEFAULT 'CNY',
  PRIMARY KEY ("id"),
  CONSTRAINT "receivableaccount_credit_limit_cents_check" CHECK ("credit_limit_cents" >= 0)
);
CREATE UNIQUE INDEX "receivableaccount_id" ON "receivable_accounts" ("id");
CREATE INDEX "receivableaccount_namespace" ON "receivable_accounts" ("namespace");
CREATE INDEX "receivableaccount_annotations" ON "receivable_accounts" USING GIN ("annotations");
CREATE UNIQUE INDEX "receivableaccount_namespace_customer_id" ON "receivable_accounts" ("namespace", "customer_id");

-- create "receivable_periods" table
CREATE TABLE "receivable_periods" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "status" receivable_period_status DEFAULT 'open',
  "period_start" timestamptz NOT NULL,
  "period_end" timestamptz NOT NULL,
  "total_cents" bigint DEFAULT 0,
  "paid_cents" bigint DEFAULT 0,
  "currency" character varying DEFAULT 'CNY',
  "receivable_account_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "receivableperiod_total_cents_check" CHECK ("total_cents" >= 0),
  CONSTRAINT "receivableperiod_paid_cents_check" CHECK ("paid_cents" >= 0)
);
CREATE UNIQUE INDEX "receivableperiod_id" ON "receivable_periods" ("id");
CREATE INDEX "receivableperiod_namespace" ON "receivable_periods" ("namespace");
CREATE UNIQUE INDEX "receivableperiod_namespace_receivable_account_id_period_start" ON "receivable_periods" ("namespace", "receivable_account_id", "period_start");
CREATE INDEX "receivableperiod_namespace_status" ON "receivable_periods" ("namespace", "status");
ALTER TABLE "receivable_periods" ADD CONSTRAINT "receivable_periods_receivable_accounts_periods" FOREIGN KEY ("receivable_account_id") REFERENCES "receivable_accounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;

-- create "offline_payments" table
CREATE TABLE "offline_payments" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "amount_cents" bigint NOT NULL,
  "currency" character varying DEFAULT 'CNY',
  "confirmed_by" character varying NOT NULL,
  "confirmed_at" timestamptz NOT NULL,
  "reference" character varying NULL,
  "note" character varying NULL,
  "receivable_account_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "offlinepayment_amount_cents_check" CHECK ("amount_cents" >= 0)
);
CREATE UNIQUE INDEX "offlinepayment_id" ON "offline_payments" ("id");
CREATE INDEX "offlinepayment_namespace" ON "offline_payments" ("namespace");
CREATE INDEX "offlinepayment_namespace_receivable_account_id" ON "offline_payments" ("namespace", "receivable_account_id");
CREATE INDEX "offlinepayment_namespace_confirmed_at" ON "offline_payments" ("namespace", "confirmed_at");
ALTER TABLE "offline_payments" ADD CONSTRAINT "offline_payments_receivable_accounts_offline_payments" FOREIGN KEY ("receivable_account_id") REFERENCES "receivable_accounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;

-- create "external_invoice_refs" table
CREATE TABLE "external_invoice_refs" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "invoice_number" character varying NOT NULL,
  "invoice_url" character varying NULL,
  "status" external_invoice_ref_status DEFAULT 'draft',
  "issued_at" timestamptz NULL,
  "receivable_period_id" character(26) NOT NULL,
  PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX "externalinvoiceref_id" ON "external_invoice_refs" ("id");
CREATE INDEX "externalinvoiceref_namespace" ON "external_invoice_refs" ("namespace");
CREATE UNIQUE INDEX "externalinvoiceref_namespace_invoice_number" ON "external_invoice_refs" ("namespace", "invoice_number");
CREATE INDEX "externalinvoiceref_namespace_receivable_period_id" ON "external_invoice_refs" ("namespace", "receivable_period_id");
ALTER TABLE "external_invoice_refs" ADD CONSTRAINT "external_invoice_refs_receivable_periods_invoice_refs" FOREIGN KEY ("receivable_period_id") REFERENCES "receivable_periods" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;

-- refund sum invariant: total non-failed refunds per order must not exceed order total_cents.
CREATE OR REPLACE FUNCTION commerce_check_refund_sum() RETURNS TRIGGER AS $$
DECLARE
    order_total BIGINT;
    existing_refunds BIGINT;
BEGIN
    SELECT total_cents INTO order_total
        FROM commerce_orders WHERE id = NEW.commerce_order_id;
    SELECT COALESCE(SUM(amount_cents), 0) INTO existing_refunds
        FROM refund_requests
        WHERE commerce_order_id = NEW.commerce_order_id
        AND status NOT IN ('failed');
    IF existing_refunds + NEW.amount_cents > order_total THEN
        RAISE EXCEPTION 'refund sum % exceeds order total %',
            existing_refunds + NEW.amount_cents, order_total
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER commerce_refund_sum_check
    BEFORE INSERT ON refund_requests
    FOR EACH ROW EXECUTE FUNCTION commerce_check_refund_sum();
