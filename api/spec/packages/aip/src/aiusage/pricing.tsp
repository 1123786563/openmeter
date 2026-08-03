import "../shared/index.tsp";
import "../currencies/index.tsp";

namespace AIUsage;

/**
 * Machine-readable error codes specific to AI Usage billing.
 *
 * These codes appear in the `error_code` field of RFC-7807 problem responses
 * returned by AI Usage endpoints.
 */
@friendlyName("AIUsageErrorCode")
enum AIUsageErrorCode {
  // The idempotency key was replayed with a different payload hash.
  IdempotencyConflict: "idempotency_conflict",

  // No rate card entry found for a provider-managed resource at the batch time.
  RateMissing: "rate_missing",

  // The submitted resource_code is not a recognized billable resource.
  ResourceUnknown: "resource_unknown",

  // Available credits are insufficient to settle the batch.
  CreditInsufficient: "credit_insufficient",

  // The customer's credit limit has been reached.
  CreditLimitExceeded: "credit_limit_exceeded",

  // The batch total exceeds the reservation ceiling after capping.
  ReservationCeilingExceeded: "reservation_ceiling_exceeded",
}

/**
 * The provider cost snapshot for a rated line (typically in USD).
 */
@friendlyName("AIUsageCostSnapshot")
model CostSnapshot {
  currency: Currencies.CurrencyCode;
  amount: Shared.Numeric;

  // Source of the cost data (e.g. `llm_cost`, `manual`).
  source: string;
}

/**
 * The customer-facing sales price snapshot (typically in CNY for display).
 */
@friendlyName("AIUsageSalesSnapshot")
model SalesSnapshot {
  currency: Currencies.CurrencyCode;
  amount: Shared.Numeric;

  // Rate card version used to resolve this price.
  rate_card_version: string;
}

/**
 * A rating snapshot captures the cost and sales price resolution for one line item
 * within a settled batch.
 */
@friendlyName("AIUsageRatingSnapshot")
model RatingSnapshot {
  resource_code: string;
  cost_snapshot: CostSnapshot;
  sales_snapshot: SalesSnapshot;

  // Integer Credit charge for this line.
  credits: int64;
}

/**
 * A reference to a credit grant that was burned during settlement.
 */
@friendlyName("AIUsageLedgerEntryRef")
model LedgerEntryRef {
  grant_id: Shared.ULID;
  amount: int64;

  // Settlement priority (0 = plan credit, 10 = gift, 20 = recharge, 30 = receivable).
  priority: int32;
}

/**
 * Settlement result for a batch, including the ledger entries burned and the
 * watermark advancement.
 */
@friendlyName("AIUsageBatchSettlementResult")
model BatchSettlementResult {
  batch_id: Shared.ULID;
  status: BatchStatus;
  total_credits: int64;
  rating_snapshots: RatingSnapshot[];
  ledger_entries: LedgerEntryRef[];
  covered_tenant_seq: int64;
}
