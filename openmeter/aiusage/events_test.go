package aiusage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestEventNameValues verifies that the event type constants match the
// approved outbox event names from the design spec.
func TestEventNameValues(t *testing.T) {
	assert.Equal(t, "ai_usage.batch.settled", EventBatchSettled)
	assert.Equal(t, "ai_usage.batch.corrected", EventBatchCorrected)
	assert.Equal(t, "credit.balance.changed", EventCreditBalanceChanged)
	assert.Equal(t, "runtime_authorization.updated", EventRuntimeAuthorizationUpdated)
}

// TestConsumedEventNames verifies the external events that trigger
// authorization package regeneration.
func TestConsumedEventNames(t *testing.T) {
	assert.Equal(t, "credit.grant.expired", EventConsumedCreditGrantExpired)
	assert.Equal(t, "subscription.updated", EventConsumedSubscriptionUpdated)
}

// TestNoEventNameCollision verifies that produced and consumed event names
// are all unique (no overlap between produced and consumed sets).
func TestNoEventNameCollision(t *testing.T) {
	produced := []string{
		EventBatchSettled, EventBatchCorrected,
		EventCreditBalanceChanged, EventRuntimeAuthorizationUpdated,
	}
	consumed := []string{
		EventConsumedCreditGrantExpired, EventConsumedSubscriptionUpdated,
	}

	seen := make(map[string]bool)
	for _, n := range produced {
		assert.False(t, seen[n], "duplicate produced event name: %s", n)
		seen[n] = true
	}
	for _, n := range consumed {
		assert.False(t, seen[n], "consumed event name collides with produced: %s", n)
		seen[n] = true
	}
}
