package commerce

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAllEventNames verifies that every approved event name is returned and
// the slice has no duplicates (reconciliation relies on the full set being
// stable and unique).
func TestAllEventNames(t *testing.T) {
	names := AllEventNames()

	expected := []string{
		EventOrderUpdated,
		EventPaymentSettled,
		EventPaymentFailed,
		EventRefundUpdated,
		EventInvoiceUpdated,
		EventSubscriptionUpdate,
	}
	assert.ElementsMatch(t, expected, names)
	assert.Len(t, names, 6)

	// No duplicates.
	seen := make(map[string]bool, len(names))
	for _, n := range names {
		assert.False(t, seen[n], "duplicate event name: %s", n)
		seen[n] = true
	}
}

// TestEventNameValues verifies the string values match the API contract.
func TestEventNameValues(t *testing.T) {
	assert.Equal(t, "order.updated", EventOrderUpdated)
	assert.Equal(t, "payment.settled", EventPaymentSettled)
	assert.Equal(t, "payment.failed", EventPaymentFailed)
	assert.Equal(t, "refund.updated", EventRefundUpdated)
	assert.Equal(t, "invoice.updated", EventInvoiceUpdated)
	assert.Equal(t, "subscription.updated", EventSubscriptionUpdate)
}
