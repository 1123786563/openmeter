package meterregistry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/aiusage"
)

type fakeMeterManager struct {
	created map[string]string // slug -> aggregation type
}

func (f *fakeMeterManager) EnsureMeter(_ context.Context, slug string, aggType string, _ string) error {
	if f.created == nil {
		f.created = make(map[string]string)
	}
	f.created[slug] = aggType
	return nil
}

func TestPublishAll16Resources(t *testing.T) {
	mgr := &fakeMeterManager{}
	reg := New(mgr)

	set := DefaultResourceSchemas()
	require.Len(t, set.Schemas, 16, "design defines exactly 16 AI resources")

	err := reg.Publish(t.Context(), set)
	require.NoError(t, err)

	assert.Len(t, mgr.created, 16, "all 16 meters should be created")
	assert.Len(t, reg.All(), 16, "registry should contain all 16 schemas")
}

func TestPublishIsIdempotent(t *testing.T) {
	mgr := &fakeMeterManager{}
	reg := New(mgr)

	set := DefaultResourceSchemas()

	err := reg.Publish(t.Context(), set)
	require.NoError(t, err)

	err = reg.Publish(t.Context(), set)
	require.NoError(t, err)

	// Registry still has 16 entries (not 32).
	assert.Len(t, reg.All(), 16)
}

func TestLookupExistingResource(t *testing.T) {
	mgr := &fakeMeterManager{}
	reg := New(mgr)

	err := reg.Publish(t.Context(), DefaultResourceSchemas())
	require.NoError(t, err)

	schema, err := reg.Lookup(aiusage.ResourceChatInputToken)
	require.NoError(t, err)
	assert.Equal(t, "token", schema.Unit)
	assert.Equal(t, AggregationSum, schema.Aggregation)
	assert.True(t, schema.ProviderManaged)
}

func TestLookupUnknownResource(t *testing.T) {
	mgr := &fakeMeterManager{}
	reg := New(mgr)

	err := reg.Publish(t.Context(), DefaultResourceSchemas())
	require.NoError(t, err)

	_, err = reg.Lookup(aiusage.ResourceCode("bogus"))
	require.ErrorIs(t, err, aiusage.ErrResourceUnknown)
}

func TestPlatformResourcesNotProviderManaged(t *testing.T) {
	mgr := &fakeMeterManager{}
	reg := New(mgr)

	err := reg.Publish(t.Context(), DefaultResourceSchemas())
	require.NoError(t, err)

	for _, code := range []aiusage.ResourceCode{
		aiusage.ResourceRAGRetrieval,
		aiusage.ResourceDocParsePage,
		aiusage.ResourceMCPCall,
		aiusage.ResourceWebSearch,
		aiusage.ResourceAgentRun,
	} {
		schema, err := reg.Lookup(code)
		require.NoError(t, err)
		assert.False(t, schema.ProviderManaged, "%s should not be provider-managed", code)
	}
}

func TestModelTokenResourcesAreProviderManaged(t *testing.T) {
	mgr := &fakeMeterManager{}
	reg := New(mgr)

	err := reg.Publish(t.Context(), DefaultResourceSchemas())
	require.NoError(t, err)

	for _, code := range []aiusage.ResourceCode{
		aiusage.ResourceChatInputToken,
		aiusage.ResourceChatOutputToken,
		aiusage.ResourceChatCacheReadToken,
		aiusage.ResourceChatCacheWriteToken,
		aiusage.ResourceChatReasoningToken,
		aiusage.ResourceEmbeddingToken,
		aiusage.ResourceRerankCall,
		aiusage.ResourceVLMInputToken,
		aiusage.ResourceVLMOutputToken,
		aiusage.ResourceVLMImage,
		aiusage.ResourceASRSeconds,
	} {
		schema, err := reg.Lookup(code)
		require.NoError(t, err)
		assert.True(t, schema.ProviderManaged, "%s should be provider-managed", code)
	}
}

func TestMeterSlugNaming(t *testing.T) {
	mgr := &fakeMeterManager{}
	reg := New(mgr)

	err := reg.Publish(t.Context(), DefaultResourceSchemas())
	require.NoError(t, err)

	_, ok := mgr.created["ai_chat_input_token"]
	assert.True(t, ok, "meter slug should be ai_<resource_code>")
}
