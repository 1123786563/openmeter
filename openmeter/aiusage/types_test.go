package aiusage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func validLineItem() UsageLineItem {
	return UsageLineItem{
		ResourceCode:    ResourceChatInputToken,
		Quantity:        1000,
		Provider:        "openai",
		Model:           "gpt-4",
		ProviderManaged: true,
	}
}

func validBatch() AIUsageBatch {
	return AIUsageBatch{
		Namespace:      "ns-1",
		CustomerID:     "cust-1",
		SubjectID:      "subj-1",
		UsageBatchID:   "batch-01J0K3X9FHTC6QZ4D2N9R7YBPW",
		TenantSeq:      1,
		OccurredAt:     time.Now(),
		RateVersion:    "v1",
		BillingMode:    BillingModeComponent,
		PayloadHash:    "abc123",
		Status:         BatchStatusPending,
		LineItems:      []UsageLineItem{validLineItem()},
	}
}

func TestAIUsageBatch_Validate(t *testing.T) {
	t.Run("valid component batch", func(t *testing.T) {
		b := validBatch()
		require.NoError(t, b.Validate())
	})

	t.Run("valid bundle batch", func(t *testing.T) {
		ceiling := int64(500)
		b := validBatch()
		b.BillingMode = BillingModeBundle
		b.CeilingCredits = &ceiling
		b.LineItems = nil
		require.NoError(t, b.Validate())
	})

	t.Run("missing namespace", func(t *testing.T) {
		b := validBatch()
		b.Namespace = ""
		err := b.Validate()
		require.Error(t, err)
	})

	t.Run("missing customer_id", func(t *testing.T) {
		b := validBatch()
		b.CustomerID = ""
		err := b.Validate()
		require.Error(t, err)
	})

	t.Run("zero tenant_seq", func(t *testing.T) {
		b := validBatch()
		b.TenantSeq = 0
		err := b.Validate()
		require.Error(t, err)
	})

	t.Run("invalid billing mode", func(t *testing.T) {
		b := validBatch()
		b.BillingMode = "invalid"
		err := b.Validate()
		require.Error(t, err)
	})

	t.Run("bundle without ceiling", func(t *testing.T) {
		b := validBatch()
		b.BillingMode = BillingModeBundle
		b.CeilingCredits = nil
		b.LineItems = nil
		err := b.Validate()
		require.Error(t, err)
	})

	t.Run("component without line items", func(t *testing.T) {
		b := validBatch()
		b.LineItems = nil
		err := b.Validate()
		require.Error(t, err)
	})

	t.Run("missing payload_hash", func(t *testing.T) {
		b := validBatch()
		b.PayloadHash = ""
		err := b.Validate()
		require.Error(t, err)
	})
}

func TestUsageLineItem_Validate(t *testing.T) {
	t.Run("valid provider-managed item", func(t *testing.T) {
		i := validLineItem()
		require.NoError(t, i.Validate())
	})

	t.Run("valid platform resource item", func(t *testing.T) {
		i := UsageLineItem{
			ResourceCode:    ResourceRAGRetrieval,
			Quantity:        5,
			ProviderManaged: false,
		}
		require.NoError(t, i.Validate())
	})

	t.Run("invalid resource code", func(t *testing.T) {
		i := validLineItem()
		i.ResourceCode = "bogus"
		err := i.Validate()
		require.Error(t, err)
	})

	t.Run("zero quantity", func(t *testing.T) {
		i := validLineItem()
		i.Quantity = 0
		err := i.Validate()
		require.Error(t, err)
	})

	t.Run("provider-managed without provider", func(t *testing.T) {
		i := validLineItem()
		i.Provider = ""
		err := i.Validate()
		require.Error(t, err)
	})

	t.Run("provider-managed without model", func(t *testing.T) {
		i := validLineItem()
		i.Model = ""
		err := i.Validate()
		require.Error(t, err)
	})
}

func TestResourceCode_Classification(t *testing.T) {
	tests := []struct {
		code             ResourceCode
		providerManaged  bool
		platformResource bool
		unit             string
	}{
		{ResourceChatInputToken, true, false, "token"},
		{ResourceChatOutputToken, true, false, "token"},
		{ResourceChatCacheReadToken, true, false, "token"},
		{ResourceChatCacheWriteToken, true, false, "token"},
		{ResourceChatReasoningToken, true, false, "token"},
		{ResourceEmbeddingToken, true, false, "token"},
		{ResourceRerankCall, true, false, "call"},
		{ResourceVLMInputToken, true, false, "token"},
		{ResourceVLMOutputToken, true, false, "token"},
		{ResourceVLMImage, true, false, "image"},
		{ResourceASRSeconds, true, false, "second"},
		{ResourceRAGRetrieval, false, true, "call"},
		{ResourceDocParsePage, false, true, "page"},
		{ResourceMCPCall, false, true, "call"},
		{ResourceWebSearch, false, true, "call"},
		{ResourceAgentRun, false, true, "call"},
	}

	for _, tc := range tests {
		t.Run(string(tc.code), func(t *testing.T) {
			require.Equal(t, tc.providerManaged, tc.code.IsProviderManaged(),
				"IsProviderManaged mismatch for %s", tc.code)
			require.Equal(t, tc.platformResource, tc.code.IsPlatformResource(),
				"IsPlatformResource mismatch for %s", tc.code)
			require.Equal(t, tc.unit, tc.code.Unit(),
				"Unit mismatch for %s", tc.code)
			require.NoError(t, tc.code.Validate())
			require.True(t, ValidResourceCode(string(tc.code)))
		})
	}

	t.Run("unknown code", func(t *testing.T) {
		c := ResourceCode("nonexistent")
		require.Error(t, c.Validate())
		require.False(t, ValidResourceCode("nonexistent"))
	})
}

func TestBillingMode_Validate(t *testing.T) {
	require.NoError(t, BillingModeComponent.Validate())
	require.NoError(t, BillingModeBundle.Validate())
	require.Error(t, BillingMode("invalid").Validate())
}

func TestBatchStatus_Validate(t *testing.T) {
	require.NoError(t, BatchStatusPending.Validate())
	require.NoError(t, BatchStatusSettled.Validate())
	require.NoError(t, BatchStatusRejected.Validate())
	require.NoError(t, BatchStatusCompensated.Validate())
	require.Error(t, BatchStatus("invalid").Validate())
}

func TestIngestBatchInput_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		i := IngestBatchInput{
			Namespace:    "ns-1",
			CustomerID:   "cust-1",
			SubjectID:    "subj-1",
			UsageBatchID: "batch-001",
			TenantSeq:    1,
			PayloadHash:  "abc",
			BillingMode:  BillingModeComponent,
			LineItems:    []UsageLineItem{validLineItem()},
		}
		require.NoError(t, i.Validate())
	})

	t.Run("missing namespace", func(t *testing.T) {
		i := IngestBatchInput{
			CustomerID:   "c",
			SubjectID:    "s",
			UsageBatchID: "b",
			TenantSeq:    1,
			PayloadHash:  "h",
			BillingMode:  BillingModeComponent,
			LineItems:    []UsageLineItem{validLineItem()},
		}
		require.Error(t, i.Validate())
	})
}
