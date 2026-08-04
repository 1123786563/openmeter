import "../shared/index.tsp";
import "./models.tsp";

namespace Commerce;

/**
 * A verified payment fact from a payment provider. Represents a confirmed monetary
 * transfer without exposing raw credentials.
 */
@friendlyName("CommercePaymentFact")
model PaymentFact {
  /**
   * Public identifier of the payment fact.
   */
  @visibility(Lifecycle.Read)
  id: Shared.ULID;

  /**
   * The order this payment belongs to.
   */
  order_id: Shared.ULID;

  /**
   * The checkout session associated with this payment.
   */
  checkout_session_id: Shared.ULID;

  /**
   * The payment channel that processed this payment.
   */
  provider: PaymentProvider;

  /**
   * Confirmed amount in fen.
   */
  amount_fen: FenAmount;

  /**
   * Currency of the payment.
   */
  currency: Shared.CurrencyCode;

  /**
   * External provider transaction identifier.
   */
  provider_transaction_id: string;

  /**
   * External provider order identifier.
   */
  provider_order_id: string;

  /**
   * Current status of the payment attempt.
   */
  status: PaymentAttemptStatus;

  /**
   * When the payment fact was created.
   */
  @visibility(Lifecycle.Read)
  created_at: Shared.DateTime;

  /**
   * When the payment was confirmed by the provider.
   */
  @visibility(Lifecycle.Read)
  settled_at?: Shared.DateTime;
}

/**
 * Acknowledgment returned to a payment provider callback. The body is a
 * provider-appropriate plain text response (e.g. WeChat XML/JSON or Alipay
 * "success").
 */
@friendlyName("CommerceProviderCallbackAck")
model ProviderCallbackAck {
  /**
   * Provider-specific acknowledgment text.
   */
  ack: string;
}
