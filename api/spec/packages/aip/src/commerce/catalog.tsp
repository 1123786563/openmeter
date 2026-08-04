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
 * Response body for listing recharge products.
 */
@friendlyName("CommerceRechargeProductList")
model RechargeProductList {
  products: RechargeProduct[];
}
