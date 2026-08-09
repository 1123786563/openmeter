package wechat

import (
	"context"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
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
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/commerce/payment"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

const (
	testAPIv3Key = "0123456789abcdef0123456789abcdef"
	testNowUnix  = int64(1_800_000_000)
)

type testKeys struct {
	merchantPrivate    *rsa.PrivateKey
	merchantPrivatePEM string
	platformPrivate    *rsa.PrivateKey
	platformPublicPEM  string
}

func newTestKeys(t *testing.T) testKeys {
	t.Helper()
	merchant, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	platform, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	merchantDER := x509.MarshalPKCS1PrivateKey(merchant)
	platformDER, err := x509.MarshalPKIXPublicKey(&platform.PublicKey)
	require.NoError(t, err)

	return testKeys{
		merchantPrivate: merchant,
		merchantPrivatePEM: string(pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: merchantDER,
		})),
		platformPrivate: platform,
		platformPublicPEM: string(pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: platformDER,
		})),
	}
}

func newTestAdapter(t *testing.T, baseURL string, keys testKeys) *Adapter {
	t.Helper()
	adapter, err := New(Config{
		Secrets: &payment.StaticSecretProvider{Secrets: map[string]string{
			SecretKeyMerchantPrivateKey:                keys.merchantPrivatePEM,
			SecretKeyAPIv3:                             testAPIv3Key,
			PlatformPublicKeySecret("platform-serial"): keys.platformPublicPEM,
		}},
		Client:           &http.Client{Timeout: 2 * time.Second},
		BaseURL:          baseURL,
		AppID:            "wx-app",
		MerchantID:       "wx-mch",
		MerchantSerial:   "merchant-serial",
		NotifyURL:        "https://merchant.example/wechat/notify",
		RefundNotifyURL:  "https://merchant.example/wechat/refund-notify",
		Now:              func() time.Time { return time.Unix(testNowUnix, 0) },
		CallbackMaxAge:   5 * time.Minute,
		MaxResponseBytes: 1 << 20,
		Logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	require.NoError(t, err)
	return adapter
}

func signWechatMessage(t *testing.T, key *rsa.PrivateKey, message string) string {
	t.Helper()
	digest := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(signature)
}

func writeSignedWechatResponse(t *testing.T, key *rsa.PrivateKey, w http.ResponseWriter, status int, body string) {
	t.Helper()
	timestamp := strconv.FormatInt(testNowUnix, 10)
	nonce := "response-nonce"
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Wechatpay-Timestamp", timestamp)
	w.Header().Set("Wechatpay-Nonce", nonce)
	w.Header().Set("Wechatpay-Serial", "platform-serial")
	w.Header().Set("Wechatpay-Signature", signWechatMessage(t, key, timestamp+"\n"+nonce+"\n"+body+"\n"))
	w.WriteHeader(status)
	_, err := io.WriteString(w, body)
	require.NoError(t, err)
}

func requireValidRequestAuthorization(t *testing.T, key *rsa.PrivateKey, r *http.Request, body []byte) {
	t.Helper()
	header := r.Header.Get("Authorization")
	require.True(t, strings.HasPrefix(header, "WECHATPAY2-SHA256-RSA2048 "))
	attributes := map[string]string{}
	for _, item := range strings.Split(strings.TrimPrefix(header, "WECHATPAY2-SHA256-RSA2048 "), ",") {
		parts := strings.SplitN(strings.TrimSpace(item), "=", 2)
		require.Len(t, parts, 2)
		attributes[parts[0]] = strings.Trim(parts[1], `"`)
	}
	require.Equal(t, "wx-mch", attributes["mchid"])
	require.Equal(t, "merchant-serial", attributes["serial_no"])
	require.NotEmpty(t, attributes["nonce_str"])
	require.Equal(t, strconv.FormatInt(testNowUnix, 10), attributes["timestamp"])

	signature, err := base64.StdEncoding.DecodeString(attributes["signature"])
	require.NoError(t, err)
	canonicalURL := r.URL.EscapedPath()
	if r.URL.RawQuery != "" {
		canonicalURL += "?" + r.URL.RawQuery
	}
	message := r.Method + "\n" + canonicalURL + "\n" + attributes["timestamp"] + "\n" + attributes["nonce_str"] + "\n" + string(body) + "\n"
	digest := sha256.Sum256([]byte(message))
	require.NoError(t, rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, digest[:], signature))
}

func encryptNotificationResource(t *testing.T, apiKey, plaintext string) encryptedResource {
	t.Helper()
	block, err := aes.NewCipher([]byte(apiKey))
	require.NoError(t, err)
	gcm, err := cipher.NewGCM(block)
	require.NoError(t, err)
	nonce := []byte("notify-nonce")
	associatedData := "transaction"
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), []byte(associatedData))
	return encryptedResource{
		Algorithm:      "AEAD_AES_256_GCM",
		Ciphertext:     base64.StdEncoding.EncodeToString(ciphertext),
		Nonce:          string(nonce),
		AssociatedData: associatedData,
		OriginalType:   "transaction",
	}
}

func encryptedCallback(t *testing.T, key *rsa.PrivateKey, apiKey string, timestamp int64) (http.Header, []byte) {
	t.Helper()
	resource := encryptNotificationResource(t, apiKey, `{"appid":"wx-app","mchid":"wx-mch","out_trade_no":"01ORDER","transaction_id":"4200000001","trade_state":"SUCCESS","success_time":"2027-01-15T08:00:00+08:00","amount":{"total":10000,"currency":"CNY"}}`)
	body, err := json.Marshal(notification{
		ID:         "notification-id",
		CreateTime: "2027-01-15T08:00:00+08:00",
		EventType:  "TRANSACTION.SUCCESS",
		Resource:   resource,
	})
	require.NoError(t, err)
	timestampString := strconv.FormatInt(timestamp, 10)
	nonce := "callback-signature-nonce"
	headers := http.Header{}
	headers.Set("Wechatpay-Timestamp", timestampString)
	headers.Set("Wechatpay-Nonce", nonce)
	headers.Set("Wechatpay-Serial", "platform-serial")
	headers.Set("Wechatpay-Signature", signWechatMessage(t, key, timestampString+"\n"+nonce+"\n"+string(body)+"\n"))
	return headers, body
}

func encryptedRefundCallback(t *testing.T, key *rsa.PrivateKey, plaintext string) (http.Header, []byte) {
	t.Helper()
	resource := encryptNotificationResource(t, testAPIv3Key, plaintext)
	resource.OriginalType = "refund"
	body, err := json.Marshal(notification{
		ID: "refund-notification-id", CreateTime: "2027-01-15T08:00:00+08:00",
		EventType: "REFUND.SUCCESS", Resource: resource,
	})
	require.NoError(t, err)
	timestamp := strconv.FormatInt(testNowUnix, 10)
	nonce := "refund-callback-nonce"
	headers := http.Header{
		"Wechatpay-Timestamp": []string{timestamp},
		"Wechatpay-Nonce":     []string{nonce},
		"Wechatpay-Serial":    []string{"platform-serial"},
	}
	headers.Set("Wechatpay-Signature", signWechatMessage(t, key, timestamp+"\n"+nonce+"\n"+string(body)+"\n"))
	return headers, body
}

func TestCreateQRCodeCallsWechatNativeAPI(t *testing.T) {
	keys := newTestKeys(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v3/pay/transactions/native", r.URL.Path)
		bodyBytes, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requireValidRequestAuthorization(t, keys.merchantPrivate, r, bodyBytes)
		var body nativeCreateRequest
		require.NoError(t, json.Unmarshal(bodyBytes, &body))
		require.Equal(t, "wx-app", body.AppID)
		require.Equal(t, "wx-mch", body.MchID)
		require.Equal(t, "01ORDER", body.OutTradeNo)
		require.Equal(t, "https://merchant.example/wechat/notify", body.NotifyURL)
		require.Equal(t, int64(10000), body.Amount.Total)
		require.Equal(t, "CNY", body.Amount.Currency)
		writeSignedWechatResponse(t, keys.platformPrivate, w, http.StatusOK, `{"code_url":"weixin://wxpay/bizpayurl?pr=test"}`)
	}))
	defer server.Close()

	fact, err := newTestAdapter(t, server.URL, keys).CreateQRCode(t.Context(), payment.CheckoutInput{
		OrderPublicID: "01ORDER", AmountMinor: 10000, Currency: " cny ",
		Description: "WeKnora recharge", IdempotencyKey: "idem-1",
	})
	require.NoError(t, err)
	require.Equal(t, "01ORDER", fact.ProviderOrderID)
	require.Empty(t, fact.ProviderPaymentID)
	require.Equal(t, "weixin://wxpay/bizpayurl?pr=test", fact.QRCodeURL)
}

func TestCreateQRCodeRejectsNonCNYBeforeCallingProvider(t *testing.T) {
	keys := newTestKeys(t)
	_, err := newTestAdapter(t, "http://127.0.0.1:1", keys).CreateQRCode(t.Context(), payment.CheckoutInput{
		OrderPublicID: "01ORDER", AmountMinor: 10000, Currency: "USD", Description: "test",
	})
	require.ErrorIs(t, err, payment.ErrPermanentProviderProtocol)
}

func TestVerifyEncryptedCallback(t *testing.T) {
	keys := newTestKeys(t)
	headers, body := encryptedCallback(t, keys.platformPrivate, testAPIv3Key, testNowUnix)

	t.Run("valid", func(t *testing.T) {
		fact, err := newTestAdapter(t, "https://api.mch.weixin.qq.com", keys).VerifyCallback(t.Context(), headers, body)
		require.NoError(t, err)
		require.Equal(t, "notification-id", fact.ProviderEventID)
		require.Equal(t, "4200000001", fact.ProviderPaymentID)
		require.Equal(t, "01ORDER", fact.ProviderOrderID)
		require.Equal(t, "wx-mch", fact.MerchantID)
		require.Equal(t, "wx-app", fact.ApplicationID)
		require.Equal(t, int64(10000), fact.AmountMinor)
		require.Equal(t, "CNY", fact.Currency)
		require.True(t, fact.Success)
		require.NotEmpty(t, fact.RawHash)
		require.NotContains(t, fmt.Sprint(fact.SignedPayload), "ciphertext")
	})

	t.Run("tampered body", func(t *testing.T) {
		tampered := append([]byte(nil), body...)
		tampered[len(tampered)-2] ^= 1
		_, err := newTestAdapter(t, "https://api.mch.weixin.qq.com", keys).VerifyCallback(t.Context(), headers, tampered)
		require.ErrorIs(t, err, payment.ErrInvalidSignature)
	})

	t.Run("unknown serial", func(t *testing.T) {
		unknownHeaders := headers.Clone()
		unknownHeaders.Set("Wechatpay-Serial", "unknown-serial")
		_, err := newTestAdapter(t, "https://api.mch.weixin.qq.com", keys).VerifyCallback(t.Context(), unknownHeaders, body)
		require.Error(t, err)
	})

	t.Run("timestamp outside five minute window", func(t *testing.T) {
		oldHeaders, oldBody := encryptedCallback(t, keys.platformPrivate, testAPIv3Key, testNowUnix-int64((5*time.Minute)/time.Second)-1)
		_, err := newTestAdapter(t, "https://api.mch.weixin.qq.com", keys).VerifyCallback(t.Context(), oldHeaders, oldBody)
		require.Error(t, err)
	})

	t.Run("wrong API v3 key", func(t *testing.T) {
		wrongHeaders, wrongBody := encryptedCallback(t, keys.platformPrivate, "abcdef0123456789abcdef0123456789", testNowUnix)
		_, err := newTestAdapter(t, "https://api.mch.weixin.qq.com", keys).VerifyCallback(t.Context(), wrongHeaders, wrongBody)
		require.Error(t, err)
	})
}

func TestVerifyEncryptedCallbackRejectsPaymentIdentityAndMoneyMismatch(t *testing.T) {
	keys := newTestKeys(t)
	tests := []struct {
		name      string
		plaintext string
	}{
		{name: "application", plaintext: `{"appid":"other-app","mchid":"wx-mch","out_trade_no":"01ORDER","transaction_id":"4201","trade_state":"SUCCESS","amount":{"total":10000,"currency":"CNY"}}`},
		{name: "merchant", plaintext: `{"appid":"wx-app","mchid":"other-mch","out_trade_no":"01ORDER","transaction_id":"4201","trade_state":"SUCCESS","amount":{"total":10000,"currency":"CNY"}}`},
		{name: "amount", plaintext: `{"appid":"wx-app","mchid":"wx-mch","out_trade_no":"01ORDER","transaction_id":"4201","trade_state":"SUCCESS","amount":{"total":0,"currency":"CNY"}}`},
		{name: "currency", plaintext: `{"appid":"wx-app","mchid":"wx-mch","out_trade_no":"01ORDER","transaction_id":"4201","trade_state":"SUCCESS","amount":{"total":10000,"currency":"USD"}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := encryptNotificationResource(t, testAPIv3Key, tt.plaintext)
			body, err := json.Marshal(notification{ID: "notification-id", CreateTime: "2027-01-15T08:00:00+08:00", Resource: resource})
			require.NoError(t, err)
			timestamp := strconv.FormatInt(testNowUnix, 10)
			headers := http.Header{
				"Wechatpay-Timestamp": []string{timestamp},
				"Wechatpay-Nonce":     []string{"nonce"},
				"Wechatpay-Serial":    []string{"platform-serial"},
			}
			headers.Set("Wechatpay-Signature", signWechatMessage(t, keys.platformPrivate, timestamp+"\nnonce\n"+string(body)+"\n"))
			_, err = newTestAdapter(t, "https://api.mch.weixin.qq.com", keys).VerifyCallback(t.Context(), headers, body)
			require.Error(t, err)
		})
	}
}

func TestQueryPaymentUsesOutTradeNumberAndMapsOnlySuccess(t *testing.T) {
	keys := newTestKeys(t)
	states := []struct {
		state    string
		success  bool
		terminal bool
	}{
		{state: "SUCCESS", success: true, terminal: true},
		{state: "NOTPAY"},
		{state: "USERPAYING"},
		{state: "CLOSED", terminal: true},
		{state: "PAYERROR", terminal: true},
	}
	for _, tt := range states {
		t.Run(tt.state, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/v3/pay/transactions/out-trade-no/ORDER%2F1", r.URL.EscapedPath())
				require.Equal(t, "wx-mch", r.URL.Query().Get("mchid"))
				requireValidRequestAuthorization(t, keys.merchantPrivate, r, nil)
				body := fmt.Sprintf(`{"appid":"wx-app","mchid":"wx-mch","out_trade_no":"ORDER/1","transaction_id":"4200000001","trade_state":%q,"success_time":"2027-01-15T08:00:00+08:00","amount":{"total":10000,"currency":"CNY"}}`, tt.state)
				writeSignedWechatResponse(t, keys.platformPrivate, w, http.StatusOK, body)
			}))
			defer server.Close()

			fact, err := newTestAdapter(t, server.URL, keys).QueryPayment(t.Context(), "ORDER/1")
			require.NoError(t, err)
			require.Equal(t, tt.success, fact.Success)
			require.Equal(t, tt.terminal, fact.Terminal)
			require.Empty(t, fact.ProviderEventID)
			require.Equal(t, "ORDER/1", fact.ProviderOrderID)
			require.Equal(t, int64(10000), fact.AmountMinor)
		})
	}
}

func TestQueryPaymentRejectsMismatchedProviderOrder(t *testing.T) {
	keys := newTestKeys(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeSignedWechatResponse(t, keys.platformPrivate, w, http.StatusOK, `{"appid":"wx-app","mchid":"wx-mch","out_trade_no":"OTHER","trade_state":"SUCCESS","amount":{"total":10000,"currency":"CNY"}}`)
	}))
	defer server.Close()
	_, err := newTestAdapter(t, server.URL, keys).QueryPayment(t.Context(), "01ORDER")
	require.ErrorIs(t, err, payment.ErrPermanentProviderProtocol)
}

func TestQueryPaymentRejectsMissingRequiredTransactionFields(t *testing.T) {
	keys := newTestKeys(t)
	for _, tt := range []struct {
		name string
		body string
	}{
		{
			name: "successful transaction ID",
			body: `{"appid":"wx-app","mchid":"wx-mch","out_trade_no":"01ORDER","trade_state":"SUCCESS","amount":{"total":10000,"currency":"CNY"}}`,
		},
		{
			name: "known trade state",
			body: `{"appid":"wx-app","mchid":"wx-mch","out_trade_no":"01ORDER","transaction_id":"4200000001","trade_state":"UNKNOWN","amount":{"total":10000,"currency":"CNY"}}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeSignedWechatResponse(t, keys.platformPrivate, w, http.StatusOK, tt.body)
			}))
			defer server.Close()

			_, err := newTestAdapter(t, server.URL, keys).QueryPayment(t.Context(), "01ORDER")
			require.ErrorIs(t, err, payment.ErrPermanentProviderProtocol)
		})
	}
}

func TestRefundAndQueryRefund(t *testing.T) {
	keys := newTestKeys(t)
	var baseURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requireValidRequestAuthorization(t, keys.merchantPrivate, r, bodyBytes)
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v3/refund/domestic/refunds":
			var body refundRequest
			require.NoError(t, json.Unmarshal(bodyBytes, &body))
			require.Equal(t, "01ORDER", body.OutTradeNo)
			require.Equal(t, "refund-idem", body.OutRefundNo)
			require.Equal(t, "customer request", body.Reason)
			require.Equal(t, "https://merchant.example/wechat/refund-notify", body.NotifyURL)
			require.Equal(t, int64(3000), body.Amount.Refund)
			require.Equal(t, int64(10000), body.Amount.Total)
			require.Equal(t, "CNY", body.Amount.Currency)
			writeSignedWechatResponse(t, keys.platformPrivate, w, http.StatusOK, `{"refund_id":"5030000001","out_refund_no":"refund-idem","out_trade_no":"01ORDER","status":"PROCESSING","amount":{"refund":3000,"total":10000,"currency":"CNY"}}`)
		case r.Method == http.MethodGet && r.URL.EscapedPath() == "/v3/refund/domestic/refunds/refund-idem":
			writeSignedWechatResponse(t, keys.platformPrivate, w, http.StatusOK, `{"refund_id":"5030000001","out_refund_no":"refund-idem","out_trade_no":"01ORDER","status":"SUCCESS","success_time":"2027-01-15T08:00:00+08:00","amount":{"refund":3000,"total":10000,"currency":"CNY"}}`)
		default:
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	}))
	defer server.Close()
	baseURL = server.URL
	adapter := newTestAdapter(t, baseURL, keys)

	submission, err := adapter.Refund(t.Context(), payment.RefundInput{
		OrderID: "internal-order", ProviderOrderID: "01ORDER",
		AmountMinor: 3000, TotalAmountMinor: 10000, Currency: " cny ",
		Reason: "customer request", IdempotencyKey: "refund-idem",
	})
	require.NoError(t, err)
	require.Equal(t, "refund-idem", submission.ProviderRefundID)
	require.Equal(t, "processing", submission.Status)

	fact, err := adapter.QueryRefund(t.Context(), payment.RefundQueryInput{
		ProviderRefundID: "refund-idem", ProviderOrderID: "01ORDER", AmountMinor: 3000, Currency: "CNY",
	})
	require.NoError(t, err)
	require.True(t, fact.Success)
	require.Equal(t, "refund-idem", fact.ProviderRefundID)
	require.Equal(t, "01ORDER", fact.ProviderOrderID)
	require.Equal(t, int64(3000), fact.AmountMinor)
	require.Equal(t, "CNY", fact.Currency)
}

func TestRefundRejectsNonCNYBeforeCallingProvider(t *testing.T) {
	keys := newTestKeys(t)
	_, err := newTestAdapter(t, "http://127.0.0.1:1", keys).Refund(t.Context(), payment.RefundInput{
		OrderID: "internal-order", ProviderOrderID: "01ORDER", AmountMinor: 3000, TotalAmountMinor: 10000,
		Currency: "USD", IdempotencyKey: "refund-idem",
	})
	require.ErrorIs(t, err, payment.ErrPermanentProviderProtocol)
}

func TestRefundRequiresDomainOrderBinding(t *testing.T) {
	keys := newTestKeys(t)
	_, err := newTestAdapter(t, "http://127.0.0.1:1", keys).Refund(t.Context(), payment.RefundInput{
		ProviderOrderID: "01ORDER", AmountMinor: 3000, TotalAmountMinor: 10000,
		Currency: "CNY", IdempotencyKey: "refund-idem",
	})
	require.ErrorIs(t, err, payment.ErrPermanentProviderProtocol)
}

func TestRefundRejectsResponseMismatches(t *testing.T) {
	keys := newTestKeys(t)
	tests := []struct {
		name string
		body string
	}{
		{name: "out refund number", body: `{"refund_id":"5030000001","out_refund_no":"other-refund","out_trade_no":"01ORDER","status":"PROCESSING","amount":{"refund":3000,"total":10000,"currency":"CNY"}}`},
		{name: "out trade number", body: `{"refund_id":"5030000001","out_refund_no":"refund-idem","out_trade_no":"OTHER","status":"PROCESSING","amount":{"refund":3000,"total":10000,"currency":"CNY"}}`},
		{name: "refund amount", body: `{"refund_id":"5030000001","out_refund_no":"refund-idem","out_trade_no":"01ORDER","status":"PROCESSING","amount":{"refund":3001,"total":10000,"currency":"CNY"}}`},
		{name: "total amount", body: `{"refund_id":"5030000001","out_refund_no":"refund-idem","out_trade_no":"01ORDER","status":"PROCESSING","amount":{"refund":3000,"total":9999,"currency":"CNY"}}`},
		{name: "currency", body: `{"refund_id":"5030000001","out_refund_no":"refund-idem","out_trade_no":"01ORDER","status":"PROCESSING","amount":{"refund":3000,"total":10000,"currency":"USD"}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeSignedWechatResponse(t, keys.platformPrivate, w, http.StatusOK, tt.body)
			}))
			defer server.Close()
			_, err := newTestAdapter(t, server.URL, keys).Refund(t.Context(), payment.RefundInput{
				OrderID: "internal-order", ProviderOrderID: "01ORDER",
				AmountMinor: 3000, TotalAmountMinor: 10000, Currency: "CNY",
				IdempotencyKey: "refund-idem",
			})
			require.ErrorIs(t, err, payment.ErrPermanentProviderProtocol)
		})
	}
}

func TestQueryRefundMapsTerminalFailuresWithRawHash(t *testing.T) {
	keys := newTestKeys(t)
	for _, status := range []string{"CLOSED", "ABNORMAL"} {
		t.Run(status, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body := fmt.Sprintf(`{"refund_id":"5030000001","out_refund_no":"refund-idem","out_trade_no":"01ORDER","status":%q,"amount":{"refund":3000,"total":10000,"currency":"CNY"}}`, status)
				writeSignedWechatResponse(t, keys.platformPrivate, w, http.StatusOK, body)
			}))
			defer server.Close()
			fact, err := newTestAdapter(t, server.URL, keys).QueryRefund(t.Context(), payment.RefundQueryInput{
				ProviderRefundID: "refund-idem", ProviderOrderID: "01ORDER", AmountMinor: 3000, Currency: "CNY",
			})
			require.NoError(t, err)
			require.False(t, fact.Success)
			require.NotEmpty(t, fact.RawHash)
		})
	}
}

func TestQueryRefundRejectsInvalidMoneyAndProviderFields(t *testing.T) {
	keys := newTestKeys(t)
	tests := []struct {
		name string
		body string
	}{
		{name: "zero total", body: `{"refund_id":"5030000001","out_refund_no":"refund-idem","out_trade_no":"01ORDER","status":"PROCESSING","amount":{"refund":3000,"total":0,"currency":"CNY"}}`},
		{name: "refund exceeds total", body: `{"refund_id":"5030000001","out_refund_no":"refund-idem","out_trade_no":"01ORDER","status":"PROCESSING","amount":{"refund":10001,"total":10000,"currency":"CNY"}}`},
		{name: "non CNY", body: `{"refund_id":"5030000001","out_refund_no":"refund-idem","out_trade_no":"01ORDER","status":"PROCESSING","amount":{"refund":3000,"total":10000,"currency":"USD"}}`},
		{name: "missing provider refund ID", body: `{"out_refund_no":"refund-idem","out_trade_no":"01ORDER","status":"PROCESSING","amount":{"refund":3000,"total":10000,"currency":"CNY"}}`},
		{name: "mismatched out refund number", body: `{"refund_id":"5030000001","out_refund_no":"other","out_trade_no":"01ORDER","status":"PROCESSING","amount":{"refund":3000,"total":10000,"currency":"CNY"}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeSignedWechatResponse(t, keys.platformPrivate, w, http.StatusOK, tt.body)
			}))
			defer server.Close()
			_, err := newTestAdapter(t, server.URL, keys).QueryRefund(t.Context(), payment.RefundQueryInput{
				ProviderRefundID: "refund-idem", ProviderOrderID: "01ORDER", AmountMinor: 3000, Currency: "CNY",
			})
			require.ErrorIs(t, err, payment.ErrPermanentProviderProtocol)
		})
	}
}

func TestVerifyRefundCallbackValidatesMoneyAndProviderFields(t *testing.T) {
	keys := newTestKeys(t)
	valid := `{"refund_id":"5030000001","out_refund_no":"refund-idem","out_trade_no":"01ORDER","status":"SUCCESS","amount":{"refund":3000,"total":10000,"currency":"CNY"}}`
	headers, body := encryptedRefundCallback(t, keys.platformPrivate, valid)
	fact, err := newTestAdapter(t, "https://api.mch.weixin.qq.com", keys).VerifyRefundCallback(t.Context(), headers, body)
	require.NoError(t, err)
	require.True(t, fact.Success)
	require.Equal(t, "refund-idem", fact.ProviderRefundID)
	require.Equal(t, "01ORDER", fact.ProviderOrderID)
	require.Equal(t, int64(3000), fact.AmountMinor)
	require.Equal(t, "CNY", fact.Currency)

	tests := []struct {
		name      string
		plaintext string
	}{
		{name: "zero total", plaintext: `{"refund_id":"5030000001","out_refund_no":"refund-idem","out_trade_no":"01ORDER","status":"PROCESSING","amount":{"refund":3000,"total":0,"currency":"CNY"}}`},
		{name: "refund exceeds total", plaintext: `{"refund_id":"5030000001","out_refund_no":"refund-idem","out_trade_no":"01ORDER","status":"PROCESSING","amount":{"refund":10001,"total":10000,"currency":"CNY"}}`},
		{name: "non CNY", plaintext: `{"refund_id":"5030000001","out_refund_no":"refund-idem","out_trade_no":"01ORDER","status":"PROCESSING","amount":{"refund":3000,"total":10000,"currency":"USD"}}`},
		{name: "missing provider refund ID", plaintext: `{"out_refund_no":"refund-idem","out_trade_no":"01ORDER","status":"PROCESSING","amount":{"refund":3000,"total":10000,"currency":"CNY"}}`},
		{name: "missing order", plaintext: `{"refund_id":"5030000001","out_refund_no":"refund-idem","status":"PROCESSING","amount":{"refund":3000,"total":10000,"currency":"CNY"}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers, body := encryptedRefundCallback(t, keys.platformPrivate, tt.plaintext)
			_, err := newTestAdapter(t, "https://api.mch.weixin.qq.com", keys).VerifyRefundCallback(t.Context(), headers, body)
			require.ErrorIs(t, err, payment.ErrPermanentProviderProtocol)
		})
	}
}

func TestQueryRefundProcessingHasNoDefinitiveFailureHash(t *testing.T) {
	keys := newTestKeys(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeSignedWechatResponse(t, keys.platformPrivate, w, http.StatusOK, `{"refund_id":"5030000001","out_refund_no":"refund-idem","out_trade_no":"01ORDER","status":"PROCESSING","amount":{"refund":3000,"total":10000,"currency":"CNY"}}`)
	}))
	defer server.Close()
	fact, err := newTestAdapter(t, server.URL, keys).QueryRefund(t.Context(), payment.RefundQueryInput{
		ProviderRefundID: "refund-idem", ProviderOrderID: "01ORDER", AmountMinor: 3000, Currency: "CNY",
	})
	require.NoError(t, err)
	require.False(t, fact.Success)
	require.Empty(t, fact.RawHash)
}

func TestSuccessfulAPIResponseRequiresValidSignature(t *testing.T) {
	keys := newTestKeys(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := io.WriteString(w, `{"code_url":"weixin://unsigned"}`)
		require.NoError(t, err)
	}))
	defer server.Close()
	_, err := newTestAdapter(t, server.URL, keys).CreateQRCode(t.Context(), payment.CheckoutInput{
		OrderPublicID: "01ORDER", AmountMinor: 10000, Currency: "CNY", Description: "test",
	})
	require.ErrorIs(t, err, payment.ErrInvalidSignature)
}

func TestAPIResponseBodyLimit(t *testing.T) {
	keys := newTestKeys(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := io.WriteString(w, strings.Repeat("x", 65))
		require.NoError(t, err)
	}))
	defer server.Close()
	adapter := newTestAdapter(t, server.URL, keys)
	adapter.maxResponseBytes = 64
	_, err := adapter.CreateQRCode(t.Context(), payment.CheckoutInput{
		OrderPublicID: "01ORDER", AmountMinor: 10000, Currency: "CNY", Description: "test",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, payment.ErrPermanentProviderProtocol)
}

func TestIdentity(t *testing.T) {
	keys := newTestKeys(t)
	identity, err := newTestAdapter(t, "https://api.mch.weixin.qq.com", keys).Identity(t.Context())
	require.NoError(t, err)
	require.Equal(t, "wx-mch", identity.MerchantID)
	require.Equal(t, "wx-app", identity.ApplicationID)
}

func TestPlatformPublicKeySecretTrimsSerial(t *testing.T) {
	require.Equal(t, "wechat_platform_public_key/serial-001", PlatformPublicKeySecret(" serial-001 "))
}

func TestNewRequiresProductionDependencies(t *testing.T) {
	valid := Config{
		Secrets: &payment.StaticSecretProvider{Secrets: map[string]string{}},
		Client:  &http.Client{Timeout: 2 * time.Second}, BaseURL: "https://api.mch.weixin.qq.com",
		AppID: "wx-app", MerchantID: "wx-mch", MerchantSerial: "serial",
		NotifyURL: "https://merchant.example/notify", RefundNotifyURL: "https://merchant.example/refund",
		Now: time.Now, CallbackMaxAge: 5 * time.Minute, MaxResponseBytes: 1024,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	_, err := New(valid)
	require.NoError(t, err)
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{name: "secrets", mutate: func(c *Config) { c.Secrets = nil }},
		{name: "client", mutate: func(c *Config) { c.Client = nil }},
		{name: "base URL", mutate: func(c *Config) { c.BaseURL = "" }},
		{name: "application ID", mutate: func(c *Config) { c.AppID = "" }},
		{name: "merchant ID", mutate: func(c *Config) { c.MerchantID = "" }},
		{name: "merchant serial", mutate: func(c *Config) { c.MerchantSerial = "" }},
		{name: "notify URL", mutate: func(c *Config) { c.NotifyURL = "" }},
		{name: "refund notify URL", mutate: func(c *Config) { c.RefundNotifyURL = "" }},
		{name: "clock", mutate: func(c *Config) { c.Now = nil }},
		{name: "callback max age", mutate: func(c *Config) { c.CallbackMaxAge = 0 }},
		{name: "response size", mutate: func(c *Config) { c.MaxResponseBytes = 0 }},
		{name: "logger", mutate: func(c *Config) { c.Logger = nil }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := valid
			tt.mutate(&cfg)
			_, err := New(cfg)
			require.Error(t, err)
		})
	}
}

func TestProviderErrorsExposeRetryClassification(t *testing.T) {
	keys := newTestKeys(t)
	for _, tt := range []struct {
		name       string
		statusCode int
		code       string
		kind       payment.ProviderErrorKind
		marker     error
	}{
		{name: "HTTP 503", statusCode: http.StatusServiceUnavailable, code: "SYSTEM_ERROR", kind: payment.ProviderErrorRetryable, marker: payment.ErrRetryableProvider},
		{name: "HTTP 429", statusCode: http.StatusTooManyRequests, code: "FREQUENCY_LIMITED", kind: payment.ProviderErrorRetryable, marker: payment.ErrRetryableProvider},
		{name: "HTTP 400", statusCode: http.StatusBadRequest, code: "PARAM_ERROR", kind: payment.ProviderErrorPermanent, marker: payment.ErrPermanentProviderProtocol},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, err := io.WriteString(w, fmt.Sprintf(`{"code":%q,"message":"sensitive-response-body-marker"}`, tt.code))
				require.NoError(t, err)
			}))
			defer server.Close()

			_, err := newTestAdapter(t, server.URL, keys).CreateQRCode(t.Context(), payment.CheckoutInput{
				OrderPublicID: "01ORDER", AmountMinor: 10000, Currency: "CNY", Description: "test",
			})
			require.ErrorIs(t, err, tt.marker)
			require.NotContains(t, err.Error(), "sensitive-response-body-marker")
			var providerErr *payment.ProviderError
			require.ErrorAs(t, err, &providerErr)
			require.Equal(t, payment.ProviderWeChat, providerErr.Provider)
			require.Equal(t, "POST /v3/pay/transactions/native", providerErr.Operation)
			require.Equal(t, tt.kind, providerErr.Kind)
			require.Equal(t, tt.statusCode, providerErr.HTTPStatus)
			require.Equal(t, tt.code, providerErr.Code)
		})
	}
}

func TestProviderTransportTimeoutIsRetryableAndPreservesCause(t *testing.T) {
	keys := newTestKeys(t)
	adapter := newTestAdapter(t, "https://api.mch.weixin.qq.com", keys)
	adapter.httpClient = &http.Client{
		Timeout: time.Second,
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("transport timeout: %w", context.DeadlineExceeded)
		}),
	}

	_, err := adapter.CreateQRCode(t.Context(), payment.CheckoutInput{
		OrderPublicID: "01ORDER", AmountMinor: 10000, Currency: "CNY", Description: "test",
	})
	require.ErrorIs(t, err, payment.ErrRetryableProvider)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	var providerErr *payment.ProviderError
	require.ErrorAs(t, err, &providerErr)
	require.Equal(t, payment.ProviderErrorRetryable, providerErr.Kind)
}

func TestRequestCanonicalURLPreservesEncodedPathAndQuery(t *testing.T) {
	u, err := url.Parse("https://example.com/v3/pay/transactions/out-trade-no/ORDER%2F1?mchid=wx-mch")
	require.NoError(t, err)
	require.Equal(t, "/v3/pay/transactions/out-trade-no/ORDER%2F1", u.EscapedPath())
}
