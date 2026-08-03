package aiusage

import (
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Batch lifecycle errors
const ErrCodeInvalidBillingMode models.ErrorCode = "aiusage_invalid_billing_mode"

var ErrInvalidBillingMode = models.NewValidationIssue(
	ErrCodeInvalidBillingMode,
	"invalid billing mode",
	models.WithFieldString("billing_mode"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeInvalidBatchStatus models.ErrorCode = "aiusage_invalid_batch_status"

var ErrInvalidBatchStatus = models.NewValidationIssue(
	ErrCodeInvalidBatchStatus,
	"invalid batch status",
	models.WithFieldString("status"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeInvalidResourceCode models.ErrorCode = "aiusage_invalid_resource_code"

var ErrInvalidResourceCode = models.NewValidationIssue(
	ErrCodeInvalidResourceCode,
	"invalid resource code",
	models.WithFieldString("resource_code"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeBatchAlreadyExists models.ErrorCode = "aiusage_batch_already_exists"

var ErrBatchAlreadyExists = models.NewValidationIssue(
	ErrCodeBatchAlreadyExists,
	"batch already exists with the same usage_batch_id and payload_hash",
	models.WithFieldString("usage_batch_id"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusConflict),
)

const ErrCodeBatchPayloadConflict models.ErrorCode = "aiusage_batch_payload_conflict"

var ErrBatchPayloadConflict = models.NewValidationIssue(
	ErrCodeBatchPayloadConflict,
	"batch with the same usage_batch_id but different payload_hash already exists",
	models.WithFieldString("payload_hash"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusConflict),
)

// Pricing errors
const ErrCodeMissingRateCard models.ErrorCode = "aiusage_missing_rate_card"

var ErrMissingRateCard = models.NewValidationIssue(
	ErrCodeMissingRateCard,
	"rate card entry not found for resource",
	models.WithFieldString("rate_card"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

// Settlement errors
const ErrCodeInsufficientCredits models.ErrorCode = "aiusage_insufficient_credits"

var ErrInsufficientCredits = models.NewValidationIssue(
	ErrCodeInsufficientCredits,
	"insufficient credits to settle batch",
	models.WithFieldString("credits"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusPaymentRequired),
)

const ErrCodeCreditInsufficient models.ErrorCode = "aiusage_credit_insufficient"

var ErrCreditInsufficient = models.NewValidationIssue(
	ErrCodeCreditInsufficient,
	"prepaid credits insufficient and no enterprise receivable available",
	models.WithFieldString("credits"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusPaymentRequired),
)

const ErrCodeCreditLimitExceeded models.ErrorCode = "aiusage_credit_limit_exceeded"

var ErrCreditLimitExceeded = models.NewValidationIssue(
	ErrCodeCreditLimitExceeded,
	"enterprise receivable credit limit exceeded",
	models.WithFieldString("receivable"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusPaymentRequired),
)

const ErrCodeCeilingExceeded models.ErrorCode = "aiusage_ceiling_exceeded"

var ErrCeilingExceeded = models.NewValidationIssue(
	ErrCodeCeilingExceeded,
	"batch total credits exceed the ceiling",
	models.WithFieldString("ceiling_credits"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

// Idempotency errors
const ErrCodeIdempotencyConflict models.ErrorCode = "aiusage_idempotency_conflict"

var ErrIdempotencyConflict = models.NewValidationIssue(
	ErrCodeIdempotencyConflict,
	"batch with the same idempotency key but a different payload hash already exists",
	models.WithFieldString("usage_batch_id"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusConflict),
)

// Watermark errors
const ErrCodeWatermarkGap models.ErrorCode = "aiusage_watermark_gap"

var ErrWatermarkGap = models.NewValidationIssue(
	ErrCodeWatermarkGap,
	"tenant_seq is ahead of the continuous watermark; a gap exists below",
	models.WithFieldString("tenant_seq"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusConflict),
)
