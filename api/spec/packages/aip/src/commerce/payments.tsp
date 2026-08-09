import "@typespec/http";
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
 * Successful WeChat Pay callback response. WeChat treats an empty 204 response
 * as the native acknowledgment and will retry any response with a body.
 */
@friendlyName("WechatCallbackSuccess")
model WechatCallbackSuccess {
  @Http.statusCode _: 204;
}

/**
 * Successful Alipay callback response. Alipay requires a plain-text success
 * token and retries when the response body differs from exactly "success".
 */
@friendlyName("AlipayCallbackSuccess")
model AlipayCallbackSuccess {
  @Http.statusCode _: 200;
  @Http.header contentType: "text/plain";
  @Http.body body: string;
}
