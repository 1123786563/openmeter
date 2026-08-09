package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"sort"
	"time"

	commercehandler "github.com/openmeterio/openmeter/api/v3/handlers/commerce"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/commerce"
	"github.com/openmeterio/openmeter/openmeter/commerce/catalog"
	"github.com/openmeterio/openmeter/openmeter/commerce/fulfillment"
	"github.com/openmeterio/openmeter/openmeter/commerce/order"
	"github.com/openmeterio/openmeter/openmeter/commerce/payment"
	"github.com/openmeterio/openmeter/openmeter/commerce/payment/alipay"
	"github.com/openmeterio/openmeter/openmeter/commerce/payment/wechat"
	"github.com/openmeterio/openmeter/openmeter/commerce/reconciliation"
	"github.com/openmeterio/openmeter/openmeter/commerce/refund"
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

	paymentProviders map[payment.Provider]payment.ProviderAdapter
	refundProviders  map[payment.Provider]refund.ProviderRefunder
}

// commerceRuntimeDependencies is the narrow seam for external automatic
// refund collaborators. Production currently passes nil; tests and future
// runtime integrations can supply complete real implementations atomically.
type commerceRuntimeDependencies struct {
	RefundFence             refund.FenceClient
	RefundCreditReverser    refund.CreditReverser
	RefundSnapshotPublisher refund.SnapshotPublisher
}

// wireCommerce builds the commerce services that have Ent-backed implementations
// (catalog, orders, wallet, fulfillment, reconciliation) and conditionally
// constructs production payment/refund services and workers.
//
// The defaultNamespace is used to resolve the namespace for both the handler
// (via StaticNamespaceDecoder) and the worker jobs.
func wireCommerce(
	entClient *entdb.Client,
	defaultNamespace string,
	grantConnector credit.GrantConnector,
	cfg config.CommerceConfiguration,
	runtimeDeps *commerceRuntimeDependencies,
	logger *slog.Logger,
) (*CommerceWiring, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("commerce configuration: %w", err)
	}
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

	paymentProviders := map[payment.Provider]payment.ProviderAdapter{}
	refundProviders := map[payment.Provider]refund.ProviderRefunder{}
	var paymentSvc payment.Service
	var refundSvc refund.Service
	var paymentRecovery *payment.Recovery
	var refundWorker *refundWorkerAdapter
	if cfg.Enabled {
		if err := validateAutomaticRefundDependencies(runtimeDeps); err != nil {
			return nil, err
		}
		paymentProviders, refundProviders, err = wirePaymentProviders(cfg, logger)
		if err != nil {
			return nil, err
		}

		if len(paymentProviders) > 0 {
			attemptRepo := paymentAttemptRepoAdapter{entAdapter}
			paymentSvc, err = payment.New(payment.Config{
				Attempts:  attemptRepo,
				Facts:     paymentFactRepoAdapter{entAdapter},
				Orders:    fulfillmentOrderAdapter{entAdapter},
				TxRunner:  &entPaidTxRunner{adapter: entAdapter},
				Providers: paymentProviders,
				Logger:    logger,
			})
			if err != nil {
				return nil, fmt.Errorf("payment service: %w", err)
			}
			paymentRecovery = payment.NewRecovery(attemptRepo, paymentSvc, cfg.Payment.PendingStaleAfter)

			refundSvc, err = refund.New(refund.Config{
				Repo:             refundRepoAdapter{entAdapter},
				Orders:           refundOrderReader{entAdapter},
				Wallet:           refundWalletAdapter{entAdapter},
				Fence:            runtimeDeps.RefundFence,
				Reverser:         runtimeDeps.RefundCreditReverser,
				Providers:        refundProviders,
				ProviderResolver: paymentProviderResolverAdapter{entAdapter},
				Snapshots:        runtimeDeps.RefundSnapshotPublisher,
				Logger:           logger,
			})
			if err != nil {
				return nil, fmt.Errorf("refund service: %w", err)
			}
			refundWorker = &refundWorkerAdapter{svc: refundSvc, repo: entAdapter}
		}
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
	wiring := &CommerceWiring{
		Handler: handler, Catalog: catalogSvc,
		paymentProviders: paymentProviders, refundProviders: refundProviders,
	}
	if !cfg.Enabled {
		wiring.Handler = readOnlyCommerceHandler{delegate: handler}
		return wiring, nil
	}

	// Build the worker manager with runners backed by real domain services.
	// Payment and refund runners are registered only when at least one payment
	// channel was configured and the complete refund dependency bundle passed
	// validation above.
	reconWrapper := &reconWorkerAdapter{checker: reconChecker}
	leaseWrapper := &leaseRecoveryAdapter{repo: entAdapter, namespace: defaultNamespace}
	workerMgr, err := worker.RegisterCommerceWorkers(worker.CommerceWorkerDeps{
		Namespace:      defaultNamespace,
		Fulfillment:    &fulfillmentWorkerAdapter{svc: fulfillmentSvc},
		Payment:        paymentRecovery,
		Refund:         refundWorker,
		Reconciliation: reconWrapper,
		LeaseRecovery:  leaseWrapper,
		Logger:         logger,
	})
	if err != nil {
		return nil, err
	}

	wiring.WorkerManager = workerMgr
	return wiring, nil
}

// readOnlyCommerceHandler keeps commerce query routes available while making
// the disabled configuration a hard mutation boundary. The short-circuit
// handlers intentionally do not parse request bodies or call domain services.
type readOnlyCommerceHandler struct {
	delegate commercehandler.Handler
}

func (h readOnlyCommerceHandler) GetCustomerWallet() http.HandlerFunc {
	return h.delegate.GetCustomerWallet()
}

func (h readOnlyCommerceHandler) ListRechargeProducts() http.HandlerFunc {
	return h.delegate.ListRechargeProducts()
}

func (readOnlyCommerceHandler) CreateProduct() http.HandlerFunc { return commerceDisabledMutation() }

func (readOnlyCommerceHandler) UpdateProduct() http.HandlerFunc { return commerceDisabledMutation() }

func (readOnlyCommerceHandler) CreateOrder() http.HandlerFunc { return commerceDisabledMutation() }

func (h readOnlyCommerceHandler) GetOrder() http.HandlerFunc { return h.delegate.GetOrder() }

func (readOnlyCommerceHandler) CreateCheckoutSession() http.HandlerFunc {
	return commerceDisabledMutation()
}

func (h readOnlyCommerceHandler) GetCheckoutSession() http.HandlerFunc {
	return h.delegate.GetCheckoutSession()
}

func (readOnlyCommerceHandler) AlipayPaymentCallback() http.HandlerFunc {
	return commerceDisabledMutation()
}

func (readOnlyCommerceHandler) WechatPaymentCallback() http.HandlerFunc {
	return commerceDisabledMutation()
}

func (readOnlyCommerceHandler) CreateRefund() http.HandlerFunc { return commerceDisabledMutation() }

func (h readOnlyCommerceHandler) GetRefund() http.HandlerFunc { return h.delegate.GetRefund() }

func (readOnlyCommerceHandler) CreateOfflinePayment() http.HandlerFunc {
	return commerceDisabledMutation()
}

func (h readOnlyCommerceHandler) ListReceivablePeriods() http.HandlerFunc {
	return h.delegate.ListReceivablePeriods()
}

func (readOnlyCommerceHandler) UpdateExternalInvoice() http.HandlerFunc {
	return commerceDisabledMutation()
}

func commerceDisabledMutation() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func wirePaymentProviders(cfg config.CommerceConfiguration, logger *slog.Logger) (map[payment.Provider]payment.ProviderAdapter, map[payment.Provider]refund.ProviderRefunder, error) {
	paymentProviders := map[payment.Provider]payment.ProviderAdapter{}
	refundProviders := map[payment.Provider]refund.ProviderRefunder{}
	secretPaths := map[string]string{}
	if cfg.Payment.WeChat.Enabled {
		secretPaths[wechat.SecretKeyMerchantPrivateKey] = cfg.Payment.WeChat.MerchantPrivateKeyFile
		secretPaths[wechat.SecretKeyAPIv3] = cfg.Payment.WeChat.APIv3KeyFile
		for serial, path := range cfg.Payment.WeChat.PlatformPublicKeyFiles {
			secretPaths[wechat.PlatformPublicKeySecret(serial)] = path
		}
	}
	if cfg.Payment.Alipay.Enabled {
		secretPaths[alipay.SecretKeyAppPrivateKey] = cfg.Payment.Alipay.AppPrivateKeyFile
		secretPaths[alipay.SecretKeyAlipayPublicKey] = cfg.Payment.Alipay.AlipayPublicKeyFile
	}
	if len(secretPaths) == 0 {
		return paymentProviders, refundProviders, nil
	}

	secretProvider, err := payment.NewFileSecretProvider(secretPaths)
	if err != nil {
		return nil, nil, fmt.Errorf("commerce payment secrets: %w", err)
	}
	if cfg.Payment.WeChat.Enabled {
		platformSerials := make([]string, 0, len(cfg.Payment.WeChat.PlatformPublicKeyFiles))
		for serial := range cfg.Payment.WeChat.PlatformPublicKeyFiles {
			platformSerials = append(platformSerials, serial)
		}
		sort.Strings(platformSerials)
		adapter, err := wechat.New(wechat.Config{
			Secrets: secretProvider, Client: &http.Client{Timeout: cfg.Payment.HTTPTimeout},
			BaseURL: cfg.Payment.WeChat.BaseURL, AppID: cfg.Payment.WeChat.AppID,
			MerchantID: cfg.Payment.WeChat.MerchantID, MerchantSerial: cfg.Payment.WeChat.MerchantSerial,
			PlatformPublicKeySerials: platformSerials,
			NotifyURL:                cfg.Payment.WeChat.NotifyURL, RefundNotifyURL: cfg.Payment.WeChat.RefundNotifyURL,
			Now: time.Now, CallbackMaxAge: cfg.Payment.WeChat.CallbackMaxAge,
			MaxResponseBytes: cfg.Payment.MaxResponseBytes, Logger: logger,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("commerce WeChat adapter: %w", err)
		}
		paymentProviders[payment.ProviderWeChat] = adapter
		refundProviders[payment.ProviderWeChat] = adapter
	}
	if cfg.Payment.Alipay.Enabled {
		adapter, err := alipay.New(alipay.Config{
			Secrets: secretProvider, Client: &http.Client{Timeout: cfg.Payment.HTTPTimeout},
			GatewayURL: cfg.Payment.Alipay.GatewayURL, AppID: cfg.Payment.Alipay.AppID,
			SellerID: cfg.Payment.Alipay.SellerID, NotifyURL: cfg.Payment.Alipay.NotifyURL,
			Now: time.Now, MaxResponseBytes: cfg.Payment.MaxResponseBytes, Logger: logger,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("commerce Alipay adapter: %w", err)
		}
		paymentProviders[payment.ProviderAlipay] = adapter
		refundProviders[payment.ProviderAlipay] = adapter
	}

	return paymentProviders, refundProviders, nil
}

// validateCommerceProviderConfiguration performs the provider-only portion of
// production assembly for --validate. It reads and validates configured local
// key material, but does not construct domain services, workers, or listeners.
// Automatic refund dependency validation remains in wireCommerce so normal
// startup continues to fail closed when those real dependencies are absent.
func validateCommerceProviderConfiguration(cfg config.CommerceConfiguration, logger *slog.Logger) error {
	if !cfg.Enabled {
		return nil
	}
	if _, _, err := wirePaymentProviders(cfg, logger); err != nil {
		return fmt.Errorf("commerce provider configuration: %w", err)
	}
	return nil
}

func validateAutomaticRefundDependencies(deps *commerceRuntimeDependencies) error {
	var fence refund.FenceClient
	var reverser refund.CreditReverser
	var publisher refund.SnapshotPublisher
	if deps != nil {
		fence = deps.RefundFence
		reverser = deps.RefundCreditReverser
		publisher = deps.RefundSnapshotPublisher
	}
	for _, dependency := range []struct {
		name  string
		value any
	}{
		{name: "refund fence", value: fence},
		{name: "credit reverser", value: reverser},
		{name: "snapshot publisher", value: publisher},
	} {
		if unavailableCommerceDependency(dependency.value) {
			return fmt.Errorf("commerce automatic refund disabled: real %s is required", dependency.name)
		}
	}
	return nil
}

func unavailableCommerceDependency(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		if reflected.IsNil() {
			return true
		}
	}
	if marker, ok := value.(interface{ IsNoop() bool }); ok {
		return marker.IsNoop()
	}
	return false
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

// refundWorkerAdapter exposes the refund service through the worker's narrow
// batch boundary. ProcessOne deliberately returns the service error unchanged.
type refundWorkerAdapter struct {
	svc  refundProcessService
	repo *commerce.EntAdapter
}

type refundProcessService interface {
	ProcessOne(ctx context.Context, namespace, refundID string) (*refund.RefundRequest, error)
}

func (a *refundWorkerAdapter) ListProviderProcessing(ctx context.Context, namespace string) ([]string, error) {
	refunds, err := a.repo.ListProviderProcessingRefundRequests(ctx, namespace, 100)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(refunds))
	for i, item := range refunds {
		ids[i] = item.ID
	}
	return ids, nil
}

func (a *refundWorkerAdapter) ProcessOne(ctx context.Context, namespace, refundID string) error {
	_, err := a.svc.ProcessOne(ctx, namespace, refundID)
	return err
}

// reconWorkerAdapter bridges reconciliation.Checker to the worker's reconRunner
// interface. Run returns the number of findings produced by the checker.
type reconWorkerAdapter struct {
	checker *reconciliation.Checker
}

func (a *reconWorkerAdapter) Run(ctx context.Context, namespace string) (int, error) {
	report := a.checker.Run(ctx, namespace)
	return len(report.Findings), nil
}

// leaseRecoveryAdapter bridges the fulfillment Repository's lease recovery
// to the worker's LeaseRecoverer interface.
type leaseRecoveryAdapter struct {
	repo      *commerce.EntAdapter
	namespace string
}

func (a leaseRecoveryAdapter) RecoverExpiredLeases(ctx context.Context) (int, error) {
	return a.repo.RecoverExpiredFulfillmentLeases(ctx, a.namespace, fulfillment.ProcessingLeaseTimeout)
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
	result, err := r.adapter.RunPaidTransition(ctx, commerce.PaidTransitionParams{
		Namespace:         in.Namespace,
		CustomerID:        in.Attempt.CustomerID,
		OrderID:           in.Attempt.OrderID,
		PaymentAttemptID:  in.Attempt.ID,
		PaymentFactID:     in.Fact.ID,
		RawHash:           in.Fact.RawHash,
		Provider:          string(in.Fact.Provider),
		ProviderOrderID:   in.Fact.ProviderOrderID,
		ProviderPaymentID: in.Fact.ProviderPaymentID,
		ProviderEventID:   in.Fact.ProviderEventID,
		MerchantID:        in.Fact.MerchantID,
		ApplicationID:     in.Fact.ApplicationID,
		AmountMinor:       in.Fact.AmountMinor,
		Currency:          in.Fact.Currency,
		Success:           in.Fact.Success,
		SignedPayload:     in.Fact.SignedPayload,
		Timestamp:         in.Fact.Timestamp,
		CreatedAt:         in.Fact.CreatedAt,
	})
	if err != nil {
		return payment.PaidTransitionResult{}, err
	}
	return payment.PaidTransitionResult{
		Fact:        mapWireToPaymentFact(result.Fact),
		AlreadyPaid: result.AlreadyPaid,
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
		ID:                    attempt.ID,
		Namespace:             attempt.Namespace,
		OrderID:               attempt.OrderID,
		CustomerID:            attempt.CustomerID,
		Provider:              string(attempt.Provider),
		ExpectedMerchantID:    attempt.ExpectedMerchantID,
		ExpectedApplicationID: attempt.ExpectedApplicationID,
		Status:                string(attempt.Status),
		IdempotencyKey:        attempt.IdempotencyKey,
		AmountMinor:           attempt.AmountMinor,
		Currency:              attempt.Currency,
		CreatedAt:             attempt.CreatedAt,
		UpdatedAt:             attempt.UpdatedAt,
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

func (a paymentAttemptRepoAdapter) ListStalePendingAttempts(ctx context.Context, namespace string, cutoff time.Time, limit int) ([]payment.PaymentAttempt, error) {
	wires, err := a.EntAdapter.ListStalePendingPaymentAttempts(ctx, namespace, cutoff, limit)
	if err != nil {
		return nil, err
	}
	attempts := make([]payment.PaymentAttempt, len(wires))
	for i := range wires {
		attempts[i] = *mapWireToPaymentAttempt(&wires[i])
	}
	return attempts, nil
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
		ID:                    w.ID,
		Namespace:             w.Namespace,
		OrderID:               w.OrderID,
		CustomerID:            w.CustomerID,
		Provider:              payment.Provider(w.Provider),
		ProviderOrderID:       w.ProviderOrderID,
		ProviderPaymentID:     w.ProviderPaymentID,
		ProviderSessionID:     w.ProviderSessionID,
		ExpectedMerchantID:    w.ExpectedMerchantID,
		ExpectedApplicationID: w.ExpectedApplicationID,
		Status:                payment.AttemptStatus(w.Status),
		IdempotencyKey:        w.IdempotencyKey,
		AmountMinor:           w.AmountMinor,
		Currency:              w.Currency,
		CreatedAt:             w.CreatedAt,
		UpdatedAt:             w.UpdatedAt,
	}
}

type paymentProviderResolverAdapter struct {
	*commerce.EntAdapter
}

func (a paymentProviderResolverAdapter) ResolveProviderForOrder(ctx context.Context, namespace, orderID string) (payment.Provider, error) {
	provider, err := a.EntAdapter.ResolvePaymentProviderForOrder(ctx, namespace, orderID)
	if err != nil {
		return "", err
	}
	return payment.Provider(provider), nil
}

// ---------------------------------------------------------------------------
// Payment: FactRepository adapter (EntAdapter → payment.FactRepository)
// ---------------------------------------------------------------------------

type paymentFactRepoAdapter struct {
	*commerce.EntAdapter
}

func (a paymentFactRepoAdapter) InsertFact(ctx context.Context, fact payment.PaymentFactRecord) (*payment.PaymentFactRecord, bool, error) {
	w, fresh, err := a.EntAdapter.InsertPaymentFact(ctx, commerce.PaymentFactWire{
		ID:                fact.ID,
		Namespace:         fact.Namespace,
		AttemptID:         fact.AttemptID,
		Provider:          string(fact.Provider),
		ProviderOrderID:   fact.ProviderOrderID,
		ProviderPaymentID: fact.ProviderPaymentID,
		ProviderEventID:   fact.ProviderEventID,
		MerchantID:        fact.MerchantID,
		ApplicationID:     fact.ApplicationID,
		AmountMinor:       fact.AmountMinor,
		Currency:          fact.Currency,
		Success:           fact.Success,
		RawHash:           fact.RawHash,
		SignedPayload:     fact.SignedPayload,
		Timestamp:         fact.Timestamp,
		CreatedAt:         fact.CreatedAt,
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
	if w == nil {
		return nil
	}
	return &payment.PaymentFactRecord{
		ID:                w.ID,
		Namespace:         w.Namespace,
		AttemptID:         w.AttemptID,
		Provider:          payment.Provider(w.Provider),
		ProviderOrderID:   w.ProviderOrderID,
		ProviderPaymentID: w.ProviderPaymentID,
		ProviderEventID:   w.ProviderEventID,
		MerchantID:        w.MerchantID,
		ApplicationID:     w.ApplicationID,
		AmountMinor:       w.AmountMinor,
		Currency:          w.Currency,
		Success:           w.Success,
		RawHash:           w.RawHash,
		SignedPayload:     w.SignedPayload,
		Timestamp:         w.Timestamp,
		CreatedAt:         w.CreatedAt,
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
		ID:               req.ID,
		Namespace:        req.Namespace,
		OrderID:          req.CommerceOrderID,
		CustomerID:       req.CustomerID,
		AmountMinor:      req.AmountCents,
		Currency:         req.Currency,
		Status:           string(req.Status),
		Reason:           req.Reason,
		IdempotencyKey:   req.IdempotencyKey,
		CreditQuantum:    req.CreditQuantum,
		RefundQuantumFen: req.RefundQuantumFen,
		CreatedAt:        req.CreatedAt,
		UpdatedAt:        req.UpdatedAt,
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
		ID:              w.ID,
		Namespace:       w.Namespace,
		RefundRequestID: w.RefundRequestID,
		Provider:        payment.Provider(w.Provider),
		RawHash:         w.RawHash,
		SignedPayload:   w.SignedPayload,
		Timestamp:       w.Timestamp,
		CreatedAt:       w.CreatedAt,
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
			"source":          string(in.Source),
			"order_id":        in.OrderID,
			"idempotency_key": in.IdempotencyKey,
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
