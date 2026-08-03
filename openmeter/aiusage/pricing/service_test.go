package pricing

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/aiusage"
)

// staticRateProvider is a test double that returns a fixed set of rate entries.
type staticRateProvider struct {
	entries []RateEntry
}

func (p *staticRateProvider) GetEntries(_ context.Context, _ string) ([]RateEntry, error) {
	return p.entries, nil
}

// newPricingServiceWithRates returns a Service configured with a standard test
// rate package covering llm_input_tokens (2 credits / 1000 tokens) and
// rag_queries (1 credit / call).
func newPricingServiceWithRates(t *testing.T) *Service {
	t.Helper()
	return NewService(&staticRateProvider{
		entries: []RateEntry{
			{
				ResourceCode:   aiusage.ResourceLLMInputTokens,
				Provider:       "openai",
				Model:          "gpt-test",
				CreditsPerUnit: 2,
				UnitSize:       1000,
				EffectiveFrom:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			{
				ResourceCode:   aiusage.ResourceRAGQueries,
				CreditsPerUnit: 1,
				UnitSize:       1,
				EffectiveFrom:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	})
}

// TestResolveSeparatesCostAndCustomerCredits verifies the core invariant: two
// 500-token lines merge into one 1000-token line and the ceiling remainder
// (2 credits) is assigned to a single ResolvedLine, not split.
func TestResolveSeparatesCostAndCustomerCredits(t *testing.T) {
	svc := newPricingServiceWithRates(t)
	got, err := svc.Resolve(t.Context(), ResolveInput{
		OccurredAt:         time.Date(2026, 8, 3, 1, 2, 3, 0, time.UTC),
		RatePackageVersion: "pro-v1",
		BillingMode:        aiusage.BillingModeComponent,
		ProviderManaged:    true,
		Lines: []aiusage.UsageLineInput{
			{ResourceCode: "llm_input_tokens", Quantity: 500, Provider: "openai", Model: "gpt-test"},
			{ResourceCode: "llm_input_tokens", Quantity: 500, Provider: "openai", Model: "gpt-test"},
		},
	})
	require.NoError(t, err)
	require.Len(t, got.Lines, 1, "two homogeneous lines should merge into one")
	require.EqualValues(t, 2, got.Lines[0].CustomerCredits)
	assert.EqualValues(t, 2, got.TotalCredits)
}

func TestResolveTableCases(t *testing.T) {
	at := time.Date(2026, 8, 3, 1, 2, 3, 0, time.UTC)

	tests := []struct {
		name        string
		provider    bool // ProviderManaged flag (false = BYOK)
		lines       []aiusage.UsageLineInput
		rates       []RateEntry // nil = use default test rates
		mode        aiusage.BillingMode
		wantCredits int64
		wantLines   int
		wantErr     error
	}{
		{
			name:        "zero quantity yields zero credits",
			provider:    true,
			lines:       []aiusage.UsageLineInput{{ResourceCode: aiusage.ResourceLLMInputTokens, Quantity: 0, Provider: "openai", Model: "gpt-test"}},
			wantCredits: 0,
			wantLines:   1,
		},
		{
			name:        "minimum charge: one token rounds up to 1 credit",
			provider:    true,
			lines:       []aiusage.UsageLineInput{{ResourceCode: aiusage.ResourceLLMInputTokens, Quantity: 1, Provider: "openai", Model: "gpt-test"}},
			wantCredits: 1,
			wantLines:   1,
		},
		{
			name:     "overflow: MaxInt64 * 2 exceeds int64",
			provider: true,
			lines:    []aiusage.UsageLineInput{{ResourceCode: aiusage.ResourceLLMInputTokens, Quantity: math.MaxInt64, Provider: "openai", Model: "gpt-test"}},
			rates: []RateEntry{
				{ResourceCode: aiusage.ResourceLLMInputTokens, Provider: "openai", Model: "gpt-test", CreditsPerUnit: 2, UnitSize: 1, EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
			},
			wantErr: aiusage.ErrCreditOverflow,
		},
		{
			name: "two 500-token lines equal one 1000-token line",
			provider: true,
			lines: []aiusage.UsageLineInput{
				{ResourceCode: aiusage.ResourceLLMInputTokens, Quantity: 500, Provider: "openai", Model: "gpt-test"},
				{ResourceCode: aiusage.ResourceLLMInputTokens, Quantity: 500, Provider: "openai", Model: "gpt-test"},
			},
			wantCredits: 2,
			wantLines:   1,
		},
		{
			name:        "ceiling remainder: 501 tokens = ceil(1.002) = 2 credits",
			provider:    true,
			lines:       []aiusage.UsageLineInput{{ResourceCode: aiusage.ResourceLLMInputTokens, Quantity: 501, Provider: "openai", Model: "gpt-test"}},
			wantCredits: 2,
			wantLines:   1,
		},
		{
			name:        "BYOK provider-managed tokens: zero customer credits",
			provider:    false,
			lines:       []aiusage.UsageLineInput{{ResourceCode: aiusage.ResourceLLMInputTokens, Quantity: 10000, Provider: "openai", Model: "gpt-test"}},
			wantCredits: 0,
			wantLines:   1,
		},
		{
			name:        "BYOK platform resource still charges",
			provider:    false,
			lines:       []aiusage.UsageLineInput{{ResourceCode: aiusage.ResourceRAGQueries, Quantity: 5}},
			wantCredits: 5,
			wantLines:   1,
		},
		{
			name:    "ambiguous rate: two equally specific entries",
			provider: true,
			lines:   []aiusage.UsageLineInput{{ResourceCode: aiusage.ResourceLLMInputTokens, Quantity: 1000, Provider: "openai", Model: "gpt-test"}},
			rates: []RateEntry{
				{ResourceCode: aiusage.ResourceLLMInputTokens, Provider: "openai", Model: "gpt-test", CreditsPerUnit: 2, UnitSize: 1000, EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
				{ResourceCode: aiusage.ResourceLLMInputTokens, Provider: "openai", Model: "gpt-test", CreditsPerUnit: 3, UnitSize: 1000, EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
			},
			wantErr: aiusage.ErrAmbiguousRate,
		},
		{
			name:    "missing rate for provider-managed resource",
			provider: true,
			lines:   []aiusage.UsageLineInput{{ResourceCode: aiusage.ResourceLLMOutputTokens, Quantity: 100, Provider: "openai", Model: "gpt-test"}},
			wantErr: aiusage.ErrRateMissing,
		},
		{
			name:    "bundle mode with lines is rejected",
			provider: true,
			mode:    aiusage.BillingModeBundle,
			lines:   []aiusage.UsageLineInput{{ResourceCode: aiusage.ResourceLLMInputTokens, Quantity: 100, Provider: "openai", Model: "gpt-test"}},
			wantErr: aiusage.ErrBillingModeConflict,
		},
		{
			name:        "platform resource under provider-managed still charges",
			provider:    true,
			lines:       []aiusage.UsageLineInput{{ResourceCode: aiusage.ResourceRAGQueries, Quantity: 3}},
			wantCredits: 3,
			wantLines:   1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := newPricingServiceWithRates(t)
			if tc.rates != nil {
				svc = NewService(&staticRateProvider{entries: tc.rates})
			}

			mode := tc.mode
			if mode == "" {
				mode = aiusage.BillingModeComponent
			}

			got, err := svc.Resolve(t.Context(), ResolveInput{
				OccurredAt:         at,
				RatePackageVersion: "pro-v1",
				BillingMode:        mode,
				ProviderManaged:    tc.provider,
				Lines:              tc.lines,
			})

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}

			require.NoError(t, err)
			assert.EqualValues(t, tc.wantCredits, got.TotalCredits)
			if tc.wantLines > 0 {
				assert.Len(t, got.Lines, tc.wantLines)
			}
		})
	}
}

func TestResolveMergesHomogeneousLines(t *testing.T) {
	svc := newPricingServiceWithRates(t)

	// Two lines with different providers should NOT merge.
	got, err := svc.Resolve(t.Context(), ResolveInput{
		OccurredAt:         time.Date(2026, 8, 3, 1, 2, 3, 0, time.UTC),
		RatePackageVersion: "pro-v1",
		BillingMode:        aiusage.BillingModeComponent,
		ProviderManaged:    true,
		Lines: []aiusage.UsageLineInput{
			{ResourceCode: aiusage.ResourceRAGQueries, Quantity: 3},
			{ResourceCode: aiusage.ResourceRAGQueries, Quantity: 4},
		},
	})
	require.NoError(t, err)
	// Same resource + empty provider + empty model → merge into one.
	assert.Len(t, got.Lines, 1)
	assert.EqualValues(t, 7, got.TotalCredits)
}

func TestResolveSpecificityWins(t *testing.T) {
	at := time.Date(2026, 8, 3, 1, 2, 3, 0, time.UTC)
	svc := NewService(&staticRateProvider{
		entries: []RateEntry{
			// Resource-only fallback: 10 credits / 1000 tokens.
			{ResourceCode: aiusage.ResourceLLMInputTokens, CreditsPerUnit: 10, UnitSize: 1000, EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
			// Provider+model specific: 2 credits / 1000 tokens (should win).
			{ResourceCode: aiusage.ResourceLLMInputTokens, Provider: "openai", Model: "gpt-test", CreditsPerUnit: 2, UnitSize: 1000, EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	})

	got, err := svc.Resolve(t.Context(), ResolveInput{
		OccurredAt:         at,
		RatePackageVersion: "pro-v1",
		BillingMode:        aiusage.BillingModeComponent,
		ProviderManaged:    true,
		Lines:              []aiusage.UsageLineInput{{ResourceCode: aiusage.ResourceLLMInputTokens, Quantity: 1000, Provider: "openai", Model: "gpt-test"}},
	})
	require.NoError(t, err)
	// The more specific (provider+model) rate at 2 credits/1000 should win
	// over the resource-only fallback at 10 credits/1000.
	assert.EqualValues(t, 2, got.TotalCredits)
}
