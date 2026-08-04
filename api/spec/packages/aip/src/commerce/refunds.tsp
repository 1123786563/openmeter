import "../shared/index.tsp";
import "./models.tsp";

namespace Commerce;

/**
 * Request body for creating a refund.
 *
 * Refunds are only permitted for unspent recharge-source credits. The system
 * enforces a billing fence before reversing any ledger entries.
 */
@friendlyName("CommerceRefundCreate")
model RefundCreate {
  /**
   * Client-generated idempotency key.
   */
  idempotency_key: string;

  /**
   * The billing customer requesting the refund.
   */
  billing_customer_id: Shared.ULID;

  /**
   * The original order being refunded (must be a wallet_top_up).
   */
  order_id: Shared.ULID;

  /**
   * Refund amount in fen. Must not exceed the unspent portion of the original
   * payment, in multiples of the credit quantum.
   */
  amount_fen: FenAmount;

  /**
   * The reason for the refund.
   */
  reason: string;
}

/**
 * A refund request, progressing through the billing fence and provider processing
 * before the ledger is reversed.
 */
@friendlyName("CommerceRefund")
model Refund {
  /**
   * Public identifier of the refund.
   */
  @visibility(Lifecycle.Read)
  id: Shared.ULID;

  /**
   * The idempotency key from the create request.
   */
  idempotency_key: string;

  billing_customer_id: Shared.ULID;
  order_id: Shared.ULID;

  /**
   * Refund amount in fen.
   */
  amount_fen: FenAmount;

  /**
   * Credits being reversed.
   */
  credits_reversed: int64;

  reason: string;

  /**
   * The payment channel that will process the refund.
   */
  provider: PaymentProvider;

  /**
   * Current refund status.
   */
  status: RefundStatus;

  /**
   * External provider refund identifier.
   */
  provider_refund_id?: string;

  /**
   * Human-readable business tracking number.
   */
  business_tracking_number?: string;

  /**
   * When the refund was created.
   */
  @visibility(Lifecycle.Read)
  created_at: Shared.DateTime;

  /**
   * When the refund was last updated.
   */
  @visibility(Lifecycle.Read)
  updated_at: Shared.DateTime;

  /**
   * When the refund was fulfilled.
   */
  @visibility(Lifecycle.Read)
  fulfilled_at?: Shared.DateTime;
}
