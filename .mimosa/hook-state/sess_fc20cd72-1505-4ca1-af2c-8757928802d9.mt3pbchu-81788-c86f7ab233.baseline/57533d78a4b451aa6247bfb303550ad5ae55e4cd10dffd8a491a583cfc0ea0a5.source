package creditreservations

import (
	"context"
	"errors"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/creditreservation"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/models"
)

func httpStatusForError(err error) int {
	switch {
	case errors.Is(err, creditreservation.ErrInsufficientFunds):
		return http.StatusPaymentRequired
	case errors.Is(err, creditreservation.ErrIdempotencyConflict), errors.Is(err, creditreservation.ErrStateConflict):
		return http.StatusConflict
	case models.IsGenericNotFoundError(err),
		entdb.IsNotFound(err),
		errors.Is(err, creditreservation.ErrRateNotFound),
		errors.Is(err, creditreservation.ErrSubscriptionNotFound),
		errors.Is(err, creditreservation.ErrRefundFenceNotFound):
		return http.StatusNotFound
	case models.IsGenericValidationError(err), isCreditReservationValidationError(err):
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}

func isCreditReservationValidationError(err error) bool {
	return errors.Is(err, creditreservation.ErrCreditCurrencyRequired) ||
		errors.Is(err, creditreservation.ErrAmbiguousRate) ||
		errors.Is(err, creditreservation.ErrRateNotFound) ||
		errors.Is(err, creditreservation.ErrSubscriptionNotFound) ||
		errors.Is(err, creditreservation.ErrAmbiguousSubscription) ||
		errors.Is(err, creditreservation.ErrUnitPriceRequired) ||
		errors.Is(err, creditreservation.ErrInvalidQuantity) ||
		errors.Is(err, creditreservation.ErrCreditOverflow) ||
		errors.Is(err, creditreservation.ErrResourceLinesRequired) ||
		errors.Is(err, creditreservation.ErrInvalidCommandIdentity) ||
		errors.Is(err, creditreservation.ErrTransitionEvidenceRequired)
}

func writeError(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
	status := httpStatusForError(err)
	if status == http.StatusInternalServerError {
		// Temporary debug: expose the actual error for diagnosis
		problem := &apierrors.BaseAPIError{
			Type:   "https://openmeter.io/problems/credit-reservation",
			Status: status,
			Title:  http.StatusText(status),
			Detail: err.Error(),
		}
		problem.HandleAPIError(w, r)
		return
	}

	problem := &apierrors.BaseAPIError{
		Type:   "https://openmeter.io/problems/credit-reservation",
		Status: status,
		Title:  http.StatusText(status),
		Detail: err.Error(),
	}
	problem.HandleAPIError(w, r)
}
