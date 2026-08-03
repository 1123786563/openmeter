import "../shared/index.tsp";

namespace AIUsage;

/**
 * Determines how a usage batch is charged.
 *
 * - `component`: Per-resource pricing; each line item is rated individually.
 * - `bundle`: Flat ceiling charge; no per-resource breakdown.
 */
@friendlyName("AIUsageBillingMode")
enum BillingMode {
  // Per-resource pricing; each line item is rated individually.
  Component: "component",

  // Flat ceiling charge; no per-resource breakdown.
  Bundle: "bundle",
}

/**
 * Settlement status of a usage batch.
 *
 * - `settled`: Batch processed and credits deducted successfully.
 * - `corrected`: Batch reprocessed via a correction or compensation entry.
 */
@friendlyName("AIUsageBatchStatus")
enum BatchStatus {
  // Batch processed and credits deducted successfully.
  Settled: "settled",

  // Batch reprocessed via a correction or compensation entry.
  Corrected: "corrected",
}

/**
 * One billable resource line within a usage batch creation request.
 */
@friendlyName("AIUsageUsageLineCreate")
model UsageLineCreate {
  /**
   * Resource code identifying the billable resource type.
   *
   * Valid codes include:
   * `chat_input_token`, `chat_output_token`, `chat_cache_read_token`,
   * `chat_cache_write_token`, `chat_reasoning_token`, `embedding_token`,
   * `rerank_call`, `vlm_input_token`, `vlm_output_token`, `vlm_image`,
   * `asr_seconds`, `rag_retrieval`, `doc_parse_page`, `mcp_call`,
   * `web_search`, `agent_run`.
   */
  resource_code: string;

  /**
   * Raw consumption quantity (tokens, calls, seconds, pages, or images).
   */
  @example(1500)
  quantity: int64;

  /**
   * LLM provider (e.g. `openai`, `anthropic`). Required for provider-managed
   * resources.
   */
  provider?: string;

  /**
   * Canonical model identifier (e.g. `gpt-4o`, `claude-3-5-sonnet`). Required
   * for provider-managed resources.
   */
  `model`?: string;

  /**
   * Optional resource-specific pricing dimensions (e.g. region, tier).
   */
  pricing_dimensions?: Record<string>;

  /**
   * Stable zero-based position used for deterministic ceiling allocation when
   * the batch total exceeds the reservation ceiling.
   */
  canonical_line_index: int32;
}

/**
 * A rated usage line returned in a settled batch, with resolved cost and sales
 * snapshots.
 */
@friendlyName("AIUsageUsageLine")
model UsageLine {
  ...UsageLineCreate;

  /**
   * Integer Credit charge for this line.
   */
  credits: int64;

  /**
   * Provider cost snapshot. Omitted for BYOK resources (cost is zero).
   */
  cost_snapshot?: CostSnapshot;

  /**
   * Customer sales price snapshot.
   */
  sales_snapshot?: SalesSnapshot;
}

/**
 * Request body for submitting a Canonical AI Usage Batch.
 *
 * A batch represents one business action (a chat turn, an agent run, etc.) and
 * is the atomic unit of AI billing settlement.
 */
@friendlyName("AIUsageUsageBatchCreate")
model UsageBatchCreate {
  /**
   * Client-generated idempotency key. Replaying the same key with the same
   * `payload_hash` returns the stored result with HTTP 200. A replay with a
   * different `payload_hash` for the same key returns HTTP 409.
   */
  idempotency_key: string;

  /**
   * SHA-256 hex digest of the canonical request body, used for idempotency
   * verification.
   */
  payload_hash: string;

  /**
   * The billing customer (payer) in OpenMeter.
   */
  billing_customer_id: Shared.ULID;

  /**
   * The subject or tenant key that produced the usage.
   */
  subject_key: string;

  /**
   * Monotonic per-tenant sequence number for watermark tracking. Must be
   * strictly increasing within a tenant.
   */
  tenant_seq: int64;

  /**
   * When the business action occurred. Used for rate package version resolution.
   */
  occurred_at: Shared.DateTime;

  /**
   * Links to the WeKnora runtime reservation, if any.
   */
  reservation_id: string;

  /**
   * Caps the total Credit charge for this batch. The platform absorbs any
   * excess above the ceiling.
   */
  reservation_ceiling_credits: int64;

  /**
   * Rate package version snapshot used for settlement.
   */
  rate_package_version: string;

  /**
   * Billing mode for this batch.
   */
  billing_mode: BillingMode;

  /**
   * Whether model resources are platform-managed. Set to `false` for
   * bring-your-own-key (BYOK) models.
   */
  provider_managed: boolean;

  /**
   * The individual resource consumption entries. Must contain at least one line
   * for `component` billing mode.
   */
  @minItems(1)
  lines: UsageLineCreate[];
}

/**
 * A settled AI Usage Batch with its rating and settlement results.
 */
@friendlyName("AIUsageUsageBatch")
model UsageBatch {
  /**
   * Unique identifier of the batch.
   */
  @visibility(Lifecycle.Read)
  id: Shared.ULID;

  /**
   * The idempotency key from the create request.
   */
  idempotency_key: string;

  /**
   * SHA-256 hex digest of the canonical request body.
   */
  payload_hash: string;

  billing_customer_id: Shared.ULID;
  subject_key: string;
  tenant_seq: int64;
  occurred_at: Shared.DateTime;
  reservation_id: string;
  reservation_ceiling_credits: int64;
  rate_package_version: string;
  billing_mode: BillingMode;
  provider_managed: boolean;

  /**
   * The consumed line items with resolved pricing.
   */
  lines: UsageLine[];

  /**
   * Settlement status.
   */
  status: BatchStatus;

  /**
   * Total integer Credit charge for the batch, after applying the reservation
   * ceiling.
   */
  total_credits: int64;

  /**
   * The highest continuously covered tenant sequence number after settlement.
   */
  covered_tenant_seq: int64;

  /**
   * Creation timestamp.
   */
  @visibility(Lifecycle.Read)
  created_at: Shared.DateTime;
}
