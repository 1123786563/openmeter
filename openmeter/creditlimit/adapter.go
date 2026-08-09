package creditlimit

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/customercreditlimit"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Adapter interface {
	GetActive(context.Context, GetActiveInput) (*Limit, error)
	Create(context.Context, CreateInput) (*Limit, error)
	Disable(context.Context, models.NamespacedID) error
}

type adapter struct{ db *entdb.Client }

func NewAdapter(db *entdb.Client) Adapter { return &adapter{db: db} }

func (a *adapter) GetActive(ctx context.Context, input GetActiveInput) (*Limit, error) {
	currency, err := input.Currency.MarshalText()
	if err != nil {
		return nil, fmt.Errorf("serialize currency: %w", err)
	}
	row, err := a.db.CustomerCreditLimit.Query().
		Where(
			customercreditlimit.NamespaceEQ(input.Namespace),
			customercreditlimit.CustomerIDEQ(input.CustomerID),
			customercreditlimit.CurrencyEQ(string(currency)),
			customercreditlimit.Enabled(true),
			customercreditlimit.DeletedAtIsNil(),
			customercreditlimit.EffectiveFromLTE(input.AsOf),
			customercreditlimit.Or(customercreditlimit.EffectiveToIsNil(), customercreditlimit.EffectiveToGT(input.AsOf)),
		).
		Order(customercreditlimit.ByEffectiveFrom(sql.OrderDesc())).
		First(ctx)
	if entdb.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get active credit limit: %w", err)
	}
	return mapLimit(row)
}

func (a *adapter) Create(ctx context.Context, input CreateInput) (*Limit, error) {
	currency, err := input.Currency.MarshalText()
	if err != nil {
		return nil, fmt.Errorf("serialize currency: %w", err)
	}
	overlaps, err := a.db.CustomerCreditLimit.Query().Where(
		customercreditlimit.NamespaceEQ(input.Namespace),
		customercreditlimit.CustomerIDEQ(input.CustomerID),
		customercreditlimit.CurrencyEQ(string(currency)),
		customercreditlimit.Enabled(true),
		customercreditlimit.DeletedAtIsNil(),
		customercreditlimit.Or(customercreditlimit.EffectiveToIsNil(), customercreditlimit.EffectiveToGT(input.EffectiveFrom)),
	).Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("check overlapping credit limit: %w", err)
	}
	if overlaps {
		// Existing end > new start is necessary. When this new interval has an
		// end, only existing starts before it overlap; the query below checks
		// that additional half of the interval intersection.
		if input.EffectiveTo == nil {
			return nil, fmt.Errorf("an overlapping active credit limit already exists for customer and currency")
		}
		overlaps, err = a.db.CustomerCreditLimit.Query().Where(
			customercreditlimit.NamespaceEQ(input.Namespace),
			customercreditlimit.CustomerIDEQ(input.CustomerID),
			customercreditlimit.CurrencyEQ(string(currency)),
			customercreditlimit.Enabled(true),
			customercreditlimit.DeletedAtIsNil(),
			customercreditlimit.EffectiveFromLT(*input.EffectiveTo),
			customercreditlimit.Or(customercreditlimit.EffectiveToIsNil(), customercreditlimit.EffectiveToGT(input.EffectiveFrom)),
		).Exist(ctx)
		if err != nil {
			return nil, fmt.Errorf("check overlapping credit limit: %w", err)
		}
		if overlaps {
			return nil, fmt.Errorf("an overlapping active credit limit already exists for customer and currency")
		}
	}
	row, err := a.db.CustomerCreditLimit.Create().
		SetNamespace(input.Namespace).
		SetCustomerID(input.CustomerID).
		SetCurrency(string(currency)).
		SetCustomCurrencyID(*input.Currency.CustomCurrencyID).
		SetAmount(input.Amount).
		SetEffectiveFrom(input.EffectiveFrom).
		SetNillableEffectiveTo(input.EffectiveTo).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create credit limit: %w", err)
	}
	return mapLimit(row)
}

func (a *adapter) Disable(ctx context.Context, id models.NamespacedID) error {
	_, err := a.db.CustomerCreditLimit.Update().
		Where(customercreditlimit.NamespaceEQ(id.Namespace), customercreditlimit.IDEQ(id.ID), customercreditlimit.DeletedAtIsNil()).
		SetEnabled(false).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("disable credit limit: %w", err)
	}
	return nil
}

func mapLimit(row *entdb.CustomerCreditLimit) (*Limit, error) {
	currency, err := currencies.ParseCurrencyReference([]byte(row.Currency))
	if err != nil {
		return nil, fmt.Errorf("parse stored currency reference: %w", err)
	}
	return &Limit{ID: row.ID, Namespace: row.Namespace, CustomerID: row.CustomerID, Currency: currency, Amount: row.Amount, EffectiveFrom: row.EffectiveFrom, EffectiveTo: row.EffectiveTo, Enabled: row.Enabled}, nil
}
