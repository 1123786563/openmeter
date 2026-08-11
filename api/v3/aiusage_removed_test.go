package v3_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
)

func TestAIUsageContractIsRemoved(t *testing.T) {
	spec, err := api.GetSpec()
	require.NoError(t, err)

	retiredPaths := []string{
		"/ai-usage-batches",
		"/ai-usage-batches/{batchId}",
		"/customers/{customerId}/runtime-authorization",
		"/customers/{customerId}/credit-balance",
		"/customers/{customerId}/credit-transactions",
	}
	for _, path := range retiredPaths {
		require.Nilf(t, spec.Paths.Find(path), "retired AIUsage path %s must not be published", path)
	}

	for name := range spec.Components.Schemas {
		require.Falsef(t, strings.HasPrefix(name, "AIUsage"), "retired AIUsage schema %s must not be published", name)
	}

	for _, tag := range spec.Tags {
		require.NotEqual(t, "AI Usage", tag.Name)
	}
}
