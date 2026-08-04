package commerce

import (
	"context"
	"fmt"

	entsql "entgo.io/ent/dialect/sql"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/commerceorder"
	"github.com/openmeterio/openmeter/openmeter/ent/db/fulfillment"
	"github.com/openmeterio/openmeter/openmeter/ent/db/paymentfact"
	"github.com/openmeterio/openmeter/pkg/clock"
)

// PaidTransitionParams carries the primitive inputs for the atomic paid
// transition. This is separated from the payment.PaidTransitionInput type to
// avoid an import cycle (payment imports commerce, so commerce cannot import
// payment).
type PaidTransitionParams struct {
	Namespace        string
	CustomerID       string
	OrderID          string
	PaymentAttemptID string
	RawHash          string
	Provider         string // "wechat", "alipay", or "offline"
	SignedPayload    map[string]any
}

// RunPaidTransition executes the atomic paid transition (C2) within a single
// customer-locked transaction:
//
//  1. Acquire the customer advisory lock (WithCustomerLock).
//  2. Lock the order row for update.
//  3. If already paid/fulfilled, return nil (idempotent replay).
//  4. Insert the unique Payment Fact (OnConflict DoNothing on the
//     namespace+raw_hash unique index — DB-enforced dedup).
//  5. Move the order to paid (awaiting_payment -> paid) with optimistic CC.
//  6. Insert one Fulfillment request (status=pending).
//  7. Write an Outbox record (event type "order.paid").
//
// If any step fails, the entire transaction rolls back.
func (a *EntAdapter) RunPaidTransition(ctx context.Context, p PaidTransitionParams) error {
	return a.WithCustomerLock(ctx, p.Namespace, p.CustomerID, func(txa *EntAdapter) error {
		// Lock the order row.
		eo, err := txa.db.CommerceOrder.Query().
			Where(commerceorder.IDEQ(p.OrderID), commerceorder.NamespaceEQ(p.Namespace)).
			ForUpdate().
			First(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return ErrOrderNotFound
			}
			return fmt.Errorf("paid-tx: lock order: %w", err)
		}

		// Idempotent: if already paid or fulfilled, skip.
		if eo.Status == commerceorder.StatusPaid || eo.Status == commerceorder.StatusFulfilled {
			return nil
		}

		// Only awaiting_payment can transition to paid.
		if eo.Status != commerceorder.StatusAwaitingPayment {
			return fmt.Errorf("paid-tx: order status is %s, expected awaiting_payment", eo.Status)
		}

		now := clock.Now()

		// Insert the unique Payment Fact. The unique index on (namespace,
		// raw_hash) enforces dedup at the DB level. OnConflict DoNothing means
		// a concurrent insert for the same hash is silently skipped.
		if err := txa.db.PaymentFact.Create().
			SetNamespace(p.Namespace).
			SetCreatedAt(now).
			SetPaymentAttemptID(p.PaymentAttemptID).
			SetRawHash(p.RawHash).
			SetProvider(paymentfact.Provider(p.Provider)).
			SetSignedPayload(p.SignedPayload).
			SetTimestamp(now).
			OnConflict(entsql.ConflictColumns("namespace", "raw_hash")).
			DoNothing().
			Exec(ctx); err != nil {
			return fmt.Errorf("paid-tx: insert payment fact: %w", err)
		}

		// Move order to paid with optimistic concurrency.
		n, err := txa.db.CommerceOrder.Update().
			Where(commerceorder.IDEQ(p.OrderID), commerceorder.StatusEQ(commerceorder.StatusAwaitingPayment)).
			SetStatus(commerceorder.StatusPaid).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("paid-tx: move order to paid: %w", err)
		}
		if n == 0 {
			// A concurrent transition already moved it; treat as idempotent.
			return nil
		}

		// Insert fulfillment request (status=pending).
		if _, err := txa.db.Fulfillment.Create().
			SetNamespace(p.Namespace).
			SetCreatedAt(now).
			SetUpdatedAt(now).
			SetCommerceOrderID(p.OrderID).
			SetCustomerID(p.CustomerID).
			SetStatus(fulfillment.StatusPending).
			Save(ctx); err != nil {
			return fmt.Errorf("paid-tx: create fulfillment: %w", err)
		}

		// Write outbox record.
		if err := txa.db.CommerceOutbox.Create().
			SetNamespace(p.Namespace).
			SetCreatedAt(now).
			SetAggregateType("commerce_order").
			SetAggregateID(p.OrderID).
			SetEventType("order.paid").
			SetPayload(map[string]any{
				"order_id":    p.OrderID,
				"customer_id": p.CustomerID,
				"provider":    p.Provider,
			}).
			Exec(ctx); err != nil {
			return fmt.Errorf("paid-tx: write outbox: %w", err)
		}

		return nil
	})
}
