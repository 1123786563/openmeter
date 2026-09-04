package auth

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/openmeterio/openmeter/openmeter/session"
	"github.com/openmeterio/openmeter/pkg/models"
)

// SessionOrganizationRoleAdmin is the organization role assigned to sessions
// minted from OIDC logins until provider role mapping is implemented.
const SessionOrganizationRoleAdmin = "admin"

// SessionMiddlewareConfig configures the management-plane session enforcement.
type SessionMiddlewareConfig struct {
	// Tokens verifies the OpenMeter session JWTs carried as bearer tokens.
	Tokens *SessionTokenIssuer

	Logger *slog.Logger

	// ExemptPathPrefixes lists request path prefixes that skip session
	// enforcement, for endpoints with their own authentication scheme.
	ExemptPathPrefixes []string
}

// Enabled reports whether session enforcement is configured.
func (c SessionMiddlewareConfig) Enabled() bool {
	return c.Tokens != nil
}

// Validate validates the configuration.
func (c SessionMiddlewareConfig) Validate() error {
	var errs []error

	if c.Tokens == nil {
		errs = append(errs, errors.New("tokens is required"))
	}

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	return errors.Join(errs...)
}

// NewSessionMiddleware returns HTTP middleware that requires an OpenMeter
// session token on requests outside the exempt path prefixes. Valid tokens
// attach the authentication session to the request context.
func NewSessionMiddleware(config SessionMiddlewareConfig) (func(http.Handler) http.Handler, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, prefix := range config.ExemptPathPrefixes {
				if strings.HasPrefix(r.URL.Path, prefix) {
					next.ServeHTTP(w, r)

					return
				}
			}

			token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || token == "" {
				unauthorized(w, r, config.Logger, errors.New("missing bearer token"))

				return
			}

			claims, err := config.Tokens.Verify(token)
			if err != nil {
				unauthorized(w, r, config.Logger, err)

				return
			}

			// The token organization links the identity to its tenant; the
			// namespace decoder resolves it to an OpenMeter namespace.
			authSession, err := session.NewAuthenticationSession(
				claims.Organization,
				claims.Organization,
				SessionOrganizationRoleAdmin,
				claims.Subject,
				nil,
			)
			if err != nil {
				unauthorized(w, r, config.Logger, fmt.Errorf("token carries no organization: %w", err))

				return
			}

			next.ServeHTTP(w, r.WithContext(session.WithAuthenticationSession(r.Context(), authSession)))
		})
	}, nil
}

func unauthorized(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	logger.WarnContext(r.Context(), "session authentication failed", "error", err)

	w.Header().Set("WWW-Authenticate", `Bearer realm="openmeter"`)
	models.NewStatusProblem(r.Context(), err, http.StatusUnauthorized).Respond(w)
}
