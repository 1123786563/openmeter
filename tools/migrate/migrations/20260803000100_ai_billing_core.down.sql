-- reverse: drop tables and indexes (child tables first, then parent)

-- drop "customer_ai_rate_packages"
DROP INDEX IF EXISTS "customerairatepackage_namespace_customer_id_status";
DROP INDEX IF EXISTS "customerairatepackage_namespace_customer_id_package_code";
DROP INDEX IF EXISTS "customerairatepackage_annotations";
DROP INDEX IF EXISTS "customerairatepackage_namespace";
DROP INDEX IF EXISTS "customerairatepackage_id";
DROP TABLE IF EXISTS "customer_ai_rate_packages";

-- drop "ai_usage_ratecard_entries"
DROP INDEX IF EXISTS "aiusageratecardentry_namespace_resource_code";
DROP INDEX IF EXISTS "aiusageratecardentry_namespace_customer_id_resource_code_provider_model_effective_from";
DROP INDEX IF EXISTS "aiusageratecardentry_annotations";
DROP INDEX IF EXISTS "aiusageratecardentry_namespace";
DROP INDEX IF EXISTS "aiusageratecardentry_id";
DROP TABLE IF EXISTS "ai_usage_ratecard_entries";

-- drop "ai_usage_outboxes"
DROP INDEX IF EXISTS "aiusageoutbox_namespace_customer_id";
DROP INDEX IF EXISTS "aiusageoutbox_namespace_published";
DROP INDEX IF EXISTS "aiusageoutbox_namespace";
DROP INDEX IF EXISTS "aiusageoutbox_id";
DROP TABLE IF EXISTS "ai_usage_outboxes";

-- drop "ai_usage_allocations"
DROP INDEX IF EXISTS "aiusageallocation_namespace_customer_id";
DROP INDEX IF EXISTS "aiusageallocation_namespace";
DROP INDEX IF EXISTS "aiusageallocation_id";
DROP TABLE IF EXISTS "ai_usage_allocations";

-- drop "ai_usage_rating_snapshots"
DROP INDEX IF EXISTS "aiusageratingsnapshot_annotations";
DROP INDEX IF EXISTS "aiusageratingsnapshot_namespace";
DROP INDEX IF EXISTS "aiusageratingsnapshot_id";
DROP TABLE IF EXISTS "ai_usage_rating_snapshots";

-- drop "ai_usage_line_items"
DROP INDEX IF EXISTS "aiusagelineitem_annotations";
DROP INDEX IF EXISTS "aiusagelineitem_namespace";
DROP INDEX IF EXISTS "aiusagelineitem_id";
DROP TABLE IF EXISTS "ai_usage_line_items";

-- drop "ai_usage_batches"
DROP INDEX IF EXISTS "aiusagebatch_namespace_customer_id_status";
DROP INDEX IF EXISTS "aiusagebatch_namespace_customer_id_tenant_seq";
DROP INDEX IF EXISTS "aiusagebatch_namespace_subject_id_tenant_seq";
DROP INDEX IF EXISTS "aiusagebatch_namespace_usage_batch_id";
DROP INDEX IF EXISTS "aiusagebatch_annotations";
DROP INDEX IF EXISTS "aiusagebatch_namespace";
DROP INDEX IF EXISTS "aiusagebatch_id";
DROP TABLE IF EXISTS "ai_usage_batches";

-- drop "ai_usage_watermarks"
DROP INDEX IF EXISTS "aiusagewatermark_namespace_customer_id";
DROP INDEX IF EXISTS "aiusagewatermark_namespace_subject_id";
DROP INDEX IF EXISTS "aiusagewatermark_namespace";
DROP INDEX IF EXISTS "aiusagewatermark_id";
DROP TABLE IF EXISTS "ai_usage_watermarks";

-- drop enum types
DROP TYPE IF EXISTS "customer_ai_rate_package_status";
DROP TYPE IF EXISTS "ai_usage_settlement_scope";
