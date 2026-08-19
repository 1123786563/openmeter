package subscription

import "github.com/openmeterio/openmeter/pkg/currencyx"

// CreditCurrencyCode is the internal custom currency credit reservations are
// priced in. Rate cards denominated in this currency may be materialized on
// subscriptions (see validateSubscriptionCurrencySupport); any other custom
// currency is still rejected by the temporary billing boundary.
//
// The value must stay in sync with creditreservation.creditCurrencyCode.
const CreditCurrencyCode = currencyx.Code("CREDIT")
