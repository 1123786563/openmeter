package aiusage

// ResourceCode classifies a billable AI resource.
type ResourceCode string

const (
	// Model tokens (provider-managed)
	ResourceLLMInputTokens      ResourceCode = "llm_input_tokens"
	ResourceLLMOutputTokens     ResourceCode = "llm_output_tokens"
	ResourceLLMCacheReadTokens  ResourceCode = "llm_cache_read_tokens"
	ResourceLLMCacheWriteTokens ResourceCode = "llm_cache_write_tokens"
	ResourceLLMReasoningTokens  ResourceCode = "llm_reasoning_tokens"

	// Embedding and rerank
	ResourceEmbeddingTokens ResourceCode = "embedding_tokens"
	ResourceRerankCalls     ResourceCode = "rerank_calls"

	// Multimodal
	ResourceVLMInputTokens  ResourceCode = "vlm_input_tokens"
	ResourceVLMOutputTokens ResourceCode = "vlm_output_tokens"
	ResourceVLMImages       ResourceCode = "vlm_images"

	// Speech
	ResourceASRMilliseconds ResourceCode = "asr_milliseconds"

	// Platform resources (always billed, even under BYOK)
	ResourceRAGQueries    ResourceCode = "rag_queries"
	ResourceDocParsePages ResourceCode = "doc_parse_pages"
	ResourceMCPToolCalls  ResourceCode = "mcp_tool_calls"
	ResourceWebSearches   ResourceCode = "web_searches"
	ResourceAgentRuns     ResourceCode = "agent_runs"
)

type resourceMetadata struct {
	ProviderManaged bool
	Unit            string
}

var resourceMeta = map[ResourceCode]resourceMetadata{
	ResourceLLMInputTokens:      {ProviderManaged: true, Unit: "token"},
	ResourceLLMOutputTokens:     {ProviderManaged: true, Unit: "token"},
	ResourceLLMCacheReadTokens:  {ProviderManaged: true, Unit: "token"},
	ResourceLLMCacheWriteTokens: {ProviderManaged: true, Unit: "token"},
	ResourceLLMReasoningTokens:  {ProviderManaged: true, Unit: "token"},
	ResourceEmbeddingTokens:     {ProviderManaged: true, Unit: "token"},
	ResourceRerankCalls:         {ProviderManaged: true, Unit: "call"},
	ResourceVLMInputTokens:      {ProviderManaged: true, Unit: "token"},
	ResourceVLMOutputTokens:     {ProviderManaged: true, Unit: "token"},
	ResourceVLMImages:           {ProviderManaged: true, Unit: "image"},
	ResourceASRMilliseconds:     {ProviderManaged: true, Unit: "millisecond"},
	ResourceRAGQueries:          {ProviderManaged: false, Unit: "call"},
	ResourceDocParsePages:       {ProviderManaged: false, Unit: "page"},
	ResourceMCPToolCalls:        {ProviderManaged: false, Unit: "call"},
	ResourceWebSearches:         {ProviderManaged: false, Unit: "call"},
	ResourceAgentRuns:           {ProviderManaged: false, Unit: "call"},
}

// IsProviderManaged returns true for resources backed by an external model provider
// (model tokens, embeddings, rerank, VLM, ASR). BYOK resources have provider_managed=false.
func (r ResourceCode) IsProviderManaged() bool {
	if meta, ok := resourceMeta[r]; ok {
		return meta.ProviderManaged
	}
	return false
}

// IsPlatformResource returns true for WeKnora platform resources (RAG, MCP, web search,
// agent run, document parsing). These are always billed even for BYOK models.
func (r ResourceCode) IsPlatformResource() bool {
	if meta, ok := resourceMeta[r]; ok {
		return !meta.ProviderManaged
	}
	return false
}

// Unit returns the billing unit for this resource ("token", "call", "millisecond", "page", "image").
func (r ResourceCode) Unit() string {
	if meta, ok := resourceMeta[r]; ok {
		return meta.Unit
	}
	return ""
}

// Validate checks whether the ResourceCode is a known constant.
func (r ResourceCode) Validate() error {
	if _, ok := resourceMeta[r]; !ok {
		return ErrInvalidResourceCode
	}
	return nil
}

// ValidResourceCode returns true if the string is a known ResourceCode.
func ValidResourceCode(code string) bool {
	_, ok := resourceMeta[ResourceCode(code)]
	return ok
}
