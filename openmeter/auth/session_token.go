package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/openmeterio/openmeter/pkg/models"
)

// SessionTokenIssuerName identifies OpenMeter as the issuer of session tokens.
const SessionTokenIssuerName = "openmeter"

// SessionTokenClaims is the payload of the JWT OpenMeter issues after a
// successful OIDC login. It is the credential the management API middleware
// will require once API authentication is enforced.
type SessionTokenClaims struct {
	jwt.RegisteredClaims

	// Email is the authenticated user's email address, when the provider
	// shares it.
	Email string `json:"email,omitempty"`

	// Name is the authenticated user's display name.
	Name string `json:"name,omitempty"`

	// Organization is the provider-side organization the user belongs to
	// (Casdoor "owner"). It maps to an OpenMeter namespace.
	Organization string `json:"organization,omitempty"`
}

// IssueSessionTokenInput describes the identity a session token is issued for.
type IssueSessionTokenInput struct {
	UserID       string
	Email        string
	Name         string
	Organization string
}

// Validate validates the input.
func (i IssueSessionTokenInput) Validate() error {
	var errs []error

	if i.UserID == "" {
		errs = append(errs, errors.New("userId is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// SessionTokenIssuer issues and verifies OpenMeter session JWTs.
type SessionTokenIssuer struct {
	secret []byte
	expire time.Duration
}

// NewSessionTokenIssuer returns an issuer signing tokens with the given HMAC
// secret. The secret must come from the environment or a secret service, never
// from source control.
func NewSessionTokenIssuer(secret string, expire time.Duration) (*SessionTokenIssuer, error) {
	if secret == "" {
		return nil, errors.New("secret must not be empty")
	}

	if expire == 0 {
		return nil, errors.New("expire must not be 0")
	}

	return &SessionTokenIssuer{
		secret: []byte(secret),
		expire: expire,
	}, nil
}

// Issue signs a session token for the given identity.
func (s *SessionTokenIssuer) Issue(input IssueSessionTokenInput) (string, error) {
	if err := input.Validate(); err != nil {
		return "", err
	}

	now := time.Now()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, SessionTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   input.UserID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.expire)),
			Issuer:    SessionTokenIssuerName,
		},
		Email:        input.Email,
		Name:         input.Name,
		Organization: input.Organization,
	})

	tokenString, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("sign session token: %w", err)
	}

	return tokenString, nil
}

// Verify parses and validates a session token.
func (s *SessionTokenIssuer) Verify(tokenString string) (*SessionTokenClaims, error) {
	opts := []jwt.ParserOption{
		jwt.WithStrictDecoding(),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(SessionTokenIssuerName),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
	}

	keyFunc := func(token *jwt.Token) (any, error) {
		return s.secret, nil
	}

	jwtToken, err := jwt.ParseWithClaims(tokenString, &SessionTokenClaims{}, keyFunc, opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot parse session token: %w", err)
	}

	claims, ok := jwtToken.Claims.(*SessionTokenClaims)
	if !ok {
		return nil, fmt.Errorf("not session token claims")
	}

	return claims, nil
}
