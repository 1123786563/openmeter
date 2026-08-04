// Package order implements order creation, immutable product snapshots, and the
// order lifecycle state machine. The state machine enforces that only valid
// transitions are applied; invalid transitions return a stable conflict error.
package order

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/commerce"
)

// ValidTransitions defines the order state machine. Each entry maps a source
// status to the set of statuses it may transition to.
//
//	created           -> awaiting_payment | cancelled | expired
//	awaiting_payment  -> paid | cancelled | expired
//	paid              -> fulfilled
//	fulfilled         -> refund_pending
//	refund_pending    -> partially_refunded | refunded
//	partially_refunded -> refunded
//
// "paid does not imply fulfilled" and "fulfilled cannot return to paid" are
// encoded by the absence of those transitions.
var ValidTransitions = map[commerce.OrderStatus]map[commerce.OrderStatus]bool{
	commerce.OrderStatusCreated: {
		commerce.OrderStatusAwaitingPayment: true,
		commerce.OrderStatusCancelled:       true,
		commerce.OrderStatusExpired:         true,
	},
	commerce.OrderStatusAwaitingPayment: {
		commerce.OrderStatusPaid:      true,
		commerce.OrderStatusCancelled: true,
		commerce.OrderStatusExpired:   true,
	},
	commerce.OrderStatusPaid: {
		commerce.OrderStatusFulfilled: true,
	},
	commerce.OrderStatusFulfilled: {
		commerce.OrderStatusRefundPending: true,
	},
	commerce.OrderStatusRefundPending: {
		commerce.OrderStatusPartiallyRefunded: true,
		commerce.OrderStatusRefunded:          true,
	},
	commerce.OrderStatusPartiallyRefunded: {
		commerce.OrderStatusRefunded: true,
	},
	// Terminal states: no outgoing transitions.
	commerce.OrderStatusCancelled: {},
	commerce.OrderStatusExpired:   {},
	commerce.OrderStatusRefunded:  {},
}

// CanTransition returns true if transitioning from `from` to `to` is allowed
// by the state machine.
func CanTransition(from, to commerce.OrderStatus) bool {
	dests, ok := ValidTransitions[from]
	if !ok {
		return false
	}
	return dests[to]
}

// MustTransition returns an error if the transition is not allowed. The error
// wraps commerce.ErrInvalidOrderTransition so callers get a stable conflict code.
func MustTransition(from, to commerce.OrderStatus) error {
	if CanTransition(from, to) {
		return nil
	}
	return fmt.Errorf("%w: %s -> %s", commerce.ErrInvalidOrderTransition, from, to)
}
