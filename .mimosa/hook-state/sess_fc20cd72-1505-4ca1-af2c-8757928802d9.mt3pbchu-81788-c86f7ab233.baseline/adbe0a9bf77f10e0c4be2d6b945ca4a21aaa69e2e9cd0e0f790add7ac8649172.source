package errorsx

import (
	"errors"
	"fmt"
)

// WithPrefix annotates an error with a prefix.
func WithPrefix(err error, prefix string) error {
	if err == nil {
		return nil
	}

	type unwrapper interface {
		Unwrap() []error
	}

	// Deliberately checking for the unwrapper interface instead of the errors.Is function.
	// We only want to check the top-level error otherwise we may accidentally drop other wrappers from the error chain.
	e, ok := err.(unwrapper)
	if !ok {
		return fmt.Errorf("%s: %w", prefix, err)
	}

	errs := e.Unwrap()

	// Unwrap may return the error's live backing slice (errors.Join does);
	// copy it so prefixing does not rewrite the caller's error tree.
	prefixed := make([]error, len(errs))
	for i, err := range errs {
		prefixed[i] = WithPrefix(err, prefix)
	}

	return errors.Join(prefixed...)
}

var _ error = (*warnError)(nil)

type warnError struct {
	Err error
}

func (w *warnError) Error() string {
	return w.Err.Error()
}

func (w *warnError) Unwrap() error {
	return w.Err
}

func NewWarnError(err error) error {
	if err == nil {
		return nil
	}

	return &warnError{Err: err}
}
