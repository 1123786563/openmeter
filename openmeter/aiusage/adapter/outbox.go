package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/aiusage"
)

// AppendOutbox creates outbox rows linked to a batch via the edge relationship.
// All rows are written within the caller's transaction.
func (t *txAdapter) AppendOutbox(ctx context.Context, namespace, customerID, subjectID string, events []aiusage.OutboxEvent, batchID string) error {
	for _, evt := range events {
		payload := evt.Payload
		if payload == nil {
			payload = map[string]any{}
		}

		if _, err := t.db.AIUsageOutbox.Create().
			SetNamespace(namespace).
			SetCustomerID(customerID).
			SetSubjectID(subjectID).
			SetEventType(evt.EventType).
			SetPayload(payload).
			SetBatchID(batchID).
			Save(ctx); err != nil {
			return fmt.Errorf("create outbox event: %w", err)
		}
	}

	return nil
}
