import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/parameters.tsp";
import "../common/pagination.tsp";
import "../shared/index.tsp";
import "./batch.tsp";
import "./pricing.tsp";
import "./runtime_authorization.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace AIUsage;

/**
 * Batch submission and retrieval operations for Canonical AI Usage.
 */
interface AIUsageBatchOperations {
  /**
   * Submit a Canonical AI Usage Batch for settlement.
   *
   * The first submit for a given `idempotency_key` returns HTTP 201 with the
   * settled batch. An identical replay (same `idempotency_key` and
   * `payload_hash`) returns HTTP 200 with the stored result. A replay with the
   * same `idempotency_key` but a different `payload_hash` returns HTTP 409.
   */
  @extension(Shared.UnstableExtension, true)
  @post
  @operationId("create-ai-usage-batch")
  @summary("Submit an AI usage batch")
  create(@body batch: UsageBatchCreate):
    | Shared.CreateResponse<UsageBatch>
    | Shared.GetResponse<UsageBatch>
    | Common.BadRequest
    | Common.NotFound
    | Common.Conflict
    | Common.ErrorResponses;

  /**
   * Retrieve a settled AI Usage Batch by its ID.
   */
  @extension(Shared.UnstableExtension, true)
  @get
  @operationId("get-ai-usage-batch")
  @summary("Get an AI usage batch")
  get(@path batchId: Shared.ULID):
    | Shared.GetResponse<UsageBatch>
    | Common.NotFound
    | Common.ErrorResponses;
}

/**
 * Runtime authorization check before submitting a usage batch.
 */
interface RuntimeAuthorizationOperations {
  /**
   * Check whether a customer is authorized to consume AI resources.
   *
   * Returns the current integer credit balance, reservation ceiling, and the
   * covered tenant sequence watermark.
   */
  @extension(Shared.UnstableExtension, true)
  @get
  @operationId("get-customer-runtime-authorization")
  @summary("Get runtime authorization")
  get(
    @path customerId: Shared.ULID,

    /**
     * Filter the authorization check by subject and reservation.
     */
    @query(#{ style: "deepObject", explode: true })
    filter?: RuntimeAuthorizationQuery,
  ):
    | Shared.GetResponse<RuntimeAuthorization>
    | Common.NotFound
    | Common.ErrorResponses;
}

/**
 * AI Usage-scoped credit balance operations. These mirror the OpenMeter
 * `Customers.Credits.Balance` operations but use AI Usage-specific operation IDs
 * and route prefixes for the WeKnora integration.
 */
interface CreditBalanceOperations {
  /**
   * Get a customer's credit balance for AI usage. Returns the same balance
   * model as the OpenMeter Credits endpoint but scoped to the AI Usage route.
   */
  @extension(Shared.UnstableExtension, true)
  @get
  @operationId("get-ai-usage-credit-balance")
  @summary("Get AI usage credit balance")
  get(
    @path customerId: Shared.ULID,

    /**
     * Return the credit balance as of this timestamp. Defaults to now.
     */
    @query
    timestamp?: Shared.DateTime,
  ):
    | Shared.GetResponse<AICreditBalance>
    | Common.NotFound
    | Common.ErrorResponses;
}

/**
 * AI Usage-scoped credit transaction listing. These mirror the OpenMeter
 * `Customers.Credits.Transactions` operations but use AI Usage-specific
 * operation IDs and route prefixes for the WeKnora integration.
 */
interface CreditTransactionOperations {
  /**
   * List credit transactions for a customer's AI usage. Returns the same
   * transaction model as the OpenMeter Credits endpoint but scoped to the AI
   * Usage route.
   */
  @extension(Shared.UnstableExtension, true)
  @get
  @operationId("list-ai-usage-credit-transactions")
  @summary("List AI usage credit transactions")
  list(@path customerId: Shared.ULID, ...Common.CursorPaginationQuery):
    | Shared.CursorPaginatedResponse<AICreditTransaction>
    | Common.NotFound
    | Common.ErrorResponses;
}
