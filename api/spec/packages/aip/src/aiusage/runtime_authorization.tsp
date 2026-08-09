import "@typespec/openapi";
import "../shared/index.tsp";

using TypeSpec.OpenAPI;

namespace AIUsage;

/**
 * The type of a credit movement on the AI Usage integer credit ledger.
 */
@friendlyName("AIUsageCreditTransactionType")
enum AICreditTransactionType {
  // Credits funded and available for consumption.
  Funded: "funded",

  // Credits consumed by AI usage settlement.
  Consumed: "consumed",

  // Credits removed because they expired.
  Expired: "expired",

  // Credits removed because the grant was voided.
  Voided: "voided",
}

/**
 * Integer credit balance for AI usage, scoped to a single currency.
 *
 * Unlike the OpenMeter Credits decimal balance, AI Usage tracks credits as signed
 * 64-bit integers to avoid floating-point rounding in settlement.
 */
@friendlyName("AIUsageCreditBalance")
model AICreditBalance {
  /**
   * The timestamp of the balance retrieval.
   */
  @visibility(Lifecycle.Read)
  retrieved_at: Shared.DateTime;

  /**
   * Credits available for immediate consumption.
   */
  available_credits: int64;

  /**
   * Credits that have been booked on the ledger.
   */
  settled_credits: int64;

  /**
   * Credits granted but not yet written to the ledger.
   */
  pending_credits: int64;
}

/**
 * An immutable credit movement on the AI Usage integer credit ledger.
 */
@friendlyName("AIUsageCreditTransaction")
model AICreditTransaction {
  /**
   * Unique identifier of the transaction.
   */
  @visibility(Lifecycle.Read)
  id: Shared.ULID;

  /**
   * When the transaction was booked.
   */
  @visibility(Lifecycle.Read)
  booked_at: Shared.DateTime;

  /**
   * The type of credit movement.
   */
  @visibility(Lifecycle.Read)
  type: AICreditTransactionType;

  /**
   * Signed credit amount. Positive adds balance, negative reduces it.
   */
  @visibility(Lifecycle.Read)
  amount: int64;

  /**
   * Available balance before this transaction.
   */
  @visibility(Lifecycle.Read)
  available_balance_before: int64;

  /**
   * Available balance after this transaction.
   */
  @visibility(Lifecycle.Read)
  available_balance_after: int64;
}

/**
 * Runtime authorization decision for a customer's AI usage.
 *
 * Returned before processing a usage batch to indicate whether the customer is
 * authorized to consume AI resources, along with their current credit balance and
 * reservation ceiling.
 */
@friendlyName("AIUsageRuntimeAuthorization")
model RuntimeAuthorization {
  /**
   * The frozen Phase 1 contract version string (currently `weknora-billing-p1-v1`).
   * Clients can use this to detect contract changes.
   */
  contract_version: string;

  /**
   * The timestamp of the authorization check.
   */
  @visibility(Lifecycle.Read)
  retrieved_at: Shared.DateTime;

  /**
   * Whether the customer is authorized to consume AI resources.
   */
  authorized: boolean;

  /**
   * Available integer Credit balance for AI consumption.
   */
  available_credits: int64;

  /**
   * The reservation ceiling in Credits, if a reservation is active. When set, usage
   * is capped at this amount per reservation period.
   */
  reservation_ceiling_credits?: int64;

  /**
   * The highest continuously covered tenant sequence number, used for watermark
   * tracking.
   */
  covered_tenant_seq: int64;

  /**
  * The reason authorization was denied, when `authorized` is `false`.
  */
 denial_reason?: string;

  /**
   * Canonical JSON payload for consumer-side verification. The raw bytes of
   * the canonicalized authorization object, used to derive the SHA-256 hash
   * and verify the signature.
  */
  @extension("x-go-name", "CanonicalPayload")
  @extension("x-go-type", "json.RawMessage")
  @extension("x-go-type-skip-optional-pointer", true)
  canonical_payload?: Record<unknown>;

  /**
   * SHA-256 hash of the canonical payload, hex-encoded.
   */
  @extension("x-go-name", "CanonicalSHA256")
  @extension("x-go-type-skip-optional-pointer", true)
  canonical_sha256?: string;

  /**
   * The key ID used to sign the authorization.
   */
  @extension("x-go-name", "KeyID")
  @extension("x-go-type-skip-optional-pointer", true)
  key_id?: string;

  /**
   * Base64-encoded signature over the canonical payload.
   */
  @extension("x-go-type-skip-optional-pointer", true)
  signature?: string;

  /**
   * The snapshot version for watermark tracking.
   */
  @extension("x-go-type-skip-optional-pointer", true)
  snapshot_version?: int64;

  /**
   * The subject or tenant key this authorization applies to.
   */
  @extension("x-go-type-skip-optional-pointer", true)
  subject_key?: string;

  /**
   * The timestamp until which this authorization is valid.
   */
  valid_until?: Shared.DateTime;
}

/**
 * Query parameters for the runtime authorization endpoint.
 */
@friendlyName("AIUsageRuntimeAuthorizationQuery")
model RuntimeAuthorizationQuery {
  /**
   * The subject or tenant key that will produce the usage. Required to check the
   * tenant's sequence watermark and reservation status.
   */
  subject_key?: string;

  /**
   * The reservation ID to check the ceiling against.
   */
  reservation_id?: string;
}
