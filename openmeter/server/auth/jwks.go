package auth

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// jwksMinRefreshInterval throttles JWKS refresh attempts. It is short enough
// to pick up key rotations promptly and long enough to keep an unreachable
// JWKS endpoint from being hammered on every request.
const jwksMinRefreshInterval = time.Minute

// jwksFetchTimeout bounds a single JWKS refresh independently of the caller's
// request context.
const jwksFetchTimeout = 10 * time.Second

// jwksMaxResponseSize caps the JWKS response body to bound memory usage from a
// hostile or misbehaving endpoint.
const jwksMaxResponseSize = 1 << 20

// jwksCache fetches and caches the issuer's RSA public keys from a JWKS
// endpoint, keyed by kid. Keys are fetched lazily on the first lookup of an
// unknown kid, which covers both the initial load and later rotations without
// background workers.
type jwksCache struct {
	jwksURL string
	client  *http.Client

	mu          sync.Mutex
	keys        map[string]*rsa.PublicKey
	lastRefresh time.Time
	refresh     singleflight.Group
}

func newJWKSCache(jwksURL string, client *http.Client) *jwksCache {
	return &jwksCache{
		jwksURL: jwksURL,
		client:  client,
	}
}

// publicKey returns the RSA public key for kid, lazily refreshing the cached
// key set at most once per jwksMinRefreshInterval. Refreshes run outside the
// cache lock on a context detached from the triggering request, so a slow
// endpoint or a disconnecting client can neither stall nor cancel other
// requests' lookups. A failed refresh only rejects requests for unknown
// kids; known kids keep serving from the stale key set.
func (c *jwksCache) publicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	refreshErr := c.refreshIfStale(ctx)

	if key, ok := c.cachedKey(kid); ok {
		return key, nil
	}

	if refreshErr != nil {
		return nil, fmt.Errorf("refresh jwks from %s: %w", c.jwksURL, refreshErr)
	}

	return nil, fmt.Errorf("unknown signing key id %q", kid)
}

// refreshIfStale shares a single in-flight refresh across concurrent callers
// and skips it entirely while the throttle window holds.
func (c *jwksCache) refreshIfStale(ctx context.Context) error {
	_, err, _ := c.refresh.Do("refresh", func() (any, error) {
		if !c.shouldRefresh() {
			return nil, nil
		}

		fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), jwksFetchTimeout)
		defer cancel()

		return nil, c.refreshKeys(fetchCtx)
	})

	return err
}

func (c *jwksCache) cachedKey(kid string) (*rsa.PublicKey, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key, ok := c.keys[kid]

	return key, ok
}

func (c *jwksCache) shouldRefresh() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return time.Since(c.lastRefresh) >= jwksMinRefreshInterval
}

// refreshKeys replaces the cached key set. The throttle timestamp advances
// on failure too, so an unreachable endpoint is not retried per request.
func (c *jwksCache) refreshKeys(ctx context.Context) error {
	keys, err := c.fetch(ctx)

	c.mu.Lock()
	c.lastRefresh = time.Now()
	if err == nil {
		c.keys = keys
	}
	c.mu.Unlock()

	return err
}

// fetch downloads and parses the key set from the JWKS endpoint.
func (c *jwksCache) fetch(ctx context.Context) (map[string]*rsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.jwksURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	var payload struct {
		Keys []jsonWebKey `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, jwksMaxResponseSize)).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode jwks: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(payload.Keys))
	for _, jwk := range payload.Keys {
		// Skip unusable entries instead of failing the whole key set: Casdoor
		// may advertise keys for algorithms this middleware does not accept.
		if jwk.Kid == "" {
			continue
		}

		key, err := jwk.publicKey()
		if err != nil {
			continue
		}

		keys[jwk.Kid] = key
	}

	return keys, nil
}

// jsonWebKey is the subset of the JWK format needed for RSA public keys.
type jsonWebKey struct {
	Kid string   `json:"kid"`
	Kty string   `json:"kty"`
	N   string   `json:"n"`
	E   string   `json:"e"`
	X5c []string `json:"x5c"`
}

// publicKey materializes the RSA public key from the modulus/exponent pair, or
// from the first X.509 certificate when n/e are absent.
func (k jsonWebKey) publicKey() (*rsa.PublicKey, error) {
	if k.Kty != "RSA" {
		return nil, fmt.Errorf("unsupported key type %q", k.Kty)
	}

	if k.N != "" && k.E != "" {
		modulus, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			return nil, fmt.Errorf("decode modulus: %w", err)
		}

		exponent, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			return nil, fmt.Errorf("decode exponent: %w", err)
		}

		e := new(big.Int).SetBytes(exponent)
		if !e.IsInt64() {
			return nil, errors.New("exponent overflows int64")
		}

		return &rsa.PublicKey{
			N: new(big.Int).SetBytes(modulus),
			E: int(e.Int64()),
		}, nil
	}

	if len(k.X5c) > 0 {
		der, err := base64.StdEncoding.DecodeString(k.X5c[0])
		if err != nil {
			return nil, fmt.Errorf("decode x5c certificate: %w", err)
		}

		cert, err := x509.ParseCertificate(der)
		if err != nil {
			return nil, fmt.Errorf("parse x5c certificate: %w", err)
		}

		key, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("x5c certificate public key is not RSA")
		}

		return key, nil
	}

	return nil, errors.New("missing n/e and x5c key material")
}
