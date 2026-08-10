// Package runtime assembles the reservation lifecycle from explicit production
// dependencies. The application container must construct this bundle before it
// enables the HTTP gate; there is intentionally no nil or noop fallback.
package runtime

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/creditlimit"
	"github.com/openmeterio/openmeter/openmeter/creditreservation"
	reservationadapter "github.com/openmeterio/openmeter/openmeter/creditreservation/adapter"
	reservationservice "github.com/openmeterio/openmeter/openmeter/creditreservation/service"
	"github.com/openmeterio/openmeter/openmeter/creditreservation/worker"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ingest"
	ledgercollector "github.com/openmeterio/openmeter/openmeter/ledger/collector"
	"github.com/openmeterio/openmeter/openmeter/subscription"
)

type Config struct {
	Configuration config.CreditReservationConfiguration
	Client        *entdb.Client
	Logger        *slog.Logger
	Subscriptions subscription.QueryService
	Currencies    currencies.Service
	Collector     ledgercollector.Service
	CreditLimit   creditlimit.AllowanceResolver
	Ingest        ingest.Collector
	OwnerID       string
	Meter         metric.Meter
}

type Bundle struct {
	Service       creditreservation.Service
	OutboxWorker  *worker.Worker
	SweepInterval time.Duration
}

func New(cfg Config) (Bundle, error) {
	if !cfg.Configuration.Enabled {
		return Bundle{}, errors.New("credit reservation runtime is disabled")
	}
	if err := cfg.Configuration.Validate(); err != nil {
		return Bundle{}, fmt.Errorf("validate credit reservation configuration: %w", err)
	}
	if cfg.Client == nil || cfg.Logger == nil || cfg.Subscriptions == nil || cfg.Currencies == nil || cfg.Collector == nil || cfg.Ingest == nil || cfg.OwnerID == "" {
		return Bundle{}, errors.New("credit reservation runtime requires client, logger, subscriptions, currencies, collector, ingest, and owner id")
	}

	pollInterval, err := time.ParseDuration(cfg.Configuration.Worker.PollInterval)
	if err != nil {
		return Bundle{}, fmt.Errorf("parse credit reservation poll interval: %w", err)
	}
	leaseDuration, err := time.ParseDuration(cfg.Configuration.Worker.LeaseDuration)
	if err != nil {
		return Bundle{}, fmt.Errorf("parse credit reservation lease duration: %w", err)
	}
	unknownManualReviewAfter, err := time.ParseDuration(cfg.Configuration.UnknownManualReviewAfter)
	if err != nil {
		return Bundle{}, fmt.Errorf("parse credit reservation unknown review duration: %w", err)
	}
	authorizationTTL, err := time.ParseDuration(cfg.Configuration.AuthorizationTTL)
	if err != nil {
		return Bundle{}, fmt.Errorf("parse credit reservation authorization ttl: %w", err)
	}
	executionDeadline, err := time.ParseDuration(cfg.Configuration.ExecutionDeadline)
	if err != nil {
		return Bundle{}, fmt.Errorf("parse credit reservation execution deadline: %w", err)
	}

	adapter, err := reservationadapter.New(reservationadapter.Config{Client: cfg.Client, Logger: cfg.Logger})
	if err != nil {
		return Bundle{}, fmt.Errorf("create credit reservation adapter: %w", err)
	}
	service, err := reservationservice.New(reservationservice.Config{
		Adapter:                  adapter,
		Prices:                   creditreservation.NewCatalogPriceResolver(cfg.Subscriptions, cfg.Currencies),
		Collector:                cfg.Collector,
		SettlementCollector:      cfg.Collector,
		CreditLimit:              cfg.CreditLimit,
		AuthorizationTTL:         authorizationTTL,
		ExecutionDeadline:        executionDeadline,
		UnknownManualReviewAfter: unknownManualReviewAfter,
		Meter:                    cfg.Meter,
	})
	if err != nil {
		return Bundle{}, fmt.Errorf("create credit reservation service: %w", err)
	}
	repo, err := worker.NewEntRepository(cfg.Client)
	if err != nil {
		return Bundle{}, err
	}
	outboxWorker, err := worker.New(worker.Config{Repo: repo, Collector: cfg.Ingest, OwnerID: cfg.OwnerID, BatchSize: cfg.Configuration.Worker.BatchSize, LeaseDuration: leaseDuration, MaxClaimCount: cfg.Configuration.Worker.MaxClaimCount, Meter: cfg.Meter})
	if err != nil {
		return Bundle{}, fmt.Errorf("create credit reservation outbox worker: %w", err)
	}

	return Bundle{Service: service, OutboxWorker: outboxWorker, SweepInterval: pollInterval}, nil
}
