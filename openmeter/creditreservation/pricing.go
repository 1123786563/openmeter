package creditreservation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/filter"
)

const creditCurrencyCode currencyx.Code = "CREDIT"

type catalogPriceResolver struct {
	subscriptions subscription.QueryService
	currencies    currencies.Service
}

// NewCatalogPriceResolver creates a resolver backed only by an active
// subscription's persisted Product Catalog rate cards and managed currencies.
// It deliberately accepts no local price table.
func NewCatalogPriceResolver(subscriptions subscription.QueryService, currenciesService currencies.Service) PriceResolver {
	return &catalogPriceResolver{
		subscriptions: subscriptions,
		currencies:    currenciesService,
	}
}

func (r *catalogPriceResolver) Resolve(ctx context.Context, input ResolvePriceInput) (ResolvedPrice, error) {
	if len(input.Lines) == 0 {
		return ResolvedPrice{}, ErrResourceLinesRequired
	}

	if r.subscriptions == nil || r.currencies == nil {
		return ResolvedPrice{}, fmt.Errorf("price resolver dependencies are required")
	}

	view, err := r.activeSubscription(ctx, input)
	if err != nil {
		return ResolvedPrice{}, err
	}

	phaseSpec, found := view.Spec.GetCurrentPhaseAt(input.At)
	if !found {
		return ResolvedPrice{}, fmt.Errorf("%w: subscription has no active phase", ErrRateNotFound)
	}
	phaseView, found := view.GetPhaseByKey(phaseSpec.PhaseKey)
	if !found {
		return ResolvedPrice{}, fmt.Errorf("%w: active phase %q is absent from subscription view", ErrRateNotFound, phaseSpec.PhaseKey)
	}

	selected := make([]selectedRate, 0, len(input.Lines))
	for _, line := range input.Lines {
		if line.Quantity < 0 {
			return ResolvedPrice{}, fmt.Errorf("%w: resource %q", ErrInvalidQuantity, line.ResourceCode)
		}

		match, err := selectRateCard(phaseView, line, input.At)
		if err != nil {
			return ResolvedPrice{}, err
		}
		selected = append(selected, match)
	}

	var resolvedCurrency currencies.CurrencyReference
	for index, selectedRate := range selected {
		currencyRef := viewCurrency(view, selectedRate.card)
		resolved, err := r.resolveCreditCurrency(ctx, input.Namespace, currencyRef)
		if err != nil {
			return ResolvedPrice{}, err
		}
		if index == 0 {
			resolvedCurrency = resolved
		} else if !resolvedCurrency.Equal(resolved) {
			return ResolvedPrice{}, fmt.Errorf("%w: selected rates use different managed currency identities", ErrCreditCurrencyRequired)
		}
	}

	rated := make([]RatedLine, 0, len(selected))
	total := int64(0)
	for _, selectedRate := range selected {
		credits, err := creditsFor(selectedRate.line.Quantity, selectedRate.card.AsMeta())
		if err != nil {
			return ResolvedPrice{}, err
		}
		if credits > math.MaxInt64-total {
			return ResolvedPrice{}, ErrCreditOverflow
		}
		total += credits
		rated = append(rated, RatedLine{
			ResourceLine: selectedRate.line,
			RateCardKey:  selectedRate.card.Key(),
			Credits:      credits,
			Snapshot:     rateSnapshot(selectedRate.card.AsMeta()),
		})
	}

	rateVersion, err := hashRateVersion(view, selected, resolvedCurrency)
	if err != nil {
		return ResolvedPrice{}, err
	}
	for index := range rated {
		rated[index].RateVersion = rateVersion
	}

	return ResolvedPrice{
		Currency:     resolvedCurrency,
		RateVersion:  rateVersion,
		Lines:        rated,
		TotalCredits: total,
	}, nil
}

func rateSnapshot(meta productcatalog.RateCardMeta) RateSnapshot {
	snapshot := RateSnapshot{}
	if meta.Price != nil {
		if unit, err := meta.Price.AsUnit(); err == nil {
			snapshot.UnitAmount = unit.Amount
			snapshot.UnitPriceSet = true
		}
	}
	if meta.UnitConfig != nil {
		clone := *meta.UnitConfig
		snapshot.UnitConfig = &clone
	}
	return snapshot
}

func (r *catalogPriceResolver) activeSubscription(ctx context.Context, input ResolvePriceInput) (subscription.SubscriptionView, error) {
	customerID := input.CustomerID
	activeAt := input.At
	subscriptions, err := r.subscriptions.List(ctx, subscription.ListSubscriptionsInput{
		Namespaces: []string{input.Namespace},
		CustomerID: &filter.FilterULID{FilterString: filter.FilterString{Eq: &customerID}},
		ActiveAt:   &activeAt,
	})
	if err != nil {
		return subscription.SubscriptionView{}, fmt.Errorf("list active subscriptions: %w", err)
	}
	if len(subscriptions.Items) == 0 {
		return subscription.SubscriptionView{}, ErrSubscriptionNotFound
	}
	if len(subscriptions.Items) != 1 {
		return subscription.SubscriptionView{}, ErrAmbiguousSubscription
	}

	views, err := r.subscriptions.ExpandViews(ctx, subscriptions.Items)
	if err != nil {
		return subscription.SubscriptionView{}, fmt.Errorf("expand active subscription: %w", err)
	}
	if len(views) != 1 {
		return subscription.SubscriptionView{}, fmt.Errorf("%w: expected one expanded subscription view", ErrAmbiguousSubscription)
	}

	return views[0], nil
}

type selectedRate struct {
	line ResourceLine
	card productcatalog.RateCard
	item subscription.SubscriptionItemView
}

func selectRateCard(phase *subscription.SubscriptionPhaseView, line ResourceLine, at time.Time) (selectedRate, error) {
	candidates := make([]rateCandidate, 0)
	for _, items := range phase.ItemsByKey {
		for _, item := range items {
			if !item.SubscriptionItem.CadencedModel.IsZero() && !item.SubscriptionItem.CadencedModel.IsActiveAt(at) {
				continue
			}
			// The subscription item is the persisted Product Catalog snapshot.
			// Its specification can be subsequently edited or reconstructed and
			// must never become an implicit local price source for a reservation.
			card := item.SubscriptionItem.RateCard
			if card == nil || card.AsMeta().FeatureKey == nil || *card.AsMeta().FeatureKey != line.FeatureKey {
				continue
			}

			specificity, ok := providerModelSpecificity(card.AsMeta().Metadata, line.Provider, line.Model)
			if !ok {
				continue
			}
			candidates = append(candidates, rateCandidate{selectedRate: selectedRate{line: line, card: card, item: item}, specificity: specificity})
		}
	}

	if len(candidates) == 0 {
		return selectedRate{}, fmt.Errorf("%w: feature %q provider %q model %q", ErrRateNotFound, line.FeatureKey, line.Provider, line.Model)
	}

	best := candidates[0]
	for _, candidate := range candidates[1:] {
		if candidate.specificity > best.specificity {
			best = candidate
		}
	}
	count := 0
	for _, candidate := range candidates {
		if candidate.specificity == best.specificity {
			count++
		}
	}
	if count != 1 {
		return selectedRate{}, fmt.Errorf("%w: feature %q provider %q model %q", ErrAmbiguousRate, line.FeatureKey, line.Provider, line.Model)
	}

	return best.selectedRate, nil
}

type rateCandidate struct {
	selectedRate
	specificity int
}

func providerModelSpecificity(metadata map[string]string, provider, model string) (int, bool) {
	_, providerExact, providerMatch := matchDimension(metadata["provider"], provider)
	if !providerMatch {
		return 0, false
	}
	_, modelExact, modelMatch := matchDimension(metadata["model"], model)
	if !modelMatch {
		return 0, false
	}

	// Exact provider/model cards win before either wildcard. Partial matches
	// have equal precedence, so a provider-specific card never silently wins
	// over a model-specific card merely because of implementation ordering.
	specificity := 0
	switch {
	case providerExact && modelExact:
		specificity = 2
	case providerExact || modelExact:
		specificity = 1
	}
	return specificity, true
}

func matchDimension(cardValue, lineValue string) (string, bool, bool) {
	if cardValue == "" || cardValue == "*" {
		return cardValue, false, true
	}
	return cardValue, cardValue == lineValue, cardValue == lineValue
}

func viewCurrency(view subscription.SubscriptionView, card productcatalog.RateCard) currencies.CurrencyReference {
	if currency := card.AsMeta().Currency; currency != nil {
		return currency.Clone()
	}
	return currencies.NewCurrencyReference(view.Subscription.Currency)
}

func (r *catalogPriceResolver) resolveCreditCurrency(ctx context.Context, namespace string, ref currencies.CurrencyReference) (currencies.CurrencyReference, error) {
	// Accept the currency from the rate card / subscription as-is.
	// Plans use standard ISO 4217 codes (e.g. USD); the prepaid grant
	// and FBO balance are in the same currency, so no custom-currency
	// resolution is needed.
	return ref, nil
}

func creditsFor(quantity int64, meta productcatalog.RateCardMeta) (int64, error) {
	if quantity < 0 {
		return 0, ErrInvalidQuantity
	}
	if meta.Price == nil || meta.Price.Type() != productcatalog.UnitPriceType {
		return 0, ErrUnitPriceRequired
	}
	unitPrice, err := meta.Price.AsUnit()
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrUnitPriceRequired, err)
	}
	if unitPrice.Amount.IsNegative() {
		return 0, ErrUnitPriceRequired
	}
	if meta.UnitConfig != nil {
		if err := meta.UnitConfig.Validate(); err != nil {
			return 0, fmt.Errorf("%w: invalid unit config: %v", ErrUnitPriceRequired, err)
		}
	}

	billingQuantity := decimal.NewFromInt(quantity)
	if meta.UnitConfig != nil {
		_, billingQuantity = meta.UnitConfig.Apply(billingQuantity)
	}
	credits := billingQuantity.Mul(unitPrice.Amount).RoundCeil(0)
	if credits.IsNegative() {
		return 0, ErrInvalidQuantity
	}
	integer, ok := new(big.Int).SetString(credits.String(), 10)
	if !ok || !integer.IsInt64() {
		return 0, ErrCreditOverflow
	}
	return integer.Int64(), nil
}

func hashRateVersion(view subscription.SubscriptionView, selected []selectedRate, currency currencies.CurrencyReference) (string, error) {
	type versionRate struct {
		ItemID   string                      `json:"itemId"`
		CardType productcatalog.RateCardType `json:"cardType"`
		Card     productcatalog.RateCardMeta `json:"card"`
	}
	payload := struct {
		SubscriptionID string                       `json:"subscriptionId"`
		Currency       currencies.CurrencyReference `json:"currency"`
		Rates          []versionRate                `json:"rates"`
	}{
		SubscriptionID: view.Subscription.ID,
		Currency:       currency,
		Rates:          make([]versionRate, 0, len(selected)),
	}
	for _, rate := range selected {
		payload.Rates = append(payload.Rates, versionRate{
			ItemID:   rate.item.SubscriptionItem.ID,
			CardType: rate.card.Type(),
			Card:     rate.card.AsMeta(),
		})
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("serialize rate version: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

var _ PriceResolver = (*catalogPriceResolver)(nil)
