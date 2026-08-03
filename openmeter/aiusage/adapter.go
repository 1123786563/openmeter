package aiusage

import (
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

// ConnectorConfig holds the dependencies for the AI Usage connector.
type ConnectorConfig struct {
	Repo             Repository
	RateCardResolver RateCardResolver
	CostResolver     CostResolver
	SettlementEngine SettlementEngine
	Logger           *slog.Logger
	Tracer           trace.Tracer
}

// Connector provides the AI Usage service and its dependencies.
type Connector struct {
	Service Service
}

func NewConnector(cfg ConnectorConfig) *Connector {
	svc := NewService(ServiceConfig{
		Repo:             cfg.Repo,
		RateCardResolver: cfg.RateCardResolver,
		CostResolver:     cfg.CostResolver,
		SettlementEngine: cfg.SettlementEngine,
		Logger:           cfg.Logger,
		Tracer:           cfg.Tracer,
	})

	return &Connector{
		Service: svc,
	}
}
