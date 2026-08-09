package wechat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/openmeterio/openmeter/openmeter/commerce/payment"
)

func (a *Adapter) doJSON(ctx context.Context, method, path string, query url.Values, requestValue, responseValue any) ([]byte, error) {
	var requestBody []byte
	var err error
	if requestValue != nil {
		requestBody, err = json.Marshal(requestValue)
		if err != nil {
			return nil, fmt.Errorf("wechat: marshal request: %w", err)
		}
	}

	requestURL := strings.TrimRight(a.baseURL, "/") + path
	if len(query) != 0 {
		requestURL += "?" + query.Encode()
	}
	request, err := http.NewRequestWithContext(ctx, method, requestURL, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("wechat: create request: %w", err)
	}
	if requestValue != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	request.Header.Set("Accept", "application/json")
	if err := a.authorizeRequest(ctx, request, requestBody); err != nil {
		return nil, err
	}

	response, err := a.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("wechat: execute %s %s: %w", method, path, err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, a.maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("wechat: read response: %w", err)
	}
	if int64(len(responseBody)) > a.maxResponseBytes {
		return nil, fmt.Errorf("%w: response exceeds %d bytes", payment.ErrPermanentProviderProtocol, a.maxResponseBytes)
	}

	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		if err := a.verifyHTTPMessage(ctx, response.Header, responseBody); err != nil {
			return nil, err
		}
		if responseValue != nil {
			if err := json.Unmarshal(responseBody, responseValue); err != nil {
				return nil, fmt.Errorf("%w: invalid JSON response", payment.ErrPermanentProviderProtocol)
			}
		}
		return responseBody, nil
	}

	var providerError wechatErrorResponse
	if err := json.Unmarshal(responseBody, &providerError); err != nil {
		return nil, fmt.Errorf("wechat: HTTP %d returned an invalid error response", response.StatusCode)
	}
	return nil, fmt.Errorf("wechat: HTTP %d code=%q message=%q", response.StatusCode, providerError.Code, providerError.Message)
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
