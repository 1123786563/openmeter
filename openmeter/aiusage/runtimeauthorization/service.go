// Package runtimeauthorization builds and signs the runtime authorization
// package consumed by the WeKnora runtime for capacity enforcement.
//
// The authorization package bundles the billing customer's subscription plan,
// entitlement codes, credit balances, rate package, and watermark position into
// a single Ed25519-signed document. The runtime uses authorization_capacity_credits
// (= spendable prepaid + available Enterprise) to gate API calls before usage
// reaches the settlement pipeline.
package runtimeauthorization

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/aiusage/signing"
)

// CreditBalance holds the credit amounts included in the authorization package.
type CreditBalance struct {
	SpendableCredits           int64
	EnterpriseAvailableCredits int64
}

// SubscriptionInfo holds the plan and period metadata.
type SubscriptionInfo struct {
	PlanCode           string
	SubscriptionCode   string
	SubscriptionStatus string
	EntitlementCodes   []string
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time
}

// RatePackageSnapshot holds the active rate package version and entries.
type RatePackageSnapshot struct {
	Version string
	Entries []signing.SignedRateEntry
}

// SubjectBalanceReader reads the credit balance for the billing customer.
type SubjectBalanceReader interface {
	ReadCreditBalance(ctx context.Context, namespace, customerID string) (CreditBalance, error)
}

// SubscriptionReader reads the customer's subscription plan and entitlements.
type SubscriptionReader interface {
	ReadSubscription(ctx context.Context, namespace, customerID string) (SubscriptionInfo, error)
}

// RatePackageReader reads the active rate package for the customer.
type RatePackageReader interface {
	ReadRatePackage(ctx context.Context, namespace, customerID string) (RatePackageSnapshot, error)
}

// CoveredSeqReader reads the highest continuously settled tenant_seq.
type CoveredSeqReader interface {
	ReadCoveredSeq(ctx context.Context, namespace, customerID string) (int64, error)
}

// SnapshotVersionProvider generates strictly increasing snapshot versions.
// Implementations must guarantee monotonicity across restarts (e.g. a DB
// sequence or an epoch-millis + counter scheme).
type SnapshotVersionProvider interface {
	Next(ctx context.Context) (int64, error)
}

// Clock abstracts time.Now for testability.
type Clock interface {
	Now() time.Time
}

// Config wires the authorization service.
type Config struct {
	BalanceReader   SubjectBalanceReader
	Subscription    SubscriptionReader
	RatePackage     RatePackageReader
	CoveredSeq      CoveredSeqReader
	SnapshotVersion SnapshotVersionProvider
	Signer          signing.Signer
	Clock           Clock
	Namespace       string
	Logger          *slog.Logger
	Tracer          trace.Tracer
}

func (c Config) validate() error {
	if c.BalanceReader == nil {
		return fmt.Errorf("runtimeauthorization: balance reader is required")
	}
	if c.Subscription == nil {
		return fmt.Errorf("runtimeauthorization: subscription reader is required")
	}
	if c.RatePackage == nil {
		return fmt.Errorf("runtimeauthorization: rate package reader is required")
	}
	if c.CoveredSeq == nil {
		return fmt.Errorf("runtimeauthorization: covered seq reader is required")
	}
	if c.SnapshotVersion == nil {
		return fmt.Errorf("runtimeauthorization: snapshot version provider is required")
	}
	if c.Signer == nil {
		return fmt.Errorf("runtimeauthorization: signer is required")
	}
	return nil
}

// Service generates signed runtime authorization packages.
type Service interface {
	Get(ctx context.Context, customerID string, subjectKeys []string) (signing.AuthorizationPackage, error)
}

type service struct {
	balance      SubjectBalanceReader
	subscription SubscriptionReader
	ratePackage  RatePackageReader
	coveredSeq   CoveredSeqReader
	snapshotVer  SnapshotVersionProvider
	signer       signing.Signer
	clock        Clock
	namespace    string
	logger       *slog.Logger
	tracer       trace.Tracer
}

// New creates an authorization Service from Config.
func New(cfg Config) (Service, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	clk := cfg.Clock
	if clk == nil {
		clk = systemClock{}
	}
	return &service{
		balance:      cfg.BalanceReader,
		subscription: cfg.Subscription,
		ratePackage:  cfg.RatePackage,
		coveredSeq:   cfg.CoveredSeq,
		snapshotVer:  cfg.SnapshotVersion,
		signer:       cfg.Signer,
		clock:        clk,
		namespace:    cfg.Namespace,
		logger:       logger,
		tracer:       cfg.Tracer,
	}, nil
}

// Get assembles the authorization package from the billing customer's current
// state, signs it, and returns it. The authorization_capacity_credits field is
// the sum of spendable prepaid Credit and available Enterprise credit.
func (s *service) Get(ctx context.Context, customerID string, subjectKeys []string) (signing.AuthorizationPackage, error) {
	ctx, span := s.startSpan(ctx, "runtimeauthorization.Get")
	defer span.End()

	if customerID == "" {
		return signing.AuthorizationPackage{}, fmt.Errorf("runtimeauthorization: customer_id must not be empty")
	}

	// Gather all inputs in parallel-safe sequence.
	balance, err := s.balance.ReadCreditBalance(ctx, s.namespace, customerID)
	if err != nil {
		return signing.AuthorizationPackage{}, fmt.Errorf("runtimeauthorization: read balance: %w", err)
	}

	sub, err := s.subscription.ReadSubscription(ctx, s.namespace, customerID)
	if err != nil {
		return signing.AuthorizationPackage{}, fmt.Errorf("runtimeauthorization: read subscription: %w", err)
	}

	ratePkg, err := s.ratePackage.ReadRatePackage(ctx, s.namespace, customerID)
	if err != nil {
		return signing.AuthorizationPackage{}, fmt.Errorf("runtimeauthorization: read rate package: %w", err)
	}

	covered, err := s.coveredSeq.ReadCoveredSeq(ctx, s.namespace, customerID)
	if err != nil {
		return signing.AuthorizationPackage{}, fmt.Errorf("runtimeauthorization: read covered seq: %w", err)
	}

	version, err := s.snapshotVer.Next(ctx)
	if err != nil {
		return signing.AuthorizationPackage{}, fmt.Errorf("runtimeauthorization: next snapshot version: %w", err)
	}

	now := s.clock.Now()

	pkg := signing.AuthorizationPackage{
		BillingCustomerID:            customerID,
		SubjectKeys:                  subjectKeys,
		PlanCode:                     sub.PlanCode,
		SubscriptionCode:             sub.SubscriptionCode,
		SubscriptionStatus:           sub.SubscriptionStatus,
		EntitlementCodes:             sub.EntitlementCodes,
		SpendableCredits:             balance.SpendableCredits,
		EnterpriseAvailableCredits:   balance.EnterpriseAvailableCredits,
		AuthorizationCapacityCredits: balance.SpendableCredits + balance.EnterpriseAvailableCredits,
		CurrentPeriodStart:           sub.CurrentPeriodStart,
		CurrentPeriodEnd:             sub.CurrentPeriodEnd,
		SnapshotVersion:              version,
		CoveredTenantSeq:             covered,
		RatePackageVersion:           ratePkg.Version,
		RatePackage:                  ratePkg.Entries,
		ExpiresAt:                    now.Add(s.signer.TTL()),
	}

	signed, err := s.signer.Sign(pkg)
	if err != nil {
		return signing.AuthorizationPackage{}, fmt.Errorf("runtimeauthorization: sign package: %w", err)
	}

	return signed, nil
}

func (s *service) startSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	if s.tracer != nil {
		return s.tracer.Start(ctx, name)
	}
	return ctx, trace.SpanFromContext(ctx)
}

// systemClock implements Clock using time.Now.
type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

var _ Service = (*service)(nil)
