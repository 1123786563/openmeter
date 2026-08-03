package common

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/aiusage"
	aiusageadapter "github.com/openmeterio/openmeter/openmeter/aiusage/adapter"
	"github.com/openmeterio/openmeter/openmeter/aiusage/pricing"
	"github.com/openmeterio/openmeter/openmeter/aiusage/runtimeauthorization"
	aiusageservice "github.com/openmeterio/openmeter/openmeter/aiusage/service"
	"github.com/openmeterio/openmeter/openmeter/aiusage/settlement"
	"github.com/openmeterio/openmeter/openmeter/aiusage/signing"
	"github.com/openmeterio/openmeter/openmeter/aiusage/worker"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/account"
	ledgercollector "github.com/openmeterio/openmeter/openmeter/ledger/collector"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
)

// AIUsage is the wire provider set for the AI Usage subsystem.
var AIUsage = wire.NewSet(
	NewAIUsageConfig,
	NewAIUsageRepository,
	NewAIUsageAdapter,
	NewAIUsagePricingResolver,
	NewAIUsageProfileResolver,
	NewAIUsageAllocationFetcher,
	NewAIUsageCollector,
	NewAIUsageSettlementService,
	NewAIUsageAppService,
	NewAIUsageServiceAdapter,
	NewAIUsageSigner,
	NewRuntimeAuthorizationService,
	NewAIUsageWorker,
)

// NewAIUsageConfig extracts the AIUsageConfiguration from the top-level Configuration.
func NewAIUsageConfig(conf config.Configuration) config.AIUsageConfiguration {
	return conf.AIUsage
}

// NewAIUsageRepository creates the ent-backed repository for AI Usage batches.
func NewAIUsageRepository(
	aiUsageConfig config.AIUsageConfiguration,
	db *entdb.Client,
	logger *slog.Logger,
) (aiusage.Repository, error) {
	if !aiUsageConfig.Enabled {
		return nil, nil
	}
	if db == nil {
		return nil, fmt.Errorf("ai_usage: postgresql adapter (ent client) is required when ai_usage is enabled")
	}
	return aiusageadapter.NewRepository(db, logger), nil
}

// NewAIUsageAdapter creates the persistence adapter with customer-locked
// transaction support used by the application service.
func NewAIUsageAdapter(
	aiUsageConfig config.AIUsageConfiguration,
	db *entdb.Client,
	logger *slog.Logger,
) (aiusageadapter.Adapter, error) {
	if !aiUsageConfig.Enabled {
		return nil, nil
	}
	if db == nil {
		return nil, fmt.Errorf("ai_usage: ent client is required when ai_usage is enabled")
	}
	return aiusageadapter.New(aiusageadapter.Config{Client: db, Logger: logger})
}

// NewAIUsagePricingResolver creates the pricing service backed by config rate
// entries.
func NewAIUsagePricingResolver(
	aiUsageConfig config.AIUsageConfiguration,
) (aiusageservice.PricingResolver, error) {
	if !aiUsageConfig.Enabled {
		return nil, nil
	}
	provider, err := newConfigRateEntryProvider(aiUsageConfig)
	if err != nil {
		return nil, fmt.Errorf("ai_usage: rate entry provider: %w", err)
	}
	return pricing.NewService(provider), nil
}

// NewAIUsageProfileResolver creates the customer profile resolver backed by
// config defaults.
func NewAIUsageProfileResolver(
	aiUsageConfig config.AIUsageConfiguration,
) (aiusageservice.CustomerProfileResolver, error) {
	if !aiUsageConfig.Enabled {
		return nil, nil
	}
	return newConfigCustomerProfileResolver(aiUsageConfig)
}

// NewAIUsageAllocationFetcher creates the ent-backed allocation fetcher used
// by the Correct flow.
func NewAIUsageAllocationFetcher(
	aiUsageConfig config.AIUsageConfiguration,
	db *entdb.Client,
) (aiusageservice.AllocationFetcher, error) {
	if !aiUsageConfig.Enabled {
		return nil, nil
	}
	if db == nil {
		return nil, fmt.Errorf("ai_usage: ent client is required when ai_usage is enabled")
	}
	return &dbAllocationFetcher{db: db}, nil
}

// NewAIUsageCollector creates the ledger collector service used by the
// settlement service for credit source selection and ledger commit.
func NewAIUsageCollector(
	aiUsageConfig config.AIUsageConfiguration,
	db *entdb.Client,
	ledgerSvc ledger.Ledger,
	balanceQuerier ledger.BalanceQuerier,
	accountResolver ledger.AccountResolver,
	accountSvc account.Service,
) (ledgercollector.Service, error) {
	if !aiUsageConfig.Enabled {
		return nil, nil
	}
	collector, err := ledgercollector.NewService(ledgercollector.Config{
		Ledger: ledgerSvc,
		Dependencies: transactions.ResolverDependencies{
			AccountService: accountResolver,
			AccountCatalog: accountSvc,
			BalanceQuerier: balanceQuerier,
		},
		AccountLocker:      accountSvc,
		TransactionManager: enttx.NewCreator(db),
	})
	if err != nil {
		return nil, fmt.Errorf("ai_usage: collector service: %w", err)
	}
	return collector, nil
}

// NewAIUsageSettlementService creates the settlement service that delegates
// credit allocation to the ledger collector.
func NewAIUsageSettlementService(
	aiUsageConfig config.AIUsageConfiguration,
	collector ledgercollector.Service,
	logger *slog.Logger,
	tracer trace.Tracer,
) (settlement.Service, error) {
	if !aiUsageConfig.Enabled {
		return nil, nil
	}
	if collector == nil {
		return nil, fmt.Errorf("ai_usage: collector service is required when ai_usage is enabled")
	}
	return settlement.New(settlement.ServiceConfig{
		Collector: collector,
		Logger:    logger,
		Tracer:    tracer,
	}), nil
}

// NewAIUsageAppService creates the application-level service that coordinates
// validate → price → settle → persist in one atomic customer-locked
// transaction.
func NewAIUsageAppService(
	aiUsageConfig config.AIUsageConfiguration,
	adp aiusageadapter.Adapter,
	pricingResolver aiusageservice.PricingResolver,
	settlementSvc settlement.Service,
	profileResolver aiusageservice.CustomerProfileResolver,
	allocFetcher aiusageservice.AllocationFetcher,
	logger *slog.Logger,
	tracer trace.Tracer,
) (aiusageservice.Service, error) {
	if !aiUsageConfig.Enabled {
		return nil, nil
	}
	if adp == nil {
		return nil, fmt.Errorf("ai_usage: persistence adapter is required when ai_usage is enabled")
	}
	if pricingResolver == nil {
		return nil, fmt.Errorf("ai_usage: pricing resolver is required when ai_usage is enabled")
	}
	if settlementSvc == nil {
		return nil, fmt.Errorf("ai_usage: settlement service is required when ai_usage is enabled")
	}
	if profileResolver == nil {
		return nil, fmt.Errorf("ai_usage: profile resolver is required when ai_usage is enabled")
	}
	if allocFetcher == nil {
		return nil, fmt.Errorf("ai_usage: allocation fetcher is required when ai_usage is enabled")
	}

	return aiusageservice.New(aiusageservice.Config{
		Adapter:           adp,
		Pricing:           pricingResolver,
		Settlement:        settlementSvc,
		ProfileResolver:   profileResolver,
		ScopeResolver:     nil, // nil = always formal scope
		AllocationFetcher: allocFetcher,
		Logger:            logger,
		Tracer:            tracer,
	}), nil
}

// NewAIUsageServiceAdapter wraps the application-level service.Service to
// implement the aiusage.Service interface consumed by the HTTP handler and
// Wire DI graph.
func NewAIUsageServiceAdapter(
	aiUsageConfig config.AIUsageConfiguration,
	appService aiusageservice.Service,
	repo aiusage.Repository,
) (aiusage.Service, error) {
	if !aiUsageConfig.Enabled {
		return nil, nil
	}
	if appService == nil {
		return nil, fmt.Errorf("ai_usage: application service is required when ai_usage is enabled")
	}
	if repo == nil {
		return nil, fmt.Errorf("ai_usage: repository is required when ai_usage is enabled")
	}
	return &aiusageServiceAdapter{appService: appService, repo: repo}, nil
}

// NewAIUsageSigner constructs the Ed25519 signer from the configured key material.
func NewAIUsageSigner(
	aiUsageConfig config.AIUsageConfiguration,
) (signing.Signer, error) {
	if !aiUsageConfig.Enabled {
		return nil, nil
	}

	seed, err := hex.DecodeString(aiUsageConfig.Signing.CurrentSeed)
	if err != nil {
		return nil, fmt.Errorf("ai_usage: decode signing seed: %w", err)
	}

	ttl := signing.DefaultTTL
	if aiUsageConfig.AuthorizationTTL != "" {
		parsed, err := time.ParseDuration(aiUsageConfig.AuthorizationTTL)
		if err != nil {
			return nil, fmt.Errorf("ai_usage: parse authorization_ttl: %w", err)
		}
		ttl = parsed
	}

	return signing.New(signing.Config{
		CurrentKey: signing.KeyPair{
			KeyID: aiUsageConfig.Signing.CurrentKeyID,
			Seed:  seed,
		},
		TTL: ttl,
	})
}

// NewRuntimeAuthorizationService constructs the runtime authorization service.
func NewRuntimeAuthorizationService(
	aiUsageConfig config.AIUsageConfiguration,
	signer signing.Signer,
	db *entdb.Client,
	logger *slog.Logger,
	tracer trace.Tracer,
) (runtimeauthorization.Service, error) {
	if !aiUsageConfig.Enabled || signer == nil {
		return nil, nil
	}

	return runtimeauthorization.New(runtimeauthorization.Config{
		BalanceReader:   noopBalanceReader{},
		Subscription:    noopSubscriptionReader{},
		RatePackage:     noopRatePackageReader{},
		CoveredSeq:      noopCoveredSeqReader{},
		SnapshotVersion: &dbSnapshotVersionProvider{db: db},
		Signer:          signer,
		Logger:          logger,
		Tracer:          tracer,
	})
}

// NewAIUsageWorker constructs the outbox relay worker with the ent-backed
// outbox repository and Kafka/ClickHouse projections.
func NewAIUsageWorker(
	aiUsageConfig config.AIUsageConfiguration,
	db *entdb.Client,
	logger *slog.Logger,
	tracer trace.Tracer,
) (*worker.Worker, error) {
	if !aiUsageConfig.Enabled {
		return nil, nil
	}
	if db == nil {
		return nil, fmt.Errorf("ai_usage: ent client is required when ai_usage is enabled")
	}

	repo := &entOutboxRepository{db: db}

	projections := []worker.Projection{
		&noopClickHouseProjection{logger: logger},
	}

	leaseDuration := 30 * time.Second
	if aiUsageConfig.Worker.LeaseDuration != "" {
		parsed, err := time.ParseDuration(aiUsageConfig.Worker.LeaseDuration)
		if err == nil {
			leaseDuration = parsed
		}
	}

	batchSize := aiUsageConfig.Worker.BatchSize
	if batchSize <= 0 {
		batchSize = 50
	}

	w, err := worker.New(worker.Config{
		Repo:          repo,
		Projections:   projections,
		OwnerID:       resolveWorkerOwnerID(),
		BatchSize:     batchSize,
		LeaseDuration: leaseDuration,
		Logger:        logger,
		Tracer:        tracer,
	})
	if err != nil {
		return nil, fmt.Errorf("ai_usage: worker: %w", err)
	}
	return w, nil
}

// AIUsageWorkerGroup builds the execute/intercept pair for the run.Group
// lifecycle. When the worker is nil (disabled), the execute function blocks
// on the context and the intercept function is a no-op.
func AIUsageWorkerGroup(
	ctx context.Context,
	w *worker.Worker,
) (func() error, func(error)) {
	workerRun := func() error {
		if w != nil {
			w.Start(ctx)
		}
		<-ctx.Done()
		return ctx.Err()
	}

	workerStop := func(_ error) {
		if w != nil {
			w.Stop()
		}
	}

	return workerRun, workerStop
}

// Compile-time interface checks.
var (
	_ aiusageservice.PricingResolver         = (*pricing.Service)(nil)
	_ aiusageservice.CustomerProfileResolver = (*configCustomerProfileResolver)(nil)
	_ aiusageservice.AllocationFetcher       = (*dbAllocationFetcher)(nil)
	_ worker.OutboxRepository                = (*entOutboxRepository)(nil)
)

