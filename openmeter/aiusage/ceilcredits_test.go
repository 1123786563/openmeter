package aiusage

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCeilCredits(t *testing.T) {
	tests := []struct {
		name          string
		quantity      int64
		creditsPerU   int64
		unitSize      int64
		expected      int64
		expectedError error
	}{
		{"exact division", 1000, 2, 1000, 2, nil},
		{"rounds up from remainder", 501, 2, 1000, 2, nil},
		{"one unit rounds up", 1, 2, 1000, 1, nil},
		{"zero quantity", 0, 2, 1000, 0, nil},
		{"negative quantity", -1, 2, 1000, 0, ErrInvalidQuantity},
		{"negative rate", 100, -1, 1000, 0, ErrInvalidQuantity},
		{"zero unit size", 100, 2, 0, 0, ErrInvalidQuantity},
		{"overflow", math.MaxInt64, 2, 1, 0, ErrCreditOverflow},
		{"large exact", 999999999999999999, 1, 3, 333333333333333333, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CeilCredits(tc.quantity, tc.creditsPerU, tc.unitSize)
			if tc.expectedError != nil {
				require.ErrorIs(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
				assert.EqualValues(t, tc.expected, got)
			}
		})
	}
}

func TestCeilCreditsNoFloatingPoint(t *testing.T) {
	// Verify that large integer values are exact (no float truncation).
	got, err := CeilCredits(9007199254740993, 1, 1) // 2^53 + 1
	require.NoError(t, err)
	assert.EqualValues(t, 9007199254740993, got)
}

func TestCeilCreditsMergeEquivalence(t *testing.T) {
	// ceil(a+b) should equal ceil(a) + ceil(b) when no remainder is lost.
	// For 500+500 tokens at rate 2/1000: ceil(1000*2/1000) = 2.
	// Each half: ceil(500*2/1000) = ceil(1) = 1, total = 2.
	single, err := CeilCredits(1000, 2, 1000)
	require.NoError(t, err)

	half1, err := CeilCredits(500, 2, 1000)
	require.NoError(t, err)
	half2, err := CeilCredits(500, 2, 1000)
	require.NoError(t, err)

	assert.Equal(t, single, half1+half2)
}
