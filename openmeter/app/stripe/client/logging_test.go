package client

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v80"
	stripecclient "github.com/stripe/stripe-go/v80/client"

	"github.com/openmeterio/openmeter/pkg/framework/transport/httpclient"
)

// redirectTransport rewrites outbound requests to point at a mock server,
// preserving the path and query. It is only used in tests to redirect
// stripe-go's hardcoded api.stripe.com calls to an httptest.Server.
type redirectTransport struct {
	target *url.URL
	base   http.RoundTripper
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.URL.Scheme = t.target.Scheme
	cloned.URL.Host = t.target.Host
	cloned.Host = t.target.Host
	return t.base.RoundTrip(cloned)
}

// stripeErrorBody is the minimal JSON Stripe returns for an error response.
const stripeErrorBody = `{
	"error": {
		"type": "invalid_request_error",
		"message": "test error"
	}
}`

// TestStripeClient_Logging verifies that a Stripe API call produces the
// expected structured log line via the shared LoggingTransport, and that
// the log level reflects the response status code.
func TestStripeClient_Logging(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		wantLevel  string
		wantStatus string
		// expectErr is true when the Stripe client surfaces an error for this
		// status (4xx/5xx); the 2xx case succeeds.
		expectErr bool
	}{
		{
			name:       "success 2xx logs INFO",
			status:     http.StatusOK,
			wantLevel:  "level=INFO",
			wantStatus: "status_code=200",
			expectErr:  false,
		},
		{
			name:       "client error 4xx logs WARN",
			status:     http.StatusNotFound,
			wantLevel:  "level=WARN",
			wantStatus: "status_code=404",
			expectErr:  true,
		},
		{
			name:       "server error 5xx logs ERROR",
			status:     http.StatusBadGateway,
			wantLevel:  "level=ERROR",
			wantStatus: "status_code=502",
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock Stripe API: respond to GET /v1/account with tt.status.
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				if tt.status == http.StatusOK {
					_, _ = w.Write([]byte(`{
						"id": "acct_test123",
						"object": "account",
						"country": "US"
					}`))
				} else {
					_, _ = w.Write([]byte(stripeErrorBody))
				}
			}))
			defer server.Close()

			target, err := url.Parse(server.URL)
			require.NoError(t, err)

			// Capture slog output so we can assert on it.
			var logBuf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

			// LoggingTransport wraps the redirect transport, mirroring production wiring.
			loggingTransport := httpclient.NewLoggingTransport(
				&redirectTransport{target: target, base: http.DefaultTransport},
				logger.With("subsystem", "stripe"),
			)

			backend := stripe.GetBackendWithConfig(stripe.APIBackend, &stripe.BackendConfig{
				LeveledLogger: leveledLogger{logger: logger},
				HTTPClient:    &http.Client{Transport: loggingTransport},
			})

			api := &stripecclient.API{}
			api.Init("sk_test_xxx", &stripe.Backends{API: backend, Connect: backend, Uploads: backend})

			sc := &stripeClient{client: api, namespace: "ns-test"}

			_, err = sc.GetAccount(context.Background())
			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			out := logBuf.String()
			t.Logf("captured logs:\n%s", out)

			for _, want := range []string{
				"external http request completed",
				"subsystem=stripe",
				"method=GET",
				tt.wantStatus,
				tt.wantLevel,
				"duration=",
			} {
				require.True(t, strings.Contains(out, want), "log missing %q\nfull output:\n%s", want, out)
			}
		})
	}
}
