package aiusage

// ResourceCode classifies a billable AI resource.
type ResourceCode string

const (
	// Model tokens
	ResourceChatInputToken      ResourceCode = "chat_input_token"
	ResourceChatOutputToken     ResourceCode = "chat_output_token"
	ResourceChatCacheReadToken  ResourceCode = "chat_cache_read_token"
	ResourceChatCacheWriteToken ResourceCode = "chat_cache_write_token"
	ResourceChatReasoningToken  ResourceCode = "chat_reasoning_token"

	// Embedding and rerank
	ResourceEmbeddingToken ResourceCode = "embedding_token"
	ResourceRerankCall     ResourceCode = "rerank_call"

	// Multimodal
	ResourceVLMInputToken  ResourceCode = "vlm_input_token"
	ResourceVLMOutputToken ResourceCode = "vlm_output_token"
	ResourceVLMImage       ResourceCode = "vlm_image"

	// Speech
	ResourceASRSeconds ResourceCode = "asr_seconds"

	// Platform resources
	ResourceRAGRetrieval ResourceCode = "rag_retrieval"
	ResourceDocParsePage ResourceCode = "doc_parse_page"
	ResourceMCPCall      ResourceCode = "mcp_call"
	ResourceWebSearch    ResourceCode = "web_search"
	ResourceAgentRun     ResourceCode = "agent_run"
)

type resourceMetadata struct {
	ProviderManaged bool
	Unit            string
}

var resourceMeta = map[ResourceCode]resourceMetadata{
	ResourceChatInputToken:      {ProviderManaged: true, Unit: "token"},
	ResourceChatOutputToken:     {ProviderManaged: true, Unit: "token"},
	ResourceChatCacheReadToken:  {ProviderManaged: true, Unit: "token"},
	ResourceChatCacheWriteToken: {ProviderManaged: true, Unit: "token"},
	ResourceChatReasoningToken:  {ProviderManaged: true, Unit: "token"},
	ResourceEmbeddingToken:      {ProviderManaged: true, Unit: "token"},
	ResourceRerankCall:          {ProviderManaged: true, Unit: "call"},
	ResourceVLMInputToken:       {ProviderManaged: true, Unit: "token"},
	ResourceVLMOutputToken:      {ProviderManaged: true, Unit: "token"},
	ResourceVLMImage:            {ProviderManaged: true, Unit: "image"},
	ResourceASRSeconds:          {ProviderManaged: true, Unit: "second"},
	ResourceRAGRetrieval:        {ProviderManaged: false, Unit: "call"},
	ResourceDocParsePage:        {ProviderManaged: false, Unit: "page"},
	ResourceMCPCall:             {ProviderManaged: false, Unit: "call"},
	ResourceWebSearch:           {ProviderManaged: false, Unit: "call"},
	ResourceAgentRun:            {ProviderManaged: false, Unit: "call"},
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

// Unit returns the billing unit for this resource ("token", "call", "second", "page", "image").
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
