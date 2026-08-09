package creditreservation_test

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/creditreservation"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	currencytestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// This catches fractional unit pricing being truncated instead of rounded up.
func TestCatalogPriceResolverUsesActiveSubscriptionCreditRateCard(t *testing.T) {
	resolver := newResolverFixture(t,
		unitRateCard("llm-input", "ai_usage", "CREDIT", "openai", "gpt-5", "0.002"),
	)

	result, err := resolver.Resolve(t.Context(), creditreservation.ResolvePriceInput{
		Namespace:  "ns",
		CustomerID: "cust",
		At:         time.Date(2026, 8, 9, 1, 0, 0, 0, time.UTC),
		Lines: []creditreservation.ResourceLine{{
			FeatureKey: "ai_usage", ResourceCode: "llm_input_tokens", Provider: "openai", Model: "gpt-5", Quantity: 1001,
		}},
	})
	require.NoError(t, err)
	require.Equal(t, int64(3), result.TotalCredits)
	require.NotEmpty(t, result.RateVersion)
	require.True(t, result.Currency.IsCustom())
	require.True(t, result.Currency.IsResolved())
}

// This catches accepting money as credits and arbitrary selection between equal matches.
func TestCatalogPriceResolverRejectsFiatAndAmbiguousMatches(t *testing.T) {
	_, err := resolverWithCurrency(t, "USD").Resolve(t.Context(), validResolveInput())
	require.ErrorIs(t, err, creditreservation.ErrCreditCurrencyRequired)

	_, err = resolverWithDuplicateExactCards(t).Resolve(t.Context(), validResolveInput())
	require.ErrorIs(t, err, creditreservation.ErrAmbiguousRate)
}

// This catches a wildcard card taking precedence over the exact provider/model card.
func TestCatalogPriceResolverPrefersExactProviderAndModelOverWildcard(t *testing.T) {
	resolver := newResolverFixture(t,
		unitRateCard("wildcard", "ai_usage", "CREDIT", "", "", "9"),
		unitRateCard("exact", "ai_usage", "CREDIT", "openai", "gpt-5", "1"),
	)

	result, err := resolver.Resolve(t.Context(), validResolveInput())
	require.NoError(t, err)
	require.Equal(t, "exact", result.Lines[0].RateCardKey)
	require.Equal(t, int64(1), result.TotalCredits)
}

// This catches applying ceil before, rather than after, Product Catalog's unit conversion.
func TestCatalogPriceResolverAppliesUnitConfigBeforeCeilingCredits(t *testing.T) {
	card := unitRateCard("converted", "ai_usage", "CREDIT", "openai", "gpt-5", "0.002")
	unitConfig := &productcatalog.UnitConfig{
		Operation:        productcatalog.UnitConfigOperationMultiply,
		ConversionFactor: decimal.RequireFromString("0.5"),
	}
	card.(*productcatalog.UsageBasedRateCard).UnitConfig = unitConfig

	result, err := newResolverFixture(t, card).Resolve(t.Context(), creditreservation.ResolvePriceInput{
		Namespace: "ns", CustomerID: "cust", At: validResolveInput().At,
		Lines: []creditreservation.ResourceLine{{FeatureKey: "ai_usage", Provider: "openai", Model: "gpt-5", Quantity: 1001}},
	})
	require.NoError(t, err)
	require.Equal(t, int64(2), result.TotalCredits)
}

func TestCatalogPriceResolverRejectsNegativeQuantityAndOverflow(t *testing.T) {
	resolver := newResolverFixture(t, unitRateCard("rate", "ai_usage", "CREDIT", "openai", "gpt-5", "2"))

	negative := validResolveInput()
	negative.Lines[0].Quantity = -1
	_, err := resolver.Resolve(t.Context(), negative)
	require.ErrorIs(t, err, creditreservation.ErrInvalidQuantity)

	overflow := validResolveInput()
	overflow.Lines[0].Quantity = math.MaxInt64
	_, err = resolver.Resolve(t.Context(), overflow)
	require.ErrorIs(t, err, creditreservation.ErrCreditOverflow)
}

func TestCatalogPriceResolverRejectsNonUnitPrice(t *testing.T) {
	featureKey := "ai_usage"
	currencyRef := currencies.NewCurrencyReference(currencyx.Code("CREDIT"))
	flatCard := &productcatalog.UsageBasedRateCard{RateCardMeta: productcatalog.RateCardMeta{
		Key:        "flat",
		FeatureKey: &featureKey,
		Metadata:   models.Metadata{"provider": "openai", "model": "gpt-5"},
		Currency:   &currencyRef,
		Price:      productcatalog.NewPriceFrom(productcatalog.FlatPrice{Amount: decimal.RequireFromString("1")}),
	}}

	_, err := newResolverFixture(t, flatCard).Resolve(t.Context(), validResolveInput())
	require.ErrorIs(t, err, creditreservation.ErrUnitPriceRequired)
}

// This catches falling back to the subscription specification instead of the
// persisted rate card snapshot held by the subscription item.
func TestCatalogPriceResolverRequiresPersistedSubscriptionItemRateCard(t *testing.T) {
	at := validResolveInput().At
	featureKey := "ai_usage"
	view := subscription.SubscriptionView{
		Subscription: subscription.Subscription{NamespacedID: models.NamespacedID{Namespace: "ns", ID: "sub-1"}, CustomerId: "cust"},
		Spec: subscription.SubscriptionSpec{
			CreateSubscriptionCustomerInput: subscription.CreateSubscriptionCustomerInput{CustomerId: "cust", ActiveFrom: at.Add(-time.Hour)},
			Phases: map[string]*subscription.SubscriptionPhaseSpec{
				"phase": {CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{PhaseKey: "phase"}},
			},
		},
		Phases: []subscription.SubscriptionPhaseView{{
			SubscriptionPhase: subscription.SubscriptionPhase{Key: "phase"},
			ItemsByKey: map[string][]subscription.SubscriptionItemView{"usage": {{
				SubscriptionItem: subscription.SubscriptionItem{NamespacedID: models.NamespacedID{Namespace: "ns", ID: "item-a"}},
				Spec: subscription.SubscriptionItemSpec{CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
					CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{RateCard: unitRateCard("spec-only", featureKey, "CREDIT", "openai", "gpt-5", "1")},
				}},
			}}},
		}},
	}
	credit := currencytestutils.NewManagedCurrency(t, "ns", "credit-currency", "CREDIT")
	resolver := creditreservation.NewCatalogPriceResolver(
		&subscriptionQueryStub{views: []subscription.SubscriptionView{view}},
		&currencyServiceStub{currencies: []currencies.Currency{credit}},
	)

	_, err := resolver.Resolve(t.Context(), validResolveInput())
	require.ErrorIs(t, err, creditreservation.ErrRateNotFound)
}

func validResolveInput() creditreservation.ResolvePriceInput {
	return creditreservation.ResolvePriceInput{
		Namespace: "ns", CustomerID: "cust", At: time.Date(2026, 8, 9, 1, 0, 0, 0, time.UTC),
		Lines: []creditreservation.ResourceLine{{
			FeatureKey: "ai_usage", ResourceCode: "llm_input_tokens", Provider: "openai", Model: "gpt-5", Quantity: 1,
		}},
	}
}

func resolverWithCurrency(t *testing.T, currency string) creditreservation.PriceResolver {
	t.Helper()
	return newResolverFixture(t, unitRateCard("llm-input", "ai_usage", currency, "openai", "gpt-5", "1"))
}

func resolverWithDuplicateExactCards(t *testing.T) creditreservation.PriceResolver {
	t.Helper()
	return newResolverFixture(t,
		unitRateCard("llm-input-a", "ai_usage", "CREDIT", "openai", "gpt-5", "1"),
		unitRateCard("llm-input-b", "ai_usage", "CREDIT", "openai", "gpt-5", "1"),
	)
}

func newResolverFixture(t *testing.T, cards ...productcatalog.RateCard) creditreservation.PriceResolver {
	t.Helper()

	at := time.Date(2026, 8, 9, 1, 0, 0, 0, time.UTC)
	items := make([]subscription.SubscriptionItemView, 0, len(cards))
	for index, card := range cards {
		items = append(items, subscription.SubscriptionItemView{
			SubscriptionItem: subscription.SubscriptionItem{
				NamespacedID: models.NamespacedID{Namespace: "ns", ID: "item-" + string(rune('a'+index))},
				RateCard:     card,
			},
			Spec: subscription.SubscriptionItemSpec{
				CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
					CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{RateCard: card},
				},
			},
		})
	}

	view := subscription.SubscriptionView{
		Subscription: subscription.Subscription{NamespacedID: models.NamespacedID{Namespace: "ns", ID: "sub-1"}, CustomerId: "cust"},
		Spec: subscription.SubscriptionSpec{
			CreateSubscriptionCustomerInput: subscription.CreateSubscriptionCustomerInput{CustomerId: "cust", ActiveFrom: at.Add(-time.Hour)},
			Phases: map[string]*subscription.SubscriptionPhaseSpec{
				"phase": {CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{PhaseKey: "phase"}},
			},
		},
		Phases: []subscription.SubscriptionPhaseView{{
			SubscriptionPhase: subscription.SubscriptionPhase{Key: "phase"},
			ItemsByKey:        map[string][]subscription.SubscriptionItemView{"usage": items},
		}},
	}

	credit := currencytestutils.NewManagedCurrency(t, "ns", "credit-currency", "CREDIT")
	return creditreservation.NewCatalogPriceResolver(
		&subscriptionQueryStub{views: []subscription.SubscriptionView{view}},
		&currencyServiceStub{currencies: []currencies.Currency{credit}},
	)
}

func unitRateCard(key, featureKey, currency, provider, model, amount string) productcatalog.RateCard {
	currencyRef := currencies.NewCurrencyReference(currencyx.Code(currency))
	return &productcatalog.UsageBasedRateCard{RateCardMeta: productcatalog.RateCardMeta{
		Key:        key,
		FeatureKey: &featureKey,
		Metadata:   models.Metadata{"provider": provider, "model": model},
		Currency:   &currencyRef,
		Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: decimal.RequireFromString(amount),
		}),
	}}
}

type subscriptionQueryStub struct {
	views []subscription.SubscriptionView
}

func (s *subscriptionQueryStub) Get(context.Context, models.NamespacedID) (subscription.Subscription, error) {
	return subscription.Subscription{}, errors.New("not implemented")
}

func (s *subscriptionQueryStub) GetView(context.Context, models.NamespacedID) (subscription.SubscriptionView, error) {
	return subscription.SubscriptionView{}, errors.New("not implemented")
}

func (s *subscriptionQueryStub) List(context.Context, subscription.ListSubscriptionsInput) (subscription.SubscriptionList, error) {
	items := make([]subscription.Subscription, 0, len(s.views))
	for _, view := range s.views {
		items = append(items, view.Subscription)
	}
	return pagination.Result[subscription.Subscription]{Items: items, TotalCount: len(items)}, nil
}

func (s *subscriptionQueryStub) ExpandViews(context.Context, []subscription.Subscription) ([]subscription.SubscriptionView, error) {
	return s.views, nil
}

type currencyServiceStub struct {
	currencies.Service
	currencies []currencies.Currency
}

func (s *currencyServiceStub) ListCurrencies(context.Context, currencies.ListCurrenciesInput) (pagination.Result[currencies.Currency], error) {
	return pagination.Result[currencies.Currency]{Items: s.currencies, TotalCount: len(s.currencies)}, nil
}
