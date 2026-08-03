package aiusage

import (
	"math"

	"github.com/alpacahq/alpacadecimal"
)

// Deprecated: CalculateCredits uses math.Ceil(float64) which can lose
// precision for large amounts. Prefer CeilCredits (ceilcredits.go) which
// uses exact big.Int arithmetic per the pricing design formula:
//   line_credits = ceil(quantity * credits_per_unit / unit_size)
//
// CalculateCredits converts a sales amount (in CNY) to integer Credits.
// Formula: credits = ceil(salesAmountCNY * creditRate)
// Rounding is always up (ceil) to protect platform margin.
// Returns 0 for non-positive amounts.
func CalculateCredits(salesAmountCNY alpacadecimal.Decimal, creditRate int64) int64 {
	if salesAmountCNY.IsZero() || salesAmountCNY.IsNegative() {
		return 0
	}
	if creditRate <= 0 {
		return 0
	}

	rate := alpacadecimal.NewFromInt(creditRate)
	product := salesAmountCNY.Mul(rate)

	// Use ceiling: even fractional credits round up.
	return int64(math.Ceil(product.InexactFloat64()))
}

// CalculateLineCredits rates one line item and returns its Credit cost.
// Formula: credits = ceil(quantity * pricePerUnitCNY * creditRate)
func CalculateLineCredits(quantity int64, pricePerUnitCNY alpacadecimal.Decimal, creditRate int64) int64 {
	if quantity <= 0 {
		return 0
	}

	totalPrice := pricePerUnitCNY.Mul(alpacadecimal.NewFromInt(quantity))
	return CalculateCredits(totalPrice, creditRate)
}
