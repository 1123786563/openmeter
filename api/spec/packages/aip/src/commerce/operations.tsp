import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/pagination.tsp";
import "../common/parameters.tsp";
import "../shared/index.tsp";
import "./models.tsp";
import "./wallet.tsp";
import "./catalog.tsp";
import "./orders.tsp";
import "./payments.tsp";
import "./refunds.tsp";
import "./enterprise.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Commerce;

/**
 * Wallet read-model operations. The Wallet aggregates credit buckets and
 * transactions from the immutable Credit Ledger.
 */
interface WalletOperations {
  /**
   * Get a customer's Wallet view, including all credit buckets and recent
   * transactions.
   */
  @extension(Shared.UnstableExtension, true)
  @get
  @operationId("get-customer-wallet")
  @summary("Get customer wallet")
  get(@path customerId: Shared.ULID):
    | Shared.GetResponse<Wallet>
    | Common.NotFound
    | Common.ErrorResponses;
}

/**
 * Recharge catalog operations. Lists purchasable credit packages.
 */
interface CatalogOperations {
  /**
   * List all active recharge products available for purchase.
   */
  @extension(Shared.UnstableExtension, true)
  @get
  @operationId("list-recharge-products")
  @summary("List recharge products")
  list(
    /**
     * Filter by currency to show only products priced in the customer's currency.
     */
    @query
    currency?: Shared.CurrencyCode,
  ): Shared.GetResponse<RechargeProductList> | Common.ErrorResponses;
}

/**
 * Order creation and retrieval operations.
 */
interface OrderOperations {
  /**
   * Create a new order (plan purchase, subscription renewal, or wallet top-up).
   *
   * Returns HTTP 201 on first creation. Replaying the same idempotency key returns
   * the stored order with HTTP 200.
   */
  @extension(Shared.UnstableExtension, true)
  @post
  @operationId("create-order")
  @summary("Create order")
  create(@body order: OrderCreate):
    | Shared.CreateResponse<Order>
    | Shared.GetResponse<Order>
    | Common.BadRequest
    | Common.Conflict
    | Common.ErrorResponses;

  /**
   * Retrieve an order by its ID.
   */
  @extension(Shared.UnstableExtension, true)
  @get
  @operationId("get-order")
  @summary("Get order")
  get(@path orderId: Shared.ULID):
    | Shared.GetResponse<Order>
    | Common.NotFound
    | Common.ErrorResponses;
}

/**
 * Checkout session creation for an order.
 */
interface CheckoutSessionCreateOperations {
  /**
   * Create a checkout session for an order, initiating a payment attempt with the
   * specified provider.
   */
  @extension(Shared.UnstableExtension, true)
  @post
  @operationId("create-checkout-session")
  @summary("Create checkout session")
  create(@path orderId: Shared.ULID, @body session: CheckoutSessionCreate):
    | Shared.CreateResponse<CheckoutSession>
    | Common.NotFound
    | Common.ErrorResponses;
}

/**
 * Checkout session retrieval for polling payment status.
 */
interface CheckoutSessionGetOperations {
  /**
   * Retrieve a checkout session by its ID (for polling payment status after QR code
   * expiry or page reload).
   */
  @extension(Shared.UnstableExtension, true)
  @get
  @operationId("get-checkout-session")
  @summary("Get checkout session")
  get(@path sessionId: Shared.ULID):
    | Shared.GetResponse<CheckoutSession>
    | Common.NotFound
    | Common.ErrorResponses;
}

/**
 * Refund operations. Refunds are only permitted for unspent recharge credits and
 * must pass through a billing fence.
 */
interface RefundOperations {
  /**
   * Create a refund request for an order.
   */
  @extension(Shared.UnstableExtension, true)
  @post
  @operationId("create-refund")
  @summary("Create refund")
  create(@body refund: RefundCreate):
    | Shared.CreateResponse<Refund>
    | Common.BadRequest
    | Common.Conflict
    | Common.ErrorResponses;

  /**
   * Retrieve a refund by its ID.
   */
  @extension(Shared.UnstableExtension, true)
  @get
  @operationId("get-refund")
  @summary("Get refund")
  get(@path refundId: Shared.ULID):
    | Shared.GetResponse<Refund>
    | Common.NotFound
    | Common.ErrorResponses;
}

/**
 * WeChat Pay payment callback operation.
 */
interface WechatCallbackOperations {
  /**
   * WeChat Pay payment callback. OpenMeter verifies the signature, confirms the
   * payment fact, and fulfills the order.
   */
  @extension(Shared.UnstableExtension, true)
  @post
  @operationId("wechat-payment-callback")
  @summary("WeChat Pay callback")
  callback(@body body: string):
    | WechatCallbackSuccess
    | Common.BadRequest
    | Common.PayloadTooLarge
    | Common.ErrorResponses;
}

/**
 * Alipay payment callback operation.
 */
interface AlipayCallbackOperations {
  /**
   * Alipay payment callback. OpenMeter verifies the signature, confirms the payment
   * fact, and fulfills the order.
   */
  @extension(Shared.UnstableExtension, true)
  @post
  @operationId("alipay-payment-callback")
  @summary("Alipay callback")
  callback(@body body: string):
    | AlipayCallbackSuccess
    | Common.BadRequest
    | Common.PayloadTooLarge
    | Common.ErrorResponses;
}

/**
 * Enterprise receivable period operations for monthly settlement cycles.
 */
interface EnterpriseReceivableOperations {
  /**
   * List receivable periods for a customer.
   */
  @extension(Shared.UnstableExtension, true)
  @get
  @operationId("list-receivable-periods")
  @summary("List receivable periods")
  list(@path customerId: Shared.ULID, ...Common.CursorPaginationQuery):
    | Shared.CursorPaginatedResponse<ReceivablePeriod>
    | Common.NotFound
    | Common.ErrorResponses;
}

/**
 * Offline payment recording for enterprise customers.
 */
interface OfflinePaymentOperations {
  /**
   * Record an offline payment (bank transfer, enterprise remittance) for a customer.
   * The payment is held for reconciliation before being applied to a receivable
   * period.
   */
  @extension(Shared.UnstableExtension, true)
  @post
  @operationId("create-offline-payment")
  @summary("Create offline payment")
  create(@path customerId: Shared.ULID, @body payment: OfflinePaymentCreate):
    | Shared.CreateResponse<OfflinePayment>
    | Common.BadRequest
    | Common.Conflict
    | Common.ErrorResponses;
}

/**
 * External invoice operations for attaching VAT invoice references to receivable
 * periods.
 */
interface ExternalInvoiceOperations {
  /**
   * Attach or update an external invoice reference on a receivable period.
   */
  @extension(Shared.UnstableExtension, true)
  @put
  @operationId("update-external-invoice")
  @summary("Update external invoice")
  update(
    @path customerId: Shared.ULID,
    @path periodId: Shared.ULID,
    @body invoice: ExternalInvoiceUpdate,
  ):
    | Shared.UpdateResponse<ExternalInvoice>
    | Common.NotFound
    | Common.ErrorResponses;
}
