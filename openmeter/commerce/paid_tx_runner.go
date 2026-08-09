package commerce

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/commerceorder"
	"github.com/openmeterio/openmeter/openmeter/ent/db/fulfillment"
	"github.com/openmeterio/openmeter/openmeter/ent/db/paymentattempt"
	"github.com/openmeterio/openmeter/openmeter/ent/db/paymentfact"
	"github.com/openmeterio/openmeter/pkg/clock"
)

// PaidTransitionParams carries the primitive inputs for the atomic paid
// transition. This is separated from the payment.PaidTransitionInput type to
// avoid an import cycle (payment imports commerce, so commerce cannot import
// payment).
type PaidTransitionParams struct {
	Namespace         string
	CustomerID        string
	OrderID           string
	PaymentAttemptID  string
	PaymentFactID     string
	RawHash           string
	Provider          string // "wechat", "alipay", or "offline"
	ProviderOrderID   string
	ProviderPaymentID string
	ProviderEventID   string
	MerchantID        string
	ApplicationID     string
	AmountMinor       int64
	Currency          string
	Success           bool
	SignedPayload     map[string]any
	Timestamp         time.Time
	CreatedAt         time.Time
}

type PaidTransitionResult struct {
	Fact        *PaymentFactWire
	AlreadyPaid bool
}

// RunPaidTransition executes the atomic paid transition (C2) within a single
// customer-locked transaction:
//
//  1. Acquire the customer advisory lock (WithCustomerLock).
//  2. Lock the payment attempt and order rows for update.
//  3. Insert the unique Payment Fact.
//  4. Move the attempt to succeeded (pending -> succeeded).
//  5. Move the order to paid (awaiting_payment -> paid).
//  6. Insert one Fulfillment request (status=pending).
//  7. Write an Outbox record (event type "order.paid").
//
// If any step fails, the entire transaction rolls back.
func (a *EntAdapter) RunPaidTransition(ctx context.Context, p PaidTransitionParams) (PaidTransitionResult, error) {
	var result PaidTransitionResult
	err := a.WithCustomerLock(ctx, p.Namespace, p.CustomerID, func(txa *EntAdapter) error {
		ea, err := txa.db.PaymentAttempt.Query().
			Where(paymentattempt.IDEQ(p.PaymentAttemptID), paymentattempt.NamespaceEQ(p.Namespace)).
			ForUpdate().
			First(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return ErrPaymentAttemptNotFound
			}
			return fmt.Errorf("paid-tx: lock payment attempt: %w", err)
		}
		if ea.CommerceOrderID != p.OrderID || ea.CustomerID != p.CustomerID || string(ea.Provider) != p.Provider {
			return errors.New("paid-tx: payment attempt does not match order, customer, or provider")
		}
		if ea.Status != paymentattempt.StatusPending && ea.Status != paymentattempt.StatusSucceeded {
			return fmt.Errorf("paid-tx: payment attempt status is %s, expected pending or succeeded", ea.Status)
		}

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

		if eo.CustomerID != p.CustomerID {
			return errors.New("paid-tx: order does not match customer")
		}
		alreadyPaid := eo.Status == commerceorder.StatusPaid || eo.Status == commerceorder.StatusFulfilled
		if !alreadyPaid && eo.Status != commerceorder.StatusAwaitingPayment {
			return fmt.Errorf("paid-tx: order status is %s, expected awaiting_payment", eo.Status)
		}

		now := clock.Now()
		factCreate := txa.db.PaymentFact.Create().
			SetNamespace(p.Namespace).
			SetCreatedAt(p.CreatedAt).
			SetPaymentAttemptID(p.PaymentAttemptID).
			SetRawHash(p.RawHash).
			SetProvider(paymentfact.Provider(p.Provider)).
			SetProviderOrderID(p.ProviderOrderID).
			SetNillableProviderPaymentID(nonEmptyPtr(p.ProviderPaymentID)).
			SetNillableProviderEventID(nonEmptyPtr(p.ProviderEventID)).
			SetNillableMerchantID(nonEmptyPtr(p.MerchantID)).
			SetNillableApplicationID(nonEmptyPtr(p.ApplicationID)).
			SetAmountMinor(p.AmountMinor).
			SetCurrency(p.Currency).
			SetSuccess(p.Success).
			SetSignedPayload(p.SignedPayload).
			SetTimestamp(p.Timestamp)
		if p.PaymentFactID != "" {
			factCreate.SetID(p.PaymentFactID)
		}
		if err := factCreate.
			OnConflict().
			DoNothing().
			Exec(ctx); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("paid-tx: insert payment fact: %w", err)
		}
		fact, err := txa.getPaidTransitionFact(ctx, p)
		if err != nil {
			return err
		}
		if fact.PaymentAttemptID != p.PaymentAttemptID || fact.ProviderOrderID != p.ProviderOrderID {
			return errors.New("paid-tx: deduplicated payment fact belongs to a different attempt or provider order")
		}
		result.Fact = mapEntPaymentFact(fact)

		if ea.Status == paymentattempt.StatusPending {
			n, err := txa.db.PaymentAttempt.Update().
				Where(
					paymentattempt.IDEQ(p.PaymentAttemptID),
					paymentattempt.NamespaceEQ(p.Namespace),
					paymentattempt.StatusEQ(paymentattempt.StatusPending),
				).
				SetStatus(paymentattempt.StatusSucceeded).
				SetUpdatedAt(now).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("paid-tx: move payment attempt to succeeded: %w", err)
			}
			if n != 1 {
				return errors.New("paid-tx: payment attempt transition did not update exactly one row")
			}
		}

		if alreadyPaid {
			result.AlreadyPaid = true
			return nil
		}

		n, err := txa.db.CommerceOrder.Update().
			Where(
				commerceorder.IDEQ(p.OrderID),
				commerceorder.NamespaceEQ(p.Namespace),
				commerceorder.StatusEQ(commerceorder.StatusAwaitingPayment),
			).
			SetStatus(commerceorder.StatusPaid).
			SetUpdatedAt(now).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("paid-tx: move order to paid: %w", err)
		}
		if n != 1 {
			return errors.New("paid-tx: order transition did not update exactly one row")
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
	if err != nil {
		return PaidTransitionResult{}, err
	}
	return result, nil
}

func (a *EntAdapter) getPaidTransitionFact(ctx context.Context, p PaidTransitionParams) (*entdb.PaymentFact, error) {
	fact, err := a.db.PaymentFact.Query().
		Where(paymentfact.NamespaceEQ(p.Namespace), paymentfact.RawHashEQ(p.RawHash)).
		First(ctx)
	if err == nil {
		return fact, nil
	}
	if !entdb.IsNotFound(err) {
		return nil, fmt.Errorf("paid-tx: get payment fact by raw hash: %w", err)
	}
	if p.ProviderEventID == "" {
		return nil, errors.New("paid-tx: inserted payment fact not found by raw hash")
	}
	fact, err = a.db.PaymentFact.Query().
		Where(
			paymentfact.NamespaceEQ(p.Namespace),
			paymentfact.ProviderEQ(paymentfact.Provider(p.Provider)),
			paymentfact.ProviderEventIDEQ(p.ProviderEventID),
		).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("paid-tx: get payment fact by provider event: %w", err)
	}
	return fact, nil
}
