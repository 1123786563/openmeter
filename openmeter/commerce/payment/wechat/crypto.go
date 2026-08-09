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
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	"github.com/openmeterio/openmeter/openmeter/commerce/payment"
)

func randomNonce() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	return hex.EncodeToString(value), nil
}

func signRSA(privateKey *rsa.PrivateKey, message []byte) (string, error) {
	digest := sha256.Sum256(message)
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("RSA-SHA256 sign: %w", err)
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func (a *Adapter) verifyWechatSignature(ctx context.Context, serial string, message []byte, signatureBase64 string) error {
	serial = strings.TrimSpace(serial)
	if serial == "" {
		return fmt.Errorf("%w: missing Wechatpay-Serial", payment.ErrInvalidSignature)
	}
	publicKeyPEM, err := a.secrets.Get(ctx, PlatformPublicKeySecret(serial))
	if err != nil {
		return fmt.Errorf("%w: platform public key for serial %q is unavailable", payment.ErrInvalidSignature, serial)
	}
	publicKey, err := parseRSAPublicKey([]byte(publicKeyPEM))
	if err != nil {
		return fmt.Errorf("%w: invalid platform public key for serial %q", payment.ErrInvalidSignature, serial)
	}
	signature, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil {
		return fmt.Errorf("%w: invalid base64 signature", payment.ErrInvalidSignature)
	}
	digest := sha256.Sum256(message)
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, digest[:], signature); err != nil {
		return fmt.Errorf("%w: RSA-SHA256 verification failed", payment.ErrInvalidSignature)
	}
	return nil
}

func decryptResource(apiKey string, resource encryptedResource) ([]byte, error) {
	if resource.Algorithm != "AEAD_AES_256_GCM" {
		return nil, fmt.Errorf("unsupported resource algorithm %q", resource.Algorithm)
	}
	if len(apiKey) != 32 {
		return nil, errors.New("API v3 key must contain exactly 32 bytes")
	}
	ciphertext, err := base64.StdEncoding.DecodeString(resource.Ciphertext)
	if err != nil {
		return nil, errors.New("resource ciphertext is not valid base64")
	}
	block, err := aes.NewCipher([]byte(apiKey))
	if err != nil {
		return nil, fmt.Errorf("initialize AES-256: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("initialize AES-GCM: %w", err)
	}
	plaintext, err := gcm.Open(nil, []byte(resource.Nonce), ciphertext, []byte(resource.AssociatedData))
	if err != nil {
		return nil, errors.New("AES-GCM resource authentication failed")
	}
	return plaintext, nil
}

func parseRSAPrivateKey(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not RSA")
	}
	return rsaKey, nil
}

func parseRSAPublicKey(pemBytes []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}
	if certificate, err := x509.ParseCertificate(block.Bytes); err == nil {
		publicKey, ok := certificate.PublicKey.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("certificate public key is not RSA")
		}
		return publicKey, nil
	}
	if key, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		publicKey, ok := key.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("public key is not RSA")
		}
		return publicKey, nil
	}
	publicKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	return publicKey, nil
}
