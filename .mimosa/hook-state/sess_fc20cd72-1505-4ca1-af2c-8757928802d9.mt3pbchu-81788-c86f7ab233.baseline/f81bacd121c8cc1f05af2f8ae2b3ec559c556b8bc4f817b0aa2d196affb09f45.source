package service

import (
	"context"
	"errors"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"

	"github.com/openmeterio/openmeter/openmeter/creditlimit"
	"github.com/openmeterio/openmeter/openmeter/creditreservation"
	reservationadapter "github.com/openmeterio/openmeter/openmeter/creditreservation/adapter"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger/collector"
	"github.com/openmeterio/openmeter/pkg/models"
)

type collectableReader interface {
	GetCollectableAmount(context.Context, collector.GetCollectableAmountInput) (decimal.Decimal, error)
}

type Config struct {
	Adapter                  reservationadapter.Adapter
	Prices                   creditreservation.PriceResolver
	Collector                collectableReader
	SettlementCollector      settlementCollector
	CreditLimit              creditlimit.AllowanceResolver
	Now                      func() time.Time
	AuthorizationTTL         time.Duration
	ExecutionDeadline        time.Duration
	UnknownManualReviewAfter time.Duration
	Meter                    metric.Meter
}

func (c Config) Validate() error {
	if c.Adapter == nil || c.Prices == nil || c.Collector == nil {
		return errors.New("reservation adapter, price resolver, and collector are required")
	}
	return nil
}

type service struct {
	adapter                  reservationadapter.Adapter
	prices                   creditreservation.PriceResolver
	collector                collectableReader
	settlement               settlementCollector
	creditLimit              creditlimit.AllowanceResolver
	now                      func() time.Time
	authorizationTTL         time.Duration
	executionDeadline        time.Duration
	unknownManualReviewAfter time.Duration
	metrics                  lifecycleMetrics
}

var _ creditreservation.Service = (*service)(nil)

func New(config Config) (creditreservation.Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	now := config.Now
	if now == nil {
		now = time.Now
	}
	limit := config.CreditLimit
	if limit == nil {
		limit = creditlimit.NoopAllowanceResolver{}
	}
	unknownManualReviewAfter := config.UnknownManualReviewAfter
	if unknownManualReviewAfter <= 0 {
		unknownManualReviewAfter = time.Hour
	}
	authorizationTTL := config.AuthorizationTTL
	if authorizationTTL <= 0 {
		authorizationTTL = 5 * time.Minute
	}
	executionDeadline := config.ExecutionDeadline
	if executionDeadline <= 0 {
		executionDeadline = 10 * time.Minute
	}
	meter := config.Meter
	if meter == nil {
		meter = metricnoop.NewMeterProvider().Meter("openmeter.credit_reservation")
	}
	metrics, err := newLifecycleMetrics(meter)
	if err != nil {
		return nil, err
	}
	return &service{adapter: config.Adapter, prices: config.Prices, collector: config.Collector, settlement: config.SettlementCollector, creditLimit: limit, now: now, authorizationTTL: authorizationTTL, executionDeadline: executionDeadline, unknownManualReviewAfter: unknownManualReviewAfter, metrics: metrics}, nil
}

func (s *service) Get(ctx context.Context, id models.NamespacedID) (creditreservation.Reservation, error) {
	return s.adapter.GetReservation(ctx, id)
}

func (s *service) withReservation(ctx context.Context, id models.NamespacedID, fn func(reservationadapter.TxAdapter, creditreservation.Reservation) (creditreservation.Reservation, error)) (creditreservation.Reservation, error) {
	initial, err := s.adapter.GetReservation(ctx, id)
	if err != nil {
		return creditreservation.Reservation{}, err
	}
	var result creditreservation.Reservation
	err = s.adapter.WithCustomerLock(ctx, customer.CustomerID{Namespace: initial.Namespace, ID: initial.CustomerID}, func(tx reservationadapter.TxAdapter) error {
		current, err := tx.GetReservation(ctx, id)
		if err != nil {
			return err
		}
		result, err = fn(tx, current)
		return err
	})
	return result, err
}
