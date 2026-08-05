// Package httpclient provides utilities for outbound HTTP clients.
package httpclient

import (
	"log/slog"
	"net/http"
	"time"
)

// DefaultSlowRequestThreshold is the latency above which an outbound request is
// considered slow and logged at WARN level with slow=true.
const DefaultSlowRequestThreshold = 2 * time.Second

// LoggingTransport is an [http.RoundTripper] that logs each outbound request's
// method, URL, response status code, and duration.
//
// It wraps a base RoundTripper (defaulting to [http.DefaultTransport] when nil)
// and is safe for concurrent use. The log level reflects the outcome:
//   - successful responses (< 400): Info
//   - client errors (4xx): Warn
//   - server errors (>= 500): Error
//   - transport errors (no response): Warn
//
// Requests slower than the configured slow threshold are always logged at least
// at WARN level and tagged with slow=true, so slow external API calls stand out
// regardless of status code. Set the threshold to 0 via [WithSlowRequestThreshold]
// to disable this behavior.
type LoggingTransport struct {
	base          http.RoundTripper
	logger        *slog.Logger
	slowThreshold time.Duration
}

// Option configures a [LoggingTransport].
type Option func(*LoggingTransport)

// WithSlowRequestThreshold overrides the latency above which a request is
// considered slow. Use 0 to disable slow-request detection.
func WithSlowRequestThreshold(d time.Duration) Option {
	return func(t *LoggingTransport) {
		t.slowThreshold = d
	}
}

// NewLoggingTransport wraps base with request logging. If base is nil,
// [http.DefaultTransport] is used.
func NewLoggingTransport(base http.RoundTripper, logger *slog.Logger, opts ...Option) *LoggingTransport {
	if base == nil {
		base = http.DefaultTransport
	}

	t := &LoggingTransport{
		base:          base,
		logger:        logger,
		slowThreshold: DefaultSlowRequestThreshold,
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// RoundTrip implements [http.RoundTripper].
func (t *LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := t.base.RoundTrip(req)
	duration := time.Since(start)

	attrs := []any{
		slog.String("method", req.Method),
		slog.String("url", req.URL.String()),
		slog.Duration("duration", duration),
	}

	// Flag slow requests so they stand out when triaging external API latency.
	slow := t.slowThreshold > 0 && duration >= t.slowThreshold
	if slow {
		attrs = append(attrs, slog.Bool("slow", true))
	}

	if err != nil {
		t.logger.Log(req.Context(), slog.LevelWarn, "external http request failed", append(attrs, slog.Any("error", err))...)
		return resp, err
	}

	level := slog.LevelInfo
	switch {
	case resp.StatusCode >= http.StatusInternalServerError:
		level = slog.LevelError
	case resp.StatusCode >= http.StatusBadRequest:
		level = slog.LevelWarn
	}

	// Escalate slow-but-successful calls to WARN so they are not buried in INFO.
	if slow && level < slog.LevelWarn {
		level = slog.LevelWarn
	}

	t.logger.Log(req.Context(), level, "external http request completed", append(attrs, slog.Int("status_code", resp.StatusCode))...)

	return resp, err
}
