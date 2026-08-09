import "../shared/index.tsp";
import "./models.tsp";

namespace Commerce;

/**
 * Request body for creating a new order.
 */
@friendlyName("CommerceOrderCreate")
model OrderCreate {
  /**
   * Client-generated idempotency key. Replaying the same key returns the stored
   * order; a different payload for the same key returns HTTP 409.
   */
  idempotency_key: string;

  /**
   * The billing customer placing the order.
   */
  billing_customer_id: Shared.ULID;

  /**
   * The business type of the order.
   */
  kind: OrderKind;

  /**
   * For `plan_purchase` or `subscription_renewal`: the plan and billing period being
   * purchased.
   */
  plan?: OrderCreatePlanRef;

  /**
   * For `wallet_top_up`: the recharge product being purchased.
   */
  recharge_product_id?: Shared.ULID;

  /**
   * The currency for this order (three-letter ISO 4217).
   */
  currency: Shared.CurrencyCode;
}

/**
 * A commerce order representing a purchase intent (plan, renewal, or top-up).
 */
@friendlyName("CommerceOrder")
model Order {
  /**
   * Public identifier of the order.
   */
  @visibility(Lifecycle.Read)
  id: Shared.ULID;

  /**
   * The idempotency key from the create request.
   */
  idempotency_key: string;

  billing_customer_id: Shared.ULID;
  kind: OrderKind;
  plan?: PlanRef;
  recharge_product_id?: Shared.ULID;
  currency: Shared.CurrencyCode;

  /**
   * Total order amount in fen (smallest currency unit).
   */
  amount_fen: FenAmount;

  /**
   * Credits that will be granted on fulfillment (for top-up and plan orders).
   */
  credits?: int64;

  /**
   * Current lifecycle status.
   */
  status: OrderStatus;

  /**
   * Human-readable business tracking number shown to the customer.
   */
  business_tracking_number?: string;

  /**
   * When the order was created.
   */
  @visibility(Lifecycle.Read)
  created_at: Shared.DateTime;

  /**
   * When the order was last updated.
   */
  @visibility(Lifecycle.Read)
  updated_at: Shared.DateTime;

  /**
   * When the order expired or was cancelled.
   */
  @visibility(Lifecycle.Read)
  expired_at?: Shared.DateTime;
}

/**
 * Request body for creating a checkout session for an order.
 */
@friendlyName("CommerceCheckoutSessionCreate")
model CheckoutSessionCreate {
  /**
   * Client-generated idempotency key for the checkout attempt.
   */
  idempotency_key: string;

  /**
   * The payment channel to use for this checkout.
   */
  provider: PaymentProvider;

  /**
   * For offline payments: the external reference (e.g. bank transfer number).
   */
  external_reference?: string;
}

/**
 * A checkout session for an order, representing a single payment attempt.
 */
@friendlyName("CommerceCheckoutSession")
model CheckoutSession {
  /**
   * Public identifier of the checkout session.
   */
  @visibility(Lifecycle.Read)
  id: Shared.ULID;

  /**
   * The order this checkout session belongs to.
   */
  order_id: Shared.ULID;

  /**
   * The payment channel for this attempt.
   */
  provider: PaymentProvider;

  /**
   * Current payment attempt status.
   */
  status: PaymentAttemptStatus;

  /**
   * Provider-specific payment URL or QR code payload (e.g. WeChat code_url).
   */
  payment_url?: string;

  /**
   * External provider order identifier.
   */
  provider_order_id?: string;

  /**
   * For offline: the external payment reference.
   */
  external_reference?: string;

  /**
   * When the checkout session was created.
   */
  @visibility(Lifecycle.Read)
  created_at: Shared.DateTime;

  /**
   * When the checkout session expires (QR code timeout).
   */
  @visibility(Lifecycle.Read)
  expires_at?: Shared.DateTime;
}
