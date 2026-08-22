package order

import (
	"errors"
	"testing"

	"github.com/openmeterio/openmeter/openmeter/commerce"
)

// TestValidTransitions verifies the full transition table from the brief.
func TestValidTransitions(t *testing.T) {
	tests := []struct {
		name string
		from commerce.OrderStatus
		to   commerce.OrderStatus
		want bool
	}{
		// Happy path
		{"created -> awaiting_payment", commerce.OrderStatusCreated, commerce.OrderStatusAwaitingPayment, true},
		{"awaiting_payment -> paid", commerce.OrderStatusAwaitingPayment, commerce.OrderStatusPaid, true},
		{"paid -> fulfilled", commerce.OrderStatusPaid, commerce.OrderStatusFulfilled, true},
		{"fulfilled -> refund_pending", commerce.OrderStatusFulfilled, commerce.OrderStatusRefundPending, true},
		{"refund_pending -> partially_refunded", commerce.OrderStatusRefundPending, commerce.OrderStatusPartiallyRefunded, true},
		{"refund_pending -> refunded", commerce.OrderStatusRefundPending, commerce.OrderStatusRefunded, true},
		{"partially_refunded -> refunded", commerce.OrderStatusPartiallyRefunded, commerce.OrderStatusRefunded, true},

		// Cancellation from early states
		{"created -> canceled", commerce.OrderStatusCreated, commerce.OrderStatusCancelled, true},
		{"awaiting_payment -> canceled", commerce.OrderStatusAwaitingPayment, commerce.OrderStatusCancelled, true},
		{"created -> expired", commerce.OrderStatusCreated, commerce.OrderStatusExpired, true},
		{"awaiting_payment -> expired", commerce.OrderStatusAwaitingPayment, commerce.OrderStatusExpired, true},

		// Invalid transitions
		{"created -> paid (skip payment)", commerce.OrderStatusCreated, commerce.OrderStatusPaid, false},
		{"created -> fulfilled (skip everything)", commerce.OrderStatusCreated, commerce.OrderStatusFulfilled, false},
		{"paid -> canceled (too late)", commerce.OrderStatusPaid, commerce.OrderStatusCancelled, false},
		{"paid does not imply fulfilled", commerce.OrderStatusPaid, commerce.OrderStatusPaid, false},
		{"fulfilled cannot return to paid", commerce.OrderStatusFulfilled, commerce.OrderStatusPaid, false},
		{"fulfilled -> canceled", commerce.OrderStatusFulfilled, commerce.OrderStatusCancelled, false},
		{"refunded is terminal", commerce.OrderStatusRefunded, commerce.OrderStatusPaid, false},
		{"canceled is terminal", commerce.OrderStatusCancelled, commerce.OrderStatusCreated, false},
		{"expired is terminal", commerce.OrderStatusExpired, commerce.OrderStatusAwaitingPayment, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanTransition(tt.from, tt.to)
			if got != tt.want {
				t.Errorf("CanTransition(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

// TestInvalidTransitionReturnsStableError verifies that MustTransition returns
// an error wrapping commerce.ErrInvalidOrderTransition for invalid transitions.
func TestInvalidTransitionReturnsStableError(t *testing.T) {
	err := MustTransition(commerce.OrderStatusPaid, commerce.OrderStatusAwaitingPayment)
	if err == nil {
		t.Fatal("expected error for paid -> awaiting_payment")
	}
	// The error should wrap the domain sentinel so callers can check for the
	// stable conflict code.
	if !isInvalidTransitionError(err) {
		t.Errorf("error should wrap ErrInvalidOrderTransition: %v", err)
	}
}

func isInvalidTransitionError(err error) bool {
	return errors.Is(err, commerce.ErrInvalidOrderTransition)
}

// TestIsTerminal verifies terminal state detection.
func TestIsTerminal(t *testing.T) {
	terminal := []commerce.OrderStatus{
		commerce.OrderStatusCancelled,
		commerce.OrderStatusExpired,
		commerce.OrderStatusRefunded,
	}
	for _, s := range terminal {
		if !IsTerminal(s) {
			t.Errorf("%s should be terminal", s)
		}
	}
	nonTerminal := []commerce.OrderStatus{
		commerce.OrderStatusCreated,
		commerce.OrderStatusAwaitingPayment,
		commerce.OrderStatusPaid,
		commerce.OrderStatusFulfilled,
		commerce.OrderStatusRefundPending,
		commerce.OrderStatusPartiallyRefunded,
	}
	for _, s := range nonTerminal {
		if IsTerminal(s) {
			t.Errorf("%s should not be terminal", s)
		}
	}
}
