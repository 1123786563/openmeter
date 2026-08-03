package aiusage

import (
	"log/slog"

	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

// AppConfig holds configuration for constructing the AI Usage app module.
type AppConfig struct {
	DB                *db.Client
	Logger            *slog.Logger
	Tracer            trace.Tracer
	RateCardResolver  RateCardResolver
	CostResolver      CostResolver
	SettlementEngine  SettlementEngine
	TransactionManager transaction.Creator
}

// App is the assembled AI Usage application module.
type App struct {
	Service Service
	Repo    Repository
}

func NewApp(cfg AppConfig) (*App, error) {
	return &App{
		Service: NewService(ServiceConfig{
			RateCardResolver: cfg.RateCardResolver,
			CostResolver:     cfg.CostResolver,
			SettlementEngine: cfg.SettlementEngine,
			Logger:           cfg.Logger,
			Tracer:           cfg.Tracer,
		}),
	}, nil
}
