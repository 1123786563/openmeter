package commerce

import (
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Order lifecycle errors

const ErrCodeInvalidTransition models.ErrorCode = "commerce_invalid_order_transition"

var ErrInvalidOrderTransition = models.NewValidationIssue(
	ErrCodeInvalidTransition,
	"order status transition is not allowed",
	models.WithFieldString("status"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusConflict),
)

const ErrCodeOrderNotFound models.ErrorCode = "commerce_order_not_found"

var ErrOrderNotFound = models.NewValidationIssue(
	ErrCodeOrderNotFound,
	"order not found",
	models.WithFieldString("order_id"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusNotFound),
)

const ErrCodeOrderIdempotencyConflict models.ErrorCode = "commerce_order_idempotency_conflict"

var ErrOrderIdempotencyConflict = models.NewValidationIssue(
	ErrCodeOrderIdempotencyConflict,
	"order with the same idempotency key but different payload already exists",
	models.WithFieldString("idempotency_key"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusConflict),
)

const ErrCodeProductNotFound models.ErrorCode = "commerce_product_not_found"

var ErrProductNotFound = models.NewValidationIssue(
	ErrCodeProductNotFound,
	"product not found",
	models.WithFieldString("product_id"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusNotFound),
)

const ErrCodeSKUNotUnique models.ErrorCode = "commerce_sku_not_unique"

var ErrSKUNotUnique = models.NewValidationIssue(
	ErrCodeSKUNotUnique,
	"product with this SKU already exists in this namespace",
	models.WithFieldString("sku"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusConflict),
)

const ErrCodeInsufficientCredits models.ErrorCode = "commerce_insufficient_credits"

var ErrInsufficientCredits = models.NewValidationIssue(
	ErrCodeInsufficientCredits,
	"insufficient credits and no enterprise receivable available",
	models.WithFieldString("credits"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusPaymentRequired),
)
