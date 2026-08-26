package server

import (
	"net/http"
	"strings"

	"github.com/go-chi/cors"

	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
)

type corsOptions struct {
	cors.Options
	AllowedPaths []string
}

func corsHandler(options corsOptions) func(next http.Handler) http.Handler {
	ch := cors.Handler(options.Options)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If AllowedPaths is empty, apply CORS to all paths
			if len(options.AllowedPaths) == 0 {
				ch(next).ServeHTTP(w, r)
				return
			}

			// Check if the request path starts with any of the allowed prefixes
			for _, path := range options.AllowedPaths {
				if strings.HasPrefix(r.URL.Path, path) {
					ch(next).ServeHTTP(w, r)
					return
				}
			}

			// If none of the prefixes match, call the next handler
			next.ServeHTTP(w, r)
		})
	}
}

// apiAllowedHeaders are the request headers browsers may send to the main API
// surface cross-origin. The API authenticates with a bearer token and selects
// namespaces with a header, so both must be preflight-approved.
var apiAllowedHeaders = []string{"Accept", "Authorization", "Content-Type", namespacedriver.NamespaceHeader}

// portalPathPrefix is owned by the portal CORS handler (corsHandler with
// AllowedPaths), which answers credentialed preflights for its routes. The API
// CORS middleware must not intercept those preflights with a
// non-credentialed answer.
const portalPathPrefix = "/api/v1/portal/"

// NewAPICORSMiddleware returns a CORS middleware for the main API surface (the
// v1 and v3 API groups), or nil when no origins are configured, in which case
// callers skip CORS entirely.
//
// A "*" origin is served without credentials (Access-Control-Allow-Origin: *):
// the API uses the Authorization bearer header instead of cookies, so wildcard
// responses must not opt into credentialed requests.
func NewAPICORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	if len(allowedOrigins) == 0 {
		return nil
	}

	apiCORS := cors.Handler(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: apiAllowedHeaders,
		MaxAge:         300,
	})

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Portal routes carry their own credentialed CORS handling.
			if strings.HasPrefix(r.URL.Path, portalPathPrefix) {
				next.ServeHTTP(w, r)

				return
			}

			apiCORS(next).ServeHTTP(w, r)
		})
	}
}
