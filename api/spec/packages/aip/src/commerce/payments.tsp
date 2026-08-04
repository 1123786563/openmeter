import "@typespec/http";
import "@typespec/openapi";
import "../shared/index.tsp";
import "./models.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

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

/**
 * The raw body of a payment provider callback (WeChat Pay or Alipay). The exact
 * format is provider-specific; OpenMeter verifies the signature, confirms the
 * payment fact, and returns a provider-appropriate acknowledgment. The client does
 * not need to understand this format.
 *
 * This is an opaque string-typed body because the callback payloads are
 * provider-specific (XML, form-encoded, or JSON) and are never exposed in
 * client-facing SDK types.
 */
@friendlyName("CommerceProviderCallbackBody")
model ProviderCallbackBody {
  @Http.body
  body: string;
}

/**
 * The acknowledgment returned to a payment provider callback. WeChat expects a
 * specific XML/JSON structure; Alipay expects "success". The actual response
 * format is negotiated by the server at runtime.
 */
@friendlyName("CommerceProviderCallbackResponse")
model ProviderCallbackResponse {
  @Http.statusCode _: 200;

  @Http.body
  body: string;
}
