import "../shared/index.tsp";
import "./models.tsp";

namespace Commerce;

/**
 * A recharge product (SKU) available for purchase. Each product defines a credit
 * package and its retail price in integer fen.
 */
@friendlyName("CommerceRechargeProduct")
model RechargeProduct {
  /**
   * Public identifier of the recharge product.
   */
  @visibility(Lifecycle.Read)
  id: Shared.ULID;

  /**
   * Human-readable product name (e.g. "1,000 Points Pack").
   */
  name: string;

  /**
   * Number of Credits granted on successful purchase.
   */
  credits: int64;

  /**
   * Retail price in the smallest currency unit (fen).
   */
  price_fen: FenAmount;

  /**
   * Three-letter ISO 4217 currency code for the price.
   */
  currency: Shared.CurrencyCode;

  /**
   * Whether this product is currently available for purchase.
   */
  active: boolean;

  /**
   * Sort order for display.
   */
  display_order?: int32;
}

/**
 * A recharge product with promotional bonus credits (e.g. "buy 1000 get 100
 * extra"). Bonus credits are granted as a separate gift-source bucket.
 */
@friendlyName("CommerceRechargeProductWithBonus")
model RechargeProductWithBonus {
  ...RechargeProduct;

  /**
   * Bonus gift credits granted on purchase, if any.
   */
  bonus_credits?: int64;
}
/**
 * Request body for creating a new recharge product.
 */
@friendlyName("CommerceRechargeProductCreate")
model RechargeProductCreate {
  /**
   * Unique stock keeping unit for the product within the namespace.
   */
  sku: string;

  /**
   * Human-readable product name (e.g. "1,000 Points Pack").
   */
  display_name: string;

  /**
   * The business kind of the product. It must match the order kind the product can
   * fulfill (`wallet_top_up` for recharge products).
   */
  kind: OrderKind;

  /**
   * Number of Credits granted on successful purchase.
   */
  credits: int64;

  /**
   * Retail price in the smallest currency unit (fen).
   */
  amount_fen: FenAmount;

  /**
   * Three-letter ISO 4217 currency code for the price.
   */
  currency: Shared.CurrencyCode;

  /**
   * Sort order for display.
   */
  display_order?: int32;

  /**
   * Refund policy for purchases of this product (`none`, `unspent`, or
   * `full_window`). Omit to use the default policy.
   */
  refund_policy?: string;

  /**
   * Longer marketing description of the product.
   */
  description?: string;
}

/**
 * Request body for updating a recharge product's mutable fields. Omitted fields
 * keep their current values; `sku`, `kind`, and `credits` are immutable.
 */
@friendlyName("CommerceRechargeProductUpdate")
model RechargeProductUpdate {
  /**
   * New human-readable product name.
   */
  display_name?: string;

  /**
   * New retail price in the smallest currency unit (fen). Existing orders keep their
   * creation-time price snapshot.
   */
  amount_fen?: FenAmount;

  /**
   * Whether the product is currently available for purchase (listing or delisting
   * the product).
   */
  active?: boolean;
}

/**
 * Response body for listing recharge products.
 */
@friendlyName("CommerceRechargeProductList")
model RechargeProductList {
  products: RechargeProduct[];
}
