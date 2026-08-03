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
	"github.com/openmeterio/openmeter/openmeter/aiusage/runtimeauthorization"
	"github.com/openmeterio/openmeter/openmeter/aiusage/signing"
	"github.com/openmeterio/openmeter/openmeter/aiusage/worker"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ledger"
)

// AIUsage is the wire provider set for the AI Usage subsystem.
var AIUsage = wire.NewSet(
	NewAIUsageConfig,
	NewAIUsageRepository,
	NewAIUsageService,
	NewAIUsageSigner,
	NewRuntimeAuthorizationService,
	NewAIUsageWorker,
)

// NewAIUsageConfig extracts the AIUsageConfiguration from the top-level Configuration.
func NewAIUsageConfig(conf config.Configuration) config.AIUsageConfiguration {
	return conf.AIUsage
}

// AIUsageRegistry bundles the AI Usage services consumed by the server and
// worker layers.
type AIUsageRegistry struct {
	Service                     aiusage.Service
	Repository                  aiusage.Repository
	RuntimeAuthorizationService runtimeauthorization.Service
	Signer                      signing.Signer
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

// NewAIUsageService creates the AI Usage settlement service from the repository
// and existing credit stack. When ai_usage is disabled it returns nil.
func NewAIUsageService(
	aiUsageConfig config.AIUsageConfiguration,
	repo aiusage.Repository,
	logger *slog.Logger,
	tracer trace.Tracer,
	ledgerSvc ledger.Ledger,
) (aiusage.Service, error) {
	if !aiUsageConfig.Enabled {
		return nil, nil
	}
	if repo == nil {
		return nil, fmt.Errorf("ai_usage: repository is required when ai_usage is enabled")
	}
	if ledgerSvc == nil {
		return nil, fmt.Errorf("ai_usage: credit stack (ledger) is required when ai_usage is enabled")
	}

	// Validate that the signing key is present — the process must refuse
	// ai_usage.enabled=true without it.
	if aiUsageConfig.Signing.CurrentKeyID == "" {
		return nil, fmt.Errorf("ai_usage: signing.current_key_id is required when ai_usage is enabled")
	}

	// Phase 1 uses noop rate-card, cost, and settlement implementations until
	// the pricing, llmcost, and collector services are wired into the DI graph.
	return aiusage.NewService(aiusage.ServiceConfig{
		Repo:             repo,
		RateCardResolver: noopRateCardResolver{},
		CostResolver:     noopCostResolver{},
		SettlementEngine: &noopSettlementEngine{logger: logger},
		Logger:           logger,
		Tracer:           tracer,
	}), nil
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
// It wires the signer and no-op readers; production wiring supplies real
// balance/subscription/rate-package readers.
func NewRuntimeAuthorizationService(
	aiUsageConfig config.AIUsageConfiguration,
	signer signing.Signer,
	ledgerSvc ledger.Ledger,
	logger *slog.Logger,
	tracer trace.Tracer,
) (runtimeauthorization.Service, error) {
	if !aiUsageConfig.Enabled || signer == nil {
		return nil, nil
	}
	if ledgerSvc == nil {
		return nil, fmt.Errorf("ai_usage: credit stack (ledger) is required when ai_usage is enabled")
	}

	return runtimeauthorization.New(runtimeauthorization.Config{
		BalanceReader:   noopBalanceReader{},
		Subscription:    noopSubscriptionReader{},
		RatePackage:     noopRatePackageReader{},
		CoveredSeq:      noopCoveredSeqReader{},
		SnapshotVersion: &atomicSnapshotVersionProvider{},
		Signer:          signer,
		Logger:          logger,
		Tracer:          tracer,
	})
}

// NewAIUsageWorker constructs the outbox relay worker. When ai_usage is
// disabled or the outbox repository is not yet wired, it returns nil.
//
// The worker uses a no-op projection until Kafka/ClickHouse projections are
// wired in the full DI graph. This keeps the lifecycle correct: the worker
// starts, polls, finds zero unpublished rows, and shuts down within the
// 5-second grace period on cancellation.
func NewAIUsageWorker(
	aiUsageConfig config.AIUsageConfiguration,
) (*worker.Worker, error) {
	if !aiUsageConfig.Enabled {
		return nil, nil
	}

	// The worker requires at least one projection and an outbox repository.
	// Until the full DI graph supplies a real Kafka/ClickHouse projection and
	// ent-backed outbox repo, we return nil so the lifecycle does not start a
	// worker that would fail validation.
	// Production wiring will pass a real repo and projections here.
	return nil, nil
}

// AIUsageWorkerGroup builds the execute/intercept pair for the run.Group
// lifecycle, matching the pattern used by kafkaingest.KafkaProducerGroup.
// When the worker is nil (not yet wired or disabled), the execute function
// blocks on the context and the intercept function is a no-op, so the
// lifecycle is always correct.
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
