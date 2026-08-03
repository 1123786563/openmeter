package pricing

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/aiusage"
)

var testOccurredAt = time.Date(2026, 8, 3, 1, 2, 3, 0, time.UTC)

// TestCeilCreditsProperties verifies the algebraic properties of CeilCredits
// that the pricing engine depends on. These are deterministic property checks
// (not randomized fuzz), complementing the aiusage package's own fuzz tests.
func TestCeilCreditsProperties(t *testing.T) {
	t.Run("zero quantity always returns zero", func(t *testing.T) {
		for _, cpu := range []int64{0, 1, 2, 100, math.MaxInt64} {
			for _, us := range []int64{1, 1000, math.MaxInt64} {
				got, err := aiusage.CeilCredits(0, cpu, us)
				require.NoError(t, err)
				assert.EqualValues(t, 0, got, "cpu=%d us=%d", cpu, us)
			}
		}
	})

	t.Run("exact division has no ceiling overhead", func(t *testing.T) {
		for _, tc := range []struct {
			qty, cpu, us int64
		}{
			{1000, 2, 1000},
			{5000, 1, 100},
			{1, 5, 1},
		} {
			got, err := aiusage.CeilCredits(tc.qty, tc.cpu, tc.us)
			require.NoError(t, err)
			// exact division: ceil == floor == qty*cpu/us
			expected := tc.qty * tc.cpu / tc.us
			assert.EqualValues(t, expected, got, "qty=%d cpu=%d us=%d", tc.qty, tc.cpu, tc.us)
		}
	})

	t.Run("remainder rounds up by exactly one", func(t *testing.T) {
		for _, tc := range []struct {
			qty, cpu, us int64
		}{
			{501, 2, 1000},  // floor=1, ceil=2
			{1, 2, 1000},    // floor=0, ceil=1
			{999, 2, 1000},  // floor=1, ceil=2
			{1001, 2, 1000}, // floor=2, ceil=3
		} {
			got, err := aiusage.CeilCredits(tc.qty, tc.cpu, tc.us)
			require.NoError(t, err)
			floor := tc.qty * tc.cpu / tc.us
			assert.EqualValues(t, floor+1, got, "qty=%d cpu=%d us=%d", tc.qty, tc.cpu, tc.us)
		}
	})

	t.Run("merge equivalence: ceil(a+b) == ceil(a)+ceil(b) when divisible", func(t *testing.T) {
		// When both halves divide evenly, merging is equivalent to summing.
		cpu, us := int64(2), int64(1000)
		merged, err := aiusage.CeilCredits(1000, cpu, us)
		require.NoError(t, err)

		half, err := aiusage.CeilCredits(500, cpu, us)
		require.NoError(t, err)

		assert.Equal(t, merged, half+half, "two 500-token lines should equal one 1000-token line")
	})

	t.Run("monotonic: more quantity never yields fewer credits", func(t *testing.T) {
		cpu, us := int64(2), int64(1000)
		var prev int64 = 0
		for q := int64(0); q <= 3000; q += 50 {
			got, err := aiusage.CeilCredits(q, cpu, us)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, got, prev, "quantity=%d", q)
			prev = got
		}
	})
}

// FuzzResolveNeverPanics is a smoke-level fuzz that ensures Resolve never
// panics for arbitrary quantity values, even at the int64 boundary.
func FuzzResolveNeverPanics(f *testing.F) {
	f.Add(int64(0))
	f.Add(int64(1))
	f.Add(int64(500))
	f.Add(int64(1000))
	f.Add(int64(-1))          // invalid — should be handled gracefully
	f.Add(int64(math.MaxInt64))

	f.Fuzz(func(t *testing.T, qty int64) {
		svc := newPricingServiceWithRates(t)
		// We don't assert on error here; we only verify no panic.
		_, _ = svc.Resolve(t.Context(), ResolveInput{
			OccurredAt:         testOccurredAt,
			RatePackageVersion: "pro-v1",
			BillingMode:        aiusage.BillingModeComponent,
			ProviderManaged:    true,
			Lines: []aiusage.UsageLineInput{
				{ResourceCode: aiusage.ResourceLLMInputTokens, Quantity: qty, Provider: "openai", Model: "gpt-test"},
			},
		})
	})
}
