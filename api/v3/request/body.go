package request

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
)

// maxBodySize bounds request bodies parsed through this package to prevent
// unbounded per-request memory use.
const maxBodySize = commonhttp.MaxJSONRequestBodySize

func ParseBody(r *http.Request, payload any) error {
	// A nil ResponseWriter is safe here: MaxBytesReader only consults it to
	// request connection teardown after the limit is exceeded.
	decoder := json.NewDecoder(http.MaxBytesReader(nil, r.Body, maxBodySize))
	if err := decoder.Decode(&payload); err != nil {
		return bodyDecodeError(r.Context(), err)
	}

	return nil
}

// ParseOptionalBody parses the request body if present, leaving payload unchanged if the body is empty.
func ParseOptionalBody(r *http.Request, payload any) error {
	if r.Body == nil {
		return nil
	}

	// A nil ResponseWriter is safe here: MaxBytesReader only consults it to
	// request connection teardown after the limit is exceeded.
	decoder := json.NewDecoder(http.MaxBytesReader(nil, r.Body, maxBodySize))
	if err := decoder.Decode(&payload); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}

		return bodyDecodeError(r.Context(), err)
	}

	return nil
}

func bodyDecodeError(ctx context.Context, err error) error {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return apierrors.NewBadRequestError(ctx, err,
			apierrors.InvalidParameters{
				apierrors.InvalidParameter{
					Reason: fmt.Sprintf("request body exceeds limit of %d bytes", maxBytesErr.Limit),
					Source: apierrors.InvalidParamSourceBody,
				},
			},
		)
	}

	return apierrors.NewBadRequestError(ctx, err,
		apierrors.InvalidParameters{
			apierrors.InvalidParameter{
				Reason: "unable to parse body",
				Source: apierrors.InvalidParamSourceBody,
			},
		},
	)
}
