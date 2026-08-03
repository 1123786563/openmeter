package aiusage

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"
)

func ptrString(s string) *string {
	return &s
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func validRateCardEntry() CustomerRateCardEntry {
	return CustomerRateCardEntry{
		Namespace:       "ns-1",
		ResourceCode:    ResourceLLMInputTokens,
		PricePerUnitCNY: alpacadecimal.NewFromFloat(0.001),
		CreditRate:      1000,
		EffectiveFrom:   time.Now().Add(-24 * time.Hour),
	}
}

func TestCustomerRateCardEntry_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		require.NoError(t, validRateCardEntry().Validate())
	})

	t.Run("missing namespace", func(t *testing.T) {
		e := validRateCardEntry()
		e.Namespace = ""
		require.Error(t, e.Validate())
	})

	t.Run("invalid resource code", func(t *testing.T) {
		e := validRateCardEntry()
		e.ResourceCode = "bogus"
		require.Error(t, e.Validate())
	})

	t.Run("zero credit rate", func(t *testing.T) {
		e := validRateCardEntry()
		e.CreditRate = 0
		require.Error(t, e.Validate())
	})

	t.Run("zero effective_from", func(t *testing.T) {
		e := validRateCardEntry()
		e.EffectiveFrom = time.Time{}
		require.Error(t, e.Validate())
	})

	t.Run("negative price", func(t *testing.T) {
		e := validRateCardEntry()
		e.PricePerUnitCNY = alpacadecimal.NewFromFloat(-0.001)
		require.Error(t, e.Validate())
	})

	t.Run("effective_to before effective_from", func(t *testing.T) {
		e := validRateCardEntry()
		e.EffectiveTo = ptrTime(e.EffectiveFrom.Add(-1 * time.Hour))
		require.Error(t, e.Validate())
	})
}

func TestCustomerRateCardEntry_MatchPriority(t *testing.T) {
	tests := []struct {
		name     string
		entry    CustomerRateCardEntry
		expected int
	}{
		{
			name: "customer + provider + model",
			entry: CustomerRateCardEntry{
				CustomerID: ptrString("cust-1"),
				Provider:   ptrString("openai"),
				Model:      ptrString("gpt-4"),
			},
			expected: 100,
		},
		{
			name: "customer + provider only",
			entry: CustomerRateCardEntry{
				CustomerID: ptrString("cust-1"),
				Provider:   ptrString("openai"),
			},
			expected: 80,
		},
		{
			name: "customer + resource only",
			entry: CustomerRateCardEntry{
				CustomerID: ptrString("cust-1"),
			},
			expected: 50,
		},
		{
			name: "namespace + provider + model",
			entry: CustomerRateCardEntry{
				Provider: ptrString("openai"),
				Model:    ptrString("gpt-4"),
			},
			expected: 50,
		},
		{
			name:     "namespace default only",
			entry:    CustomerRateCardEntry{},
			expected: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, tc.entry.MatchPriority())
		})
	}
}

func TestCustomerRateCardEntry_IsActiveAt(t *testing.T) {
	now := time.Now()
	e := CustomerRateCardEntry{
		EffectiveFrom: now.Add(-1 * time.Hour),
	}

	require.True(t, e.IsActiveAt(now))
	require.False(t, e.IsActiveAt(now.Add(-2*time.Hour)))

	e.EffectiveTo = ptrTime(now.Add(1 * time.Hour))
	require.True(t, e.IsActiveAt(now))
	require.False(t, e.IsActiveAt(now.Add(2*time.Hour)))
}

func TestCustomerRateCardEntry_Matches(t *testing.T) {
	cust := "cust-1"
	e := CustomerRateCardEntry{
		Namespace:    "ns-1",
		CustomerID:   &cust,
		ResourceCode: ResourceLLMInputTokens,
		Provider:     ptrString("openai"),
		Model:        ptrString("gpt-4"),
	}

	require.True(t, e.Matches("ns-1", "cust-1", ResourceLLMInputTokens, "openai", "gpt-4"))
	require.False(t, e.Matches("ns-2", "cust-1", ResourceLLMInputTokens, "openai", "gpt-4"))
	require.False(t, e.Matches("ns-1", "cust-2", ResourceLLMInputTokens, "openai", "gpt-4"))
	require.False(t, e.Matches("ns-1", "cust-1", ResourceLLMOutputTokens, "openai", "gpt-4"))
	require.False(t, e.Matches("ns-1", "cust-1", ResourceLLMInputTokens, "anthropic", "gpt-4"))
	require.False(t, e.Matches("ns-1", "cust-1", ResourceLLMInputTokens, "openai", "claude-3"))

	// Namespace default (no customer) should match any customer
	nsDefault := CustomerRateCardEntry{
		Namespace:    "ns-1",
		ResourceCode: ResourceLLMInputTokens,
	}
	require.True(t, nsDefault.Matches("ns-1", "any-customer", ResourceLLMInputTokens, "any", "any"))
}
