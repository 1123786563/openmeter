package wechat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/openmeterio/openmeter/openmeter/commerce/payment"
)

func (a *Adapter) doJSON(ctx context.Context, method, path string, query url.Values, requestValue, responseValue any) ([]byte, error) {
	operation := method + " " + path
	var requestBody []byte
	var err error
	if requestValue != nil {
		requestBody, err = json.Marshal(requestValue)
		if err != nil {
			return nil, permanentProviderError(operation, 0, "", fmt.Errorf("marshal request: %w", err))
		}
	}

	requestURL := strings.TrimRight(a.baseURL, "/") + path
	if len(query) != 0 {
		requestURL += "?" + query.Encode()
	}
	request, err := http.NewRequestWithContext(ctx, method, requestURL, bytes.NewReader(requestBody))
	if err != nil {
		return nil, permanentProviderError(operation, 0, "", fmt.Errorf("create request: %w", err))
	}
	if requestValue != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	request.Header.Set("Accept", "application/json")
	if err := a.authorizeRequest(ctx, request, requestBody); err != nil {
		return nil, permanentProviderError(operation, 0, "", err)
	}

	response, err := a.httpClient.Do(request)
	if err != nil {
		kind := payment.ProviderErrorRetryable
		if errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			kind = payment.ProviderErrorPermanent
		}
		return nil, &payment.ProviderError{
			Provider: payment.ProviderWeChat, Operation: operation, Kind: kind, Cause: err,
		}
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, a.maxResponseBytes+1))
	if err != nil {
		return nil, &payment.ProviderError{
			Provider: payment.ProviderWeChat, Operation: operation, Kind: payment.ProviderErrorRetryable,
			HTTPStatus: response.StatusCode, Cause: err,
		}
	}
	if int64(len(responseBody)) > a.maxResponseBytes {
		return nil, permanentProviderError(operation, response.StatusCode, "", fmt.Errorf("response exceeds %d bytes: %w", a.maxResponseBytes, payment.ErrPermanentProviderProtocol))
	}

	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		if err := a.verifyHTTPMessage(ctx, response.Header, responseBody); err != nil {
			return nil, permanentProviderError(operation, response.StatusCode, "", err)
		}
		if _, err := a.validateSignatureTime(response.Header.Get("Wechatpay-Timestamp")); err != nil {
			return nil, permanentProviderError(operation, response.StatusCode, "", err)
		}
		if responseValue != nil {
			if err := json.Unmarshal(responseBody, responseValue); err != nil {
				return nil, permanentProviderError(operation, response.StatusCode, "", fmt.Errorf("invalid JSON response: %w", payment.ErrPermanentProviderProtocol))
			}
		}
		return responseBody, nil
	}

	var providerError wechatErrorResponse
	if err := json.Unmarshal(responseBody, &providerError); err != nil {
		providerError.Code = ""
	}
	kind := payment.ProviderErrorPermanent
	if response.StatusCode == http.StatusRequestTimeout || response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= http.StatusInternalServerError {
		kind = payment.ProviderErrorRetryable
	}
	return nil, &payment.ProviderError{
		Provider: payment.ProviderWeChat, Operation: operation, Kind: kind,
		HTTPStatus: response.StatusCode, Code: strings.TrimSpace(providerError.Code),
		Cause: providerErrorMarker(kind),
	}
}

func permanentProviderError(operation string, httpStatus int, code string, cause error) error {
	return &payment.ProviderError{
		Provider: payment.ProviderWeChat, Operation: operation, Kind: payment.ProviderErrorPermanent,
		HTTPStatus: httpStatus, Code: code, Cause: cause,
	}
}

func providerErrorMarker(kind payment.ProviderErrorKind) error {
	if kind == payment.ProviderErrorRetryable {
		return payment.ErrRetryableProvider
	}
	return payment.ErrPermanentProviderProtocol
}

func (a *Adapter) authorizeRequest(ctx context.Context, request *http.Request, body []byte) error {
	privateKeyPEM, err := a.secrets.Get(ctx, SecretKeyMerchantPrivateKey)
	if err != nil {
		return fmt.Errorf("wechat: get merchant private key: %w", err)
	}
	privateKey, err := parseRSAPrivateKey([]byte(privateKeyPEM))
	if err != nil {
		return fmt.Errorf("wechat: parse merchant private key: %w", err)
	}
	nonce, err := randomNonce()
	if err != nil {
		return fmt.Errorf("wechat: %w", err)
	}
	timestamp := strconv.FormatInt(a.now().Unix(), 10)
	message := request.Method + "\n" + request.URL.RequestURI() + "\n" + timestamp + "\n" + nonce + "\n" + string(body) + "\n"
	signature, err := signRSA(privateKey, []byte(message))
	if err != nil {
		return fmt.Errorf("wechat: sign request: %w", err)
	}
	request.Header.Set("Authorization", fmt.Sprintf(
		`WECHATPAY2-SHA256-RSA2048 mchid=%q,nonce_str=%q,signature=%q,timestamp=%q,serial_no=%q`,
		a.merchantID, nonce, signature, timestamp, a.merchantSerial,
	))
	return nil
}

func (a *Adapter) verifyHTTPMessage(ctx context.Context, headers http.Header, body []byte) error {
	timestamp := headers.Get("Wechatpay-Timestamp")
	nonce := headers.Get("Wechatpay-Nonce")
	signature := headers.Get("Wechatpay-Signature")
	serial := headers.Get("Wechatpay-Serial")
	if timestamp == "" || nonce == "" || signature == "" || serial == "" {
		return fmt.Errorf("%w: missing required Wechatpay response headers", payment.ErrInvalidSignature)
	}
	message := timestamp + "\n" + nonce + "\n" + string(body) + "\n"
	return a.verifyWechatSignature(ctx, serial, []byte(message), signature)
}
