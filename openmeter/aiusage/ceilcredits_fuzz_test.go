package aiusage

import (
	"math"
	"math/big"
	"testing"
)

// FuzzCeilCredits verifies that CeilCredits never panics and always returns
// a result consistent with the mathematical definition:
//
//	ceil(quantity * creditsPerUnit / unitSize)
//
// for all int64 inputs.
func FuzzCeilCredits(f *testing.F) {
	f.Add(int64(0), int64(2), int64(1000))
	f.Add(int64(500), int64(2), int64(1000))
	f.Add(int64(1), int64(1), int64(3))
	f.Add(int64(math.MaxInt64), int64(1), int64(1))
	f.Add(int64(-1), int64(2), int64(1000))

	f.Fuzz(func(t *testing.T, quantity, creditsPerUnit, unitSize int64) {
		got, err := CeilCredits(quantity, creditsPerUnit, unitSize)

		// Invalid inputs must return error.
		if quantity < 0 || creditsPerUnit < 0 || unitSize <= 0 {
			if err == nil {
				t.Fatalf("expected error for invalid input: q=%d cpu=%d us=%d", quantity, creditsPerUnit, unitSize)
			}
			return
		}

		// Zero quantity must return 0 without error.
		if quantity == 0 {
			if err != nil || got != 0 {
				t.Fatalf("zero quantity: expected (0, nil), got (%d, %v)", got, err)
			}
			return
		}

		// Valid input: only allowed error is overflow.
		if err != nil {
			if err != ErrCreditOverflow {
				t.Fatalf("unexpected error: %v", got)
			}
			return
		}

		if got < 0 {
			t.Fatalf("negative result: %d", got)
		}

		// Use big.Int for the floor/ceiling check to avoid test-level overflow.
		product := new(big.Int).Mul(big.NewInt(quantity), big.NewInt(creditsPerUnit))
		floor, rem := new(big.Int), new(big.Int)
		floor.QuoRem(product, big.NewInt(unitSize), rem)

		expectedFloor := floor.Int64()
		if got < expectedFloor {
			t.Fatalf("result %d < floor %d for q=%d cpu=%d us=%d", got, expectedFloor, quantity, creditsPerUnit, unitSize)
		}

		// Ceiling: result is at most floor + 1 (when there is a remainder).
		if rem.Sign() > 0 {
			if got != expectedFloor+1 {
				t.Fatalf("remainder case: expected floor+1=%d, got %d for q=%d cpu=%d us=%d",
					expectedFloor+1, got, quantity, creditsPerUnit, unitSize)
			}
		} else {
			if got != expectedFloor {
				t.Fatalf("exact case: expected floor=%d, got %d for q=%d cpu=%d us=%d",
					expectedFloor, got, quantity, creditsPerUnit, unitSize)
			}
		}
	})
}
