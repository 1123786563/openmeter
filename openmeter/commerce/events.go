package commerce

// Approved commerce event names. Event IDs always equal the Outbox row ID so
// retries do not produce a second domain effect downstream. These are the only
// event types a successful commerce state transition may publish.
const (
	EventOrderUpdated       = "order.updated"
	EventPaymentSettled     = "payment.settled"
	EventPaymentFailed      = "payment.failed"
	EventRefundUpdated      = "refund.updated"
	EventInvoiceUpdated     = "invoice.updated"
	EventSubscriptionUpdate = "subscription.updated"
)

// AllEventNames returns every approved commerce event name. Reconciliation uses
// this to verify the outbox never publishes an unknown event type.
func AllEventNames() []string {
	return []string{
		EventOrderUpdated,
		EventPaymentSettled,
		EventPaymentFailed,
		EventRefundUpdated,
		EventInvoiceUpdated,
		EventSubscriptionUpdate,
	}
}
