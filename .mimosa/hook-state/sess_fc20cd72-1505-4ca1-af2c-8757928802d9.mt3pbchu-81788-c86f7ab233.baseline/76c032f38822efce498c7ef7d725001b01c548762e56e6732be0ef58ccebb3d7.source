package commerce

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
)

// TestCommerceErrorCodes verifies that every domain error has the correct
// stable error code string — these codes are part of the API contract and
// must never change without a version bump.
func TestCommerceErrorCodes(t *testing.T) {
	tests := []struct {
		name string
		err  models.ValidationIssue
		code models.ErrorCode
	}{
		{"invalid transition", ErrInvalidOrderTransition, ErrCodeInvalidTransition},
		{"order not found", ErrOrderNotFound, ErrCodeOrderNotFound},
		{"order idempotency conflict", ErrOrderIdempotencyConflict, ErrCodeOrderIdempotencyConflict},
		{"product not found", ErrProductNotFound, ErrCodeProductNotFound},
		{"sku not unique", ErrSKUNotUnique, ErrCodeSKUNotUnique},
		{"insufficient credits", ErrInsufficientCredits, ErrCodeInsufficientCredits},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Error(t, tc.err)
			assert.Equal(t, tc.code, tc.err.Code())
			assert.NotEmpty(t, tc.err.Message())
		})
	}
}

// TestCommerceErrorHTTPStatus verifies the HTTP status code mapping for each
// domain error. The handler layer relies on these attributes to produce the
// correct HTTP response without hardcoding status codes.
func TestCommerceErrorHTTPStatus(t *testing.T) {
	tests := []struct {
		name       string
		err        models.ValidationIssue
		wantStatus int
	}{
		{"invalid transition -> 409", ErrInvalidOrderTransition, http.StatusConflict},
		{"order not found -> 404", ErrOrderNotFound, http.StatusNotFound},
		{"idempotency conflict -> 409", ErrOrderIdempotencyConflict, http.StatusConflict},
		{"product not found -> 404", ErrProductNotFound, http.StatusNotFound},
		{"sku not unique -> 409", ErrSKUNotUnique, http.StatusConflict},
		{"insufficient credits -> 402", ErrInsufficientCredits, http.StatusPaymentRequired},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handled := commonhttp.HandleIssueIfHTTPStatusKnown(t.Context(), tc.err, rec)
			require.True(t, handled, "error must be recognized as an HTTP-status issue")
			assert.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}
