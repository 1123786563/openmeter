package chargeadapter_test

import (
	"context"
	"testing"

	"github.com/alpacahq/alpacadecimal"

	chargeflatfee "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargeusagebased "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/creditlimit"
)

type allowanceResolverForTest struct{ remaining *alpacadecimal.Decimal }

func (r allowanceResolverForTest) Remaining(context.Context, creditlimit.RemainingInput) (*alpacadecimal.Decimal, error) {
	return r.remaining, nil
}

func generousAllowanceForTest() creditlimit.AllowanceResolver {
	remaining := alpacadecimal.NewFromInt(1_000_000)
	return allowanceResolverForTest{remaining: &remaining}
}

func noActiveLimitForTest() creditlimit.AllowanceResolver { return creditlimit.NoopAllowanceResolver{} }

func editFlatFeeBaseIntentForTest(t testing.TB, charge *chargeflatfee.Charge, edit func(*chargeflatfee.Intent)) {
	t.Helper()

	intent := charge.Intent.GetBaseIntent()
	edit(&intent)
	charge.Intent = chargeflatfee.NewOverridableIntent(intent, charge.Intent.GetOverrideLayerMutableFields())
}

func editFlatFeeBaseLayerForTest(t testing.TB, charge *chargeflatfee.Charge, edit func(*chargeflatfee.IntentMutableFields)) {
	t.Helper()

	editFlatFeeBaseIntentForTest(t, charge, func(intent *chargeflatfee.Intent) {
		edit(&intent.IntentMutableFields)
	})
}

func editUsageBasedBaseIntentForTest(t testing.TB, charge *chargeusagebased.Charge, edit func(*chargeusagebased.Intent)) {
	t.Helper()

	intent := charge.Intent.GetBaseIntent()
	edit(&intent)
	charge.Intent = chargeusagebased.NewOverridableIntent(intent, charge.Intent.GetOverrideLayerMutableFields())
}

func editUsageBasedBaseLayerForTest(t testing.TB, charge *chargeusagebased.Charge, edit func(*chargeusagebased.IntentMutableFields)) {
	t.Helper()

	editUsageBasedBaseIntentForTest(t, charge, func(intent *chargeusagebased.Intent) {
		edit(&intent.IntentMutableFields)
	})
}

func setUsageBasedSubscriptionForTest(t testing.TB, charge *chargeusagebased.Charge, subscription meta.SubscriptionReference) {
	t.Helper()

	editUsageBasedBaseIntentForTest(t, charge, func(intent *chargeusagebased.Intent) {
		intent.Subscription = &subscription
	})
}
