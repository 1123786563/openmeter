package payment

import (
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Payment fact errors

const ErrCodeInvalidSignature models.ErrorCode = "commerce_invalid_payment_signature"

var ErrInvalidSignature = models.NewValidationIssue(
	ErrCodeInvalidSignature,
	"provider callback signature verification failed",
	models.WithFieldString("signature"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeDuplicateProviderEvent models.ErrorCode = "commerce_duplicate_provider_event"

var ErrDuplicateProviderEvent = models.NewValidationIssue(
	ErrCodeDuplicateProviderEvent,
	"duplicate provider event id",
	models.WithFieldString("provider_event_id"),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusOK),
)

const ErrCodeContradictoryPaymentFact models.ErrorCode = "commerce_contradictory_payment_fact"

var ErrContradictoryPaymentFact = models.NewValidationIssue(
	ErrCodeContradictoryPaymentFact,
	"contradictory payment fact for the same provider order",
	models.WithFieldString("provider_order_id"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusConflict),
)

const ErrCodePaymentFactMismatch models.ErrorCode = "commerce_payment_fact_mismatch"

var ErrPaymentFactMismatch = models.NewValidationIssue(
	ErrCodePaymentFactMismatch,
	"payment fact does not match the expected order (amount, currency, or identity)",
	models.WithFieldString("payment_fact"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodePaymentAttemptNotFound models.ErrorCode = "commerce_payment_attempt_not_found"

var ErrPaymentAttemptNotFound = models.NewValidationIssue(
	ErrCodePaymentAttemptNotFound,
	"payment attempt not found",
	models.WithFieldString("payment_attempt_id"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusNotFound),
)

const ErrCodePaymentNotVerified models.ErrorCode = "commerce_payment_not_verified"

var ErrPaymentNotVerified = models.NewValidationIssue(
	ErrCodePaymentNotVerified,
	"payment has not been verified by the provider",
	models.WithFieldString("payment"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusPaymentRequired),
)

const ErrCodeFulfillmentAlreadyDone models.ErrorCode = "commerce_fulfillment_already_done"

var ErrFulfillmentAlreadyDone = models.NewValidationIssue(
	ErrCodeFulfillmentAlreadyDone,
	"order has already been fulfilled",
	models.WithFieldString("order_id"),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusOK),
)

const ErrCodeFulfillmentFailed models.ErrorCode = "commerce_fulfillment_failed"

var ErrFulfillmentFailed = models.NewValidationIssue(
	ErrCodeFulfillmentFailed,
	"fulfillment failed",
	models.WithFieldString("fulfillment"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusInternalServerError),
)
