package service

import (
	"context"
	"errors"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"

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
	Adapter             reservationadapter.Adapter
	Prices              creditreservation.PriceResolver
	Collector           collectableReader
	SettlementCollector settlementCollector
	CreditLimit         creditlimit.AllowanceResolver
	Now                 func() time.Time
}

func (c Config) Validate() error {
	if c.Adapter == nil || c.Prices == nil || c.Collector == nil {
		return errors.New("reservation adapter, price resolver, and collector are required")
	}
	return nil
}

type service struct {
	adapter     reservationadapter.Adapter
	prices      creditreservation.PriceResolver
	collector   collectableReader
	settlement  settlementCollector
	creditLimit creditlimit.AllowanceResolver
	now         func() time.Time
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
	return &service{adapter: config.Adapter, prices: config.Prices, collector: config.Collector, settlement: config.SettlementCollector, creditLimit: limit, now: now}, nil
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
