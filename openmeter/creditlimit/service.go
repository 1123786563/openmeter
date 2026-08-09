// Package creditlimit owns explicit, time-bounded enterprise receivable
// allowances. It deliberately stores policy only; the ledger is authoritative
// for what has already been issued against an allowance.
package creditlimit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Limit struct {
	ID            string
	Namespace     string
	CustomerID    string
	Currency      currencies.CurrencyReference
	Amount        alpacadecimal.Decimal
	EffectiveFrom time.Time
	EffectiveTo   *time.Time
	Enabled       bool
}

type GetActiveInput struct {
	Namespace  string
	CustomerID string
	Currency   currencies.CurrencyReference
	AsOf       time.Time
}

type CreateInput struct {
	Namespace     string
	CustomerID    string
	Currency      currencies.CurrencyReference
	Amount        alpacadecimal.Decimal
	EffectiveFrom time.Time
	EffectiveTo   *time.Time
}

type Service interface {
	AllowanceResolver
	GetActive(ctx context.Context, input GetActiveInput) (*Limit, error)
	Create(ctx context.Context, input CreateInput) (*Limit, error)
	Disable(ctx context.Context, id models.NamespacedID) error
}

// AllowanceResolver converts active policy and the ledger's current
// receivable balance into the remaining amount that a collector may advance.
type AllowanceResolver interface {
	Remaining(ctx context.Context, input RemainingInput) (*alpacadecimal.Decimal, error)
}

// ActiveHoldReader is the reservation boundary for enterprise receivables.
// Task 3's reservation persistence must implement it using the same customer,
// currency, and booking-time identity. It intentionally has no feature key:
// a customer/currency enterprise limit is shared across every feature.
type ActiveHoldReader interface {
	ActiveHeldAmount(ctx context.Context, input ActiveHoldInput) (alpacadecimal.Decimal, error)
}

type ActiveHoldInput struct {
	Namespace  string
	CustomerID string
	Currency   currencies.CurrencyReference
	AsOf       time.Time
}

var ErrActiveHoldReaderUnavailable = errors.New("active enterprise hold reader is required")

// NoActiveHoldReader is the Phase 1 production reader while Reservation
// persistence does not yet exist. It is explicit wiring, not a nil fallback:
// there is no durable reservation source in this phase, therefore its active
// held amount is authoritatively zero. Task 3 must replace this reader once it
// introduces reservation persistence; a missing reader remains fail-closed.
type NoActiveHoldReader struct{}

func (NoActiveHoldReader) ActiveHeldAmount(context.Context, ActiveHoldInput) (alpacadecimal.Decimal, error) {
	return alpacadecimal.Zero, nil
}

// NoopAllowanceResolver is an explicit "no active limit" dependency for
// callers that do not provision enterprise credit policy.
type NoopAllowanceResolver struct{}

func (NoopAllowanceResolver) Remaining(context.Context, RemainingInput) (*alpacadecimal.Decimal, error) {
	return nil, nil
}

type RemainingInput struct {
	Namespace  string
	CustomerID string
	Currency   currencies.CurrencyReference
	FeatureKey string
	AsOf       time.Time
}

type Config struct {
	Client           *entdb.Client
	BalanceQuerier   ledger.BalanceQuerier
	AccountService   ledger.AccountResolver
	AccountLocker    ledger.AccountLocker
	ActiveHoldReader ActiveHoldReader
}

func (c Config) Validate() error {
	if c.Client == nil {
		return errors.New("ent client is required")
	}
	if c.BalanceQuerier == nil {
		return errors.New("balance querier is required")
	}
	if c.AccountService == nil {
		return errors.New("account service is required")
	}
	if c.AccountLocker == nil {
		return errors.New("account locker is required")
	}
	return nil
}

type service struct {
	adapter          Adapter
	balanceQuerier   ledger.BalanceQuerier
	accountService   ledger.AccountResolver
	accountLocker    ledger.AccountLocker
	activeHoldReader ActiveHoldReader
}

func NewService(config Config) (Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &service{
		adapter:          NewAdapter(config.Client),
		balanceQuerier:   config.BalanceQuerier,
		accountService:   config.AccountService,
		accountLocker:    config.AccountLocker,
		activeHoldReader: config.ActiveHoldReader,
	}, nil
}

func (s *service) GetActive(ctx context.Context, input GetActiveInput) (*Limit, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	return s.adapter.GetActive(ctx, input)
}

func (s *service) Create(ctx context.Context, input CreateInput) (*Limit, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	return s.adapter.Create(ctx, input)
}

func (s *service) Disable(ctx context.Context, id models.NamespacedID) error {
	if err := id.Validate(); err != nil {
		return err
	}
	return s.adapter.Disable(ctx, id)
}

func (s *service) Remaining(ctx context.Context, input RemainingInput) (*alpacadecimal.Decimal, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	if s.activeHoldReader == nil {
		return nil, ErrActiveHoldReaderUnavailable
	}

	limit, err := s.GetActive(ctx, GetActiveInput{
		Namespace: input.Namespace, CustomerID: input.CustomerID, Currency: input.Currency, AsOf: input.AsOf,
	})
	if err != nil || limit == nil {
		return nil, err
	}

	accounts, err := s.accountService.GetCustomerAccounts(ctx, customer.CustomerID{Namespace: input.Namespace, ID: input.CustomerID})
	if err != nil {
		return nil, fmt.Errorf("get customer accounts: %w", err)
	}
	if err := s.accountLocker.LockAccountsForPosting(ctx, []ledger.Account{accounts.ReceivableAccount}); err != nil {
		return nil, fmt.Errorf("lock customer receivable account: %w", err)
	}

	// Enterprise limits are scoped by customer and managed currency, not by
	// feature. FeatureKey is retained in RemainingInput only as event context.
	route := ledger.RouteFilter{Currency: input.Currency}
	asOf := input.AsOf
	balance, err := s.balanceQuerier.GetAccountBalance(ctx, accounts.ReceivableAccount, route, ledger.BalanceQuery{AsOf: &asOf})
	if err != nil {
		return nil, fmt.Errorf("get customer receivable balance: %w", err)
	}

	held, err := s.activeHoldReader.ActiveHeldAmount(ctx, ActiveHoldInput{
		Namespace: input.Namespace, CustomerID: input.CustomerID, Currency: input.Currency, AsOf: input.AsOf,
	})
	if err != nil {
		return nil, fmt.Errorf("get active enterprise holds: %w", err)
	}
	remaining := remainingAllowance(limit.Amount, balance, held)
	return &remaining, nil
}

func remainingAllowance(limit, receivableBalance, held alpacadecimal.Decimal) alpacadecimal.Decimal {
	remaining := limit.Add(receivableBalance).Sub(held)
	if remaining.IsNegative() {
		remaining = alpacadecimal.Zero
	}
	return remaining
}

func (i GetActiveInput) Validate() error {
	if i.Namespace == "" || i.CustomerID == "" || i.AsOf.IsZero() {
		return errors.New("namespace, customer id, and as of are required")
	}
	return validateManagedCustomCurrency(i.Currency)
}

func (i CreateInput) Validate() error {
	if i.Namespace == "" || i.CustomerID == "" || i.EffectiveFrom.IsZero() {
		return errors.New("namespace, customer id, and effective from are required")
	}
	if !i.Amount.IsPositive() {
		return errors.New("amount must be positive")
	}
	if i.EffectiveTo != nil && !i.EffectiveTo.After(i.EffectiveFrom) {
		return errors.New("effective to must be after effective from")
	}
	return validateManagedCustomCurrency(i.Currency)
}

func (i RemainingInput) Validate() error {
	return GetActiveInput{Namespace: i.Namespace, CustomerID: i.CustomerID, Currency: i.Currency, AsOf: i.AsOf}.Validate()
}

func validateManagedCustomCurrency(currency currencies.CurrencyReference) error {
	if err := currency.Validate(); err != nil {
		return fmt.Errorf("currency: %w", err)
	}
	if !currency.IsCustom() || !currency.IsResolved() || currency.CustomCurrencyID == nil {
		return errors.New("currency must be a resolved managed custom currency")
	}
	return nil
}
