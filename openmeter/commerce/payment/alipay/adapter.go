// Package alipay implements Alipay face-to-face payments using the RSA2
// OpenAPI protocol. Secrets are resolved for each operation through the
// configured SecretProvider and never retained in adapter state.
package alipay

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/openmeterio/openmeter/openmeter/commerce/payment"
)

const (
	SecretKeyAppPrivateKey   = "alipay_app_private_key"
	SecretKeyAlipayPublicKey = "alipay_public_key"
)

// Config wires production dependencies and non-secret Alipay identities.
type Config struct {
	Secrets          payment.SecretProvider
	Client           *http.Client
	GatewayURL       string
	AppID            string
	SellerID         string
	NotifyURL        string
	Now              func() time.Time
	MaxResponseBytes int64
	Logger           *slog.Logger
}

// Adapter implements the Alipay face-to-face payment workflow.
type Adapter struct {
	secrets          payment.SecretProvider
	httpClient       *http.Client
	gatewayURL       string
	appID            string
	sellerID         string
	notifyURL        string
	now              func() time.Time
	maxResponseBytes int64
	logger           *slog.Logger
}

// New creates a production Alipay adapter. The injected HTTP client must have
// a positive total timeout; the caller owns its transport configuration.
func New(cfg Config) (*Adapter, error) {
	if cfg.Secrets == nil {
		return nil, errors.New("alipay adapter: secrets provider is required")
	}
	if cfg.Client == nil {
		return nil, errors.New("alipay adapter: HTTP client is required")
	}
	if cfg.Client.Timeout <= 0 {
		return nil, errors.New("alipay adapter: HTTP client total timeout must be positive")
	}
	parsedGateway, err := url.Parse(strings.TrimSpace(cfg.GatewayURL))
	if err != nil || parsedGateway.Scheme == "" || parsedGateway.Host == "" {
		return nil, errors.New("alipay adapter: gateway URL must be absolute")
	}
	if strings.TrimSpace(cfg.AppID) == "" {
		return nil, errors.New("alipay adapter: application ID is required")
	}
	if strings.TrimSpace(cfg.SellerID) == "" {
		return nil, errors.New("alipay adapter: seller ID is required")
	}
	notifyURL, err := url.Parse(strings.TrimSpace(cfg.NotifyURL))
	if err != nil || notifyURL.Scheme == "" || notifyURL.Host == "" {
		return nil, errors.New("alipay adapter: notify URL must be absolute")
	}
	if cfg.Now == nil {
		return nil, errors.New("alipay adapter: clock is required")
	}
	if cfg.MaxResponseBytes <= 0 {
		return nil, errors.New("alipay adapter: max response bytes must be positive")
	}
	if cfg.Logger == nil {
		return nil, errors.New("alipay adapter: logger is required")
	}
	if err := validateConfiguredKeyMaterial(cfg.Secrets); err != nil {
		return nil, err
	}
	return &Adapter{
		secrets:          cfg.Secrets,
		httpClient:       cfg.Client,
		gatewayURL:       parsedGateway.String(),
		appID:            strings.TrimSpace(cfg.AppID),
		sellerID:         strings.TrimSpace(cfg.SellerID),
		notifyURL:        notifyURL.String(),
		now:              cfg.Now,
		maxResponseBytes: cfg.MaxResponseBytes,
		logger:           cfg.Logger,
	}, nil
}

func validateConfiguredKeyMaterial(secrets payment.SecretProvider) error {
	ctx := context.Background()
	privateKeyPEM, err := secrets.Get(ctx, SecretKeyAppPrivateKey)
	if err != nil {
		return errors.New("alipay adapter: application private key is unavailable")
	}
	privateKey, err := parseRSAPrivateKey([]byte(privateKeyPEM))
	if err != nil || privateKey.Validate() != nil {
		return errors.New("alipay adapter: application private key is invalid")
	}

	publicKeyPEM, err := secrets.Get(ctx, SecretKeyAlipayPublicKey)
	if err != nil {
		return errors.New("alipay adapter: Alipay public key is unavailable")
	}
	if _, err := parseRSAPublicKey([]byte(publicKeyPEM)); err != nil {
		return errors.New("alipay adapter: Alipay public key is invalid")
	}
	return nil
}

// Name returns the provider identifier.
func (a *Adapter) Name() payment.Provider { return payment.ProviderAlipay }

// Identity returns the non-secret identities used to bind signed facts to the
// payment attempt.
func (a *Adapter) Identity(context.Context) (payment.ProviderIdentity, error) {
	return payment.ProviderIdentity{MerchantID: a.sellerID, ApplicationID: a.appID}, nil
}

// CreateQRCode calls alipay.trade.precreate and returns its signed qr_code.
func (a *Adapter) CreateQRCode(ctx context.Context, input payment.CheckoutInput) (payment.CheckoutFact, error) {
	orderID := strings.TrimSpace(input.OrderPublicID)
	currency := strings.ToUpper(strings.TrimSpace(input.Currency))
	subject := strings.TrimSpace(input.Description)
	if orderID == "" || input.AmountMinor <= 0 || currency != "CNY" || subject == "" {
		return payment.CheckoutFact{}, fmt.Errorf("%w: invalid Alipay precreate input", payment.ErrPermanentProviderProtocol)
	}
	request := precreateRequest{OutTradeNo: orderID, Amount: formatAmountMinor(input.AmountMinor), Subject: subject}
	var response precreateResponse
	if _, err := a.call(ctx, "alipay.trade.precreate", "alipay_trade_precreate_response", request, &response); err != nil {
		return payment.CheckoutFact{}, err
	}
	if err := validateProviderSuccess("alipay.trade.precreate", response.providerResponse); err != nil {
		return payment.CheckoutFact{}, err
	}
	if response.OutTradeNo != orderID || strings.TrimSpace(response.QRCode) == "" {
		return payment.CheckoutFact{}, permanentProviderError("alipay.trade.precreate", "response does not match the order", payment.ErrPermanentProviderProtocol)
	}
	return payment.CheckoutFact{
		Provider:        payment.ProviderAlipay,
		ProviderOrderID: response.OutTradeNo,
		QRCodeURL:       response.QRCode,
	}, nil
}

// VerifyCallback validates the RSA2 signature before interpreting any signed
// business field.
func (a *Adapter) VerifyCallback(ctx context.Context, _ http.Header, body []byte) (payment.PaymentFact, error) {
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return payment.PaymentFact{}, fmt.Errorf("%w: invalid callback form", payment.ErrPermanentProviderProtocol)
	}
	signature := values.Get("sign")
	if signature == "" {
		return payment.PaymentFact{}, fmt.Errorf("%w: callback is missing sign", payment.ErrInvalidSignature)
	}
	if err := a.verifySignature(ctx, []byte(notificationSignContent(values)), signature); err != nil {
		return payment.PaymentFact{}, err
	}
	if values.Get("sign_type") != "RSA2" {
		return payment.PaymentFact{}, fmt.Errorf("%w: callback sign_type must be RSA2", payment.ErrPermanentProviderProtocol)
	}
	if values.Get("app_id") != a.appID {
		return payment.PaymentFact{}, fmt.Errorf("%w: callback application ID mismatch", payment.ErrPermanentProviderProtocol)
	}
	if values.Get("seller_id") != a.sellerID {
		return payment.PaymentFact{}, fmt.Errorf("%w: callback seller ID mismatch", payment.ErrPermanentProviderProtocol)
	}
	orderID := strings.TrimSpace(values.Get("out_trade_no"))
	if orderID == "" {
		return payment.PaymentFact{}, fmt.Errorf("%w: callback order ID is required", payment.ErrPermanentProviderProtocol)
	}
	amount, err := parseAmountMinor(values.Get("total_amount"))
	if err != nil {
		return payment.PaymentFact{}, fmt.Errorf("%w: callback amount is invalid", payment.ErrPermanentProviderProtocol)
	}
	status := values.Get("trade_status")
	if !validTradeStatus(status) {
		return payment.PaymentFact{}, fmt.Errorf("%w: callback trade status is invalid", payment.ErrPermanentProviderProtocol)
	}
	if successfulTradeStatus(status) && strings.TrimSpace(values.Get("trade_no")) == "" {
		return payment.PaymentFact{}, fmt.Errorf("%w: successful callback trade number is required", payment.ErrPermanentProviderProtocol)
	}
	timestamp, err := time.ParseInLocation("2006-01-02 15:04:05", values.Get("notify_time"), a.now().Location())
	if err != nil {
		return payment.PaymentFact{}, fmt.Errorf("%w: callback notify_time is invalid", payment.ErrPermanentProviderProtocol)
	}
	rawHash := hashBody(body)
	eventID := strings.TrimSpace(values.Get("notify_id"))
	if eventID == "" {
		a.logger.WarnContext(ctx, "Alipay callback has no notify_id", "provider", payment.ProviderAlipay, "raw_hash", rawHash)
	}
	return payment.PaymentFact{
		Provider:          payment.ProviderAlipay,
		ProviderOrderID:   orderID,
		ProviderPaymentID: values.Get("trade_no"),
		ProviderEventID:   eventID,
		MerchantID:        values.Get("seller_id"),
		ApplicationID:     values.Get("app_id"),
		AmountMinor:       amount,
		Currency:          "CNY",
		Success:           successfulTradeStatus(status),
		Terminal:          terminalTradeStatus(status),
		RawHash:           rawHash,
		Timestamp:         timestamp,
		SignedPayload:     valuesToMap(values),
	}, nil
}

// QueryPayment calls alipay.trade.query. Pending and closed transactions are
// returned as verified non-success facts.
func (a *Adapter) QueryPayment(ctx context.Context, providerOrderID string) (payment.PaymentFact, error) {
	providerOrderID = strings.TrimSpace(providerOrderID)
	if providerOrderID == "" {
		return payment.PaymentFact{}, fmt.Errorf("%w: provider order ID is required", payment.ErrPermanentProviderProtocol)
	}
	var response tradeQueryResponse
	rawResponse, err := a.call(ctx, "alipay.trade.query", "alipay_trade_query_response", tradeQueryRequest{OutTradeNo: providerOrderID}, &response)
	if err != nil {
		return payment.PaymentFact{}, err
	}
	if err := validateProviderSuccess("alipay.trade.query", response.providerResponse); err != nil {
		return payment.PaymentFact{}, err
	}
	if response.OutTradeNo != providerOrderID || !validTradeStatus(response.TradeState) {
		return payment.PaymentFact{}, permanentProviderError("alipay.trade.query", "response does not match the request", payment.ErrPermanentProviderProtocol)
	}
	amount, err := parseAmountMinor(response.Amount)
	if err != nil {
		return payment.PaymentFact{}, permanentProviderError("alipay.trade.query", "response amount is invalid", payment.ErrPermanentProviderProtocol)
	}
	payload, err := payment.ExtractSignedPayload(rawResponse)
	if err != nil {
		return payment.PaymentFact{}, permanentProviderError("alipay.trade.query", "signed response payload is invalid", payment.ErrPermanentProviderProtocol)
	}
	return payment.PaymentFact{
		Provider:          payment.ProviderAlipay,
		ProviderOrderID:   response.OutTradeNo,
		ProviderPaymentID: response.TradeNo,
		MerchantID:        a.sellerID,
		ApplicationID:     a.appID,
		AmountMinor:       amount,
		Currency:          "CNY",
		Success:           successfulTradeStatus(response.TradeState),
		Terminal:          terminalTradeStatus(response.TradeState),
		RawHash:           hashBody(rawResponse),
		Timestamp:         a.now(),
		SignedPayload:     payload,
	}, nil
}

// Refund submits an idempotent refund using out_request_no.
func (a *Adapter) Refund(ctx context.Context, input payment.RefundInput) (payment.RefundSubmission, error) {
	orderID := strings.TrimSpace(input.ProviderOrderID)
	requestID := strings.TrimSpace(input.IdempotencyKey)
	currency := strings.ToUpper(strings.TrimSpace(input.Currency))
	if orderID == "" || requestID == "" || input.AmountMinor <= 0 || currency != "CNY" {
		return payment.RefundSubmission{}, fmt.Errorf("%w: invalid Alipay refund input", payment.ErrPermanentProviderProtocol)
	}
	request := refundRequest{
		OutTradeNo: orderID, Amount: formatAmountMinor(input.AmountMinor), OutRequestNo: requestID,
		Reason: strings.TrimSpace(input.Reason),
	}
	var response refundResponse
	if _, err := a.call(ctx, "alipay.trade.refund", "alipay_trade_refund_response", request, &response); err != nil {
		return payment.RefundSubmission{}, err
	}
	if err := validateProviderSuccess("alipay.trade.refund", response.providerResponse); err != nil {
		return payment.RefundSubmission{}, err
	}
	amount, err := parseAmountMinor(response.RefundFee)
	if err != nil || amount != input.AmountMinor || response.OutTradeNo != orderID {
		return payment.RefundSubmission{}, permanentProviderError("alipay.trade.refund", "response does not match the request", payment.ErrPermanentProviderProtocol)
	}
	return payment.RefundSubmission{Provider: payment.ProviderAlipay, ProviderRefundID: requestID, Status: "success"}, nil
}

// QueryRefund queries an idempotent refund by out_request_no and binds the
// response to the persisted provider order, amount, and currency.
func (a *Adapter) QueryRefund(ctx context.Context, input payment.RefundQueryInput) (payment.RefundFact, error) {
	providerRefundID := strings.TrimSpace(input.ProviderRefundID)
	providerOrderID := strings.TrimSpace(input.ProviderOrderID)
	currency := strings.ToUpper(strings.TrimSpace(input.Currency))
	if providerRefundID == "" || providerOrderID == "" || input.AmountMinor <= 0 || currency != "CNY" {
		return payment.RefundFact{}, fmt.Errorf("%w: invalid Alipay refund query input", payment.ErrPermanentProviderProtocol)
	}
	var response refundQueryResponse
	rawResponse, err := a.call(ctx, "alipay.trade.fastpay.refund.query", "alipay_trade_fastpay_refund_query_response", refundQueryRequest{
		OutTradeNo: providerOrderID, OutRequestNo: providerRefundID,
	}, &response)
	if err != nil {
		return payment.RefundFact{}, err
	}
	if err := validateProviderSuccess("alipay.trade.fastpay.refund.query", response.providerResponse); err != nil {
		return payment.RefundFact{}, err
	}
	if response.OutRequestNo != providerRefundID || response.OutTradeNo != providerOrderID {
		return payment.RefundFact{}, permanentProviderError("alipay.trade.fastpay.refund.query", "response does not match the request", payment.ErrPermanentProviderProtocol)
	}
	amount, err := parseAmountMinor(response.RefundAmount)
	if err != nil || amount != input.AmountMinor {
		return payment.RefundFact{}, permanentProviderError("alipay.trade.fastpay.refund.query", "response amount is invalid", payment.ErrPermanentProviderProtocol)
	}
	if response.RefundStatus != "" && response.RefundStatus != "REFUND_SUCCESS" && response.RefundStatus != "REFUND_PROCESSING" && response.RefundStatus != "REFUND_FAIL" {
		return payment.RefundFact{}, permanentProviderError("alipay.trade.fastpay.refund.query", "response status is invalid", payment.ErrPermanentProviderProtocol)
	}
	payload, err := payment.ExtractSignedPayload(rawResponse)
	if err != nil {
		return payment.RefundFact{}, permanentProviderError("alipay.trade.fastpay.refund.query", "signed response payload is invalid", payment.ErrPermanentProviderProtocol)
	}
	success := response.RefundStatus == "" || response.RefundStatus == "REFUND_SUCCESS"
	rawHash := ""
	if success || response.RefundStatus == "REFUND_FAIL" {
		rawHash = hashBody(rawResponse)
	}
	return payment.RefundFact{
		Provider:         payment.ProviderAlipay,
		ProviderRefundID: response.OutRequestNo,
		ProviderOrderID:  response.OutTradeNo,
		AmountMinor:      amount,
		Currency:         currency,
		Success:          success,
		RawHash:          rawHash,
		Timestamp:        a.now(),
		SignedPayload:    payload,
	}, nil
}

// VerifyRefundCallback verifies an Alipay refund notification using the same
// RSA2 canonicalization as payment notifications.
func (a *Adapter) VerifyRefundCallback(ctx context.Context, _ http.Header, body []byte) (payment.RefundFact, error) {
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return payment.RefundFact{}, fmt.Errorf("%w: invalid refund callback form", payment.ErrPermanentProviderProtocol)
	}
	if err := a.verifySignature(ctx, []byte(notificationSignContent(values)), values.Get("sign")); err != nil {
		return payment.RefundFact{}, err
	}
	if values.Get("sign_type") != "RSA2" {
		return payment.RefundFact{}, fmt.Errorf("%w: refund callback sign_type must be RSA2", payment.ErrPermanentProviderProtocol)
	}
	if values.Get("app_id") != a.appID {
		return payment.RefundFact{}, fmt.Errorf("%w: refund callback application ID mismatch", payment.ErrPermanentProviderProtocol)
	}
	if values.Get("seller_id") != a.sellerID {
		return payment.RefundFact{}, fmt.Errorf("%w: refund callback seller ID mismatch", payment.ErrPermanentProviderProtocol)
	}
	providerOrderID := strings.TrimSpace(values.Get("out_trade_no"))
	providerRefundID := strings.TrimSpace(values.Get("out_request_no"))
	if providerOrderID == "" || providerRefundID == "" {
		return payment.RefundFact{}, fmt.Errorf("%w: refund callback order and request IDs are required", payment.ErrPermanentProviderProtocol)
	}
	if currency := values.Get("refund_currency"); currency != "" && strings.ToUpper(strings.TrimSpace(currency)) != "CNY" {
		return payment.RefundFact{}, fmt.Errorf("%w: refund callback currency must be CNY", payment.ErrPermanentProviderProtocol)
	}
	amount, err := parseAmountMinor(values.Get("refund_fee"))
	if err != nil || amount <= 0 {
		return payment.RefundFact{}, fmt.Errorf("%w: refund callback amount is invalid", payment.ErrPermanentProviderProtocol)
	}
	status := values.Get("refund_status")
	if status != "REFUND_SUCCESS" && status != "REFUND_PROCESSING" && status != "REFUND_FAIL" {
		return payment.RefundFact{}, fmt.Errorf("%w: refund callback status is invalid", payment.ErrPermanentProviderProtocol)
	}
	timestamp, err := time.ParseInLocation("2006-01-02 15:04:05", values.Get("gmt_refund"), a.now().Location())
	if err != nil {
		return payment.RefundFact{}, fmt.Errorf("%w: refund callback time is invalid", payment.ErrPermanentProviderProtocol)
	}
	rawHash := ""
	if status != "REFUND_PROCESSING" {
		rawHash = hashBody(body)
	}
	return payment.RefundFact{
		Provider: payment.ProviderAlipay, ProviderRefundID: providerRefundID,
		ProviderOrderID: providerOrderID, AmountMinor: amount, Currency: "CNY", Success: status == "REFUND_SUCCESS",
		RawHash: rawHash, Timestamp: timestamp, SignedPayload: valuesToMap(values),
	}, nil
}

func validateProviderSuccess(operation string, response providerResponse) error {
	if response.Code == "10000" {
		return nil
	}
	kind := payment.ProviderErrorPermanent
	if retryableProviderResponse(response.Code, response.SubCode) {
		kind = payment.ProviderErrorRetryable
	}
	return &payment.ProviderError{
		Provider: payment.ProviderAlipay, Operation: operation, Kind: kind, HTTPStatus: http.StatusOK,
		Code: response.Code, SubCode: response.SubCode,
	}
}

func retryableProviderResponse(code, subCode string) bool {
	if code == "20000" {
		return true
	}
	switch strings.ToUpper(strings.TrimSpace(subCode)) {
	case "ACQ.SYSTEM_ERROR", "SYSTEM_ERROR", "ISP.UNKNOW-ERROR", "ISP.UNKNOWN-ERROR", "ISP.NETWORK-ERROR", "AOP.UNKNOWN-ERROR":
		return true
	default:
		return false
	}
}

func validTradeStatus(status string) bool {
	switch status {
	case "TRADE_SUCCESS", "TRADE_FINISHED", "WAIT_BUYER_PAY", "TRADE_CLOSED":
		return true
	default:
		return false
	}
}

func successfulTradeStatus(status string) bool {
	return status == "TRADE_SUCCESS" || status == "TRADE_FINISHED"
}

func terminalTradeStatus(status string) bool {
	return successfulTradeStatus(status) || status == "TRADE_CLOSED"
}

func formatAmountMinor(amount int64) string {
	return strconv.FormatInt(amount/100, 10) + fmt.Sprintf(".%02d", amount%100)
}

func parseAmountMinor(value string) (int64, error) {
	if value == "" || strings.TrimSpace(value) != value || strings.HasPrefix(value, "-") || strings.HasPrefix(value, "+") {
		return 0, errors.New("invalid amount")
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, errors.New("invalid amount")
	}
	if !decimalDigits(parts[0]) {
		return 0, errors.New("invalid amount")
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
		if fraction == "" || len(fraction) > 2 || !decimalDigits(fraction) {
			return 0, errors.New("invalid amount")
		}
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, errors.New("invalid amount")
	}
	if len(fraction) == 1 {
		fraction += "0"
	} else if fraction == "" {
		fraction = "00"
	}
	fractionMinor, err := strconv.ParseInt(fraction, 10, 64)
	if err != nil || whole > (math.MaxInt64-fractionMinor)/100 {
		return 0, errors.New("invalid amount")
	}
	minor := whole*100 + fractionMinor
	if minor <= 0 {
		return 0, errors.New("invalid amount")
	}
	return minor, nil
}

func decimalDigits(value string) bool {
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func hashBody(body []byte) string {
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}

func valuesToMap(values url.Values) map[string]any {
	result := make(map[string]any, len(values))
	for key, entries := range values {
		if key == "sign" {
			continue
		}
		if len(entries) == 1 {
			result[key] = entries[0]
		} else {
			result[key] = append([]string(nil), entries...)
		}
	}
	return result
}

var _ payment.ProviderAdapter = (*Adapter)(nil)
