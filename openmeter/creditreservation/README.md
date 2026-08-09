# Credit Reservation

This domain owns synchronous authorization state and command idempotency for
CREDIT-denominated resource calls. Product Catalog owns prices, Ledger owns
booked money, and this package owns only temporary holds and their lifecycle.

An ACTIVE authorization may expire before execution. Once EXECUTING, timeout
can only preserve the hold as UNKNOWN. No transition inferred from elapsed time
may book or release money after execution intent exists.

Every reservation and charge carries a `CommandIdentity`, binding an
idempotency key to a SHA-256 payload hash. Retries must present the same pair;
a changed payload must use a new key. An UNKNOWN reservation can settle or
release only through an evidence-bearing transition with the matching external
settlement or release reference; manual review remains the only evidence-free
recovery path.

`NewCatalogPriceResolver` reads the active subscription's persisted Product
Catalog rate cards. It accepts only the resolved, managed custom currency code
`CREDIT`, unit prices, non-negative quantities, and a unique rate-card match.
It applies `UnitConfig` before rounding each line's credits upward. It never
consults a local rate table.
