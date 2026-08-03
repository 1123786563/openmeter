package aiusage

import "errors"

// Pricing calculation errors used by the pricing and settlement engines.

// ErrCreditOverflow is returned when credit calculation exceeds int64 range.
var ErrCreditOverflow = errors.New("credit calculation overflow: result exceeds int64")

// ErrAmbiguousRate is returned when two rate entries match with equal specificity.
var ErrAmbiguousRate = errors.New("ambiguous rate: two entries match with equal specificity")

// ErrInvalidQuantity is returned when quantity or rate parameters are invalid.
var ErrInvalidQuantity = errors.New("invalid quantity: must be non-negative with positive unit size")

// ErrRateMissing is returned when no rate entry matches the requested resource.
var ErrRateMissing = errors.New("rate missing: no rate entry found for resource")

// ErrResourceUnknown is returned when a resource code is not recognized.
var ErrResourceUnknown = errors.New("resource unknown: resource code not in meter registry")

// ErrBillingModeConflict is returned when component and bundle lines are mixed in the same batch.
var ErrBillingModeConflict = errors.New("billing mode conflict: component and bundle are mutually exclusive")
