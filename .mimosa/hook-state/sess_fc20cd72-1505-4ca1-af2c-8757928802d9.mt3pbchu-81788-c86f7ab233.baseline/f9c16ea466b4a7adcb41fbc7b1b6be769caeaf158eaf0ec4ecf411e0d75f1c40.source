package httpclient

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestLoggingTransport_LevelsByStatus(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		wantLevel  string
		wantStatus string
	}{
		{name: "success", status: http.StatusOK, wantLevel: "level=INFO", wantStatus: "status_code=200"},
		{name: "redirect", status: http.StatusMovedPermanently, wantLevel: "level=INFO", wantStatus: "status_code=301"},
		{name: "client error", status: http.StatusNotFound, wantLevel: "level=WARN", wantStatus: "status_code=404"},
		{name: "server error", status: http.StatusBadGateway, wantLevel: "level=ERROR", wantStatus: "status_code=502"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer server.Close()

			var buf bytes.Buffer
			client := &http.Client{Transport: NewLoggingTransport(http.DefaultTransport, newTestLogger(&buf))}

			resp, err := client.Get(server.URL + "/path")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer resp.Body.Close()

			out := buf.String()
			for _, want := range []string{tt.wantLevel, tt.wantStatus, "method=GET", "url=", "duration="} {
				if !strings.Contains(out, want) {
					t.Errorf("log output missing %q\ngot: %s", want, out)
				}
			}
		})
	}
}

func TestLoggingTransport_TransportError(t *testing.T) {
	// A server that immediately closes the connection is unreachable, but to
	// deterministically trigger a transport error we point the client at a
	// closed server.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	var buf bytes.Buffer
	client := &http.Client{Transport: NewLoggingTransport(http.DefaultTransport, newTestLogger(&buf))}

	resp, err := client.Get(server.URL)
	if resp != nil {
		resp.Body.Close()
	}
	if err == nil {
		t.Fatal("expected a transport error")
	}

	out := buf.String()
	for _, want := range []string{"level=WARN", "method=GET", "url=", "duration=", "error="} {
		if !strings.Contains(out, want) {
			t.Errorf("log output missing %q\ngot: %s", want, out)
		}
	}
}

func TestLoggingTransport_SlowRequestFlagged(t *testing.T) {
	tests := []struct {
		name         string
		threshold    time.Duration
		serverDelay  time.Duration
		wantSlow     bool
		wantLevel    string
		wantDuration string
	}{
		{
			name:         "fast request no slow flag",
			threshold:    200 * time.Millisecond,
			serverDelay:  0,
			wantSlow:     false,
			wantLevel:    "level=INFO",
			wantDuration: "", // duration is always logged; not asserted here
		},
		{
			name:         "slow request flagged and escalated to WARN",
			threshold:    50 * time.Millisecond,
			serverDelay:  120 * time.Millisecond,
			wantSlow:     true,
			wantLevel:    "level=WARN",
			wantDuration: "",
		},
		{
			name:         "threshold disabled never flags",
			threshold:    0,
			serverDelay:  120 * time.Millisecond,
			wantSlow:     false,
			wantLevel:    "level=INFO",
			wantDuration: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.serverDelay > 0 {
					time.Sleep(tt.serverDelay)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			var buf bytes.Buffer
			client := &http.Client{
				Transport: NewLoggingTransport(
					http.DefaultTransport,
					newTestLogger(&buf),
					WithSlowRequestThreshold(tt.threshold),
				),
			}

			resp, err := client.Get(server.URL + "/path")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer resp.Body.Close()

			out := buf.String()
			t.Logf("captured: %s", out)

			if !strings.Contains(out, tt.wantLevel) {
				t.Errorf("expected %q in log, got: %s", tt.wantLevel, out)
			}
			if tt.wantSlow && !strings.Contains(out, "slow=true") {
				t.Errorf("expected slow=true in log, got: %s", out)
			}
			if !tt.wantSlow && strings.Contains(out, "slow=true") {
				t.Errorf("did not expect slow=true in log, got: %s", out)
			}
		})
	}
}

func TestLoggingTransport_NilBaseUsesDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var buf bytes.Buffer
	// base nil should fall back to http.DefaultTransport and still succeed.
	client := &http.Client{Transport: NewLoggingTransport(nil, newTestLogger(&buf))}

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if !strings.Contains(buf.String(), "status_code=200") {
		t.Errorf("expected successful log, got: %s", buf.String())
	}
}
