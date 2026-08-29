// Package auth implements Casdoor OIDC bearer authentication for the fork's
// admin-facing API surface (see docs/adr/0001-casdoor-oidc-auth-middleware.md).
//
// Access tokens are verified locally against a cached JWKS key set. Beyond
// signature and standard claim validation, tokens are narrowed to an
// organization allowlist and a two-level role model: Viewer may only issue
// read requests (GET/HEAD/OPTIONS), Operator has full access.
package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Default claim names following Casdoor's token layout: the organization a user
// belongs to is the "owner" claim and its roles are the "roles" claim.
const (
	DefaultOrganizationClaim = "owner"
	DefaultRoleClaim         = "roles"
)

// Role is the coarse-grained API access level derived from the token's role claim.
type Role string

const (
	// RoleViewer allows read-only requests.
	RoleViewer Role = "Viewer"
	// RoleOperator allows read and write requests.
	RoleOperator Role = "Operator"
)

// readMethods are HTTP methods a Viewer may issue.
var readMethods = []string{http.MethodGet, http.MethodHead, http.MethodOptions}

// Identity is the authenticated caller placed into the request context after
// the middleware passes.
type Identity struct {
	// Subject is the token sub claim (the Casdoor user name).
	Subject string
	// Organization is the token organization claim (empty when the token has none).
	Organization string
	// Role is the mapped access level.
	Role Role
}

type identityContextKey struct{}

// FromContext returns the authenticated identity from the request context.
func FromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityContextKey{}).(Identity)

	return identity, ok
}

// Config configures the OIDC bearer middleware.
type Config struct {
	// Logger receives rejected-request diagnostics.
	Logger *slog.Logger

	// Issuer is the expected JWT iss claim.
	Issuer string

	// JwksURL is the JWKS endpoint serving the issuer's public RSA keys.
	JwksURL string

	// Audience validates the JWT aud claim when non-empty; empty skips the check.
	Audience string

	// AllowedOrganizations restricts access to tokens whose organization claim
	// is listed here. Empty skips the organization check.
	AllowedOrganizations []string

	// OrganizationClaim names the token claim carrying the organization.
	// Defaults to DefaultOrganizationClaim when empty.
	OrganizationClaim string

	// ViewerRoles are token roles mapped to RoleViewer.
	ViewerRoles []string

	// OperatorRoles are token roles mapped to RoleOperator.
	OperatorRoles []string

	// RoleClaim names the token claim carrying role names.
	// Defaults to DefaultRoleClaim when empty.
	RoleClaim string

	// HTTPClient fetches the JWKS endpoint. Defaults to a client with a short
	// timeout when nil.
	HTTPClient *http.Client
}

var _ models.Validator = (*Config)(nil)

// Validate validates the configuration.
func (c Config) Validate() error {
	var errs []error

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	if c.Issuer == "" {
		errs = append(errs, errors.New("issuer is required"))
	}

	if c.JwksURL == "" {
		errs = append(errs, errors.New("jwks url is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// Middleware authenticates requests with Casdoor-issued OIDC bearer tokens.
type Middleware struct {
	logger *slog.Logger

	parser     *jwt.Parser
	jwks       *jwksCache
	audience   string
	allowedOrg []string
	orgClaim   string
	viewer     []string
	operator   []string
	roleClaim  string
	// rolesConfigured is true when either role list is set. When false, passing
	// the organization check grants Operator.
	rolesConfigured bool
}

// New builds the middleware from a validated configuration.
func New(config Config) (*Middleware, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid oidc auth config: %w", err)
	}

	parserOptions := []jwt.ParserOption{
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithIssuer(config.Issuer),
		jwt.WithExpirationRequired(),
	}
	if config.Audience != "" {
		parserOptions = append(parserOptions, jwt.WithAudience(config.Audience))
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	return &Middleware{
		logger:          config.Logger,
		parser:          jwt.NewParser(parserOptions...),
		jwks:            newJWKSCache(config.JwksURL, httpClient),
		audience:        config.Audience,
		allowedOrg:      config.AllowedOrganizations,
		orgClaim:        defaultString(config.OrganizationClaim, DefaultOrganizationClaim),
		viewer:          config.ViewerRoles,
		operator:        config.OperatorRoles,
		roleClaim:       defaultString(config.RoleClaim, DefaultRoleClaim),
		rolesConfigured: len(config.ViewerRoles) > 0 || len(config.OperatorRoles) > 0,
	}, nil
}

// NewOptional returns the authentication middleware when enabled, and nil when
// disabled so callers can mount the result unconditionally: a nil middleware
// fully bypasses authentication.
func NewOptional(enabled bool, config Config) (func(http.Handler) http.Handler, error) {
	if !enabled {
		return nil, nil
	}

	middleware, err := New(config)
	if err != nil {
		return nil, err
	}

	return middleware.Handle, nil
}

// publicPaths are exact request paths that bypass authentication: the OpenAPI
// documents and the debug metrics endpoint used by container healthchecks.
var publicPaths = []string{
	"/api/swagger.json",
	"/api/v3/openapi.json",
	"/api/v3/openapi.yaml",
	"/api/v1/debug/metrics",
}

// publicPathPrefixes are request path prefixes that bypass authentication.
// Only the portal meter queries carry their own PortalTokenAuth; the portal
// token management endpoints (mint/list/invalidate) have no operation-level
// security upstream and must stay behind OIDC authentication here.
var publicPathPrefixes = []string{
	"/api/v1/portal/meters/",
}

// bypassesAuthentication reports whether the request skips OIDC checks.
// CORS preflight requests cannot carry credentials, so OPTIONS always passes through.
func bypassesAuthentication(r *http.Request) bool {
	if r.Method == http.MethodOptions {
		return true
	}

	if slices.Contains(publicPaths, r.URL.Path) {
		return true
	}

	for _, prefix := range publicPathPrefixes {
		if strings.HasPrefix(r.URL.Path, prefix) {
			return true
		}
	}

	return false
}

// Handle wraps next with OIDC bearer authentication.
func (m *Middleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if bypassesAuthentication(r) {
			next.ServeHTTP(w, r)

			return
		}

		bearer, err := bearerToken(r)
		if err != nil {
			m.reject(w, r, http.StatusUnauthorized, err)

			return
		}

		identity, err := m.authenticate(r.Context(), bearer)
		if err != nil {
			m.reject(w, r, http.StatusUnauthorized, err)

			return
		}

		if err := m.authorize(identity, r); err != nil {
			m.reject(w, r, http.StatusForbidden, err)

			return
		}

		ctx := context.WithValue(r.Context(), identityContextKey{}, identity)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// authenticate verifies signature and standard claims, then extracts the
// identity from the token payload.
func (m *Middleware) authenticate(ctx context.Context, bearer string) (Identity, error) {
	token, err := m.parser.Parse(bearer, func(token *jwt.Token) (any, error) {
		kid, _ := token.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("token header is missing the kid")
		}

		return m.jwks.publicKey(ctx, kid)
	})
	if err != nil {
		return Identity{}, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return Identity{}, errors.New("invalid token claims")
	}

	subject, err := claims.GetSubject()
	if err != nil {
		return Identity{}, fmt.Errorf("invalid subject claim: %w", err)
	}

	if subject == "" {
		return Identity{}, errors.New("token is missing the subject")
	}

	organization, _ := claims[m.orgClaim].(string)
	roles := claimStrings(claims[m.roleClaim])

	return Identity{
		Subject:      subject,
		Organization: organization,
		Role:         m.resolveRole(roles),
	}, nil
}

// authorize enforces the organization allowlist and the Viewer read-only
// restriction. The role itself is resolved from the claims in authenticate;
// unmapped tokens resolve to the empty role and fail here.
func (m *Middleware) authorize(identity Identity, r *http.Request) error {
	if len(m.allowedOrg) > 0 && !slices.Contains(m.allowedOrg, identity.Organization) {
		return fmt.Errorf("organization %q is not allowed", identity.Organization)
	}

	if identity.Role == "" {
		return errors.New("token has no allowed role")
	}

	if identity.Role == RoleViewer && !slices.Contains(readMethods, r.Method) {
		return fmt.Errorf("viewer role is not allowed to send %s requests", r.Method)
	}

	return nil
}

// resolveRole maps token roles to an access level. Operator wins over Viewer
// when both match. Without configured role lists, every authenticated token is
// an Operator; otherwise an empty result means the token matched no role.
func (m *Middleware) resolveRole(roles []string) Role {
	if !m.rolesConfigured {
		return RoleOperator
	}

	if containsAny(roles, m.operator) {
		return RoleOperator
	}

	if containsAny(roles, m.viewer) {
		return RoleViewer
	}

	return ""
}

// Response bodies carry static text only: the internal errors (JWKS URLs,
// network details, claim specifics) must not leak to unauthenticated callers.
var (
	errUnauthorizedResponse = errors.New("unauthorized")
	errForbiddenResponse    = errors.New("forbidden")
)

// reject responds with an RFC 7807 problem and logs the real rejection cause.
func (m *Middleware) reject(w http.ResponseWriter, r *http.Request, status int, err error) {
	m.logger.WarnContext(r.Context(), "oidc authentication rejected",
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.Int("status", status),
		slog.Any("error", err),
	)

	responseErr := errUnauthorizedResponse
	if status == http.StatusForbidden {
		responseErr = errForbiddenResponse
	}

	models.NewStatusProblem(r.Context(), responseErr, status).Respond(w)
}

// bearerToken extracts the token from the Authorization header.
func bearerToken(r *http.Request) (string, error) {
	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if authorization == "" {
		return "", errors.New("missing authorization header")
	}

	scheme, token, found := strings.Cut(authorization, " ")
	if !found || scheme != "Bearer" || token == "" {
		return "", errors.New("invalid authorization header: expected a bearer token")
	}

	return token, nil
}

// claimStrings reads a claim that may appear as a single string or as a JSON
// array of strings; Casdoor emits both shapes depending on the role source.
func claimStrings(claim any) []string {
	switch value := claim.(type) {
	case string:
		return []string{value}
	case []any:
		values := make([]string, 0, len(value))
		for _, item := range value {
			if s, ok := item.(string); ok {
				values = append(values, s)
			}
		}

		return values
	default:
		return nil
	}
}

// containsAny reports whether any of values is contained in candidates.
func containsAny(values, candidates []string) bool {
	for _, value := range values {
		if slices.Contains(candidates, value) {
			return true
		}
	}

	return false
}

func defaultString(value, fallback string) string {
	return defaultx.IfZero(value, fallback)
}
