package migrate_test

import (
	"database/sql"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestPaymentFactProviderFieldsMigrationBackfillsHistoricalFacts(t *testing.T) {
	orderID := ulid.Make().String()
	attemptID := ulid.Make().String()
	factID := ulid.Make().String()
	fallbackOrderID := ulid.Make().String()
	fallbackAttemptID := ulid.Make().String()
	fallbackFactID := ulid.Make().String()
	duplicateEventFactID := ulid.Make().String()

	runner{
		stops: stops{
			{
				version:   20260809000300,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					_, err := db.Exec(`
						INSERT INTO commerce_orders (
							id, namespace, created_at, updated_at, public_id, customer_id,
							kind, status, total_cents, currency, idempotency_key
						) VALUES ($1, 'default', NOW(), NOW(), $1, 'customer-1',
							'wallet_top_up', 'paid', 999, 'CNY', 'order-key')
					`, orderID)
					require.NoError(t, err)

					_, err = db.Exec(`
						INSERT INTO payment_attempts (
							id, namespace, created_at, updated_at, customer_id, provider,
							status, idempotency_key, amount_cents, currency, commerce_order_id
						) VALUES ($1, 'default', NOW(), NOW(), 'customer-1', 'wechat',
							'succeeded', 'attempt-key', 999, 'CNY', $2)
					`, attemptID, orderID)
					require.NoError(t, err)

					_, err = db.Exec(`
						INSERT INTO payment_facts (
							id, namespace, created_at, raw_hash, provider, signed_payload,
							timestamp, payment_attempt_id
						) VALUES (
							$1, 'default', NOW(), 'historical-raw-hash', 'wechat',
							'{
								"out_trade_no":"wx-order-1",
								"event_id":"wx-legacy-event-1",
								"transaction_id":"wx-payment-1",
								"mchid":"merchant-1",
								"appid":"application-1",
								"amount":{"total":1234,"currency":"CNY"},
								"trade_state":"SUCCESS"
							}'::jsonb,
							NOW(), $2
						)
					`, factID, attemptID)
					require.NoError(t, err)

					_, err = db.Exec(`
						INSERT INTO commerce_orders (
							id, namespace, created_at, updated_at, public_id, customer_id,
							kind, status, total_cents, currency, idempotency_key
						) VALUES ($1, 'default', NOW(), NOW(), $1, 'customer-2',
							'wallet_top_up', 'paid', 321, 'CNY', 'fallback-order-key')
					`, fallbackOrderID)
					require.NoError(t, err)

					_, err = db.Exec(`
						INSERT INTO payment_attempts (
							id, namespace, created_at, updated_at, customer_id, provider,
							status, idempotency_key, amount_cents, currency, commerce_order_id
						) VALUES ($1, 'default', NOW(), NOW(), 'customer-2', 'alipay',
							'succeeded', 'fallback-attempt-key', 321, 'CNY', $2)
					`, fallbackAttemptID, fallbackOrderID)
					require.NoError(t, err)

					_, err = db.Exec(`
						INSERT INTO payment_facts (
							id, namespace, created_at, raw_hash, provider, signed_payload,
							timestamp, payment_attempt_id
						) VALUES ($1, 'default', NOW(), 'fallback-raw-hash', 'alipay',
							'{}'::jsonb, NOW(), $2)
					`, fallbackFactID, fallbackAttemptID)
					require.NoError(t, err)
				},
			},
			{
				version:   20260809113159,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					var (
						providerOrderID   string
						providerPaymentID sql.NullString
						providerEventID   sql.NullString
						merchantID        sql.NullString
						applicationID     sql.NullString
						amountMinor       int64
						currency          string
						success           bool
					)

					err := db.QueryRow(`
						SELECT provider_order_id, provider_payment_id, provider_event_id,
							merchant_id, application_id, amount_minor, currency, success
						FROM payment_facts
						WHERE id = $1
					`, factID).Scan(
						&providerOrderID,
						&providerPaymentID,
						&providerEventID,
						&merchantID,
						&applicationID,
						&amountMinor,
						&currency,
						&success,
					)
					require.NoError(t, err)
					require.Equal(t, "wx-order-1", providerOrderID)
					require.Equal(t, "wx-payment-1", providerPaymentID.String)
					require.Equal(t, "wx-payment-1", providerEventID.String)
					require.Equal(t, "merchant-1", merchantID.String)
					require.Equal(t, "application-1", applicationID.String)
					require.Equal(t, int64(1234), amountMinor)
					require.Equal(t, "CNY", currency)
					require.True(t, success)

					var expectedMerchantID, expectedApplicationID sql.NullString
					err = db.QueryRow(`
						SELECT expected_merchant_id, expected_application_id
						FROM payment_attempts
						WHERE id = $1
					`, attemptID).Scan(&expectedMerchantID, &expectedApplicationID)
					require.NoError(t, err)
					require.Equal(t, "merchant-1", expectedMerchantID.String)
					require.Equal(t, "application-1", expectedApplicationID.String)

					var fallbackProviderOrderID, fallbackCurrency string
					var fallbackAmountMinor int64
					var fallbackSuccess bool
					err = db.QueryRow(`
						SELECT provider_order_id, amount_minor, currency, success
						FROM payment_facts
						WHERE id = $1
					`, fallbackFactID).Scan(
						&fallbackProviderOrderID,
						&fallbackAmountMinor,
						&fallbackCurrency,
						&fallbackSuccess,
					)
					require.NoError(t, err)
					require.Empty(t, fallbackProviderOrderID)
					require.Equal(t, int64(321), fallbackAmountMinor)
					require.Equal(t, "CNY", fallbackCurrency)
					require.False(t, fallbackSuccess)

					_, err = db.Exec(`
						INSERT INTO payment_facts (
							id, namespace, created_at, raw_hash, provider, signed_payload,
							timestamp, payment_attempt_id, provider_order_id,
							provider_event_id, amount_minor, currency, success
						) VALUES (
							$1, 'default', NOW(), 'duplicate-event-raw-hash', 'wechat',
							'{}'::jsonb, NOW(), $2, 'different-provider-order',
							'wx-payment-1', 1234, 'CNY', false
						)
					`, duplicateEventFactID, attemptID)
					require.Error(t, err, "provider event IDs must be unique per namespace and provider")
				},
			},
		},
	}.Test(t)
}
