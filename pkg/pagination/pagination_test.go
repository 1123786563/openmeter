package pagination_test

import (
	"bytes"
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/pkg/pagination"
)

func TestShouldFlattenPageInfoWhenMarshalling(t *testing.T) {
	assert := assert.New(t)
	pagedRes := pagination.Result[int]{
		Items:      []int{1, 2, 3},
		TotalCount: 3,
		Page: pagination.Page{
			PageSize:   10,
			PageNumber: 1,
		},
	}

	expected := `{"pageSize":10,"page":1,"totalCount":3,"items":[1,2,3]}
`

	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(true)

	if err := enc.Encode(pagedRes); err != nil {
		t.Fatalf("failed to marshal paged response: %v", err)
	}

	assert.JSONEq(expected, buf.String())
	// enforces ordering
	assert.Equal(expected, buf.String())
}

func TestPageValidateBounds(t *testing.T) {
	tests := []struct {
		name    string
		page    pagination.Page
		invalid bool
	}{
		{name: "valid page", page: pagination.NewPage(1, 100)},
		{name: "unit page size", page: pagination.NewPage(1, 1)},
		{name: "max page size", page: pagination.NewPage(1, pagination.MaxPageSize)},
		{name: "largest non-overflowing page", page: pagination.NewPage(math.MaxInt/pagination.MaxPageSize+1, pagination.MaxPageSize)},
		{name: "oversized page size", page: pagination.NewPage(1, pagination.MaxPageSize+1), invalid: true},
		{name: "negative page size", page: pagination.NewPage(1, -1), invalid: true},
		{
			name:    "offset overflow",
			page:    pagination.NewPage(math.MaxInt/pagination.MaxPageSize+2, pagination.MaxPageSize),
			invalid: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.page.Validate()

			if test.invalid {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
