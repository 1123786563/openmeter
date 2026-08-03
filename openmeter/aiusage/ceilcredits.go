package aiusage

import (
	"math/big"
)

// CeilCredits computes ceil(quantity * creditsPerUnit / unitSize) using exact
// big.Int arithmetic. It returns ErrInvalidQuantity for negative inputs or
// non-positive unitSize, and ErrCreditOverflow when the result exceeds int64.
// Zero quantity always returns 0 without error.
//
// This is the single source of truth for integer credit calculation per the
// pricing design: line_credits = ceil(quantity * credits_per_unit / unit_size).
// It MUST NOT use floating point at any step.
func CeilCredits(quantity, creditsPerUnit, unitSize int64) (int64, error) {
	if quantity < 0 || creditsPerUnit < 0 || unitSize <= 0 {
		return 0, ErrInvalidQuantity
	}
	if quantity == 0 {
		return 0, nil
	}

	product := new(big.Int).Mul(big.NewInt(quantity), big.NewInt(creditsPerUnit))
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(product, big.NewInt(unitSize), remainder)

	if remainder.Sign() > 0 {
		quotient.Add(quotient, big.NewInt(1))
	}

	if !quotient.IsInt64() {
		return 0, ErrCreditOverflow
	}

	return quotient.Int64(), nil
}
