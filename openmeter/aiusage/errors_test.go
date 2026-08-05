package aiusage

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
)

// TestAIUsageErrorCodes verifies that every AI Usage domain error has the
// correct stable error code string.
func TestAIUsageErrorCodes(t *testing.T) {
	tests := []struct {
		name string
		err  models.ValidationIssue
		code models.ErrorCode
	}{
		{"invalid billing mode", ErrInvalidBillingMode, ErrCodeInvalidBillingMode},
		{"invalid batch status", ErrInvalidBatchStatus, ErrCodeInvalidBatchStatus},
		{"invalid resource code", ErrInvalidResourceCode, ErrCodeInvalidResourceCode},
		{"batch already exists", ErrBatchAlreadyExists, ErrCodeBatchAlreadyExists},
		{"batch payload conflict", ErrBatchPayloadConflict, ErrCodeBatchPayloadConflict},
		{"missing rate card", ErrMissingRateCard, ErrCodeMissingRateCard},
		{"insufficient credits", ErrInsufficientCredits, ErrCodeInsufficientCredits},
		{"credit insufficient", ErrCreditInsufficient, ErrCodeCreditInsufficient},
		{"credit limit exceeded", ErrCreditLimitExceeded, ErrCodeCreditLimitExceeded},
		{"ceiling exceeded", ErrCeilingExceeded, ErrCodeCeilingExceeded},
		{"idempotency conflict", ErrIdempotencyConflict, ErrCodeIdempotencyConflict},
		{"watermark gap", ErrWatermarkGap, ErrCodeWatermarkGap},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Error(t, tc.err)
			assert.Equal(t, tc.code, tc.err.Code())
			assert.NotEmpty(t, tc.err.Message())
		})
	}
}

// TestAIUsageErrorHTTPStatus verifies the HTTP status code mapping.
func TestAIUsageErrorHTTPStatus(t *testing.T) {
	tests := []struct {
		name       string
		err        models.ValidationIssue
		wantStatus int
	}{
		{"invalid billing mode -> 400", ErrInvalidBillingMode, http.StatusBadRequest},
		{"batch already exists -> 409", ErrBatchAlreadyExists, http.StatusConflict},
		{"payload conflict -> 409", ErrBatchPayloadConflict, http.StatusConflict},
		{"credit insufficient -> 402", ErrCreditInsufficient, http.StatusPaymentRequired},
		{"credit limit exceeded -> 402", ErrCreditLimitExceeded, http.StatusPaymentRequired},
		{"ceiling exceeded -> 400", ErrCeilingExceeded, http.StatusBadRequest},
		{"idempotency conflict -> 409", ErrIdempotencyConflict, http.StatusConflict},
		{"watermark gap -> 409", ErrWatermarkGap, http.StatusConflict},
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
