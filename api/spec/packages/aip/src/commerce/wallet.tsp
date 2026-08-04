import "../shared/index.tsp";
import "./models.tsp";

namespace Commerce;

/**
 * A single credit bucket in a customer's Wallet. Each bucket represents credits
 * from one funding source with its own balance and expiration rules.
 *
 * The Wallet is a read-only aggregation over the immutable Credit Ledger; it never
 * holds a mutable second balance.
 */
@friendlyName("CommerceWalletBucket")
model WalletBucket {
  /**
   * The funding source of this bucket.
   */
  source: WalletBucketSource;

  /**
   * Credits currently available for consumption.
   */
  available_credits: int64;

  /**
   * When this bucket's credits expire. Omitted if the bucket does not expire.
   */
  expires_at?: Shared.DateTime;

  /**
   * Credits that are refundable (only present for recharge-source buckets).
   * Represents the unspent portion of purchased credits.
   */
  refundable_credits?: int64;
}

/**
 * An immutable credit movement on the Wallet, projected from the Credit Ledger.
 * Every transaction carries immutable ledger provenance and a timestamp.
 */
@friendlyName("CommerceWalletTransaction")
model WalletTransaction {
  /**
   * Public identifier of the transaction.
   */
  @visibility(Lifecycle.Read)
  id: Shared.ULID;

  /**
   * The type of credit movement.
   */
  kind: WalletTransactionKind;

  /**
   * Signed credit amount. Positive adds balance, negative reduces it.
   */
  amount: int64;

  /**
   * Immutable ledger provenance linking this movement to its grant and source.
   */
  provenance: LedgerProvenance;

  /**
   * When the transaction occurred.
   */
  occurred_at: Shared.DateTime;
}

/**
 * The aggregate Wallet view for a customer, combining all credit buckets and
 * recent transactions.
 */
@friendlyName("CommerceWallet")
model Wallet {
  /**
   * The billing customer this wallet belongs to.
   */
  customer_id: Shared.ULID;

  /**
   * Phase 2 contract version for client compatibility detection.
   */
  contract_version: string;

  /**
   * Total available credits across all buckets.
   */
  total_available_credits: int64;

  /**
   * Individual credit buckets, ordered by consumption priority.
   */
  buckets: WalletBucket[];

  /**
   * Recent wallet transactions (most recent first).
   */
  transactions?: WalletTransaction[];

  /**
   * When the wallet was retrieved.
   */
  @visibility(Lifecycle.Read)
  retrieved_at: Shared.DateTime;
}
