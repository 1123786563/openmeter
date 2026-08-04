package meterregistry

import (
	"context"
	"fmt"
	"sync"

	"github.com/openmeterio/openmeter/openmeter/aiusage"
)

// AggregationType defines how a resource's usage is aggregated.
type AggregationType string

const (
	AggregationSum   AggregationType = "SUM"
	AggregationCount AggregationType = "COUNT"
)

// ResourceSchema describes one billable AI resource for the meter registry.
type ResourceSchema struct {
	// Code is the canonical resource identifier (matches aiusage.ResourceCode).
	Code aiusage.ResourceCode

	// Category classifies the resource for reporting.
	Category string

	// Unit is the billing unit ("token", "call", "second", "page", "image").
	Unit string

	// Aggregation is how usage events accumulate (SUM or COUNT).
	Aggregation AggregationType

	// ProviderManaged is true when the platform calls the provider API.
	// BYOK resources have this as false for model tokens; platform resources
	// are always billed regardless.
	ProviderManaged bool
}

// ResourceSchemaSet is the complete set of resource schemas published by the registry.
type ResourceSchemaSet struct {
	Schemas []ResourceSchema
}

// Registry manages the set of known AI resource schemas and orchestrates
// meter/feature creation for them via the existing meter management service.
// It does NOT rewrite existing aggregators — it only publishes resource definitions.
type Registry interface {
	// Publish registers the given resource schema set, ensuring all meters
	// and features exist for the resources. It is idempotent.
	Publish(ctx context.Context, set ResourceSchemaSet) error

	// Lookup returns the schema for a resource code.
	Lookup(code aiusage.ResourceCode) (ResourceSchema, error)

	// All returns every registered schema.
	All() []ResourceSchema
}

// MeterManager is the interface for creating meters/features in OpenMeter.
// The registry delegates meter creation to this interface.
type MeterManager interface {
	EnsureMeter(ctx context.Context, slug string, aggregationType string, unit string) error
}

// DefaultResourceSchemas returns the 16 AI resource schemas defined in the
// OpenMeter AI billing design. These cover all WeKnora AI consumption types.
func DefaultResourceSchemas() ResourceSchemaSet {
	return ResourceSchemaSet{
		Schemas: []ResourceSchema{
			// Model tokens (provider-managed)
			{Code: aiusage.ResourceLLMInputTokens, Category: "llm", Unit: "token", Aggregation: AggregationSum, ProviderManaged: true},
			{Code: aiusage.ResourceLLMOutputTokens, Category: "llm", Unit: "token", Aggregation: AggregationSum, ProviderManaged: true},
			{Code: aiusage.ResourceLLMCacheReadTokens, Category: "llm", Unit: "token", Aggregation: AggregationSum, ProviderManaged: true},
			{Code: aiusage.ResourceLLMCacheWriteTokens, Category: "llm", Unit: "token", Aggregation: AggregationSum, ProviderManaged: true},
			{Code: aiusage.ResourceLLMReasoningTokens, Category: "llm", Unit: "token", Aggregation: AggregationSum, ProviderManaged: true},
			// Embedding and rerank
			{Code: aiusage.ResourceEmbeddingTokens, Category: "embedding", Unit: "token", Aggregation: AggregationSum, ProviderManaged: true},
			{Code: aiusage.ResourceRerankCalls, Category: "rerank", Unit: "call", Aggregation: AggregationCount, ProviderManaged: true},
			// Multimodal
			{Code: aiusage.ResourceVLMInputTokens, Category: "vlm", Unit: "token", Aggregation: AggregationSum, ProviderManaged: true},
			{Code: aiusage.ResourceVLMOutputTokens, Category: "vlm", Unit: "token", Aggregation: AggregationSum, ProviderManaged: true},
			{Code: aiusage.ResourceVLMImages, Category: "vlm", Unit: "image", Aggregation: AggregationSum, ProviderManaged: true},
			// Speech
			{Code: aiusage.ResourceASRMilliseconds, Category: "asr", Unit: "millisecond", Aggregation: AggregationSum, ProviderManaged: true},
			// Platform resources (always billed)
			{Code: aiusage.ResourceRAGQueries, Category: "rag", Unit: "call", Aggregation: AggregationCount, ProviderManaged: false},
			{Code: aiusage.ResourceDocParsePages, Category: "document", Unit: "page", Aggregation: AggregationSum, ProviderManaged: false},
			{Code: aiusage.ResourceMCPToolCalls, Category: "mcp", Unit: "call", Aggregation: AggregationCount, ProviderManaged: false},
			{Code: aiusage.ResourceWebSearches, Category: "web", Unit: "call", Aggregation: AggregationCount, ProviderManaged: false},
			{Code: aiusage.ResourceAgentRuns, Category: "agent", Unit: "call", Aggregation: AggregationCount, ProviderManaged: false},
		},
	}
}

type registry struct {
	mu      sync.RWMutex
	schemas map[aiusage.ResourceCode]ResourceSchema
	manager MeterManager
}

// New creates a Registry backed by the given MeterManager.
func New(manager MeterManager) Registry {
	return &registry{
		schemas: make(map[aiusage.ResourceCode]ResourceSchema),
		manager: manager,
	}
}

// Publish implements Registry.
func (r *registry) Publish(ctx context.Context, set ResourceSchemaSet) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, schema := range set.Schemas {
		if err := schema.Code.Validate(); err != nil {
			return fmt.Errorf("meterregistry: invalid resource code %q: %w", schema.Code, err)
		}
		// Delegate meter/feature creation to the existing meter management service.
		if r.manager != nil {
			meterSlug := fmt.Sprintf("ai_%s", schema.Code)
			if err := r.manager.EnsureMeter(ctx, meterSlug, string(schema.Aggregation), schema.Unit); err != nil {
				return fmt.Errorf("meterregistry: failed to ensure meter for %s: %w", schema.Code, err)
			}
		}
		r.schemas[schema.Code] = schema
	}

	return nil
}

// Lookup implements Registry.
func (r *registry) Lookup(code aiusage.ResourceCode) (ResourceSchema, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schema, ok := r.schemas[code]
	if !ok {
		return ResourceSchema{}, fmt.Errorf("%w: %s", aiusage.ErrResourceUnknown, code)
	}
	return schema, nil
}

// All implements Registry.
func (r *registry) All() []ResourceSchema {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ResourceSchema, 0, len(r.schemas))
	for _, s := range r.schemas {
		result = append(result, s)
	}
	return result
}

// Compile-time interface check.
var _ Registry = (*registry)(nil)
