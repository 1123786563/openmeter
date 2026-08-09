// Package wechat implements the WeChat Pay API v3 Native payment adapter.
package wechat

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/openmeterio/openmeter/openmeter/commerce/payment"
)

const (
	SecretKeyMerchantPrivateKey = "wechat_merchant_private_key"
	SecretKeyAPIv3              = "wechat_api_v3_key"
	secretKeyPlatformPublicKey  = "wechat_platform_public_key"
)

// PlatformPublicKeySecret returns the serial-specific logical key used for
// WeChat platform certificate rotation.
func PlatformPublicKeySecret(serial string) string {
	return secretKeyPlatformPublicKey + "/" + strings.TrimSpace(serial)
}

// Config wires the production dependencies and non-secret WeChat identities.
type Config struct {
	Secrets          payment.SecretProvider
	Client           *http.Client
	BaseURL          string
	AppID            string
	MerchantID       string
	MerchantSerial   string
	NotifyURL        string
	RefundNotifyURL  string
	Now              func() time.Time
	CallbackMaxAge   time.Duration
	MaxResponseBytes int64
	Logger           *slog.Logger
}

// Adapter implements the WeChat Pay API v3 Native workflow.
type Adapter struct {
	secrets          payment.SecretProvider
	httpClient       *http.Client
	baseURL          string
	appID            string
	merchantID       string
	merchantSerial   string
	notifyURL        string
	refundNotifyURL  string
	now              func() time.Time
	callbackMaxAge   time.Duration
	maxResponseBytes int64
}

// New creates a production WeChat Pay adapter. Callers own HTTP timeout and
// transport configuration through the injected client.
func New(cfg Config) (*Adapter, error) {
	if cfg.Secrets == nil {
		return nil, errors.New("wechat adapter: secrets provider is required")
	}
	if cfg.Client == nil {
		return nil, errors.New("wechat adapter: HTTP client is required")
	}
	if cfg.Client.Timeout <= 0 {
		return nil, errors.New("wechat adapter: HTTP client total timeout must be positive")
	}
	parsedBaseURL, err := url.Parse(strings.TrimSpace(cfg.BaseURL))
	if err != nil || parsedBaseURL.Scheme == "" || parsedBaseURL.Host == "" {
		return nil, errors.New("wechat adapter: base URL must be absolute")
	}
	if strings.TrimSpace(cfg.AppID) == "" {
		return nil, errors.New("wechat adapter: application ID is required")
	}
	if strings.TrimSpace(cfg.MerchantID) == "" {
		return nil, errors.New("wechat adapter: merchant ID is required")
	}
	if strings.TrimSpace(cfg.MerchantSerial) == "" {
		return nil, errors.New("wechat adapter: merchant serial is required")
	}
	if strings.TrimSpace(cfg.NotifyURL) == "" {
		return nil, errors.New("wechat adapter: notify URL is required")
	}
	if strings.TrimSpace(cfg.RefundNotifyURL) == "" {
		return nil, errors.New("wechat adapter: refund notify URL is required")
	}
	if cfg.Now == nil {
		return nil, errors.New("wechat adapter: clock is required")
	}
	if cfg.CallbackMaxAge <= 0 {
		return nil, errors.New("wechat adapter: callback max age must be positive")
	}
	if cfg.MaxResponseBytes <= 0 {
		return nil, errors.New("wechat adapter: max response bytes must be positive")
	}
	if cfg.Logger == nil {
		return nil, errors.New("wechat adapter: logger is required")
	}

	return &Adapter{
		secrets:          cfg.Secrets,
		httpClient:       cfg.Client,
		baseURL:          strings.TrimRight(parsedBaseURL.String(), "/"),
		appID:            strings.TrimSpace(cfg.AppID),
		merchantID:       strings.TrimSpace(cfg.MerchantID),
		merchantSerial:   strings.TrimSpace(cfg.MerchantSerial),
		notifyURL:        strings.TrimSpace(cfg.NotifyURL),
		refundNotifyURL:  strings.TrimSpace(cfg.RefundNotifyURL),
		now:              cfg.Now,
		callbackMaxAge:   cfg.CallbackMaxAge,
		maxResponseBytes: cfg.MaxResponseBytes,
	}, nil
}

// Name returns the provider identifier.
func (a *Adapter) Name() payment.Provider { return payment.ProviderWeChat }

// Identity returns the non-secret merchant and application identities used to
// bind later provider facts to the payment attempt.
func (a *Adapter) Identity() (merchantID, applicationID string) {
	return a.merchantID, a.appID
}

// CreateQRCode creates a Native payment and returns WeChat's code_url.
func (a *Adapter) CreateQRCode(ctx context.Context, input payment.CheckoutInput) (payment.CheckoutFact, error) {
	currency := strings.ToUpper(strings.TrimSpace(input.Currency))
	if strings.TrimSpace(input.OrderPublicID) == "" || input.AmountMinor <= 0 || currency != "CNY" || strings.TrimSpace(input.Description) == "" {
		return payment.CheckoutFact{}, fmt.Errorf("%w: invalid Native payment input", payment.ErrPermanentProviderProtocol)
	}
	request := nativeCreateRequest{
		AppID:       a.appID,
		MchID:       a.merchantID,
		Description: input.Description,
		OutTradeNo:  input.OrderPublicID,
		NotifyURL:   a.notifyURL,
		Amount: amount{
			Total:    input.AmountMinor,
			Currency: currency,
		},
	}
	var response nativeCreateResponse
	if _, err := a.doJSON(ctx, http.MethodPost, "/v3/pay/transactions/native", nil, request, &response); err != nil {
		return payment.CheckoutFact{}, err
	}
	if strings.TrimSpace(response.CodeURL) == "" {
		return payment.CheckoutFact{}, fmt.Errorf("%w: Native response has no code_url", payment.ErrPermanentProviderProtocol)
	}
	return payment.CheckoutFact{
		Provider:        payment.ProviderWeChat,
		ProviderOrderID: input.OrderPublicID,
		QRCodeURL:       response.CodeURL,
	}, nil
}

// QueryPayment queries a transaction by out_trade_no. Only SUCCESS is mapped
// to a successful fact.
func (a *Adapter) QueryPayment(ctx context.Context, providerOrderID string) (payment.PaymentFact, error) {
	providerOrderID = strings.TrimSpace(providerOrderID)
	if providerOrderID == "" {
		return payment.PaymentFact{}, fmt.Errorf("%w: provider order ID is required", payment.ErrPermanentProviderProtocol)
	}
	path := "/v3/pay/transactions/out-trade-no/" + url.PathEscape(providerOrderID)
	query := url.Values{"mchid": []string{a.merchantID}}
	var response transaction
	rawBody, err := a.doJSON(ctx, http.MethodGet, path, query, nil, &response)
	if err != nil {
		return payment.PaymentFact{}, err
	}
	if err := a.validateTransaction(response); err != nil {
		return payment.PaymentFact{}, err
	}
	if response.OutTradeNo != providerOrderID {
		return payment.PaymentFact{}, fmt.Errorf("%w: transaction out_trade_no mismatch", payment.ErrPermanentProviderProtocol)
	}
	payload, err := payment.ExtractSignedPayload(rawBody)
	if err != nil {
		return payment.PaymentFact{}, fmt.Errorf("%w: invalid transaction response", payment.ErrPermanentProviderProtocol)
	}
	return payment.PaymentFact{
		Provider:          payment.ProviderWeChat,
		ProviderOrderID:   response.OutTradeNo,
		ProviderPaymentID: response.TransactionID,
		ProviderEventID:   response.TransactionID,
		MerchantID:        response.MchID,
		ApplicationID:     response.AppID,
		AmountMinor:       response.Amount.Total,
		Currency:          response.Amount.Currency,
		Success:           response.TradeState == "SUCCESS",
		RawHash:           hashBody(rawBody),
		Timestamp:         parseProviderTime(response.SuccessTime, a.now()),
		SignedPayload:     payload,
	}, nil
}

// VerifyCallback verifies the raw notification, checks freshness, decrypts its
// resource, and validates the configured merchant and application identities.
func (a *Adapter) VerifyCallback(ctx context.Context, headers http.Header, body []byte) (payment.PaymentFact, error) {
	if err := a.verifyHTTPMessage(ctx, headers, body); err != nil {
		return payment.PaymentFact{}, err
	}
	callbackTime, err := a.validateCallbackTime(headers.Get("Wechatpay-Timestamp"))
	if err != nil {
		return payment.PaymentFact{}, err
	}

	var envelope notification
	if err := json.Unmarshal(body, &envelope); err != nil {
		return payment.PaymentFact{}, fmt.Errorf("%w: invalid notification JSON", payment.ErrPermanentProviderProtocol)
	}
	if strings.TrimSpace(envelope.ID) == "" {
		return payment.PaymentFact{}, fmt.Errorf("%w: notification ID is required", payment.ErrPermanentProviderProtocol)
	}
	plaintext, err := a.decryptNotificationResource(ctx, envelope.Resource)
	if err != nil {
		return payment.PaymentFact{}, err
	}
	var resource transaction
	if err := json.Unmarshal(plaintext, &resource); err != nil {
		return payment.PaymentFact{}, fmt.Errorf("%w: invalid decrypted transaction", payment.ErrPermanentProviderProtocol)
	}
	if err := a.validateTransaction(resource); err != nil {
		return payment.PaymentFact{}, err
	}
	payload, err := payment.ExtractSignedPayload(plaintext)
	if err != nil {
		return payment.PaymentFact{}, fmt.Errorf("%w: invalid decrypted transaction", payment.ErrPermanentProviderProtocol)
	}
	payload["notification_id"] = envelope.ID
	payload["event_type"] = envelope.EventType
	payload["create_time"] = envelope.CreateTime

	return payment.PaymentFact{
		Provider:          payment.ProviderWeChat,
		ProviderOrderID:   resource.OutTradeNo,
		ProviderPaymentID: resource.TransactionID,
		ProviderEventID:   envelope.ID,
		MerchantID:        resource.MchID,
		ApplicationID:     resource.AppID,
		AmountMinor:       resource.Amount.Total,
		Currency:          resource.Amount.Currency,
		Success:           resource.TradeState == "SUCCESS",
		RawHash:           hashBody(body),
		Timestamp:         callbackTime,
		SignedPayload:     payload,
	}, nil
}

// Refund submits an original-route refund. out_refund_no is the caller's
// idempotency key and remains the stable identifier used by QueryRefund.
func (a *Adapter) Refund(ctx context.Context, input payment.RefundInput) (payment.RefundSubmission, error) {
	currency := strings.ToUpper(strings.TrimSpace(input.Currency))
	orderID := strings.TrimSpace(input.OrderID)
	outTradeNo := strings.TrimSpace(input.ProviderOrderID)
	outRefundNo := strings.TrimSpace(input.IdempotencyKey)
	if orderID == "" || outTradeNo == "" || outRefundNo == "" || input.AmountMinor <= 0 || input.TotalAmountMinor <= 0 || input.AmountMinor > input.TotalAmountMinor || currency != "CNY" {
		return payment.RefundSubmission{}, fmt.Errorf("%w: invalid refund input", payment.ErrPermanentProviderProtocol)
	}
	request := refundRequest{
		OutTradeNo:  outTradeNo,
		OutRefundNo: outRefundNo,
		Reason:      input.Reason,
		NotifyURL:   a.refundNotifyURL,
		Amount: refundAmount{
			Refund:   input.AmountMinor,
			Total:    input.TotalAmountMinor,
			Currency: currency,
		},
	}
	var response refund
	if _, err := a.doJSON(ctx, http.MethodPost, "/v3/refund/domestic/refunds", nil, request, &response); err != nil {
		return payment.RefundSubmission{}, err
	}
	if err := validateRefund(response, outRefundNo, outTradeNo); err != nil {
		return payment.RefundSubmission{}, err
	}
	if response.Amount.Refund != input.AmountMinor || response.Amount.Total != input.TotalAmountMinor || response.Amount.Currency != currency {
		return payment.RefundSubmission{}, fmt.Errorf("%w: refund response does not match request", payment.ErrPermanentProviderProtocol)
	}
	return payment.RefundSubmission{
		Provider:         payment.ProviderWeChat,
		ProviderRefundID: response.OutRefundNo,
		Status:           strings.ToLower(response.Status),
	}, nil
}

// QueryRefund queries a refund by out_refund_no and binds the response to the
// persisted provider order, amount, and currency. PROCESSING remains a
// non-terminal fact; CLOSED and ABNORMAL are definitive failures.
func (a *Adapter) QueryRefund(ctx context.Context, input payment.RefundQueryInput) (payment.RefundFact, error) {
	providerRefundID := strings.TrimSpace(input.ProviderRefundID)
	providerOrderID := strings.TrimSpace(input.ProviderOrderID)
	currency := strings.ToUpper(strings.TrimSpace(input.Currency))
	if providerRefundID == "" || providerOrderID == "" || input.AmountMinor <= 0 || currency != "CNY" {
		return payment.RefundFact{}, fmt.Errorf("%w: invalid refund query input", payment.ErrPermanentProviderProtocol)
	}
	path := "/v3/refund/domestic/refunds/" + url.PathEscape(providerRefundID)
	var response refund
	rawBody, err := a.doJSON(ctx, http.MethodGet, path, nil, nil, &response)
	if err != nil {
		return payment.RefundFact{}, err
	}
	if err := validateRefund(response, providerRefundID, providerOrderID); err != nil {
		return payment.RefundFact{}, err
	}
	if response.Amount.Refund != input.AmountMinor || response.Amount.Currency != currency {
		return payment.RefundFact{}, fmt.Errorf("%w: refund query response does not match request", payment.ErrPermanentProviderProtocol)
	}
	payload, err := payment.ExtractSignedPayload(rawBody)
	if err != nil {
		return payment.RefundFact{}, fmt.Errorf("%w: invalid refund response", payment.ErrPermanentProviderProtocol)
	}
	rawHash := ""
	if response.Status == "SUCCESS" || response.Status == "CLOSED" || response.Status == "ABNORMAL" {
		rawHash = hashBody(rawBody)
	}
	return payment.RefundFact{
		Provider:         payment.ProviderWeChat,
		ProviderRefundID: response.OutRefundNo,
		ProviderOrderID:  response.OutTradeNo,
		AmountMinor:      response.Amount.Refund,
		Currency:         response.Amount.Currency,
		Success:          response.Status == "SUCCESS",
		RawHash:          rawHash,
		Timestamp:        parseProviderTime(response.SuccessTime, a.now()),
		SignedPayload:    payload,
	}, nil
}

// VerifyRefundCallback applies the same v3 signature, freshness, and AES-GCM
// checks to refund notifications.
func (a *Adapter) VerifyRefundCallback(ctx context.Context, headers http.Header, body []byte) (payment.RefundFact, error) {
	if err := a.verifyHTTPMessage(ctx, headers, body); err != nil {
		return payment.RefundFact{}, err
	}
	callbackTime, err := a.validateCallbackTime(headers.Get("Wechatpay-Timestamp"))
	if err != nil {
		return payment.RefundFact{}, err
	}
	var envelope notification
	if err := json.Unmarshal(body, &envelope); err != nil {
		return payment.RefundFact{}, fmt.Errorf("%w: invalid refund notification", payment.ErrPermanentProviderProtocol)
	}
	plaintext, err := a.decryptNotificationResource(ctx, envelope.Resource)
	if err != nil {
		return payment.RefundFact{}, err
	}
	var resource refund
	if err := json.Unmarshal(plaintext, &resource); err != nil {
		return payment.RefundFact{}, fmt.Errorf("%w: invalid decrypted refund", payment.ErrPermanentProviderProtocol)
	}
	if err := validateRefund(resource, "", ""); err != nil {
		return payment.RefundFact{}, err
	}
	payload, err := payment.ExtractSignedPayload(plaintext)
	if err != nil {
		return payment.RefundFact{}, fmt.Errorf("%w: invalid decrypted refund", payment.ErrPermanentProviderProtocol)
	}
	payload["notification_id"] = envelope.ID
	payload["event_type"] = envelope.EventType
	rawHash := ""
	if resource.Status == "SUCCESS" || resource.Status == "CLOSED" || resource.Status == "ABNORMAL" {
		rawHash = hashBody(body)
	}
	return payment.RefundFact{
		Provider:         payment.ProviderWeChat,
		ProviderRefundID: resource.OutRefundNo,
		ProviderOrderID:  resource.OutTradeNo,
		AmountMinor:      resource.Amount.Refund,
		Currency:         resource.Amount.Currency,
		Success:          resource.Status == "SUCCESS",
		RawHash:          rawHash,
		Timestamp:        callbackTime,
		SignedPayload:    payload,
	}, nil
}

func (a *Adapter) validateTransaction(value transaction) error {
	if value.AppID != a.appID {
		return fmt.Errorf("%w: transaction application ID mismatch", payment.ErrPermanentProviderProtocol)
	}
	if value.MchID != a.merchantID {
		return fmt.Errorf("%w: transaction merchant ID mismatch", payment.ErrPermanentProviderProtocol)
	}
	if value.OutTradeNo == "" || value.Amount.Total <= 0 {
		return fmt.Errorf("%w: transaction order or amount is invalid", payment.ErrPermanentProviderProtocol)
	}
	if value.Amount.Currency != "CNY" {
		return fmt.Errorf("%w: transaction currency must be CNY", payment.ErrPermanentProviderProtocol)
	}
	return nil
}

func validateRefund(value refund, expectedOutRefundNo, expectedOutTradeNo string) error {
	if value.RefundID == "" || value.OutRefundNo == "" || value.OutTradeNo == "" || value.Status == "" {
		return fmt.Errorf("%w: incomplete refund provider identity", payment.ErrPermanentProviderProtocol)
	}
	if expectedOutRefundNo != "" && value.OutRefundNo != expectedOutRefundNo {
		return fmt.Errorf("%w: refund out_refund_no mismatch", payment.ErrPermanentProviderProtocol)
	}
	if expectedOutTradeNo != "" && value.OutTradeNo != expectedOutTradeNo {
		return fmt.Errorf("%w: refund out_trade_no mismatch", payment.ErrPermanentProviderProtocol)
	}
	if value.Amount.Total <= 0 || value.Amount.Refund <= 0 || value.Amount.Refund > value.Amount.Total {
		return fmt.Errorf("%w: invalid refund amount", payment.ErrPermanentProviderProtocol)
	}
	if value.Amount.Currency != "CNY" {
		return fmt.Errorf("%w: refund currency must be CNY", payment.ErrPermanentProviderProtocol)
	}
	return nil
}

func (a *Adapter) validateCallbackTime(timestamp string) (time.Time, error) {
	seconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: invalid callback timestamp", payment.ErrInvalidSignature)
	}
	callbackTime := time.Unix(seconds, 0)
	difference := a.now().Sub(callbackTime)
	if difference < 0 {
		difference = -difference
	}
	if difference > a.callbackMaxAge {
		return time.Time{}, fmt.Errorf("%w: callback timestamp is outside the accepted window", payment.ErrInvalidSignature)
	}
	return callbackTime, nil
}

func (a *Adapter) decryptNotificationResource(ctx context.Context, resource encryptedResource) ([]byte, error) {
	apiKey, err := a.secrets.Get(ctx, SecretKeyAPIv3)
	if err != nil {
		return nil, fmt.Errorf("wechat: get API v3 key: %w", err)
	}
	plaintext, err := decryptResource(apiKey, resource)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot decrypt notification resource", payment.ErrPermanentProviderProtocol)
	}
	return plaintext, nil
}

func hashBody(body []byte) string {
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}

func parseProviderTime(value string, fallback time.Time) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return fallback
	}
	return parsed
}

var _ payment.ProviderAdapter = (*Adapter)(nil)
