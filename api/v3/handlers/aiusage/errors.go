package aiusage

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

// CreditBalanceView is the balance snapshot returned by CreditBalanceReader.
type CreditBalanceView struct {
	RetrievedAt      time.Time
	AvailableCredits int64
	SettledCredits   int64
	PendingCredits   int64
}

// CreditBalanceReader reads a customer's AI-usage integer credit balance and
// transactions. When nil the credit operations return 501.
type CreditBalanceReader interface {
	ReadBalance(ctx context.Context, namespace, customerID string, at time.Time) (CreditBalanceView, error)
	ListTransactions(ctx context.Context, namespace, customerID string, page Pagination) ([]CreditTransactionView, error)
}

// CreditTransactionView is a single credit movement on the AI Usage ledger.
type CreditTransactionView struct {
	ID                     string
	BookedAt               time.Time
	Type                   string
	Amount                 int64
	AvailableBalanceBefore int64
	AvailableBalanceAfter  int64
}

// Pagination is a minimal cursor/page input for listing operations.
type Pagination struct {
	After  *string
	Before *string
	Size   int
}

// noopCreditBalanceReader always returns NotImplemented.
type noopCreditBalanceReader struct{}

func (noopCreditBalanceReader) ReadBalance(context.Context, string, string, time.Time) (CreditBalanceView, error) {
	return CreditBalanceView{}, models.NewGenericNotImplementedError(nil)
}

func (noopCreditBalanceReader) ListTransactions(context.Context, string, string, Pagination) ([]CreditTransactionView, error) {
	return nil, models.NewGenericNotImplementedError(nil)
}
