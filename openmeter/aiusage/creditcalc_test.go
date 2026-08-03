package aiusage

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"
)

func TestCalculateCredits(t *testing.T) {
	tests := []struct {
		name     string
		amount   float64
		rate     int64
		expected int64
	}{
		{"ceil rounds up 0.002*1000", 0.002, 1000, 2},
		{"ceil rounds up 0.0015*1000", 0.0015, 1000, 2},
		{"exact 0.001*1000", 0.001, 1000, 1},
		{"zero amount", 0.0, 1000, 0},
		{"negative amount", -0.001, 1000, 0},
		{"zero rate", 0.001, 0, 0},
		{"fractional ceil 0.0001*1000", 0.0001, 1000, 1},
		{"large amount 100*1000", 100.0, 1000, 100000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := CalculateCredits(alpacadecimal.NewFromFloat(tc.amount), tc.rate)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestCalculateLineCredits(t *testing.T) {
	tests := []struct {
		name         string
		quantity     int64
		pricePerUnit float64
		rate         int64
		expected     int64
	}{
		{"1000 tokens * 0.000002 * 1000", 1000, 0.000002, 1000, 2},
		{"500 tokens * 0.001 * 1000", 500, 0.001, 1000, 500},
		{"1 token * 0.0001 * 1000", 1, 0.0001, 1000, 1},
		{"zero quantity", 0, 0.001, 1000, 0},
		{"negative quantity", -1, 0.001, 1000, 0},
		{"BYOK zero price", 1000, 0.0, 1000, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := CalculateLineCredits(tc.quantity, alpacadecimal.NewFromFloat(tc.pricePerUnit), tc.rate)
			require.Equal(t, tc.expected, result)
		})
	}
}
