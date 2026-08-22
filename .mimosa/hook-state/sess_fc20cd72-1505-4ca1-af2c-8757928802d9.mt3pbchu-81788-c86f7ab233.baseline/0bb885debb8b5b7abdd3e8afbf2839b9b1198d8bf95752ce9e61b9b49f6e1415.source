package creditreservations

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/creditreservation"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestHTTPStatusForError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "insufficient funds", err: creditreservation.ErrInsufficientFunds, want: http.StatusPaymentRequired},
		{name: "idempotency conflict", err: creditreservation.ErrIdempotencyConflict, want: http.StatusConflict},
		{name: "state conflict", err: creditreservation.ErrStateConflict, want: http.StatusConflict},
		{name: "validation", err: models.NewGenericValidationError(errors.New("invalid input")), want: http.StatusUnprocessableEntity},
		{name: "not found", err: models.NewGenericNotFoundError(errors.New("reservation")), want: http.StatusNotFound},
		{name: "unhandled", err: errors.New("database unavailable"), want: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, httpStatusForError(tt.err))
		})
	}
}
