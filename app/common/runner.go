package common

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/oklog/run"
)

// Runner is a helper struct that runs a group of services.
type Runner struct {
	Group  run.Group
	Logger *slog.Logger
}

// Run runs the service group until it terminates. It returns a non-nil error
// only for genuine failures: signal-triggered shutdowns and http.ErrServerClosed
// terminations are normal exits.
func (r Runner) Run() error {
	err := r.Group.Run(run.WithReverseShutdownOrder())
	if e := &(run.SignalError{}); errors.As(err, &e) {
		r.Logger.Info("received signal: shutting down", slog.String("signal", e.Signal.String()))
		return nil
	}
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	r.Logger.Error("application stopped due to error", slog.Any("error", err))
	return err
}
