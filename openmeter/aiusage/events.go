package aiusage

// Event type constants for the AI Usage transactional outbox. Events are emitted
// atomically with the batch they belong to and projected asynchronously by the
// outbox worker (see openmeter/aiusage/worker).

const (
	// EventBatchSettled is emitted when an AI usage batch is successfully settled.
	// Payload includes usage_batch_id.
	EventBatchSettled = "ai_usage.batch.settled"

	// EventBatchCorrected is emitted when a previously settled batch is reversed.
	// Payload includes original_batch_id and reason.
	EventBatchCorrected = "ai_usage.batch.corrected"

	// EventCreditBalanceChanged is emitted when a customer's prepaid or enterprise
	// credit balance changes (grant, burn, top-up, or expiry).
	EventCreditBalanceChanged = "credit.balance.changed"

	// EventRuntimeAuthorizationUpdated is emitted when a new signed runtime
	// authorization package is generated, signaling consumers to refresh their
	// cached authorization snapshot.
	EventRuntimeAuthorizationUpdated = "runtime_authorization.updated"
)

// Consumed event types originate in other domains and trigger regeneration of
// the runtime authorization package.
const (
	// EventConsumedCreditGrantExpired originates from the credit/billing domain.
	// When a prepaid grant expires, the spendable credit pool shrinks and a new
	// signed authorization package must be issued.
	EventConsumedCreditGrantExpired = "credit.grant.expired"

	// EventConsumedSubscriptionUpdated originates from the subscription domain.
	// Plan changes, upgrades, downgrades, or entitlement modifications require a
	// fresh authorization package reflecting the new entitlement codes and period.
	EventConsumedSubscriptionUpdated = "subscription.updated"
)
