import "@typespec/openapi";
import "@typespec/http";
import "../common/error.tsp";
import "../shared/index.tsp";

using TypeSpec.OpenAPI;
using TypeSpec.Http;

namespace CreditReservations;

@friendlyName("CreditReservationState")
enum ReservationState {
  Active: "active",
  Executing: "executing",
  Settled: "settled",
  Released: "released",
  Unknown: "unknown",
  Expired: "expired",
  ManualReview: "manual_review",
}

@friendlyName("CreditReservationEvidenceKind")
enum EvidenceKind {
  NotSent: "not_sent",
  ProviderConfirmedNotExecuted: "provider_confirmed_not_executed",
}

@friendlyName("CreditResourceLine")
model ResourceLine {
  feature_key: string;
  resource_code: string;
  quantity: int64;
  provider?: string;
  `model`?: string;
  dimensions?: Record<string>;
}

@friendlyName("CreditRatedLine")
model RatedLine {
  ...ResourceLine;
  rate_card_key: string;
  rate_version: string;
  credits: int64;
}

@friendlyName("CreditCurrency")
model Currency {
  code: string;
  custom_currency_id?: string;
}

@friendlyName("CreditFundingSplit")
model FundingSplit {
  prepaid_hold: int64;
  enterprise_hold: int64;
}

@friendlyName("CreditReservation")
model Reservation {
  id: string;
  customer_id: string;
  currency: Currency;
  state: ReservationState;
  rate_version: string;
  lines: RatedLine[];
  ceiling_credits: int64;
  settled_credits: int64;
  expires_at?: Shared.DateTime;
  execution_deadline?: Shared.DateTime;
  funding: FundingSplit;
}

@friendlyName("CreditReservationCreate")
model ReservationCreate {
  id: string;
  customer_id: string;
  subject_id: string;
  client_call_id: string;
  operation: string;
  idempotency_key: string;
  payload_hash: string;
  lines: ResourceLine[];
  authorization_expires_at: Shared.DateTime;
  provider?: string;
  `model`?: string;
  request_id?: string;
}

@friendlyName("CreditReservationExecute")
model ReservationExecute {
  idempotency_key: string;
  payload_hash: string;
  execution_deadline: Shared.DateTime;
}

@friendlyName("CreditReservationSettle")
model ReservationSettle {
  idempotency_key: string;
  payload_hash: string;
  actual_credits: int64;
  settled_at: Shared.DateTime;
}

@friendlyName("CreditReservationRelease")
model ReservationRelease {
  idempotency_key: string;
  payload_hash: string;
  evidence_kind: EvidenceKind;
  evidence_reference?: string;
}

@friendlyName("CreditReservationUnknown")
model ReservationUnknown {
  idempotency_key: string;
  payload_hash: string;
}

@friendlyName("CreditCharge")
model Charge {
  id: string;
  customer_id: string;
  reservation_id: string;
  currency: Currency;
  rate_version: string;
  lines: RatedLine[];
  total_credits: int64;
  state: string;
}

@friendlyName("CreditChargeCreate")
model ChargeCreate {
  id: string;
  customer_id: string;
  subject_id: string;
  operation: string;
  idempotency_key: string;
  payload_hash: string;
  lines: ResourceLine[];
  booked_at: Shared.DateTime;
}

@friendlyName("CreditChargeReverse")
model ChargeReverse {
  idempotency_key: string;
  payload_hash: string;
  reversed_at: Shared.DateTime;
}

@error
@friendlyName("CreditBalanceInsufficient")
model BalanceInsufficient extends Common.BaseError {
  @statusCode _: 402;
}
