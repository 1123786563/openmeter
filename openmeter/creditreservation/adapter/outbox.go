package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/creditreservation"
)

func (t *txAdapter) AppendUsageEvent(ctx context.Context, event creditreservation.UsageEvent) error {
	if event.EventID == "" || event.AggregateType == "" || event.AggregateID == "" || event.EventType == "" {
		return fmt.Errorf("usage event id, aggregate type, aggregate id, and event type are required")
	}
	payload := event.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	if _, err := t.db.CreditReservationOutbox.Create().
		SetNamespace(t.customerID.Namespace).
		SetEventID(event.EventID).
		SetAggregateType(event.AggregateType).
		SetAggregateID(event.AggregateID).
		SetEventType(event.EventType).
		SetPayload(payload).
		Save(ctx); err != nil {
		return fmt.Errorf("append usage event: %w", err)
	}
	return nil
}
