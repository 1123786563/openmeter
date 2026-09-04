// Package auth implements management-plane authentication: an OIDC
// (Authorization Code) client against an external identity provider such as
// Casdoor, plus issuance of OpenMeter's own session JWTs that downstream API
// middleware can verify.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi/v5"
	"github.com/samber/lo"
	"golang.org/x/oauth2"

	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	// stateCookieName carries the OIDC state between the login redirect and
	// the callback to protect against CSRF.
	stateCookieName = "om_oidc_state"

	// loginStateCookieName carries the front-end nonce (the login URL's
	// "state" query parameter) so the callback can hand it back in the
	// redirect fragment and the front-end can tie the token to the login this
	// tab started.
	loginStateCookieName = "om_oidc_login_state"

	// stateCookiePath limits the state cookies to the OIDC endpoints.
	stateCookiePath = "/auth/oidc"

	// stateMaxAge bounds how long a login redirect stays valid.
	stateMaxAge = 10 * time.Minute
)

// HandlerConfig configures the OIDC login flow.
type HandlerConfig struct {
	// Issuer is the OIDC issuer URL of the identity provider. For Casdoor this
	// is the deployment's base URL; discovery happens at
	// {issuer}/.well-known/openid-configuration.
	Issuer string

	// ClientID and ClientSecret identify this server as an OAuth2 client
	// registered in the provider. The secret must be injected through the
	// environment, never committed to source control.
	ClientID     string
	ClientSecret string

	// RedirectURL is this server's OAuth2 callback endpoint as registered in
	// the provider, for example http://localhost:8888/auth/oidc/callback.
	RedirectURL string

	// DashboardURL is the front-end URL the browser is redirected to after a
	// successful login; the session token is appended in the URL fragment.
	DashboardURL string

	// Tokens issues and verifies OpenMeter session JWTs.
	Tokens *SessionTokenIssuer
}

// Validate validates the configuration.
func (c HandlerConfig) Validate() error {
	var errs []error

	if c.Issuer == "" {
		errs = append(errs, errors.New("issuer is required"))
	}

	if c.ClientID == "" {
		errs = append(errs, errors.New("clientId is required"))
	}

	if c.ClientSecret == "" {
		errs = append(errs, errors.New("clientSecret is required"))
	}

	if c.RedirectURL == "" {
		errs = append(errs, errors.New("redirectURL is required"))
	}

	if c.DashboardURL == "" {
		errs = append(errs, errors.New("dashboardURL is required"))
	}

	if c.Tokens == nil {
		errs = append(errs, errors.New("tokens is required"))
	}

	return errors.Join(errs...)
}

// Handler serves the OIDC login and callback endpoints.
type Handler struct {
	logger       *slog.Logger
	oauth2       oauth2.Config
	verifier     *oidc.IDTokenVerifier
	tokens       *SessionTokenIssuer
	dashboardURL string
}

// NewHandler discovers the provider's OIDC endpoints from the issuer and
// returns a handler serving /auth/oidc/login and /auth/oidc/callback.
func NewHandler(ctx context.Context, config HandlerConfig, logger *slog.Logger) (*Handler, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	if logger == nil {
		return nil, errors.New("logger is required")
	}

	provider, err := oidc.NewProvider(ctx, config.Issuer)
	if err != nil {
		return nil, fmt.Errorf("discover OIDC provider %q: %w", config.Issuer, err)
	}

	return &Handler{
		logger: logger,
		oauth2: oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  config.RedirectURL,
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
		verifier:     provider.Verifier(&oidc.Config{ClientID: config.ClientID}),
		tokens:       config.Tokens,
		dashboardURL: config.DashboardURL,
	}, nil
}

// RegisterRoutes mounts the OIDC endpoints on the router.
func (h *Handler) RegisterRoutes(r chi.Router) error {
	r.Get("/auth/oidc/login", h.Login)
	r.Get("/auth/oidc/callback", h.Callback)

	return nil
}

// Login starts the Authorization Code flow by redirecting the browser to the
// provider's authorization endpoint. The state is stored in a short-lived
// cookie that the callback must present unchanged. A "state" query parameter
// supplied by the front-end (its login nonce) is carried along and returned
// in the callback redirect fragment.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	state, err := newState()
	if err != nil {
		h.logger.ErrorContext(r.Context(), "oidc login: generate state failed", "error", err)
		models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(w)

		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    state,
		Path:     stateCookiePath,
		MaxAge:   int(stateMaxAge.Seconds()),
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	// The front-end nonce is opaque to the server; it is only echoed back so
	// the callback page can reject tokens this tab never asked for.
	if loginState := r.URL.Query().Get("state"); loginState != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     loginStateCookieName,
			Value:    loginState,
			Path:     stateCookiePath,
			MaxAge:   int(stateMaxAge.Seconds()),
			HttpOnly: true,
			Secure:   r.TLS != nil,
			SameSite: http.SameSiteLaxMode,
		})
	}

	http.Redirect(w, r, h.oauth2.AuthCodeURL(state), http.StatusFound)
}

// idTokenClaims are the ID token payload fields consumed from the provider.
// Casdoor exposes the owning organization as "owner" ("organization" on newer
// versions); both are accepted.
type idTokenClaims struct {
	Email             string `json:"email"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
	Owner             string `json:"owner"`
	Organization      string `json:"organization"`
}

// Callback exchanges the authorization code for tokens, verifies the ID token
// against the provider's keys, issues an OpenMeter session token and redirects
// the browser to the dashboard with the token in the URL fragment.
func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	stateCookie, cookieErr := r.Cookie(stateCookieName)

	if cookieErr != nil || code == "" || state == "" || subtle.ConstantTimeCompare([]byte(state), []byte(stateCookie.Value)) != 1 {
		h.logger.WarnContext(r.Context(), "oidc callback: invalid state")
		models.NewStatusProblem(r.Context(), errors.New("invalid state parameter"), http.StatusBadRequest).Respond(w)

		return
	}

	// The state cookie is single use.
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    "",
		Path:     stateCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	tokens, err := h.oauth2.Exchange(r.Context(), code)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "oidc callback: code exchange failed", "error", err)
		models.NewStatusProblem(r.Context(), err, http.StatusBadGateway).Respond(w)

		return
	}

	rawIDToken, ok := tokens.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		h.logger.ErrorContext(r.Context(), "oidc callback: token response has no id_token")
		models.NewStatusProblem(r.Context(), errors.New("token response has no id_token"), http.StatusBadGateway).Respond(w)

		return
	}

	idToken, err := h.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		h.logger.WarnContext(r.Context(), "oidc callback: id token verification failed", "error", err)
		models.NewStatusProblem(r.Context(), err, http.StatusUnauthorized).Respond(w)

		return
	}

	var claims idTokenClaims
	if err := idToken.Claims(&claims); err != nil {
		h.logger.ErrorContext(r.Context(), "oidc callback: parse id token claims failed", "error", err)
		models.NewStatusProblem(r.Context(), err, http.StatusBadGateway).Respond(w)

		return
	}

	organization := lo.CoalesceOrEmpty(claims.Organization, claims.Owner)

	token, err := h.tokens.Issue(IssueSessionTokenInput{
		UserID:       idToken.Subject,
		Email:        claims.Email,
		Name:         lo.CoalesceOrEmpty(claims.Name, claims.PreferredUsername),
		Organization: organization,
	})
	if err != nil {
		h.logger.ErrorContext(r.Context(), "oidc callback: issue session token failed", "error", err)
		models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(w)

		return
	}

	// The redirect fragment follows the front-end's SSO callback contract:
	// the session token, the organization (as tenant_id) and — when the login
	// carried a front-end nonce — that nonce back for the callback page to
	// compare against the one it stored.
	fragment := url.Values{}
	fragment.Set("token", token)

	if organization != "" {
		fragment.Set("tenant_id", organization)
	}

	if loginStateCookie, err := r.Cookie(loginStateCookieName); err == nil && loginStateCookie.Value != "" {
		fragment.Set("state", loginStateCookie.Value)

		// The nonce is single use.
		http.SetCookie(w, &http.Cookie{
			Name:     loginStateCookieName,
			Value:    "",
			Path:     stateCookiePath,
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   r.TLS != nil,
			SameSite: http.SameSiteLaxMode,
		})
	}

	http.Redirect(w, r, h.dashboardURL+"#"+fragment.Encode(), http.StatusFound)
}

func newState() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate random state: %w", err)
	}

	return hex.EncodeToString(buf), nil
}
