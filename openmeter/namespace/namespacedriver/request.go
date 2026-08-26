package namespacedriver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"

	"github.com/openmeterio/openmeter/pkg/models"
)

// NamespaceHeader selects the namespace a request operates on. It is only
// honored when request-level namespace selection is enabled, i.e. the
// deployment configures a non-empty namespace allowlist.
const NamespaceHeader = "X-Namespace"

type requestNamespaceContextKey struct{}

// WithRequestNamespace stores the request-selected namespace in the context.
func WithRequestNamespace(ctx context.Context, namespace string) context.Context {
	return context.WithValue(ctx, requestNamespaceContextKey{}, namespace)
}

// RequestNamespaceFromContext returns the namespace selected for the request,
// e.g. by RequestNamespaceMiddleware.
func RequestNamespaceFromContext(ctx context.Context) (string, bool) {
	namespace, ok := ctx.Value(requestNamespaceContextKey{}).(string)

	return namespace, ok
}

// RequestNamespaceDecoder resolves the namespace from the request context
// first and falls back to the static default. It pairs with
// RequestNamespaceMiddleware: requests without a selected namespace (header
// absent, or selection disabled) keep the default namespace behavior.
type RequestNamespaceDecoder string

func (d RequestNamespaceDecoder) GetNamespace(ctx context.Context) (string, bool) {
	if namespace, ok := RequestNamespaceFromContext(ctx); ok {
		return namespace, true
	}

	return string(d), true
}

// RequestNamespaceMiddlewareConfig configures the middleware validating the
// namespace selected by the X-Namespace header.
type RequestNamespaceMiddlewareConfig struct {
	// Logger receives rejected-request diagnostics.
	Logger *slog.Logger

	// DefaultNamespace is the namespace used when a request carries no
	// X-Namespace header. It is always allowed, independently of the allowlist.
	DefaultNamespace string

	// Allowlist constrains the selectable namespaces. Empty disables
	// request-level selection entirely: the header is ignored.
	Allowlist []string
}

// RequestNamespaceMiddleware validates the X-Namespace header against the
// allowlist and stores the selected namespace in the request context.
type RequestNamespaceMiddleware struct {
	logger           *slog.Logger
	defaultNamespace string
	allowed          []string
}

var _ models.Validator = (*RequestNamespaceMiddlewareConfig)(nil)

// NewRequestNamespaceMiddleware builds the middleware from a validated configuration.
func NewRequestNamespaceMiddleware(config RequestNamespaceMiddlewareConfig) (*RequestNamespaceMiddleware, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request namespace middleware config: %w", err)
	}

	return &RequestNamespaceMiddleware{
		logger:           config.Logger,
		defaultNamespace: config.DefaultNamespace,
		allowed:          config.Allowlist,
	}, nil
}

// Validate validates the configuration.
func (c RequestNamespaceMiddlewareConfig) Validate() error {
	var errs []error

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	if c.DefaultNamespace == "" {
		errs = append(errs, errors.New("default namespace is required"))
	}

	for _, ns := range c.Allowlist {
		if ns == "" {
			errs = append(errs, errors.New("allowlist must not contain empty namespaces"))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// Handle wraps next with namespace selection.
func (m *RequestNamespaceMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// An empty allowlist disables request-level selection: the header is
		// ignored and the decoder falls back to the default namespace.
		if len(m.allowed) == 0 {
			next.ServeHTTP(w, r)

			return
		}

		namespace := r.Header.Get(NamespaceHeader)
		if namespace == "" || namespace == m.defaultNamespace || slices.Contains(m.allowed, namespace) {
			if namespace != "" {
				ctx := WithRequestNamespace(r.Context(), namespace)
				r = r.WithContext(ctx)
			}

			next.ServeHTTP(w, r)

			return
		}

		m.logger.WarnContext(r.Context(), "namespace selection rejected",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("namespace", namespace),
		)

		models.NewStatusProblem(r.Context(), fmt.Errorf("namespace %q is not allowed", namespace), http.StatusForbidden).Respond(w)
	})
}
