package alipay

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/commerce/payment"
)

var testNow = time.Date(2027, time.January, 15, 8, 0, 0, 0, time.FixedZone("CST", 8*60*60))

type testKeys struct {
	appPrivate      *rsa.PrivateKey
	appPrivatePEM   string
	alipayPrivate   *rsa.PrivateKey
	alipayPublicPEM string
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func newTestKeys(t *testing.T) testKeys {
	t.Helper()
	appPrivate, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	alipayPrivate, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	alipayPublicDER, err := x509.MarshalPKIXPublicKey(&alipayPrivate.PublicKey)
	require.NoError(t, err)
	return testKeys{
		appPrivate:    appPrivate,
		appPrivatePEM: string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(appPrivate)})),
		alipayPrivate: alipayPrivate,
		alipayPublicPEM: string(pem.EncodeToMemory(&pem.Block{
			Type: "PUBLIC KEY", Bytes: alipayPublicDER,
		})),
	}
}

func signTestRSA2(t *testing.T, key *rsa.PrivateKey, content string) string {
	t.Helper()
	digest := sha256.Sum256([]byte(content))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(signature)
}

func requireValidRequestRSA2(t *testing.T, publicKey *rsa.PublicKey, values url.Values) {
	t.Helper()
	signature, err := base64.StdEncoding.DecodeString(values.Get("sign"))
	require.NoError(t, err)
	digest := sha256.Sum256([]byte(requestSignContent(values)))
	require.NoError(t, rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, digest[:], signature))
}

func testNotificationSignContent(values url.Values) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		if key != "sign" && key != "sign_type" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		for _, value := range values[key] {
			if value != "" {
				parts = append(parts, key+"="+value)
			}
		}
	}
	return strings.Join(parts, "&")
}

func writeSignedAlipayResponse(t *testing.T, key *rsa.PrivateKey, w http.ResponseWriter, responseKey string, response map[string]any) {
	t.Helper()
	responseJSON, err := json.Marshal(response)
	require.NoError(t, err)
	envelope, err := json.Marshal(map[string]any{
		responseKey: json.RawMessage(responseJSON),
		"sign":      signTestRSA2(t, key, string(responseJSON)),
	})
	require.NoError(t, err)
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(envelope)
	require.NoError(t, err)
}

func newTestAdapter(t *testing.T, gatewayURL string, keys testKeys) *Adapter {
	t.Helper()
	adapter, err := New(Config{
		Secrets: &payment.StaticSecretProvider{Secrets: map[string]string{
			SecretKeyAppPrivateKey:   keys.appPrivatePEM,
			SecretKeyAlipayPublicKey: keys.alipayPublicPEM,
		}},
		Client:           &http.Client{Timeout: 2 * time.Second},
		GatewayURL:       gatewayURL,
		AppID:            "ali-app",
		SellerID:         "ali-seller",
		NotifyURL:        "https://merchant.example/alipay/notify",
		Now:              func() time.Time { return testNow },
		MaxResponseBytes: 1 << 20,
		Logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	require.NoError(t, err)
	return adapter
}

func buildAlipayCallback(t *testing.T, key *rsa.PrivateKey, overrides map[string]string) []byte {
	t.Helper()
	values := url.Values{
		"app_id":       {"ali-app"},
		"seller_id":    {"ali-seller"},
		"out_trade_no": {"01ORDER"},
		"trade_no":     {"2027011522001400000001"},
		"trade_status": {"TRADE_SUCCESS"},
		"total_amount": {"100.00"},
		"subject":      {"知识库充值"},
		"notify_id":    {"notify-1"},
		"notify_time":  {"2027-01-15 08:00:00"},
		"sign_type":    {"RSA2"},
	}
	for key, value := range overrides {
		values.Set(key, value)
	}
	values.Set("sign", signTestRSA2(t, key, testNotificationSignContent(values)))
	return []byte(values.Encode())
}

func buildAlipayRefundCallback(t *testing.T, key *rsa.PrivateKey, overrides map[string]string) []byte {
	t.Helper()
	values := url.Values{
		"app_id":         {"ali-app"},
		"seller_id":      {"ali-seller"},
		"out_trade_no":   {"01ORDER"},
		"out_request_no": {"refund-1"},
		"refund_fee":     {"100.00"},
		"refund_status":  {"REFUND_SUCCESS"},
		"gmt_refund":     {"2027-01-15 08:00:00"},
		"sign_type":      {"RSA2"},
	}
	for name, value := range overrides {
		values.Set(name, value)
	}
	values.Set("sign", signTestRSA2(t, key, testNotificationSignContent(values)))
	return []byte(values.Encode())
}

func TestCreateQRCodeCallsAlipayPrecreate(t *testing.T) {
	keys := newTestKeys(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		require.NoError(t, r.ParseForm())
		require.Equal(t, "alipay.trade.precreate", r.Form.Get("method"))
		require.Equal(t, "RSA2", r.Form.Get("sign_type"))
		require.Equal(t, "ali-app", r.Form.Get("app_id"))
		require.Equal(t, "2027-01-15 08:00:00", r.Form.Get("timestamp"))
		require.Equal(t, "https://merchant.example/alipay/notify", r.Form.Get("notify_url"))
		requireValidRequestRSA2(t, &keys.appPrivate.PublicKey, r.Form)

		var biz map[string]any
		require.NoError(t, json.Unmarshal([]byte(r.Form.Get("biz_content")), &biz))
		require.Equal(t, "01ORDER", biz["out_trade_no"])
		require.Equal(t, "100.00", biz["total_amount"])
		require.Equal(t, "WeKnora recharge", biz["subject"])

		writeSignedAlipayResponse(t, keys.alipayPrivate, w, "alipay_trade_precreate_response", map[string]any{
			"code": "10000", "msg": "Success", "out_trade_no": "01ORDER",
			"qr_code": "https://qr.alipay.test/01ORDER",
		})
	}))
	defer server.Close()

	fact, err := newTestAdapter(t, server.URL, keys).CreateQRCode(t.Context(), payment.CheckoutInput{
		OrderPublicID: "01ORDER", AmountMinor: 10000, Currency: " cny ", Description: "WeKnora recharge",
	})
	require.NoError(t, err)
	require.Equal(t, "https://qr.alipay.test/01ORDER", fact.QRCodeURL)
	require.Equal(t, "01ORDER", fact.ProviderOrderID)
	require.Empty(t, fact.ProviderPaymentID)
}

func TestVerifyCallbackRSA2AndExactAmount(t *testing.T) {
	keys := newTestKeys(t)
	adapter := newTestAdapter(t, "https://openapi.alipay.com/gateway.do", keys)

	fact, err := adapter.VerifyCallback(t.Context(), nil, buildAlipayCallback(t, keys.alipayPrivate, nil))
	require.NoError(t, err)
	require.Equal(t, "notify-1", fact.ProviderEventID)
	require.Equal(t, "2027011522001400000001", fact.ProviderPaymentID)
	require.Equal(t, "01ORDER", fact.ProviderOrderID)
	require.Equal(t, "ali-app", fact.ApplicationID)
	require.Equal(t, "ali-seller", fact.MerchantID)
	require.Equal(t, int64(10000), fact.AmountMinor)
	require.Equal(t, "CNY", fact.Currency)
	require.True(t, fact.Success)
	require.NotEmpty(t, fact.RawHash)
	require.Equal(t, testNow, fact.Timestamp)
	require.Equal(t, "知识库充值", fact.SignedPayload["subject"])
	require.NotContains(t, fact.SignedPayload, "sign")
}

func TestVerifyCallbackRejectsTamperingAndInvalidMoney(t *testing.T) {
	keys := newTestKeys(t)
	adapter := newTestAdapter(t, "https://openapi.alipay.com/gateway.do", keys)

	t.Run("tampered signed value", func(t *testing.T) {
		body := buildAlipayCallback(t, keys.alipayPrivate, nil)
		values, err := url.ParseQuery(string(body))
		require.NoError(t, err)
		values.Set("subject", "篡改后的主题")
		_, err = adapter.VerifyCallback(t.Context(), nil, []byte(values.Encode()))
		require.ErrorIs(t, err, payment.ErrInvalidSignature)
	})

	for _, testCase := range []struct {
		name   string
		amount string
		want   int64
	}{
		{name: "one fen", amount: "0.01", want: 1},
		{name: "one hundred yuan ten fen", amount: "100.10", want: 10010},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			fact, err := adapter.VerifyCallback(t.Context(), nil, buildAlipayCallback(t, keys.alipayPrivate, map[string]string{"total_amount": testCase.amount}))
			require.NoError(t, err)
			require.Equal(t, testCase.want, fact.AmountMinor)
		})
	}

	for _, amount := range []string{"1.001", "1e2", "one", "-1.00", "0.00", "92233720368547758.08"} {
		t.Run("reject "+amount, func(t *testing.T) {
			_, err := adapter.VerifyCallback(t.Context(), nil, buildAlipayCallback(t, keys.alipayPrivate, map[string]string{"total_amount": amount}))
			require.ErrorIs(t, err, payment.ErrPermanentProviderProtocol)
		})
	}
}

func TestVerifyCallbackValidatesIdentityOrderAndStatus(t *testing.T) {
	keys := newTestKeys(t)
	adapter := newTestAdapter(t, "https://openapi.alipay.com/gateway.do", keys)
	for _, testCase := range []struct {
		name      string
		overrides map[string]string
	}{
		{name: "wrong application", overrides: map[string]string{"app_id": "other-app"}},
		{name: "wrong seller", overrides: map[string]string{"seller_id": "other-seller"}},
		{name: "missing order", overrides: map[string]string{"out_trade_no": ""}},
		{name: "successful payment missing trade number", overrides: map[string]string{"trade_no": ""}},
		{name: "unknown status", overrides: map[string]string{"trade_status": "UNKNOWN"}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := adapter.VerifyCallback(t.Context(), nil, buildAlipayCallback(t, keys.alipayPrivate, testCase.overrides))
			require.ErrorIs(t, err, payment.ErrPermanentProviderProtocol)
		})
	}

	fact, err := adapter.VerifyCallback(t.Context(), nil, buildAlipayCallback(t, keys.alipayPrivate, map[string]string{
		"notify_id": "", "trade_status": "WAIT_BUYER_PAY",
	}))
	require.NoError(t, err)
	require.Empty(t, fact.ProviderEventID)
	require.NotEmpty(t, fact.RawHash)
	require.False(t, fact.Success)
}

func TestVerifyRefundCallbackValidatesSignedContextAndStatus(t *testing.T) {
	keys := newTestKeys(t)
	adapter := newTestAdapter(t, "https://openapi.alipay.com/gateway.do", keys)

	for _, testCase := range []struct {
		name      string
		overrides map[string]string
	}{
		{name: "non RSA2 signature", overrides: map[string]string{"sign_type": "RSA"}},
		{name: "wrong application", overrides: map[string]string{"app_id": "other-app"}},
		{name: "wrong seller", overrides: map[string]string{"seller_id": "other-seller"}},
		{name: "missing order", overrides: map[string]string{"out_trade_no": ""}},
		{name: "missing request", overrides: map[string]string{"out_request_no": ""}},
		{name: "zero refund", overrides: map[string]string{"refund_fee": "0.00"}},
		{name: "non CNY currency", overrides: map[string]string{"refund_currency": "USD"}},
		{name: "unknown status", overrides: map[string]string{"refund_status": "UNKNOWN"}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := adapter.VerifyRefundCallback(t.Context(), nil, buildAlipayRefundCallback(t, keys.alipayPrivate, testCase.overrides))
			require.ErrorIs(t, err, payment.ErrPermanentProviderProtocol)
		})
	}

	success, err := adapter.VerifyRefundCallback(t.Context(), nil, buildAlipayRefundCallback(t, keys.alipayPrivate, nil))
	require.NoError(t, err)
	require.True(t, success.Success)
	require.NotEmpty(t, success.RawHash)
	require.Equal(t, "CNY", success.Currency)

	processing, err := adapter.VerifyRefundCallback(t.Context(), nil, buildAlipayRefundCallback(t, keys.alipayPrivate, map[string]string{"refund_status": "REFUND_PROCESSING"}))
	require.NoError(t, err)
	require.False(t, processing.Success)
	require.Empty(t, processing.RawHash)

	failed, err := adapter.VerifyRefundCallback(t.Context(), nil, buildAlipayRefundCallback(t, keys.alipayPrivate, map[string]string{"refund_status": "REFUND_FAIL"}))
	require.NoError(t, err)
	require.False(t, failed.Success)
	require.NotEmpty(t, failed.RawHash)
}

func TestQueryPaymentMapsOnlySuccessfulTradeStates(t *testing.T) {
	for _, testCase := range []struct {
		status   string
		success  bool
		terminal bool
	}{
		{status: "TRADE_SUCCESS", success: true, terminal: true},
		{status: "TRADE_FINISHED", success: true, terminal: true},
		{status: "WAIT_BUYER_PAY", success: false},
		{status: "TRADE_CLOSED", success: false, terminal: true},
	} {
		t.Run(testCase.status, func(t *testing.T) {
			keys := newTestKeys(t)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.NoError(t, r.ParseForm())
				require.Equal(t, "alipay.trade.query", r.Form.Get("method"))
				requireValidRequestRSA2(t, &keys.appPrivate.PublicKey, r.Form)
				writeSignedAlipayResponse(t, keys.alipayPrivate, w, "alipay_trade_query_response", map[string]any{
					"code": "10000", "msg": "Success", "out_trade_no": "01ORDER",
					"trade_no": "2027011522001400000001", "trade_status": testCase.status,
					"total_amount": "100.00",
				})
			}))
			defer server.Close()

			fact, err := newTestAdapter(t, server.URL, keys).QueryPayment(t.Context(), "01ORDER")
			require.NoError(t, err)
			require.Equal(t, testCase.success, fact.Success)
			require.Equal(t, testCase.terminal, fact.Terminal)
			require.Equal(t, int64(10000), fact.AmountMinor)
			require.Equal(t, "CNY", fact.Currency)
			require.Equal(t, "ali-app", fact.ApplicationID)
			require.Equal(t, "ali-seller", fact.MerchantID)
			require.Empty(t, fact.ProviderEventID)
			require.NotEmpty(t, fact.RawHash)
		})
	}
}

func TestQueryPaymentAllowsSameTradeNumberToAdvanceFromWaitingToSuccess(t *testing.T) {
	keys := newTestKeys(t)
	requestNumber := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestNumber++
		status := "WAIT_BUYER_PAY"
		if requestNumber == 2 {
			status = "TRADE_SUCCESS"
		}
		writeSignedAlipayResponse(t, keys.alipayPrivate, w, "alipay_trade_query_response", map[string]any{
			"code": "10000", "msg": "Success", "out_trade_no": "01ORDER",
			"trade_no": "same-trade-no", "trade_status": status, "total_amount": "100.00",
		})
	}))
	defer server.Close()
	adapter := newTestAdapter(t, server.URL, keys)

	waiting, err := adapter.QueryPayment(t.Context(), "01ORDER")
	require.NoError(t, err)
	success, err := adapter.QueryPayment(t.Context(), "01ORDER")
	require.NoError(t, err)
	require.False(t, waiting.Success)
	require.True(t, success.Success)
	require.Empty(t, waiting.ProviderEventID)
	require.Empty(t, success.ProviderEventID)
	require.NotEqual(t, waiting.RawHash, success.RawHash)
}

func TestRefundAndQueryRefundCallAlipayGateway(t *testing.T) {
	keys := newTestKeys(t)
	var requestNumber int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestNumber++
		require.NoError(t, r.ParseForm())
		requireValidRequestRSA2(t, &keys.appPrivate.PublicKey, r.Form)
		var biz map[string]any
		require.NoError(t, json.Unmarshal([]byte(r.Form.Get("biz_content")), &biz))

		switch requestNumber {
		case 1:
			require.Equal(t, "alipay.trade.refund", r.Form.Get("method"))
			require.Equal(t, "01ORDER", biz["out_trade_no"])
			require.Equal(t, "10.01", biz["refund_amount"])
			require.Equal(t, "refund-idem-1", biz["out_request_no"])
			require.Equal(t, "customer request", biz["refund_reason"])
			writeSignedAlipayResponse(t, keys.alipayPrivate, w, "alipay_trade_refund_response", map[string]any{
				"code": "10000", "msg": "Success", "trade_no": "2027011522001400000001",
				"out_trade_no": "01ORDER", "refund_fee": "10.01", "fund_change": "Y",
			})
		case 2:
			require.Equal(t, "alipay.trade.fastpay.refund.query", r.Form.Get("method"))
			require.Equal(t, "refund-idem-1", biz["out_request_no"])
			require.Equal(t, "01ORDER", biz["out_trade_no"])
			writeSignedAlipayResponse(t, keys.alipayPrivate, w, "alipay_trade_fastpay_refund_query_response", map[string]any{
				"code": "10000", "msg": "Success", "trade_no": "2027011522001400000001",
				"out_trade_no": "01ORDER", "out_request_no": "refund-idem-1",
				"refund_amount": "10.01", "refund_status": "REFUND_SUCCESS",
			})
		default:
			t.Fatalf("unexpected request %d", requestNumber)
		}
	}))
	defer server.Close()
	adapter := newTestAdapter(t, server.URL, keys)

	submission, err := adapter.Refund(t.Context(), payment.RefundInput{
		ProviderOrderID: "01ORDER", AmountMinor: 1001, Currency: "CNY",
		Reason: "customer request", IdempotencyKey: "refund-idem-1",
	})
	require.NoError(t, err)
	require.Equal(t, "refund-idem-1", submission.ProviderRefundID)
	require.Equal(t, "success", submission.Status)

	fact, err := adapter.QueryRefund(t.Context(), payment.RefundQueryInput{
		ProviderRefundID: "refund-idem-1", ProviderOrderID: "01ORDER", AmountMinor: 1001, Currency: "CNY",
	})
	require.NoError(t, err)
	require.Equal(t, "refund-idem-1", fact.ProviderRefundID)
	require.Equal(t, "01ORDER", fact.ProviderOrderID)
	require.Equal(t, int64(1001), fact.AmountMinor)
	require.Equal(t, "CNY", fact.Currency)
	require.True(t, fact.Success)
	require.NotEmpty(t, fact.RawHash)
}

func TestRefundRejectsMismatchedSignedAmount(t *testing.T) {
	keys := newTestKeys(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeSignedAlipayResponse(t, keys.alipayPrivate, w, "alipay_trade_refund_response", map[string]any{
			"code": "10000", "msg": "Success", "trade_no": "trade-1",
			"out_trade_no": "01ORDER", "refund_fee": "10.00", "fund_change": "Y",
		})
	}))
	defer server.Close()
	_, err := newTestAdapter(t, server.URL, keys).Refund(t.Context(), payment.RefundInput{
		ProviderOrderID: "01ORDER", AmountMinor: 1001, Currency: "CNY", IdempotencyKey: "refund-idem-1",
	})
	require.ErrorIs(t, err, payment.ErrPermanentProviderProtocol)
}

func TestQueryRefundRejectsMismatchedSignedContext(t *testing.T) {
	keys := newTestKeys(t)
	for _, testCase := range []struct {
		name    string
		orderID string
		amount  string
	}{
		{name: "provider order", orderID: "OTHER", amount: "10.01"},
		{name: "refund amount", orderID: "01ORDER", amount: "10.00"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeSignedAlipayResponse(t, keys.alipayPrivate, w, "alipay_trade_fastpay_refund_query_response", map[string]any{
					"code": "10000", "msg": "Success", "trade_no": "trade-1",
					"out_trade_no": testCase.orderID, "out_request_no": "refund-idem-1",
					"refund_amount": testCase.amount, "refund_status": "REFUND_SUCCESS",
				})
			}))
			defer server.Close()
			_, err := newTestAdapter(t, server.URL, keys).QueryRefund(t.Context(), payment.RefundQueryInput{
				ProviderRefundID: "refund-idem-1", ProviderOrderID: "01ORDER", AmountMinor: 1001, Currency: "CNY",
			})
			require.ErrorIs(t, err, payment.ErrPermanentProviderProtocol)
		})
	}
}

func TestQueryRefundInfersSuccessWhenOptionalStatusIsAbsent(t *testing.T) {
	keys := newTestKeys(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeSignedAlipayResponse(t, keys.alipayPrivate, w, "alipay_trade_fastpay_refund_query_response", map[string]any{
			"code": "10000", "msg": "Success", "trade_no": "trade-1",
			"out_trade_no": "01ORDER", "out_request_no": "refund-idem-1",
			"refund_amount": "10.01",
		})
	}))
	defer server.Close()

	fact, err := newTestAdapter(t, server.URL, keys).QueryRefund(t.Context(), payment.RefundQueryInput{
		ProviderRefundID: "refund-idem-1", ProviderOrderID: "01ORDER", AmountMinor: 1001, Currency: "CNY",
	})
	require.NoError(t, err)
	require.True(t, fact.Success)
	require.NotEmpty(t, fact.RawHash)
}

func TestQueryRefundProcessingHasNoDefinitiveFailureHash(t *testing.T) {
	keys := newTestKeys(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeSignedAlipayResponse(t, keys.alipayPrivate, w, "alipay_trade_fastpay_refund_query_response", map[string]any{
			"code": "10000", "msg": "Success", "trade_no": "trade-1",
			"out_trade_no": "01ORDER", "out_request_no": "refund-idem-1",
			"refund_amount": "10.01", "refund_status": "REFUND_PROCESSING",
		})
	}))
	defer server.Close()
	fact, err := newTestAdapter(t, server.URL, keys).QueryRefund(t.Context(), payment.RefundQueryInput{
		ProviderRefundID: "refund-idem-1", ProviderOrderID: "01ORDER", AmountMinor: 1001, Currency: "CNY",
	})
	require.NoError(t, err)
	require.False(t, fact.Success)
	require.Empty(t, fact.RawHash)
}

func TestGatewayRejectsTamperedAndOversizedResponses(t *testing.T) {
	keys := newTestKeys(t)
	t.Run("tampered signed response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			responseJSON := `{"code":"10000","msg":"Success","out_trade_no":"01ORDER","qr_code":"https://qr.example/real"}`
			envelope := fmt.Sprintf(`{"alipay_trade_precreate_response":%s,"sign":%q}`, responseJSON, signTestRSA2(t, keys.alipayPrivate, responseJSON))
			envelope = strings.Replace(envelope, "https://qr.example/real", "https://qr.example/evil", 1)
			_, err := io.WriteString(w, envelope)
			require.NoError(t, err)
		}))
		defer server.Close()
		_, err := newTestAdapter(t, server.URL, keys).CreateQRCode(t.Context(), payment.CheckoutInput{
			OrderPublicID: "01ORDER", AmountMinor: 10000, Currency: "CNY", Description: "test",
		})
		require.ErrorIs(t, err, payment.ErrInvalidSignature)
	})

	t.Run("response body limit", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, err := io.WriteString(w, strings.Repeat("x", 2048))
			require.NoError(t, err)
		}))
		defer server.Close()
		adapter, err := New(Config{
			Secrets: &payment.StaticSecretProvider{Secrets: map[string]string{
				SecretKeyAppPrivateKey: keys.appPrivatePEM, SecretKeyAlipayPublicKey: keys.alipayPublicPEM,
			}},
			Client: &http.Client{Timeout: time.Second}, GatewayURL: server.URL, AppID: "ali-app",
			SellerID: "ali-seller", NotifyURL: "https://merchant.example/notify", Now: time.Now,
			MaxResponseBytes: 256, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		})
		require.NoError(t, err)
		_, err = adapter.QueryPayment(t.Context(), "01ORDER")
		require.ErrorIs(t, err, payment.ErrPermanentProviderProtocol)
	})
}

func TestGatewayErrorsExposeRetryClassification(t *testing.T) {
	keys := newTestKeys(t)
	for _, testCase := range []struct {
		name       string
		statusCode int
		code       string
		subCode    string
		kind       payment.ProviderErrorKind
		marker     error
	}{
		{name: "HTTP 503", statusCode: http.StatusServiceUnavailable, kind: payment.ProviderErrorRetryable, marker: payment.ErrRetryableProvider},
		{name: "HTTP 400", statusCode: http.StatusBadRequest, kind: payment.ProviderErrorPermanent, marker: payment.ErrPermanentProviderProtocol},
		{name: "provider unavailable", statusCode: http.StatusOK, code: "20000", kind: payment.ProviderErrorRetryable, marker: payment.ErrRetryableProvider},
		{name: "provider system error", statusCode: http.StatusOK, code: "40004", subCode: "ACQ.SYSTEM_ERROR", kind: payment.ProviderErrorRetryable, marker: payment.ErrRetryableProvider},
		{name: "provider business rejection", statusCode: http.StatusOK, code: "40004", subCode: "ACQ.INVALID_PARAMETER", kind: payment.ProviderErrorPermanent, marker: payment.ErrPermanentProviderProtocol},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(testCase.statusCode)
				if testCase.statusCode != http.StatusOK {
					_, err := io.WriteString(w, "sensitive-response-body-marker")
					require.NoError(t, err)
					return
				}
				writeSignedAlipayResponse(t, keys.alipayPrivate, w, "alipay_trade_precreate_response", map[string]any{
					"code": testCase.code, "sub_code": testCase.subCode,
					"msg": "sensitive-response-body-marker", "sub_msg": "do not expose this message",
				})
			}))
			defer server.Close()

			_, err := newTestAdapter(t, server.URL, keys).CreateQRCode(t.Context(), payment.CheckoutInput{
				OrderPublicID: "01ORDER", AmountMinor: 10000, Currency: "CNY", Description: "test",
			})
			require.ErrorIs(t, err, testCase.marker)
			require.NotContains(t, err.Error(), "sensitive-response-body-marker")
			require.NotContains(t, err.Error(), "do not expose this message")

			var providerErr *payment.ProviderError
			require.ErrorAs(t, err, &providerErr)
			require.Equal(t, payment.ProviderAlipay, providerErr.Provider)
			require.Equal(t, "alipay.trade.precreate", providerErr.Operation)
			require.Equal(t, testCase.kind, providerErr.Kind)
			require.Equal(t, testCase.statusCode, providerErr.HTTPStatus)
			require.Equal(t, testCase.code, providerErr.Code)
			require.Equal(t, testCase.subCode, providerErr.SubCode)
		})
	}
}

func TestGatewayTransportTimeoutIsRetryableAndPreservesCause(t *testing.T) {
	keys := newTestKeys(t)
	adapter, err := New(Config{
		Secrets: &payment.StaticSecretProvider{Secrets: map[string]string{
			SecretKeyAppPrivateKey: keys.appPrivatePEM, SecretKeyAlipayPublicKey: keys.alipayPublicPEM,
		}},
		Client: &http.Client{
			Timeout: time.Second,
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return nil, fmt.Errorf("transport timeout: %w", context.DeadlineExceeded)
			}),
		},
		GatewayURL: "https://openapi.alipay.test/gateway.do", AppID: "ali-app", SellerID: "ali-seller",
		NotifyURL: "https://merchant.example/notify", Now: time.Now, MaxResponseBytes: 1024,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	require.NoError(t, err)

	_, err = adapter.CreateQRCode(t.Context(), payment.CheckoutInput{
		OrderPublicID: "01ORDER", AmountMinor: 10000, Currency: "CNY", Description: "test",
	})
	require.ErrorIs(t, err, payment.ErrRetryableProvider)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	var providerErr *payment.ProviderError
	require.ErrorAs(t, err, &providerErr)
	require.Equal(t, payment.ProviderErrorRetryable, providerErr.Kind)
	require.Zero(t, providerErr.HTTPStatus)
}

func TestGatewayMalformedSuccessResponseIsPermanentProviderError(t *testing.T) {
	keys := newTestKeys(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := io.WriteString(w, "not-json-sensitive-response-body-marker")
		require.NoError(t, err)
	}))
	defer server.Close()

	_, err := newTestAdapter(t, server.URL, keys).CreateQRCode(t.Context(), payment.CheckoutInput{
		OrderPublicID: "01ORDER", AmountMinor: 10000, Currency: "CNY", Description: "test",
	})
	require.ErrorIs(t, err, payment.ErrPermanentProviderProtocol)
	require.NotContains(t, err.Error(), "sensitive-response-body-marker")
	var providerErr *payment.ProviderError
	require.ErrorAs(t, err, &providerErr)
	require.Equal(t, payment.ProviderErrorPermanent, providerErr.Kind)
}

func TestNewRequiresProductionDependenciesAndIdentity(t *testing.T) {
	keys := newTestKeys(t)
	valid := Config{
		Secrets: &payment.StaticSecretProvider{Secrets: map[string]string{
			SecretKeyAppPrivateKey: keys.appPrivatePEM, SecretKeyAlipayPublicKey: keys.alipayPublicPEM,
		}},
		Client: &http.Client{Timeout: time.Second}, GatewayURL: "https://openapi.alipay.com/gateway.do",
		AppID: "ali-app", SellerID: "ali-seller", NotifyURL: "https://merchant.example/notify",
		Now: time.Now, MaxResponseBytes: 1024, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	adapter, err := New(valid)
	require.NoError(t, err)
	identity, err := adapter.Identity(t.Context())
	require.NoError(t, err)
	require.Equal(t, "ali-seller", identity.MerchantID)
	require.Equal(t, "ali-app", identity.ApplicationID)
	require.Equal(t, payment.ProviderAlipay, adapter.Name())

	testCases := []struct {
		name   string
		mutate func(*Config)
	}{
		{name: "secrets", mutate: func(config *Config) { config.Secrets = nil }},
		{name: "client", mutate: func(config *Config) { config.Client = nil }},
		{name: "client timeout", mutate: func(config *Config) { config.Client = &http.Client{} }},
		{name: "gateway", mutate: func(config *Config) { config.GatewayURL = "" }},
		{name: "application", mutate: func(config *Config) { config.AppID = "" }},
		{name: "seller", mutate: func(config *Config) { config.SellerID = "" }},
		{name: "notify URL", mutate: func(config *Config) { config.NotifyURL = "" }},
		{name: "clock", mutate: func(config *Config) { config.Now = nil }},
		{name: "response size", mutate: func(config *Config) { config.MaxResponseBytes = 0 }},
		{name: "logger", mutate: func(config *Config) { config.Logger = nil }},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			config := valid
			testCase.mutate(&config)
			_, err := New(config)
			require.Error(t, err)
		})
	}
}

func TestNewRejectsMalformedConfiguredKeyMaterial(t *testing.T) {
	keys := newTestKeys(t)
	validSecrets := map[string]string{
		SecretKeyAppPrivateKey:   keys.appPrivatePEM,
		SecretKeyAlipayPublicKey: keys.alipayPublicPEM,
	}
	newConfig := func(secrets map[string]string) Config {
		return Config{
			Secrets: &payment.StaticSecretProvider{Secrets: secrets},
			Client:  &http.Client{Timeout: time.Second}, GatewayURL: "https://openapi.alipay.com/gateway.do",
			AppID: "ali-app", SellerID: "ali-seller", NotifyURL: "https://merchant.example/notify",
			Now: time.Now, MaxResponseBytes: 1024, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		}
	}

	for _, tt := range []struct {
		name string
		key  string
	}{
		{name: "application private key", key: SecretKeyAppPrivateKey},
		{name: "Alipay public key", key: SecretKeyAlipayPublicKey},
	} {
		t.Run(tt.name, func(t *testing.T) {
			secrets := make(map[string]string, len(validSecrets))
			for key, value := range validSecrets {
				secrets[key] = value
			}
			const secretMarker = "malformed-secret-content-marker"
			secrets[tt.key] = secretMarker

			_, err := New(newConfig(secrets))
			require.Error(t, err)
			require.NotContains(t, err.Error(), secretMarker)
		})
	}
}

func TestRequestSignContentUsesSortedDecodedValues(t *testing.T) {
	values, err := url.ParseQuery("subject=%E7%9F%A5%E8%AF%86%E5%BA%93+%26+RAG&app_id=ali-app&sign=ignored&sign_type=RSA2")
	require.NoError(t, err)
	require.Equal(t, "app_id=ali-app&sign_type=RSA2&subject=知识库 & RAG", requestSignContent(values))
}

func TestParseAmountMinor(t *testing.T) {
	for _, testCase := range []struct {
		value string
		want  int64
	}{
		{value: "0.01", want: 1},
		{value: "100.10", want: 10010},
		{value: "100", want: 10000},
	} {
		got, err := parseAmountMinor(testCase.value)
		require.NoError(t, err)
		require.Equal(t, testCase.want, got)
	}
}
