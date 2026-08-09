package alipay

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/openmeterio/openmeter/openmeter/commerce/payment"
)

func (a *Adapter) call(ctx context.Context, method, responseKey string, requestValue, responseValue any) ([]byte, error) {
	bizContent, err := json.Marshal(requestValue)
	if err != nil {
		return nil, fmt.Errorf("alipay: marshal %s request: %w", method, err)
	}
	values := url.Values{
		"app_id":      {a.appID},
		"method":      {method},
		"format":      {"JSON"},
		"charset":     {"utf-8"},
		"sign_type":   {"RSA2"},
		"timestamp":   {a.now().Format("2006-01-02 15:04:05")},
		"version":     {"1.0"},
		"notify_url":  {a.notifyURL},
		"biz_content": {string(bizContent)},
	}
	privateKeyPEM, err := a.secrets.Get(ctx, SecretKeyAppPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("alipay: get application private key: %w", err)
	}
	privateKey, err := parseRSAPrivateKey([]byte(privateKeyPEM))
	if err != nil {
		return nil, fmt.Errorf("alipay: parse application private key: %w", err)
	}
	signature, err := signRSA2(privateKey, []byte(requestSignContent(values)))
	if err != nil {
		return nil, fmt.Errorf("alipay: sign %s request: %w", method, err)
	}
	values.Set("sign", signature)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, a.gatewayURL, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, fmt.Errorf("alipay: create %s request: %w", method, err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")
	response, err := a.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("alipay: execute %s request: %w", method, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, a.maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("alipay: read %s response: %w", method, err)
	}
	if int64(len(body)) > a.maxResponseBytes {
		return nil, fmt.Errorf("%w: Alipay response exceeds %d bytes", payment.ErrPermanentProviderProtocol, a.maxResponseBytes)
	}

	responseBody, err := a.verifyResponse(ctx, body, responseKey)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("alipay: %s returned HTTP %d", method, response.StatusCode)
	}
	if err := json.Unmarshal(responseBody, responseValue); err != nil {
		return nil, fmt.Errorf("%w: invalid %s response", payment.ErrPermanentProviderProtocol, method)
	}
	return responseBody, nil
}

func (a *Adapter) verifyResponse(ctx context.Context, body []byte, responseKey string) ([]byte, error) {
	var envelope map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&envelope); err != nil {
		return nil, fmt.Errorf("%w: invalid Alipay response envelope", payment.ErrPermanentProviderProtocol)
	}
	responseBody := envelope[responseKey]
	if len(responseBody) == 0 || bytes.Equal(responseBody, []byte("null")) {
		return nil, fmt.Errorf("%w: response is missing %s", payment.ErrPermanentProviderProtocol, responseKey)
	}
	var signature string
	if err := json.Unmarshal(envelope["sign"], &signature); err != nil || strings.TrimSpace(signature) == "" {
		return nil, fmt.Errorf("%w: response is missing a valid sign", payment.ErrInvalidSignature)
	}
	if err := a.verifySignature(ctx, responseBody, signature); err != nil {
		return nil, err
	}
	return responseBody, nil
}

func (a *Adapter) verifySignature(ctx context.Context, content []byte, encodedSignature string) error {
	publicKeyPEM, err := a.secrets.Get(ctx, SecretKeyAlipayPublicKey)
	if err != nil {
		return fmt.Errorf("alipay: get Alipay public key: %w", err)
	}
	publicKey, err := parseRSAPublicKey([]byte(publicKeyPEM))
	if err != nil {
		return fmt.Errorf("alipay: parse Alipay public key: %w", err)
	}
	signature, err := base64.StdEncoding.DecodeString(encodedSignature)
	if err != nil {
		return fmt.Errorf("%w: invalid RSA2 signature encoding", payment.ErrInvalidSignature)
	}
	digest := sha256.Sum256(content)
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, digest[:], signature); err != nil {
		return fmt.Errorf("%w: RSA2 signature verification failed", payment.ErrInvalidSignature)
	}
	return nil
}

func signRSA2(key *rsa.PrivateKey, content []byte) (string, error) {
	digest := sha256.Sum256(content)
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

// requestSignContent excludes only sign. sign_type is a signed request field.
func requestSignContent(values url.Values) string {
	return canonicalSignContent(values, false)
}

// notificationSignContent follows Alipay's async notification protocol, which
// excludes both sign and sign_type from the verified content.
func notificationSignContent(values url.Values) string {
	return canonicalSignContent(values, true)
}

// canonicalSignContent sorts parameter names and joins their decoded,
// non-empty values. url.Values contains decoded values both after ParseQuery
// and before Encode, so request signing happens before the final URL encoding.
func canonicalSignContent(values url.Values, excludeSignType bool) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		if key != "sign" && (!excludeSignType || key != "sign_type") {
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

func parseRSAPrivateKey(value []byte) (*rsa.PrivateKey, error) {
	block, err := decodePEM(value, "RSA PRIVATE KEY")
	if err != nil {
		return nil, err
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not RSA")
	}
	return key, nil
}

func parseRSAPublicKey(value []byte) (*rsa.PublicKey, error) {
	block, err := decodePEM(value, "PUBLIC KEY")
	if err != nil {
		return nil, err
	}
	if parsed, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		key, ok := parsed.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("public key is not RSA")
		}
		return key, nil
	}
	key, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	return key, nil
}

func decodePEM(value []byte, blockType string) (*pem.Block, error) {
	trimmed := strings.TrimSpace(string(value))
	if !strings.Contains(trimmed, "BEGIN") {
		trimmed = "-----BEGIN " + blockType + "-----\n" + wrapBase64(trimmed) + "\n-----END " + blockType + "-----"
	}
	block, _ := pem.Decode([]byte(trimmed))
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}
	return block, nil
}

func wrapBase64(value string) string {
	var builder strings.Builder
	for len(value) > 64 {
		builder.WriteString(value[:64])
		builder.WriteByte('\n')
		value = value[64:]
	}
	builder.WriteString(value)
	return builder.String()
}
