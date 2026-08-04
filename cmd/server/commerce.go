package main

import (
	"context"
	"fmt"
	"log/slog"

	commercehandler "github.com/openmeterio/openmeter/api/v3/handlers/commerce"
	"github.com/openmeterio/openmeter/openmeter/commerce"
	"github.com/openmeterio/openmeter/openmeter/commerce/catalog"
	"github.com/openmeterio/openmeter/openmeter/commerce/fulfillment"
	"github.com/openmeterio/openmeter/openmeter/commerce/order"
	"github.com/openmeterio/openmeter/openmeter/commerce/payment"
	"github.com/openmeterio/openmeter/openmeter/commerce/reconciliation"
	"github.com/openmeterio/openmeter/openmeter/commerce/wallet"
	"github.com/openmeterio/openmeter/openmeter/commerce/worker"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
)

// CommerceWiring holds the commerce handler and worker manager constructed from
// the application's EntClient. It is built once at startup and passed to the
// server config + lifecycle.
type CommerceWiring struct {
	Handler       commercehandler.Handler
	WorkerManager *worker.Manager
}

// wireCommerce builds the commerce services that have Ent-backed implementations
// (catalog, orders, wallet, fulfillment, reconciliation) and constructs both the
// HTTP handler and the background worker manager. Services without Ent-backed
// repos (payment, refund, enterprise) are left nil; the handler returns 501 for
// those routes and the corresponding workers are not registered.
//
// The defaultNamespace is used to resolve the namespace for both the handler
// (via StaticNamespaceDecoder) and the worker jobs.
func wireCommerce(entClient *entdb.Client, defaultNamespace string, logger *slog.Logger) (*CommerceWiring, error) {
	entAdapter, err := commerce.NewEntAdapter(commerce.EntAdapterConfig{
		Client: entClient,
		Logger: logger,
	})
	if err != nil {
		return nil, err
	}

	// Catalog service: uses EntAdapter as ProductRepository.
	catalogSvc := catalog.New(catalog.Config{
		Repo:   entAdapter,
		Logger: logger,
	})

	// Order service: uses EntAdapter as OrderRepository + ProductLookup.
	orderSvc := order.New(order.Config{
		Repo:     entAdapter,
		Products: entAdapter,
		Logger:   logger,
	})

	// Wallet service: uses EntAdapter as DataPort.
	walletSvc := wallet.New(wallet.Config{
		Port:   walletDataPortAdapter{entAdapter},
		Logger: logger,
	})

	// Fulfillment service: uses stub adapters that delegate to EntAdapter where
	// possible. The EntAdapter does not yet have fulfillment tables, so repo
	// methods return empty results. As the fulfillment Ent schema is finalized,
	// these delegate to real Ent queries.
	fulfillmentSvc, err := fulfillment.New(fulfillment.Config{
		Repo:    fulfillmentRepoAdapter{entAdapter},
		Orders:  fulfillmentOrderAdapter{entAdapter},
		Grantor: fulfillmentCreditAdapter{},
		Logger:  logger,
	})
	if err != nil {
		return nil, fmt.Errorf("fulfillment service: %w", err)
	}

	// Reconciliation checker with Ent-backed probe.
	probe := reconciliation.NewEntProbeAdapter()
	reconChecker := reconciliation.New(reconciliation.Config{
		Probe:  probe,
		Logger: logger,
	})

	// Namespace resolver backed by the static default namespace.
	nsDecoder := namespacedriver.StaticNamespaceDecoder(defaultNamespace)
	resolveNamespace := func(ctx context.Context) (string, error) {
		ns, ok := nsDecoder.GetNamespace(ctx)
		if !ok {
			return "", fmt.Errorf("failed to resolve namespace")
		}
		return ns, nil
	}

	svc := commercehandler.Services{
		Wallet:  walletSvc,
		Catalog: catalogSvc,
		Orders:  orderSvc,
		// Payment and Refund are nil until their Ent-backed repositories are
		// implemented. The handler returns 501 for those routes.
	}

	handler := commercehandler.New(resolveNamespace, svc)

	// Build the worker manager with runners backed by real domain services.
	// Fulfillment and reconciliation runners are wired now. Payment/refund/
	// enterprise runners are omitted until their services are wired.
	reconWrapper := &reconWorkerAdapter{checker: reconChecker}
	leaseWrapper := &leaseRecoveryAdapter{}
	workerMgr, err := worker.RegisterCommerceWorkers(worker.CommerceWorkerDeps{
		Namespace:      defaultNamespace,
		Fulfillment:    &fulfillmentWorkerAdapter{svc: fulfillmentSvc},
		Reconciliation: reconWrapper,
		LeaseRecovery:  leaseWrapper,
		Logger:         logger,
	})
	if err != nil {
		return nil, err
	}

	return &CommerceWiring{
		Handler:       handler,
		WorkerManager: workerMgr,
	}, nil
}

// ---------------------------------------------------------------------------
// Adapter wrappers — bridge service interfaces to worker narrow interfaces
// ---------------------------------------------------------------------------

// walletDataPortAdapter wraps the EntAdapter to satisfy wallet.DataPort.
type walletDataPortAdapter struct {
	*commerce.EntAdapter
}

// fulfillmentWorkerAdapter bridges fulfillment.Service to the worker's
// fulfillmentProcessor interface.
type fulfillmentWorkerAdapter struct {
	svc fulfillment.Service
}

func (a *fulfillmentWorkerAdapter) ProcessPending(ctx context.Context, namespace string, limit int) (int, error) {
	return a.svc.ProcessPending(ctx, namespace, limit)
}

// reconWorkerAdapter bridges reconciliation.Checker to the worker's reconRunner
// interface. Run returns the number of findings (errors are never fatal).
type reconWorkerAdapter struct {
	checker *reconciliation.Checker
}

func (a *reconWorkerAdapter) Run(ctx context.Context, namespace string) (int, error) {
	report := a.checker.Run(ctx, namespace)
	return len(report.Findings), nil
}

// leaseRecoveryAdapter bridges the fulfillment Repository's lease recovery
// to the worker's LeaseRecoverer interface.
type leaseRecoveryAdapter struct{}

func (leaseRecoveryAdapter) RecoverExpiredLeases(_ context.Context) (int, error) {
	// The EntAdapter does not yet implement lease recovery queries.
	// Returning 0 is safe — no stale leases to reclaim.
	return 0, nil
}

// ---------------------------------------------------------------------------
// EntAdapter → fulfillment port adapters
// ---------------------------------------------------------------------------

// fulfillmentRepoAdapter wraps EntAdapter to satisfy fulfillment.Repository.
// The EntAdapter does not yet have the fulfillment table queries, so these
// methods return empty results. As the fulfillment Ent schema is finalized,
// each method delegates to the real Ent query.
type fulfillmentRepoAdapter struct {
	*commerce.EntAdapter
}

func (fulfillmentRepoAdapter) CreateFulfillment(_ context.Context, _ fulfillment.FulfillmentRequest) (*fulfillment.FulfillmentRecord, error) {
	return nil, nil
}

func (fulfillmentRepoAdapter) GetFulfillment(_ context.Context, _, _ string) (*fulfillment.FulfillmentRecord, error) {
	return nil, nil
}

func (fulfillmentRepoAdapter) GetFulfillmentByOrder(_ context.Context, _, _ string) (*fulfillment.FulfillmentRecord, error) {
	return nil, nil
}

func (fulfillmentRepoAdapter) ListPending(_ context.Context, _ string, _ int) ([]fulfillment.FulfillmentRecord, error) {
	return nil, nil
}

func (fulfillmentRepoAdapter) ClaimForProcessing(_ context.Context, _, _ string) (*fulfillment.FulfillmentRecord, error) {
	return nil, nil
}

func (fulfillmentRepoAdapter) MarkFulfilled(_ context.Context, _, _ string, _ fulfillment.FulfillmentResult) (*fulfillment.FulfillmentRecord, error) {
	return nil, nil
}

func (fulfillmentRepoAdapter) MarkFailed(_ context.Context, _, _ string, _ string) (*fulfillment.FulfillmentRecord, error) {
	return nil, nil
}

// fulfillmentCreditAdapter satisfies fulfillment.CreditGrantor. The real grant
// path uses the Ent credit ledger; this stub returns a no-op grant until the
// Ent fulfillment schema is finalized.
type fulfillmentCreditAdapter struct{}

func (fulfillmentCreditAdapter) GrantCredits(_ context.Context, _ fulfillment.GrantCreditsInput) (fulfillment.GrantCreditsResult, error) {
	return fulfillment.GrantCreditsResult{}, nil
}

// fulfillmentOrderAdapter satisfies fulfillment.OrderStatusUpdater by
// delegating to the EntAdapter's order methods.
type fulfillmentOrderAdapter struct {
	*commerce.EntAdapter
}

func (a fulfillmentOrderAdapter) GetOrder(ctx context.Context, namespace, id string) (*commerce.Order, error) {
	return a.EntAdapter.GetOrder(ctx, namespace, id)
}

func (a fulfillmentOrderAdapter) UpdateOrderStatus(ctx context.Context, namespace, id string, expectedFrom, to commerce.OrderStatus) (*commerce.Order, error) {
	return a.EntAdapter.UpdateOrderStatus(ctx, namespace, id, expectedFrom, to)
}

// ---------------------------------------------------------------------------
// PaidTxRunner adapter (C2)
// ---------------------------------------------------------------------------

// entPaidTxRunner bridges commerce.EntAdapter.RunPaidTransition to the
// payment.PaidTxRunner interface. It executes the atomic paid transition
// (insert fact + move order to paid + create fulfillment + write outbox) within
// a single customer-locked Ent transaction.
type entPaidTxRunner struct {
	adapter *commerce.EntAdapter
}

// RunPaidTransition implements payment.PaidTxRunner. It delegates to the
// EntAdapter's transactional RunPaidTransition, then maps the result.
func (r *entPaidTxRunner) RunPaidTransition(ctx context.Context, in payment.PaidTransitionInput) (payment.PaidTransitionResult, error) {
	err := r.adapter.RunPaidTransition(ctx, commerce.PaidTransitionParams{
		Namespace:        in.Namespace,
		CustomerID:       in.Attempt.CustomerID,
		OrderID:          in.Attempt.OrderID,
		PaymentAttemptID: in.Attempt.ID,
		RawHash:          in.Fact.RawHash,
		Provider:         string(in.Attempt.Provider),
		SignedPayload:    in.Fact.SignedPayload,
	})
	if err != nil {
		return payment.PaidTransitionResult{}, err
	}
	return payment.PaidTransitionResult{
		Fact: &in.Fact,
	}, nil
}
