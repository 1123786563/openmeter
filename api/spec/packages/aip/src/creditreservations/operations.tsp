import "@typespec/http";
import "@typespec/openapi";
import "../common/error.tsp";
import "../shared/index.tsp";
import "./models.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace CreditReservations;

alias CreditReservationErrors =
  | Common.ErrorResponses
  | Common.NotFound
  | Common.Conflict
  | BalanceInsufficient;

interface CreditReservationOperations {
  @post
  @operationId("create-credit-reservation")
  @summary("Create a credit reservation")
  @extension(Shared.UnstableExtension, true)
  create(@body body: ReservationCreate):
    | Shared.CreateResponse<Reservation>
    | CreditReservationErrors;

  @get
  @route("/{reservationId}")
  @operationId("get-credit-reservation")
  @summary("Get a credit reservation")
  @extension(Shared.UnstableExtension, true)
  get(@path reservationId: string):
    | Shared.GetResponse<Reservation>
    | CreditReservationErrors;

  @post
  @route("/{reservationId}/execute")
  @operationId("execute-credit-reservation")
  @summary("Start executing a credit reservation")
  @extension(Shared.UnstableExtension, true)
  execute(@path reservationId: string, @body body: ReservationExecute):
    | Shared.UpdateResponse<Reservation>
    | CreditReservationErrors;

  @post
  @route("/{reservationId}/settle")
  @operationId("settle-credit-reservation")
  @summary("Settle a credit reservation")
  @extension(Shared.UnstableExtension, true)
  settle(@path reservationId: string, @body body: ReservationSettle):
    | Shared.UpdateResponse<Reservation>
    | CreditReservationErrors;

  @post
  @route("/{reservationId}/release")
  @operationId("release-credit-reservation")
  @summary("Release a credit reservation")
  @extension(Shared.UnstableExtension, true)
  release(@path reservationId: string, @body body: ReservationRelease):
    | Shared.UpdateResponse<Reservation>
    | CreditReservationErrors;

  @post
  @route("/{reservationId}/unknown")
  @operationId("mark-credit-reservation-unknown")
  @summary("Mark a credit reservation as unknown")
  @extension(Shared.UnstableExtension, true)
  markUnknown(@path reservationId: string, @body body: ReservationUnknown):
    | Shared.UpdateResponse<Reservation>
    | CreditReservationErrors;
}

interface CreditChargeOperations {
  @post
  @operationId("create-credit-charge")
  @summary("Create and settle a direct credit charge")
  @extension(Shared.UnstableExtension, true)
  create(@body body: ChargeCreate):
    | Shared.CreateResponse<Charge>
    | CreditReservationErrors;

  @post
  @route("/{chargeId}/reverse")
  @operationId("reverse-credit-charge")
  @summary("Reverse a direct credit charge")
  @extension(Shared.UnstableExtension, true)
  reverse(@path chargeId: string, @body body: ChargeReverse):
    | Shared.UpdateResponse<Charge>
    | CreditReservationErrors;
}
