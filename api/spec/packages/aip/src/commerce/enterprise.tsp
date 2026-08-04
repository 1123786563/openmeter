import "../shared/index.tsp";
import "./models.tsp";

namespace Commerce;

/**
 * An enterprise receivable period representing one monthly settlement cycle.
 * Enterprise customers with approved contracts can consume on credit and settle
 * via offline payments against these periods.
 */
@friendlyName("CommerceReceivablePeriod")
model ReceivablePeriod {
  /**
   * Public identifier of the receivable period.
   */
  @visibility(Lifecycle.Read)
  id: Shared.ULID;

  /**
   * The billing customer this period belongs to.
   */
  customer_id: Shared.ULID;

  /**
   * The start of the billing period (inclusive).
   */
  period_start: Shared.DateTime;

  /**
   * The end of the billing period (exclusive).
   */
  period_end: Shared.DateTime;

  /**
   * Total credits consumed during this period.
   */
  credits_consumed: int64;

  /**
   * Total amount owed for this period in fen.
   */
  amount_due_fen: FenAmount;

  /**
   * Amount already paid toward this period in fen.
   */
  amount_paid_fen: FenAmount;

  /**
   * Currency for this period.
   */
  currency: Shared.CurrencyCode;

  /**
   * Current settlement status.
   */
  status: ReceivablePeriodStatus;

  /**
   * When the period was created.
   */
  @visibility(Lifecycle.Read)
  created_at: Shared.DateTime;

  /**
   * When the period was closed or settled.
   */
  @visibility(Lifecycle.Read)
  closed_at?: Shared.DateTime;
}

/**
 * Request body for recording an offline payment (bank transfer, enterprise
 * remittance) against a customer's account.
 */
@friendlyName("CommerceOfflinePaymentCreate")
model OfflinePaymentCreate {
  /**
   * Client-generated idempotency key.
   */
  idempotency_key: string;

  /**
   * Payment amount in fen.
   */
  amount_fen: FenAmount;

  /**
   * Currency of the payment.
   */
  currency: Shared.CurrencyCode;

  /**
   * The receivable period to apply this payment to, if applicable.
   */
  receivable_period_id?: Shared.ULID;

  /**
   * External reference (bank transfer number, remittance advice).
   */
  external_reference: string;

  /**
   * When the payment was received.
   */
  received_at: Shared.DateTime;

  /**
   * Optional note from the submitter.
   */
  note?: string;
}

/**
 * An offline payment record pending reconciliation and application.
 */
@friendlyName("CommerceOfflinePayment")
model OfflinePayment {
  /**
   * Public identifier of the offline payment.
   */
  @visibility(Lifecycle.Read)
  id: Shared.ULID;

  idempotency_key: string;
  amount_fen: FenAmount;
  currency: Shared.CurrencyCode;
  receivable_period_id?: Shared.ULID;
  external_reference: string;
  received_at: Shared.DateTime;
  note?: string;

  /**
   * Whether this payment has been reconciled and applied.
   */
  reconciled: boolean;

  @visibility(Lifecycle.Read)
  created_at: Shared.DateTime;
}

/**
 * Request body for attaching or updating an external invoice reference on a
 * receivable period (e.g. a VAT invoice number from the external tax service).
 */
@friendlyName("CommerceExternalInvoiceUpdate")
model ExternalInvoiceUpdate {
  /**
   * Client-generated idempotency key for the update.
   */
  idempotency_key: string;

  /**
   * External invoice number or reference.
   */
  invoice_number: string;

  /**
   * URL or identifier of the external invoice document.
   */
  invoice_url?: string;

  /**
   * The invoicing party or tax service.
   */
  issuer?: string;

  /**
   * When the external invoice was issued.
   */
  issued_at?: Shared.DateTime;
}

/**
 * External invoice metadata attached to a receivable period.
 */
@friendlyName("CommerceExternalInvoice")
model ExternalInvoice {
  receivable_period_id: Shared.ULID;
  invoice_number: string;
  invoice_url?: string;
  issuer?: string;
  issued_at?: Shared.DateTime;

  @visibility(Lifecycle.Read)
  updated_at: Shared.DateTime;
}
