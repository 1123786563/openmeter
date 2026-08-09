import "../shared/index.tsp";
import "../currencies/index.tsp";

namespace Commerce;

/**
 * The origin of a Wallet credit bucket. Determines consumption priority and refund
 * eligibility.
 *
 * - `plan`: Credits granted from a subscription plan, consumed first.
 * - `gift`: Promotional or compensatory credits.
 * - `recharge`: Credits purchased via top-up; only unspent recharge credits are
 *   refundable.
 * - `enterprise_receivable`: Post-pay credits backed by an enterprise contract
 *   credit line.
 */
@friendlyName("CommerceWalletBucketSource")
enum WalletBucketSource {
  Plan: "plan",
  Gift: "gift",
  Recharge: "recharge",
  EnterpriseReceivable: "enterprise_receivable",
}

/**
 * The kind of a Wallet transaction movement.
 *
 * - `funded`: Credits added to the wallet.
 * - `consumed`: Credits deducted by usage settlement.
 * - `expired`: Credits removed due to expiration.
 * - `refunded`: Credits reversed via a refund.
 * - `adjusted`: Manual adjustment by an operator.
 */
@friendlyName("CommerceWalletTransactionKind")
enum WalletTransactionKind {
  Funded: "funded",
  Consumed: "consumed",
  Expired: "expired",
  Refunded: "refunded",
  Adjusted: "adjusted",
}

/**
 * The business type of an order.
 *
 * - `plan_purchase`: Initial subscription purchase.
 * - `subscription_renewal`: Renewal of an existing subscription.
 * - `wallet_top_up`: Prepaid credit recharge.
 */
@friendlyName("CommerceOrderKind")
enum OrderKind {
  PlanPurchase: "plan_purchase",
  SubscriptionRenewal: "subscription_renewal",
  WalletTopUp: "wallet_top_up",
}

/**
 * The lifecycle status of an order.
 */
@friendlyName("CommerceOrderStatus")
enum OrderStatus {
  Created: "created",
  AwaitingPayment: "awaiting_payment",
  Paid: "paid",
  Fulfilled: "fulfilled",
  Cancelled: "cancelled",
  Expired: "expired",
  RefundPending: "refund_pending",
  PartiallyRefunded: "partially_refunded",
  Refunded: "refunded",
}

/**
 * The lifecycle status of a payment attempt.
 */
@friendlyName("CommercePaymentAttemptStatus")
enum PaymentAttemptStatus {
  Created: "created",
  Pending: "pending",
  Succeeded: "succeeded",
  Failed: "failed",
  Closed: "closed",
}

/**
 * The payment channel that processed (or will process) a payment.
 *
 * - `wechat`: WeChat Pay.
 * - `alipay`: Alipay.
 * - `offline`: Offline bank transfer or enterprise remittance.
 */
@friendlyName("CommercePaymentProvider")
enum PaymentProvider {
  Wechat: "wechat",
  Alipay: "alipay",
  Offline: "offline",
}

/**
 * The lifecycle status of a refund request.
 *
 * The refund progresses through a billing fence before the ledger is reversed:
 *
 * - `pending_fence`: Waiting for the billing fence to clear pending usage.
 * - `provider_processing`: The payment provider is processing the refund.
 * - `ledger_reversing`: Ledger entries are being reversed.
 * - `fulfilled`: Refund completed; credits and funds reversed.
 * - `failed`: The refund could not be completed.
 */
@friendlyName("CommerceRefundStatus")
enum RefundStatus {
  PendingFence: "pending_fence",
  ProviderProcessing: "provider_processing",
  LedgerReversing: "ledger_reversing",
  Fulfilled: "fulfilled",
  Failed: "failed",
}

/**
 * The lifecycle status of an enterprise receivable period (monthly settlement).
 */
@friendlyName("CommerceReceivablePeriodStatus")
enum ReceivablePeriodStatus {
  Open: "open",
  Closed: "closed",
  PartiallyPaid: "partially_paid",
  Paid: "paid",
  Overdue: "overdue",
}

/**
 * Immutable ledger provenance attached to a Wallet transaction. Records the
 * underlying ledger entry that backs the credit movement, without exposing
 * internal Ent IDs.
 */
@friendlyName("CommerceLedgerProvenance")
model LedgerProvenance {
  /**
   * Public identifier of the credit grant or ledger source.
   */
  grant_id: Shared.ULID;

  /**
   * Settlement priority (0 = plan credit, 10 = gift, 20 = recharge, 30 = enterprise
   * receivable).
   */
  priority: int32;

  /**
   * The credit source that funded this transaction.
   */
  source: WalletBucketSource;
}

/**
 * Integer amount in the smallest currency unit (e.g. CNY fen, USD cents). Used for
 * all payment, order, refund and invoice monetary fields.
 */
@friendlyName("CommerceFenAmount")
@minValue(0)
scalar FenAmount extends int64;

/**
 * A lightweight reference to a plan in the product catalog, embedded in order and
 * catalog responses without duplicating the full plan definition.
 */
@friendlyName("CommercePlanRef")
model PlanRef {
  plan_id: Shared.ULID;
  plan_key: string;
  plan_version: string;
}

/**
 * A plan reference supplied when creating an order. The catalog product ID is
 * optional so callers can resolve a plan by its key and version alone.
 */
@friendlyName("CommerceOrderCreatePlanRef")
model OrderCreatePlanRef {
  /**
   * Optional catalog product ID used to verify the key and version.
   */
  plan_id?: Shared.ULID;

  /**
   * Stable plan key, such as `pro`.
   */
  plan_key: string;

  /**
   * Billing period or catalog version, such as `monthly`.
   */
  plan_version: string;
}

/**
 * A lightweight reference to a subscription, embedded in order responses.
 */
@friendlyName("CommerceSubscriptionRef")
model SubscriptionRef {
  subscription_id: Shared.ULID;
}
