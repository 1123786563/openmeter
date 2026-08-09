// Package adapter persists CREDIT reservation commands and their transactional
// usage outbox. Ledger remains authoritative for booked money.
package adapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	entsql "entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/creditreservation"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingcustomerlock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Config struct {
	Client *entdb.Client
	Logger *slog.Logger
}

func (c Config) Validate() error {
	if c.Client == nil {
		return errors.New("ent client is required")
	}
	if c.Logger == nil {
		return errors.New("logger is required")
	}
	return nil
}

type Adapter interface {
	WithCustomerLock(ctx context.Context, id customer.CustomerID, fn func(TxAdapter) error) error
	GetReservation(ctx context.Context, id models.NamespacedID) (creditreservation.Reservation, error)
	GetCharge(ctx context.Context, id models.NamespacedID) (creditreservation.Charge, error)
	ListExpiredReservations(ctx context.Context, now time.Time, limit int) ([]creditreservation.Reservation, error)
}

type TxAdapter interface {
	GetReservation(ctx context.Context, id models.NamespacedID) (creditreservation.Reservation, error)
	GetReservationByCommand(ctx context.Context, namespace, idempotencyKey string) (creditreservation.Reservation, bool, error)
	GetChargeByCommand(ctx context.Context, namespace, idempotencyKey string) (creditreservation.Charge, bool, error)
	GetCharge(ctx context.Context, id models.NamespacedID) (creditreservation.Charge, error)
	ReverseCharge(ctx context.Context, id models.NamespacedID, ledgerGroupID string) (creditreservation.Charge, error)
	ActivePrepaidHold(ctx context.Context, currency currencies.CurrencyReference, featureKey string) (int64, error)
	HasActiveRefundFence(ctx context.Context) (bool, error)
	EstablishRefundFence(ctx context.Context, refundID string) (creditreservation.FenceResult, error)
	ReleaseRefundFence(ctx context.Context, refundID, sequence string) error
	CreateReservation(ctx context.Context, input CreateReservationInput) (creditreservation.Reservation, bool, error)
	UpdateReservation(ctx context.Context, input UpdateReservationInput) (creditreservation.Reservation, error)
	CreateCharge(ctx context.Context, input CreateChargeInput) (creditreservation.Charge, bool, error)
	AppendUsageEvent(ctx context.Context, event creditreservation.UsageEvent) error
}

type adapter struct {
	db     *entdb.Client
	logger *slog.Logger
}

type txAdapter struct {
	db         *entdb.Client
	logger     *slog.Logger
	customerID customer.CustomerID
}

var _ Adapter = (*adapter)(nil)
var _ TxAdapter = (*txAdapter)(nil)

func New(config Config) (Adapter, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &adapter{db: config.Client, logger: config.Logger}, nil
}

func (a *adapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := a.db.HijackTx(ctx, &sql.TxOptions{ReadOnly: false})
	if err != nil {
		return nil, nil, fmt.Errorf("hijack transaction: %w", err)
	}
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func (a *adapter) WithTx(_ context.Context, tx *entutils.TxDriver) *adapter {
	txClient := entdb.NewTxClientFromRawConfig(context.Background(), *tx.GetConfig())
	return &adapter{db: txClient.Client(), logger: a.logger}
}

func (a *adapter) Self() *adapter { return a }

// WithCustomerLock serializes all reservation writes for a customer with the
// existing billing sentinel row. The lock, reservations, charges, and outbox
// rows share one transaction and therefore commit or roll back together.
func (a *adapter) WithCustomerLock(ctx context.Context, id customer.CustomerID, fn func(TxAdapter) error) error {
	if err := id.Validate(); err != nil {
		return fmt.Errorf("validate customer id: %w", err)
	}
	return transaction.RunWithNoValue(ctx, a, func(ctx context.Context) error {
		return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, txa *adapter) error {
			err := txa.db.BillingCustomerLock.Create().
				SetNamespace(id.Namespace).
				SetCustomerID(id.ID).
				OnConflict(entsql.DoNothing()).
				Exec(ctx)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("upsert customer lock: %w", err)
			}
			if _, err := txa.db.BillingCustomerLock.Query().
				Where(billingcustomerlock.Namespace(id.Namespace), billingcustomerlock.CustomerID(id.ID)).
				ForUpdate().
				First(ctx); err != nil {
				return fmt.Errorf("lock customer: %w", err)
			}
			return fn(&txAdapter{db: txa.db, logger: a.logger, customerID: id})
		})
	})
}

func (a *adapter) GetReservation(ctx context.Context, id models.NamespacedID) (creditreservation.Reservation, error) {
	if err := id.Validate(); err != nil {
		return creditreservation.Reservation{}, fmt.Errorf("validate reservation id: %w", err)
	}
	return getReservation(ctx, a.db, id)
}

func (a *adapter) GetCharge(ctx context.Context, id models.NamespacedID) (creditreservation.Charge, error) {
	return getCharge(ctx, a.db, id)
}
