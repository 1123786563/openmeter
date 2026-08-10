# Enterprise credit limits

An enterprise credit limit is explicit policy for one managed custom currency,
customer, and time window. It has no mutable used balance: the ledger's open
customer receivable balance is authoritative. `Remaining` returns no value
when no active policy exists, so CreditOnly callers must reject a shortfall
instead of creating an unbounded receivable.
