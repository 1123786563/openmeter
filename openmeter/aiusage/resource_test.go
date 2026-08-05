package aiusage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResourceCodeIsProviderManaged verifies that model-provider-backed
// resources are correctly classified as provider-managed (and thus zero-rated
// under BYOK), while platform resources are not.
func TestResourceCodeIsProviderManaged(t *testing.T) {
	managed := []ResourceCode{
		ResourceLLMInputTokens, ResourceLLMOutputTokens,
		ResourceLLMCacheReadTokens, ResourceLLMCacheWriteTokens,
		ResourceLLMReasoningTokens, ResourceEmbeddingTokens,
		ResourceRerankCalls, ResourceVLMInputTokens,
		ResourceVLMOutputTokens, ResourceVLMImages,
		ResourceASRMilliseconds,
	}
	for _, r := range managed {
		assert.True(t, r.IsProviderManaged(), "%s should be provider-managed", r)
		assert.False(t, r.IsPlatformResource(), "%s should not be a platform resource", r)
	}
}

// TestResourceCodeIsPlatformResource verifies that platform resources are
// always billed, even under BYOK.
func TestResourceCodeIsPlatformResource(t *testing.T) {
	platform := []ResourceCode{
		ResourceRAGQueries, ResourceDocParsePages,
		ResourceMCPToolCalls, ResourceWebSearches,
		ResourceAgentRuns,
	}
	for _, r := range platform {
		assert.True(t, r.IsPlatformResource(), "%s should be a platform resource", r)
		assert.False(t, r.IsProviderManaged(), "%s should not be provider-managed", r)
	}
}

// TestResourceCodeUnit verifies the billing unit strings.
func TestResourceCodeUnit(t *testing.T) {
	tests := []struct {
		code ResourceCode
		unit string
	}{
		{ResourceLLMInputTokens, "token"},
		{ResourceEmbeddingTokens, "token"},
		{ResourceRerankCalls, "call"},
		{ResourceVLMImages, "image"},
		{ResourceASRMilliseconds, "millisecond"},
		{ResourceRAGQueries, "call"},
		{ResourceDocParsePages, "page"},
		{ResourceMCPToolCalls, "call"},
		{ResourceWebSearches, "call"},
		{ResourceAgentRuns, "call"},
	}
	for _, tc := range tests {
		t.Run(string(tc.code), func(t *testing.T) {
			assert.Equal(t, tc.unit, tc.code.Unit())
		})
	}
}

// TestResourceCodeUnknown verifies that an unrecognized code returns safe
// defaults (false, empty unit) and Validate returns an error.
func TestResourceCodeUnknown(t *testing.T) {
	unknown := ResourceCode("nonexistent_resource")

	assert.False(t, unknown.IsProviderManaged())
	assert.False(t, unknown.IsPlatformResource())
	assert.Equal(t, "", unknown.Unit())

	err := unknown.Validate()
	require.ErrorIs(t, err, ErrInvalidResourceCode)
}

// TestResourceCodeValidate verifies that all known codes pass validation.
func TestResourceCodeValidate(t *testing.T) {
	allCodes := []ResourceCode{
		ResourceLLMInputTokens, ResourceLLMOutputTokens,
		ResourceLLMCacheReadTokens, ResourceLLMCacheWriteTokens,
		ResourceLLMReasoningTokens, ResourceEmbeddingTokens,
		ResourceRerankCalls, ResourceVLMInputTokens,
		ResourceVLMOutputTokens, ResourceVLMImages,
		ResourceASRMilliseconds, ResourceRAGQueries,
		ResourceDocParsePages, ResourceMCPToolCalls,
		ResourceWebSearches, ResourceAgentRuns,
	}
	for _, c := range allCodes {
		require.NoError(t, c.Validate(), "%s must be valid", c)
	}
	assert.Equal(t, 16, len(allCodes), "exactly 16 resource codes expected")
}

// TestValidResourceCode verifies the string-based lookup function.
func TestValidResourceCode(t *testing.T) {
	assert.True(t, ValidResourceCode("llm_input_tokens"))
	assert.True(t, ValidResourceCode("rag_queries"))
	assert.False(t, ValidResourceCode("unknown"))
	assert.False(t, ValidResourceCode(""))
}

// TestResourceCount verifies that the 16 approved Phase 1 resource codes are
// all present — the meter registry and resource matrix tests depend on this
// exact set.
func TestResourceCount(t *testing.T) {
	expected := []string{
		"llm_input_tokens", "llm_output_tokens", "llm_cache_read_tokens",
		"llm_cache_write_tokens", "llm_reasoning_tokens", "embedding_tokens",
		"rerank_calls", "vlm_input_tokens", "vlm_output_tokens", "vlm_images",
		"asr_milliseconds", "rag_queries", "doc_parse_pages",
		"mcp_tool_calls", "web_searches", "agent_runs",
	}
	for _, s := range expected {
		assert.True(t, ValidResourceCode(s), "missing resource code: %s", s)
	}
}
