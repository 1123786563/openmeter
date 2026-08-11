package refund

import (
	"context"
	"fmt"

	reservationadapter "github.com/openmeterio/openmeter/openmeter/creditreservation/adapter"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

// OpenMeterFenceClient is the production fence bridge. Both establishment and
// release take the reservation adapter's durable BillingCustomerLock, so a
// competing Reserve can only observe the refund row before the sequence is
// written or after the entire fence transaction commits.
type OpenMeterFenceClient struct {
	Adapter reservationadapter.Adapter
}

func (c OpenMeterFenceClient) EstablishFence(ctx context.Context, namespace, customerID, refundID string) (FenceResult, error) {
	if c.Adapter == nil {
		return FenceResult{}, fmt.Errorf("reservation adapter is required")
	}
	var result FenceResult
	err := c.Adapter.WithCustomerLock(ctx, customer.CustomerID{Namespace: namespace, ID: customerID}, func(tx reservationadapter.TxAdapter) error {
		fence, err := tx.EstablishRefundFence(ctx, refundID)
		if err != nil {
			return err
		}
		result = FenceResult{Sequence: fence.Sequence, Established: fence.Established}
		return nil
	})
	return result, err
}

func (c OpenMeterFenceClient) ReleaseFence(ctx context.Context, namespace, customerID, refundID, sequence string) error {
	if c.Adapter == nil {
		return fmt.Errorf("reservation adapter is required")
	}
	return c.Adapter.WithCustomerLock(ctx, customer.CustomerID{Namespace: namespace, ID: customerID}, func(tx reservationadapter.TxAdapter) error {
		return tx.ReleaseRefundFence(ctx, refundID, sequence)
	})
}

var _ FenceClient = OpenMeterFenceClient{}
