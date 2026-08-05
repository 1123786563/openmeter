package payment

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
)

// TestPaymentErrorCodes verifies that every payment-domain error has the
// correct stable error code — these are part of the wire contract.
func TestPaymentErrorCodes(t *testing.T) {
	tests := []struct {
		name string
		err  models.ValidationIssue
		code models.ErrorCode
	}{
		{"invalid signature", ErrInvalidSignature, ErrCodeInvalidSignature},
		{"duplicate provider event", ErrDuplicateProviderEvent, ErrCodeDuplicateProviderEvent},
		{"contradictory payment fact", ErrContradictoryPaymentFact, ErrCodeContradictoryPaymentFact},
		{"payment fact mismatch", ErrPaymentFactMismatch, ErrCodePaymentFactMismatch},
		{"payment attempt not found", ErrPaymentAttemptNotFound, ErrCodePaymentAttemptNotFound},
		{"payment not verified", ErrPaymentNotVerified, ErrCodePaymentNotVerified},
		{"fulfillment already done", ErrFulfillmentAlreadyDone, ErrCodeFulfillmentAlreadyDone},
		{"fulfillment failed", ErrFulfillmentFailed, ErrCodeFulfillmentFailed},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Error(t, tc.err)
			assert.Equal(t, tc.code, tc.err.Code())
		})
	}
}

// TestPaymentErrorHTTPStatus verifies the HTTP status code mapping.
func TestPaymentErrorHTTPStatus(t *testing.T) {
	tests := []struct {
		name       string
		err        models.ValidationIssue
		wantStatus int
	}{
		{"invalid signature -> 400", ErrInvalidSignature, http.StatusBadRequest},
		{"duplicate event -> 200", ErrDuplicateProviderEvent, http.StatusOK},
		{"contradictory fact -> 409", ErrContradictoryPaymentFact, http.StatusConflict},
		{"fact mismatch -> 400", ErrPaymentFactMismatch, http.StatusBadRequest},
		{"attempt not found -> 404", ErrPaymentAttemptNotFound, http.StatusNotFound},
		{"not verified -> 402", ErrPaymentNotVerified, http.StatusPaymentRequired},
		{"already done -> 200", ErrFulfillmentAlreadyDone, http.StatusOK},
		{"failed -> 500", ErrFulfillmentFailed, http.StatusInternalServerError},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handled := commonhttp.HandleIssueIfHTTPStatusKnown(t.Context(), tc.err, rec)
			require.True(t, handled)
			assert.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}
