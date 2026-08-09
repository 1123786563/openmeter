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
	Client         *entdb.Client
	BalanceQuerier ledger.BalanceQuerier
	AccountService ledger.AccountResolver
	AccountLocker  ledger.AccountLocker
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
	adapter        Adapter
	balanceQuerier ledger.BalanceQuerier
	accountService ledger.AccountResolver
	accountLocker  ledger.AccountLocker
}

func NewService(config Config) (Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &service{
		adapter:        NewAdapter(config.Client),
		balanceQuerier: config.BalanceQuerier,
		accountService: config.AccountService,
		accountLocker:  config.AccountLocker,
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

	route := ledger.RouteFilter{Currency: input.Currency}
	if input.FeatureKey != "" {
		route.MatchFeature = input.FeatureKey
	}
	asOf := input.AsOf
	balance, err := s.balanceQuerier.GetAccountBalance(ctx, accounts.ReceivableAccount, route, ledger.BalanceQuery{AsOf: &asOf})
	if err != nil {
		return nil, fmt.Errorf("get customer receivable balance: %w", err)
	}

	remaining := limit.Amount.Add(balance)
	if remaining.IsNegative() {
		remaining = alpacadecimal.Zero
	}
	return &remaining, nil
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
