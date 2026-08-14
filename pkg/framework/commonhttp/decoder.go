package commonhttp

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/render"
)

// MaxJSONRequestBodySize bounds request bodies decoded through this package.
// It is deliberately generous so legitimate large batch payloads (for example
// event ingest) still fit, while preventing unbounded per-request memory use.
const MaxJSONRequestBodySize = 32 << 20 // 32 MiB

func JSONRequestBodyDecoder(r *http.Request, out any) error {
	// A nil ResponseWriter is safe here: MaxBytesReader only consults it to
	// request connection teardown after the limit is exceeded.
	body := http.MaxBytesReader(nil, r.Body, MaxJSONRequestBodySize)
	if err := render.DecodeJSON(body, out); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return NewHTTPError(http.StatusRequestEntityTooLarge, fmt.Errorf("request body exceeds limit of %d bytes", maxBytesErr.Limit))
		}
		return NewHTTPError(http.StatusBadRequest, fmt.Errorf("decode json: %w", err))
	}
	return nil
}
