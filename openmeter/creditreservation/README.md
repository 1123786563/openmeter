# Credit Reservation

This domain owns synchronous authorization state and command idempotency for
CREDIT-denominated resource calls. Product Catalog owns prices, Ledger owns
booked money, and this package owns only temporary holds and their lifecycle.

An ACTIVE authorization may expire before execution. Once EXECUTING, timeout
can only preserve the hold as UNKNOWN. No transition inferred from elapsed time
may book or release money after execution intent exists.

`NewCatalogPriceResolver` reads the active subscription's persisted Product
Catalog rate cards. It accepts only the resolved, managed custom currency code
`CREDIT`, unit prices, non-negative quantities, and a unique rate-card match.
It applies `UnitConfig` before rounding each line's credits upward. It never
consults a local rate table.
