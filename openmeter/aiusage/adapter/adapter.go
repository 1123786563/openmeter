package adapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	entsql "entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/aiusage"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingcustomerlock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

// Config holds the dependencies for constructing the adapter.
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

// TxAdapter exposes the operations available inside a customer-locked transaction.
type TxAdapter interface {
	GetBatchByIdempotencyKey(ctx context.Context, namespace, customerID, key string) (*aiusage.AIUsageBatch, error)
	CreateSettledBatch(ctx context.Context, in aiusage.SettledBatch) (*aiusage.AIUsageBatch, error)
	AdvanceWatermark(ctx context.Context, namespace, subjectID string, seq int64) (int64, error)
	AppendOutbox(ctx context.Context, namespace, customerID, subjectID string, events []aiusage.OutboxEvent, batchID string) error
}

// Adapter is the persistence adapter for AI Usage batches. All writes go through
// WithCustomerLock to guarantee per-customer serialisability.
type Adapter interface {
	WithCustomerLock(ctx context.Context, namespace, customerID string, fn func(TxAdapter) error) error
}

// New creates a new Adapter from the given Config.
func New(config Config) (Adapter, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &adapter{
		db:     config.Client,
		logger: config.Logger,
	}, nil
}

// adapter implements Adapter, transaction.Creator, and entutils.TxUser.
type adapter struct {
	db     *entdb.Client
	logger *slog.Logger
}

var _ Adapter = (*adapter)(nil)

// ---- transaction.Creator ----

func (a *adapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := a.db.HijackTx(ctx, &sql.TxOptions{ReadOnly: false})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

// ---- entutils.TxUser ----

func (a *adapter) WithTx(_ context.Context, tx *entutils.TxDriver) *adapter {
	txClient := entdb.NewTxClientFromRawConfig(context.Background(), *tx.GetConfig())
	return &adapter{
		db:     txClient.Client(),
		logger: a.logger,
	}
}

func (a *adapter) Self() *adapter { return a }

// ---- WithCustomerLock ----

// WithCustomerLock starts a PostgreSQL transaction, acquires a row-level lock
// on the customer (via the billing_customer_locks sentinel table), then executes
// fn with a TxAdapter backed by the same transaction. Commit on success,
// rollback on error.
func (a *adapter) WithCustomerLock(ctx context.Context, namespace, customerID string, fn func(TxAdapter) error) error {
	return transaction.RunWithNoValue(ctx, a, func(ctx context.Context) error {
		return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, txa *adapter) error {
			// Upsert sentinel lock row; DO NOTHING returns sql.ErrNoRows when the
			// row already exists, which is the expected no-op path.
			err := txa.db.BillingCustomerLock.Create().
				SetNamespace(namespace).
				SetCustomerID(customerID).
				OnConflict(entsql.DoNothing()).
				Exec(ctx)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("upsert customer lock: %w", err)
			}

			_, err = txa.db.BillingCustomerLock.Query().
				Where(
					billingcustomerlock.Namespace(namespace),
					billingcustomerlock.CustomerID(customerID),
				).
				ForUpdate().
				First(ctx)
			if err != nil {
				return fmt.Errorf("lock customer: %w", err)
			}

			return fn(&txAdapter{db: txa.db, logger: a.logger, customerID: customerID})
		})
	})
}

// txAdapter implements TxAdapter. It holds a transaction-backed *entdb.Client
// and the locked customerID (needed for watermark row creation).
type txAdapter struct {
	db         *entdb.Client
	logger     *slog.Logger
	customerID string
}

var _ TxAdapter = (*txAdapter)(nil)
