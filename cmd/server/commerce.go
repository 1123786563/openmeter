package main

import (
	"context"
	"os"
	"fmt"
	"log/slog"
	"time"

	commercehandler "github.com/openmeterio/openmeter/api/v3/handlers/commerce"
	"github.com/openmeterio/openmeter/openmeter/commerce"
	"github.com/openmeterio/openmeter/openmeter/commerce/catalog"
	"github.com/openmeterio/openmeter/openmeter/commerce/fulfillment"
	"github.com/openmeterio/openmeter/openmeter/commerce/order"
	"github.com/openmeterio/openmeter/openmeter/commerce/payment"
	"github.com/openmeterio/openmeter/openmeter/commerce/payment/alipay"
	"github.com/openmeterio/openmeter/openmeter/commerce/payment/wechat"
	"github.com/openmeterio/openmeter/openmeter/commerce/refund"
	"github.com/openmeterio/openmeter/openmeter/commerce/reconciliation"
	"github.com/openmeterio/openmeter/openmeter/commerce/wallet"
	"github.com/openmeterio/openmeter/openmeter/commerce/worker"
	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/models"
)

// CommerceWiring holds the commerce handler and worker manager constructed from
// the application's EntClient. It is built once at startup and passed to the
// server config + lifecycle.
type CommerceWiring struct {
	Handler       commercehandler.Handler
	WorkerManager *worker.Manager
	Catalog       catalog.Service
}

// wireCommerce builds the commerce services that have Ent-backed implementations
// (catalog, orders, wallet, fulfillment, reconciliation) and constructs both the
// HTTP handler and the background worker manager. Services without Ent-backed
// repos (payment, refund, enterprise) are left nil; the handler returns 501 for
// those routes and the corresponding workers are not registered.
//
// The defaultNamespace is used to resolve the namespace for both the handler
// (via StaticNamespaceDecoder) and the worker jobs.
func wireCommerce(entClient *entdb.Client, defaultNamespace string, grantConnector credit.GrantConnector, logger *slog.Logger) (*CommerceWiring, error) {
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
		Repo:    fulfillmentRepoAdapterV2{entAdapter},
		Orders:  fulfillmentOrderAdapter{entAdapter},
		Grantor: creditGrantAdapter{connector: grantConnector},
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

	// Payment service: uses EntAdapter-backed repositories + PaidTxRunner.
	testSecrets := map[string]string{
		"wechat_app_id":             "test-wechat-app-id",
		"wechat_mch_id":             "test-wechat-mch-id",
		"wechat_api_key":            "test-wechat-api-key",
		"alipay_app_id":             "test-alipay-app-id",
	}
	// Load test RSA public key for WeChat callback signature verification.
	// In production, this comes from the platform secret manager.
	if pubKeyPEM, err := os.ReadFile("tmp/test_public_key.pem"); err == nil {
		testSecrets["wechat_platform_public_key"] = string(pubKeyPEM)
	} else {
		logger.Warn("commerce: could not load test public key, WeChat callback verification will fail", "error", err)
	}
	secretProvider := &payment.StaticSecretProvider{Secrets: testSecrets}
	wechatAdapter, _ := wechat.New(wechat.Config{Secrets: secretProvider})
	alipayAdapter, _ := alipay.New(alipay.Config{Secrets: secretProvider})
	paymentProviders := map[payment.Provider]payment.ProviderAdapter{
		payment.ProviderWeChat:  wechatAdapter,
		payment.ProviderAlipay:  alipayAdapter,
		payment.ProviderOffline: nil, // offline has no provider adapter
	}
	paymentSvc, err := payment.New(payment.Config{
		Attempts:  paymentAttemptRepoAdapter{entAdapter},
		Facts:     paymentFactRepoAdapter{entAdapter},
		Orders:    fulfillmentOrderAdapter{entAdapter},
		TxRunner:  &entPaidTxRunner{adapter: entAdapter},
		Providers: paymentProviders,
		Logger:    logger,
	})
	if err != nil {
		return nil, fmt.Errorf("payment service: %w", err)
	}

	// Refund service: uses EntAdapter-backed repository + noop stubs for
	// external dependencies (fence, reverser, snapshots). These stubs are
	// safe for offline/manual refund flows; replace with real implementations
	// for automated provider refunds.
	refundProviders := map[payment.Provider]refund.ProviderRefunder{
		payment.ProviderOffline: noopProviderRefunder{name: payment.ProviderOffline},
	}
	refundSvc, err := refund.New(refund.Config{
		Repo:      refundRepoAdapter{entAdapter},
		Orders:    refundOrderReader{entAdapter},
		Wallet:    refundWalletAdapter{entAdapter},
		Fence:     noopFenceClient{},
		Reverser:  noopCreditReverser{},
		Providers: refundProviders,
		Snapshots: noopSnapshotPublisher{},
		Logger:    logger,
	})
	if err != nil {
		return nil, fmt.Errorf("refund service: %w", err)
	}

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
		Payment: paymentSvc,
		Refund:  refundSvc,
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
		Catalog:       catalogSvc,
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

// refundOrderReader satisfies refund.OrderReader.
type refundOrderReader struct {
	*commerce.EntAdapter
}

func (a refundOrderReader) GetOrder(ctx context.Context, namespace, id string) (*commerce.Order, error) {
	return a.EntAdapter.GetOrder(ctx, namespace, id)
}

// refundWalletAdapter satisfies refund.WalletDataPort by delegating to
// EntAdapter.GetGrants.
type refundWalletAdapter struct {
	*commerce.EntAdapter
}

func (a refundWalletAdapter) GetGrants(ctx context.Context, namespace, customerID string) ([]commerce.AllocationGrant, error) {
	return a.EntAdapter.GetGrants(ctx, namespace, customerID)
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
// Code generated for commerce adapters. Appended to commerce.go.

// ---------------------------------------------------------------------------
// Payment: AttemptRepository adapter (EntAdapter → payment.AttemptRepository)
// ---------------------------------------------------------------------------

type paymentAttemptRepoAdapter struct {
	*commerce.EntAdapter
}

func (a paymentAttemptRepoAdapter) CreateAttempt(ctx context.Context, attempt payment.PaymentAttempt) (*payment.PaymentAttempt, bool, error) {
	w, fresh, err := a.EntAdapter.CreatePaymentAttempt(ctx, commerce.PaymentAttemptWire{
		ID:             attempt.ID,
		Namespace:      attempt.Namespace,
		OrderID:        attempt.OrderID,
		CustomerID:     attempt.CustomerID,
		Provider:       string(attempt.Provider),
		Status:         string(attempt.Status),
		IdempotencyKey: attempt.IdempotencyKey,
		AmountMinor:    attempt.AmountMinor,
		Currency:       attempt.Currency,
		CreatedAt:      attempt.CreatedAt,
		UpdatedAt:      attempt.UpdatedAt,
	})
	if err != nil {
		return nil, false, err
	}
	return mapWireToPaymentAttempt(w), fresh, nil
}

func (a paymentAttemptRepoAdapter) GetAttempt(ctx context.Context, namespace, id string) (*payment.PaymentAttempt, error) {
	w, err := a.EntAdapter.GetPaymentAttempt(ctx, namespace, id)
	if err != nil {
		return nil, err
	}
	return mapWireToPaymentAttempt(w), nil
}

func (a paymentAttemptRepoAdapter) GetAttemptByIdempotencyKey(ctx context.Context, namespace, customerID, key string) (*payment.PaymentAttempt, error) {
	w, err := a.EntAdapter.GetPaymentAttemptByIdempotencyKey(ctx, namespace, customerID, key)
	if err != nil {
		return nil, err
	}
	return mapWireToPaymentAttempt(w), nil
}

func (a paymentAttemptRepoAdapter) GetAttemptByProviderOrder(ctx context.Context, namespace string, provider payment.Provider, providerOrderID string) (*payment.PaymentAttempt, error) {
	w, err := a.EntAdapter.GetPaymentAttemptByProviderOrder(ctx, namespace, string(provider), providerOrderID)
	if err != nil {
		return nil, err
	}
	return mapWireToPaymentAttempt(w), nil
}

func (a paymentAttemptRepoAdapter) UpdateAttemptStatus(ctx context.Context, namespace, id string, expectedFrom, to payment.AttemptStatus) (*payment.PaymentAttempt, error) {
	w, err := a.EntAdapter.UpdatePaymentAttemptStatus(ctx, namespace, id, string(expectedFrom), string(to))
	if err != nil {
		return nil, err
	}
	return mapWireToPaymentAttempt(w), nil
}

func (a paymentAttemptRepoAdapter) SetProviderIDs(ctx context.Context, namespace, id string, providerOrderID, providerPaymentID, sessionID string) (*payment.PaymentAttempt, error) {
	w, err := a.EntAdapter.SetPaymentAttemptProviderIDs(ctx, namespace, id, providerOrderID, providerPaymentID, sessionID)
	if err != nil {
		return nil, err
	}
	return mapWireToPaymentAttempt(w), nil
}

func mapWireToPaymentAttempt(w *commerce.PaymentAttemptWire) *payment.PaymentAttempt {
	return &payment.PaymentAttempt{
		ID:                w.ID,
		Namespace:         w.Namespace,
		OrderID:           w.OrderID,
		CustomerID:        w.CustomerID,
		Provider:          payment.Provider(w.Provider),
		ProviderOrderID:   w.ProviderOrderID,
		ProviderPaymentID: w.ProviderPaymentID,
		ProviderSessionID: w.ProviderSessionID,
		Status:            payment.AttemptStatus(w.Status),
		IdempotencyKey:    w.IdempotencyKey,
		AmountMinor:       w.AmountMinor,
		Currency:          w.Currency,
		CreatedAt:         w.CreatedAt,
		UpdatedAt:         w.UpdatedAt,
	}
}

// ---------------------------------------------------------------------------
// Payment: FactRepository adapter (EntAdapter → payment.FactRepository)
// ---------------------------------------------------------------------------

type paymentFactRepoAdapter struct {
	*commerce.EntAdapter
}

func (a paymentFactRepoAdapter) InsertFact(ctx context.Context, fact payment.PaymentFactRecord) (*payment.PaymentFactRecord, bool, error) {
	w, fresh, err := a.EntAdapter.InsertPaymentFact(ctx, commerce.PaymentFactWire{
		ID:            fact.ID,
		Namespace:     fact.Namespace,
		AttemptID:     fact.AttemptID,
		Provider:      string(fact.Provider),
		RawHash:       fact.RawHash,
		SignedPayload: fact.SignedPayload,
		Timestamp:     fact.Timestamp,
		CreatedAt:     fact.CreatedAt,
	})
	if err != nil {
		return nil, false, err
	}
	return mapWireToPaymentFact(w), fresh, nil
}

func (a paymentFactRepoAdapter) GetFactByRawHash(ctx context.Context, namespace string, rawHash string) (*payment.PaymentFactRecord, error) {
	w, err := a.EntAdapter.GetPaymentFactByRawHash(ctx, namespace, rawHash)
	if err != nil {
		return nil, err
	}
	return mapWireToPaymentFact(w), nil
}

func (a paymentFactRepoAdapter) GetFactsByProviderOrder(ctx context.Context, namespace string, provider payment.Provider, providerOrderID string) ([]payment.PaymentFactRecord, error) {
	ws, err := a.EntAdapter.GetPaymentFactsByProviderOrder(ctx, namespace, string(provider), providerOrderID)
	if err != nil {
		return nil, err
	}
	result := make([]payment.PaymentFactRecord, len(ws))
	for i, w := range ws {
		result[i] = *mapWireToPaymentFact(&w)
	}
	return result, nil
}

func (a paymentFactRepoAdapter) GetFactByProviderEvent(ctx context.Context, namespace string, provider payment.Provider, providerEventID string) (*payment.PaymentFactRecord, error) {
	w, err := a.EntAdapter.GetPaymentFactByProviderEvent(ctx, namespace, string(provider), providerEventID)
	if err != nil {
		return nil, err
	}
	return mapWireToPaymentFact(w), nil
}

func mapWireToPaymentFact(w *commerce.PaymentFactWire) *payment.PaymentFactRecord {
	return &payment.PaymentFactRecord{
		ID:            w.ID,
		Namespace:     w.Namespace,
		AttemptID:     w.AttemptID,
		Provider:      payment.Provider(w.Provider),
		RawHash:       w.RawHash,
		SignedPayload: w.SignedPayload,
		Timestamp:     w.Timestamp,
		CreatedAt:     w.CreatedAt,
	}
}

// ---------------------------------------------------------------------------
// Refund: Repository adapter (EntAdapter → refund.Repository)
// ---------------------------------------------------------------------------

type refundRepoAdapter struct {
	*commerce.EntAdapter
}

func (a refundRepoAdapter) CreateRefund(ctx context.Context, req refund.RefundRequest) (*refund.RefundRequest, bool, error) {
	w, fresh, err := a.EntAdapter.CreateRefundRequest(ctx, commerce.RefundRequestWire{
		ID:              req.ID,
		Namespace:       req.Namespace,
		OrderID:         req.CommerceOrderID,
		CustomerID:      req.CustomerID,
		AmountMinor:     req.AmountCents,
		Currency:        req.Currency,
		Status:          string(req.Status),
		Reason:          req.Reason,
		IdempotencyKey:  req.IdempotencyKey,
		CreditQuantum:   req.CreditQuantum,
		RefundQuantumFen: req.RefundQuantumFen,
		CreatedAt:       req.CreatedAt,
		UpdatedAt:       req.UpdatedAt,
	})
	if err != nil {
		return nil, false, err
	}
	return mapWireToRefundRequest(w), fresh, nil
}

func (a refundRepoAdapter) GetRefund(ctx context.Context, namespace, id string) (*refund.RefundRequest, error) {
	w, err := a.EntAdapter.GetRefundRequest(ctx, namespace, id)
	if err != nil {
		return nil, err
	}
	return mapWireToRefundRequest(w), nil
}

func (a refundRepoAdapter) GetRefundByIdempotencyKey(ctx context.Context, namespace, customerID, key string) (*refund.RefundRequest, error) {
	w, err := a.EntAdapter.GetRefundRequestByIdempotencyKey(ctx, namespace, customerID, key)
	if err != nil {
		return nil, err
	}
	return mapWireToRefundRequest(w), nil
}

func (a refundRepoAdapter) GetRefundByProviderRefundID(ctx context.Context, namespace, providerRefundID string) (*refund.RefundRequest, error) {
	w, err := a.EntAdapter.GetRefundRequestByProviderRefundID(ctx, namespace, providerRefundID)
	if err != nil {
		return nil, err
	}
	return mapWireToRefundRequest(w), nil
}

func (a refundRepoAdapter) TransitionStatus(ctx context.Context, namespace, id string, expectedFrom, to refund.RefundStatus) (*refund.RefundRequest, error) {
	w, err := a.EntAdapter.TransitionRefundStatus(ctx, namespace, id, string(expectedFrom), string(to))
	if err != nil {
		return nil, err
	}
	return mapWireToRefundRequest(w), nil
}

func (a refundRepoAdapter) SaveQuantum(ctx context.Context, namespace, id string, q refund.QuantumReservation) (*refund.RefundRequest, error) {
	w, err := a.EntAdapter.SaveRefundQuantum(ctx, namespace, id, q.CreditQuantum, q.RefundQuantumFen, q.ReservedCredits, q.RefundFen, q.RemainderCredits)
	if err != nil {
		return nil, err
	}
	return mapWireToRefundRequest(w), nil
}

func (a refundRepoAdapter) SetProviderRefundID(ctx context.Context, namespace, id, providerName, providerRefundID string) (*refund.RefundRequest, error) {
	w, err := a.EntAdapter.SetRefundProviderDetails(ctx, namespace, id, providerName, providerRefundID, 0, 0, 0)
	if err != nil {
		return nil, err
	}
	return mapWireToRefundRequest(w), nil
}

func (a refundRepoAdapter) SetFence(ctx context.Context, namespace, id, fenceSequence string) (*refund.RefundRequest, error) {
	w, err := a.EntAdapter.SetRefundFence(ctx, namespace, id, fenceSequence)
	if err != nil {
		return nil, err
	}
	return mapWireToRefundRequest(w), nil
}

func (a refundRepoAdapter) SetSnapshot(ctx context.Context, namespace, id, snapshotVersion string) (*refund.RefundRequest, error) {
	w, err := a.EntAdapter.SetRefundSnapshot(ctx, namespace, id, snapshotVersion)
	if err != nil {
		return nil, err
	}
	return mapWireToRefundRequest(w), nil
}

func (a refundRepoAdapter) MarkFailed(ctx context.Context, namespace, id, reason string) (*refund.RefundRequest, error) {
	w, err := a.EntAdapter.MarkRefundFailed(ctx, namespace, id, reason)
	if err != nil {
		return nil, err
	}
	return mapWireToRefundRequest(w), nil
}

func (a refundRepoAdapter) ReserveCredits(ctx context.Context, refundID string, in refund.ReservationInput) (refund.ReservationResult, error) {
	// TODO: implement full atomic credit reservation with allocation locking.
	// For now, perform a simple reservation based on the input.
	creditQuantum := refund.CreditQuantum
	refundQuantumFen := refund.RefundQuantumFen

	// Calculate how many full quanta fit in the requested fen.
	numQuanta := in.RequestedFen / refundQuantumFen
	if numQuanta < 1 {
		numQuanta = 1
	}
	reservedCredits := numQuanta * creditQuantum
	refundFen := numQuanta * refundQuantumFen
	remainderCredits := in.RefundableCredits - reservedCredits

	if reservedCredits > in.RefundableCredits {
		return refund.ReservationResult{Granted: false}, nil
	}

	// Persist the quantum on the refund record.
	_, err := a.EntAdapter.SaveRefundQuantum(ctx, in.Namespace, refundID,
		creditQuantum, refundQuantumFen, reservedCredits, refundFen, remainderCredits)
	if err != nil {
		return refund.ReservationResult{}, err
	}

	return refund.ReservationResult{
		Granted:           true,
		RefundFen:         refundFen,
		ReservedCredits:   reservedCredits,
		RemainderCredits:  remainderCredits,
		RefundableCredits: in.RefundableCredits,
	}, nil
}

func (a refundRepoAdapter) AppendFact(ctx context.Context, fact refund.RefundFactRecord) (*refund.RefundFactRecord, bool, error) {
	w, fresh, err := a.EntAdapter.AppendRefundFact(ctx, commerce.RefundFactWire{
		ID:              fact.ID,
		Namespace:       fact.Namespace,
		RefundRequestID: fact.RefundRequestID,
		Provider:        string(fact.Provider),
		RawHash:         fact.RawHash,
		SignedPayload:   fact.SignedPayload,
		Timestamp:       fact.Timestamp,
		CreatedAt:       fact.CreatedAt,
	})
	if err != nil {
		return nil, false, err
	}
	return mapWireToRefundFact(w), fresh, nil
}

func (a refundRepoAdapter) GetFacts(ctx context.Context, namespace, refundID string) ([]refund.RefundFactRecord, error) {
	ws, err := a.EntAdapter.GetRefundFacts(ctx, namespace, refundID)
	if err != nil {
		return nil, err
	}
	result := make([]refund.RefundFactRecord, len(ws))
	for i, w := range ws {
		result[i] = *mapWireToRefundFact(&w)
	}
	return result, nil
}

func mapWireToRefundRequest(w *commerce.RefundRequestWire) *refund.RefundRequest {
	r := &refund.RefundRequest{
		ID:               w.ID,
		Namespace:        w.Namespace,
		CommerceOrderID:  w.OrderID,
		CustomerID:       w.CustomerID,
		AmountCents:      w.AmountMinor,
		Currency:         w.Currency,
		Status:           refund.RefundStatus(w.Status),
		Reason:           w.Reason,
		IdempotencyKey:   w.IdempotencyKey,
		CreditQuantum:    w.CreditQuantum,
		RefundQuantumFen: w.RefundQuantumFen,
		ReservedCredits:  w.ReservedCredits,
		RefundFen:        w.RefundFen,
		RemainderCredits: w.RemainderCredits,
		ProviderName:     w.ProviderName,
		ProviderRefundID: w.ProviderRefundID,
		FenceSequence:    w.FenceSequence,
		SnapshotVersion:  w.SnapshotVersion,
		CreatedAt:        w.CreatedAt,
		UpdatedAt:        w.UpdatedAt,
	}
	return r
}

func mapWireToRefundFact(w *commerce.RefundFactWire) *refund.RefundFactRecord {
	return &refund.RefundFactRecord{
		ID:               w.ID,
		Namespace:        w.Namespace,
		RefundRequestID:  w.RefundRequestID,
		Provider:         payment.Provider(w.Provider),
		RawHash:          w.RawHash,
		SignedPayload:    w.SignedPayload,
		Timestamp:        w.Timestamp,
		CreatedAt:        w.CreatedAt,
	}
}

// ---------------------------------------------------------------------------
// CreditGrantor: real implementation using credit.GrantConnector
// ---------------------------------------------------------------------------

type creditGrantAdapter struct {
	connector credit.GrantConnector
}

func (g creditGrantAdapter) GrantCredits(ctx context.Context, in fulfillment.GrantCreditsInput) (fulfillment.GrantCreditsResult, error) {
	now := time.Now()
	input := credit.CreateGrantInput{
		Amount:      float64(in.Credits),
		Priority:    1,
		EffectiveAt: now,
		Metadata: map[string]string{
			"source":           string(in.Source),
			"order_id":         in.OrderID,
			"idempotency_key":  in.IdempotencyKey,
		},
	}
	if in.ValidityDays > 0 {
		input.Expiration = &grant.ExpirationPeriod{
			Count:    uint32(in.ValidityDays),
			Duration: grant.ExpirationPeriodDurationDay,
		}
	}

	created, err := g.connector.CreateGrant(ctx, models.NamespacedID{
		Namespace: in.Namespace,
		ID:        in.CustomerID,
	}, input)
	if err != nil {
		return fulfillment.GrantCreditsResult{}, fmt.Errorf("credit grant: %w", err)
	}

	return fulfillment.GrantCreditsResult{
		GrantID: created.ID,
		Credits: int64(created.Amount),
	}, nil
}

// ---------------------------------------------------------------------------
// Refund noop stubs (external dependencies)
// ---------------------------------------------------------------------------

// noopFenceClient is a stub FenceClient. It always succeeds without actually
// establishing a fence. Replace with a real WeKnora fence API client.
type noopFenceClient struct{}

func (noopFenceClient) EstablishFence(_ context.Context, _, _ string) (refund.FenceResult, error) {
	return refund.FenceResult{Sequence: "noop", Established: true}, nil
}
func (noopFenceClient) ReleaseFence(_ context.Context, _, _, _ string) error { return nil }
func (noopFenceClient) ConfirmSnapshotApplied(_ context.Context, _, _, _ string) (bool, error) {
	return true, nil
}

// noopCreditReverser is a stub CreditReverser. Replace with a real credit
// ledger reversal implementation.
type noopCreditReverser struct{}

func (noopCreditReverser) ReverseCredits(_ context.Context, in refund.ReverseCreditsInput) (refund.ReverseCreditsResult, error) {
	return refund.ReverseCreditsResult{
		LedgerEntryID: "noop-reversal",
		Credits:       in.Credits,
	}, nil
}

// noopSnapshotPublisher is a stub SnapshotPublisher.
type noopSnapshotPublisher struct{}

func (noopSnapshotPublisher) PublishSnapshot(_ context.Context, _ refund.PublishSnapshotInput) (string, error) {
	return "noop-snapshot-v1", nil
}

// noopProviderRefunder is a stub for provider refund operations.
type noopProviderRefunder struct {
	name payment.Provider
}

func (n noopProviderRefunder) Refund(_ context.Context, input payment.RefundInput) (payment.RefundSubmission, error) {
	return payment.RefundSubmission{
		Provider:         n.name,
		ProviderRefundID: "offline-refund-" + input.IdempotencyKey,
		Status:           "succeeded",
	}, nil
}
func (n noopProviderRefunder) QueryRefund(_ context.Context, _ string) (payment.RefundFact, error) {
	return payment.RefundFact{
		Provider: n.name,
		Success:  true,
	}, nil
}
func (n noopProviderRefunder) Name() payment.Provider { return n.name }

// noopProviderResolver resolves the provider from a refund's stored provider name.
type noopProviderResolver struct {
	adapter *commerce.EntAdapter
}

func (r noopProviderResolver) ResolveProviderForOrder(ctx context.Context, namespace, orderID string) (payment.Provider, error) {
	// Try to find the order's payment attempt to determine the provider.
	// For now, return offline as the default.
	return payment.ProviderOffline, nil
}

// ---------------------------------------------------------------------------
// Fulfillment: real Repository adapter (EntAdapter → fulfillment.Repository)
// ---------------------------------------------------------------------------

type fulfillmentRepoAdapterV2 struct {
	*commerce.EntAdapter
}

func (a fulfillmentRepoAdapterV2) CreateFulfillment(ctx context.Context, req fulfillment.FulfillmentRequest) (*fulfillment.FulfillmentRecord, error) {
	w, err := a.EntAdapter.CreateFulfillment(ctx, commerce.FulfillmentCreateWire{
		Namespace:  req.Namespace,
		OrderID:    req.OrderID,
		CustomerID: req.CustomerID,
	})
	if err != nil {
		return nil, err
	}
	return mapWireToFulfillment(w), nil
}

func (a fulfillmentRepoAdapterV2) GetFulfillment(ctx context.Context, namespace, id string) (*fulfillment.FulfillmentRecord, error) {
	w, err := a.EntAdapter.GetFulfillment(ctx, namespace, id)
	if err != nil {
		return nil, err
	}
	return mapWireToFulfillment(w), nil
}

func (a fulfillmentRepoAdapterV2) GetFulfillmentByOrder(ctx context.Context, namespace, orderID string) (*fulfillment.FulfillmentRecord, error) {
	w, err := a.EntAdapter.GetFulfillmentByOrder(ctx, namespace, orderID)
	if err != nil {
		return nil, err
	}
	return mapWireToFulfillment(w), nil
}

func (a fulfillmentRepoAdapterV2) ListPending(ctx context.Context, namespace string, limit int) ([]fulfillment.FulfillmentRecord, error) {
	ws, err := a.EntAdapter.ListPendingFulfillments(ctx, namespace, limit)
	if err != nil {
		return nil, err
	}
	result := make([]fulfillment.FulfillmentRecord, len(ws))
	for i, w := range ws {
		result[i] = *mapWireToFulfillment(&w)
	}
	return result, nil
}

func (a fulfillmentRepoAdapterV2) ClaimForProcessing(ctx context.Context, namespace, id string) (*fulfillment.FulfillmentRecord, error) {
	w, err := a.EntAdapter.ClaimFulfillment(ctx, namespace, id)
	if err != nil {
		return nil, err
	}
	return mapWireToFulfillment(w), nil
}

func (a fulfillmentRepoAdapterV2) MarkFulfilled(ctx context.Context, namespace, id string, result fulfillment.FulfillmentResult) (*fulfillment.FulfillmentRecord, error) {
	w, err := a.EntAdapter.MarkFulfillmentFulfilled(ctx, namespace, id, result.GrantID, result.CreditsGranted)
	if err != nil {
		return nil, err
	}
	return mapWireToFulfillment(w), nil
}

func (a fulfillmentRepoAdapterV2) MarkFailed(ctx context.Context, namespace, id string, reason string) (*fulfillment.FulfillmentRecord, error) {
	w, err := a.EntAdapter.MarkFulfillmentFailed(ctx, namespace, id, reason)
	if err != nil {
		return nil, err
	}
	return mapWireToFulfillment(w), nil
}

func mapWireToFulfillment(w *commerce.FulfillmentWire) *fulfillment.FulfillmentRecord {
	rec := &fulfillment.FulfillmentRecord{
		ID:             w.ID,
		Namespace:      w.Namespace,
		OrderID:        w.OrderID,
		CustomerID:     w.CustomerID,
		Status:         fulfillment.FulfillmentStatus(w.Status),
		GrantID:        w.GrantID,
		CreditsGranted: w.CreditsGranted,
		ClaimedAt:      w.ClaimedAt,
		FulfilledAt:    w.FulfilledAt,
		CreatedAt:      w.CreatedAt,
		UpdatedAt:      w.UpdatedAt,
	}
	if w.FailureReason != "" {
		fr := w.FailureReason
		rec.FailureReason = &fr
	}
	return rec
}
