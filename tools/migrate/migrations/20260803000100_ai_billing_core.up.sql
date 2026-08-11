-- AI Billing core tables: 9 entities for atomic usage batch persistence.
-- Batch, line items, and rating snapshots are immutable (no created_at/updated_at/deleted_at).

-- Enum types must be created before tables that reference them.
CREATE TYPE ai_usage_settlement_scope AS ENUM ('shadow', 'formal');
CREATE TYPE customer_ai_rate_package_status AS ENUM ('draft', 'active', 'archived');

-- create "ai_usage_watermarks" table
CREATE TABLE "ai_usage_watermarks" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "subject_id" character varying NOT NULL,
  "customer_id" character varying NOT NULL,
  "covered_seq" bigint DEFAULT 0,
  PRIMARY KEY ("id"),
  CHECK ("covered_seq" >= 0)
);
CREATE UNIQUE INDEX "aiusagewatermark_id" ON "ai_usage_watermarks" ("id");
CREATE INDEX "aiusagewatermark_namespace" ON "ai_usage_watermarks" ("namespace");
CREATE UNIQUE INDEX "aiusagewatermark_namespace_subject_id" ON "ai_usage_watermarks" ("namespace", "subject_id");
CREATE INDEX "aiusagewatermark_namespace_customer_id" ON "ai_usage_watermarks" ("namespace", "customer_id");

-- create "ai_usage_batches" table (immutable — no TimeMixin columns)
CREATE TABLE "ai_usage_batches" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "annotations" jsonb NULL,
  "customer_id" character varying NOT NULL,
  "subject_id" character varying NOT NULL,
  "usage_batch_id" character varying NOT NULL,
  "tenant_seq" bigint NOT NULL,
  "occurred_at" timestamptz NOT NULL,
  "reservation_id" character varying NULL,
  "ceiling_credits" bigint NULL,
  "rate_version" character varying DEFAULT '',
  "billing_mode" character varying NOT NULL,
  "payload_hash" character varying NOT NULL,
  "status" character varying DEFAULT 'pending',
  "total_credits" bigint DEFAULT 0,
  "covered_tenant_seq" bigint DEFAULT 0,
  "settlement_scope" ai_usage_settlement_scope DEFAULT 'formal',
  PRIMARY KEY ("id"),
  CONSTRAINT "aiusagebatch_settlement_scope" CHECK ("settlement_scope" in ('shadow', 'formal')),
  CONSTRAINT "aiusagebatch_tenant_seq_check" CHECK ("tenant_seq" >= 0),
  CONSTRAINT "aiusagebatch_total_credits_check" CHECK ("total_credits" >= 0),
  CONSTRAINT "aiusagebatch_covered_tenant_seq_check" CHECK ("covered_tenant_seq" >= 0)
);
CREATE UNIQUE INDEX "aiusagebatch_id" ON "ai_usage_batches" ("id");
CREATE INDEX "aiusagebatch_namespace" ON "ai_usage_batches" ("namespace");
CREATE INDEX "aiusagebatch_annotations" ON "ai_usage_batches" USING GIN ("annotations");
CREATE UNIQUE INDEX "aiusagebatch_namespace_customer_id_usage_batch_id" ON "ai_usage_batches" ("namespace", "customer_id", "usage_batch_id");
CREATE UNIQUE INDEX "aiusagebatch_namespace_subject_id_tenant_seq" ON "ai_usage_batches" ("namespace", "subject_id", "tenant_seq");
CREATE INDEX "aiusagebatch_namespace_customer_id_tenant_seq" ON "ai_usage_batches" ("namespace", "customer_id", "tenant_seq");
CREATE INDEX "aiusagebatch_namespace_customer_id_status" ON "ai_usage_batches" ("namespace", "customer_id", "status");

-- create "ai_usage_line_items" table (immutable — no TimeMixin columns)
CREATE TABLE "ai_usage_line_items" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "annotations" jsonb NULL,
  "resource_code" character varying NOT NULL,
  "quantity" bigint NOT NULL,
  "provider" character varying DEFAULT '',
  "model" character varying DEFAULT '',
  "provider_managed" boolean DEFAULT true,
  "dimensions" jsonb NULL,
  "ai_usage_batch_line_items" character(26) NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "aiusagelineitem_quantity_check" CHECK ("quantity" >= 0)
);
CREATE UNIQUE INDEX "aiusagelineitem_id" ON "ai_usage_line_items" ("id");
CREATE INDEX "aiusagelineitem_namespace" ON "ai_usage_line_items" ("namespace");
CREATE INDEX "aiusagelineitem_annotations" ON "ai_usage_line_items" USING GIN ("annotations");
ALTER TABLE "ai_usage_line_items" ADD CONSTRAINT "ai_usage_line_items_ai_usage_batches_line_items" FOREIGN KEY ("ai_usage_batch_line_items") REFERENCES "ai_usage_batches" ("id") ON DELETE NO ACTION;

-- create "ai_usage_rating_snapshots" table (immutable — no TimeMixin columns)
CREATE TABLE "ai_usage_rating_snapshots" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "annotations" jsonb NULL,
  "resource_code" character varying NOT NULL,
  "cost_currency" character varying DEFAULT 'USD',
  "cost_amount" numeric NOT NULL,
  "cost_source" character varying DEFAULT '',
  "sales_currency" character varying DEFAULT 'CNY',
  "sales_amount" numeric NOT NULL,
  "rate_card_version" character varying DEFAULT '',
  "credits" bigint DEFAULT 0,
  "ai_usage_batch_rating_snapshots" character(26) NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "aiusageratingsnapshot_credits_check" CHECK ("credits" >= 0)
);
CREATE UNIQUE INDEX "aiusageratingsnapshot_id" ON "ai_usage_rating_snapshots" ("id");
CREATE INDEX "aiusageratingsnapshot_namespace" ON "ai_usage_rating_snapshots" ("namespace");
CREATE INDEX "aiusageratingsnapshot_annotations" ON "ai_usage_rating_snapshots" USING GIN ("annotations");
ALTER TABLE "ai_usage_rating_snapshots" ADD CONSTRAINT "ai_usage_rating_snapshots_ai_usage_batches_rating_snapshots" FOREIGN KEY ("ai_usage_batch_rating_snapshots") REFERENCES "ai_usage_batches" ("id") ON DELETE NO ACTION;

-- create "ai_usage_allocations" table
CREATE TABLE "ai_usage_allocations" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "customer_id" character varying NOT NULL,
  "subject_id" character varying NOT NULL,
  "grant_id" character(26) NOT NULL,
  "amount" numeric NOT NULL,
  "priority" smallint DEFAULT 0,
  "funding_source" character varying DEFAULT '',
  "ai_usage_batch_allocations" character(26) NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "aiusageallocation_priority_check" CHECK ("priority" >= 0)
);
CREATE UNIQUE INDEX "aiusageallocation_id" ON "ai_usage_allocations" ("id");
CREATE INDEX "aiusageallocation_namespace" ON "ai_usage_allocations" ("namespace");
CREATE INDEX "aiusageallocation_namespace_customer_id" ON "ai_usage_allocations" ("namespace", "customer_id");
ALTER TABLE "ai_usage_allocations" ADD CONSTRAINT "ai_usage_allocations_ai_usage_batches_allocations" FOREIGN KEY ("ai_usage_batch_allocations") REFERENCES "ai_usage_batches" ("id") ON DELETE NO ACTION;

-- create "ai_usage_outboxes" table
CREATE TABLE "ai_usage_outboxes" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "customer_id" character varying NOT NULL,
  "subject_id" character varying NOT NULL,
  "event_type" character varying NOT NULL,
  "payload" jsonb NOT NULL,
  "published" boolean DEFAULT false,
  "published_at" timestamptz NULL,
  "owner" character varying DEFAULT '',
  "claim_count" bigint DEFAULT 0,
  "leased_until" timestamptz NULL,
  "dead_lettered" boolean DEFAULT false,
  "dead_letter_reason" character varying DEFAULT '',
  "ai_usage_batch_outbox_events" character(26) NULL,
  PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX "aiusageoutbox_id" ON "ai_usage_outboxes" ("id");
CREATE INDEX "aiusageoutbox_namespace" ON "ai_usage_outboxes" ("namespace");
CREATE INDEX "aiusageoutbox_namespace_published" ON "ai_usage_outboxes" ("namespace", "published");
CREATE INDEX "aiusageoutbox_namespace_customer_id" ON "ai_usage_outboxes" ("namespace", "customer_id");
CREATE INDEX "aiusageoutbox_namespace_published_dead_lettered_leased_until" ON "ai_usage_outboxes" ("namespace", "published", "dead_lettered", "leased_until");
ALTER TABLE "ai_usage_outboxes" ADD CONSTRAINT "ai_usage_outboxes_ai_usage_batches_outbox_events" FOREIGN KEY ("ai_usage_batch_outbox_events") REFERENCES "ai_usage_batches" ("id") ON DELETE NO ACTION;

-- create "ai_usage_ratecard_entries" table
CREATE TABLE "ai_usage_ratecard_entries" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "annotations" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "customer_id" character varying NULL,
  "resource_code" character varying NOT NULL,
  "provider" character varying NULL,
  "model" character varying NULL,
  "price_per_unit_cny" numeric NOT NULL,
  "credit_rate" bigint DEFAULT 1000,
  "effective_from" timestamptz NOT NULL,
  "effective_to" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "aiusageratecardentry_credit_rate_check" CHECK ("credit_rate" >= 0)
);
CREATE UNIQUE INDEX "aiusageratecardentry_id" ON "ai_usage_ratecard_entries" ("id");
CREATE INDEX "aiusageratecardentry_namespace" ON "ai_usage_ratecard_entries" ("namespace");
CREATE INDEX "aiusageratecardentry_annotations" ON "ai_usage_ratecard_entries" USING GIN ("annotations");
CREATE UNIQUE INDEX "aiusageratecardentry_namespace_customer_id_resource_code_provider_model_effective_from" ON "ai_usage_ratecard_entries" ("namespace", "customer_id", "resource_code", "provider", "model", "effective_from") WHERE "deleted_at" IS NULL;
CREATE INDEX "aiusageratecardentry_namespace_resource_code" ON "ai_usage_ratecard_entries" ("namespace", "resource_code") WHERE "deleted_at" IS NULL;

-- create "customer_ai_rate_packages" table
CREATE TABLE "customer_ai_rate_packages" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "annotations" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "customer_id" character varying NOT NULL,
  "package_code" character varying NOT NULL,
  "name" character varying NOT NULL,
  "description" character varying NULL,
  "status" customer_ai_rate_package_status DEFAULT 'draft',
  "effective_from" timestamptz NOT NULL,
  "effective_to" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "customerairatepackage_status" CHECK ("status" in ('draft', 'active', 'archived'))
);
CREATE UNIQUE INDEX "customerairatepackage_id" ON "customer_ai_rate_packages" ("id");
CREATE INDEX "customerairatepackage_namespace" ON "customer_ai_rate_packages" ("namespace");
CREATE INDEX "customerairatepackage_annotations" ON "customer_ai_rate_packages" USING GIN ("annotations");
CREATE UNIQUE INDEX "customerairatepackage_namespace_customer_id_package_code" ON "customer_ai_rate_packages" ("namespace", "customer_id", "package_code") WHERE "deleted_at" IS NULL;
CREATE INDEX "customerairatepackage_namespace_customer_id_status" ON "customer_ai_rate_packages" ("namespace", "customer_id", "status") WHERE "deleted_at" IS NULL;

-- create "manual_resource_costs" table
CREATE TABLE "manual_resource_costs" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "annotations" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "resource_code" character varying NOT NULL,
  "provider" character varying NULL,
  "model" character varying NULL,
  "cost_currency" character varying DEFAULT 'USD',
  "cost_amount" numeric NOT NULL,
  "source" character varying DEFAULT 'manual',
  "effective_from" timestamptz NOT NULL,
  "effective_to" timestamptz NULL,
  PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX "manualresourcecost_id" ON "manual_resource_costs" ("id");
CREATE INDEX "manualresourcecost_namespace" ON "manual_resource_costs" ("namespace");
CREATE INDEX "manualresourcecost_annotations" ON "manual_resource_costs" USING GIN ("annotations");
CREATE UNIQUE INDEX "manualresourcecost_namespace_resource_code_provider_model_effective_from" ON "manual_resource_costs" ("namespace", "resource_code", "provider", "model", "effective_from") WHERE "deleted_at" IS NULL;
CREATE INDEX "manualresourcecost_namespace_resource_code" ON "manual_resource_costs" ("namespace", "resource_code") WHERE "deleted_at" IS NULL;
