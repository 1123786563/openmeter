package pagination

import (
	"fmt"
	"math"
)

// MaxPageSize bounds user-supplied page sizes so a single request cannot
// force the server to materialize an arbitrarily large page. Internal
// callers that legitimately page through larger windows construct Page
// values directly without going through Validate.
const MaxPageSize = 1000

type InvalidError struct {
	p   Page
	msg string
}

func (e InvalidError) Error() string {
	return fmt.Sprintf("invalid page: %+v, %s", e.p, e.msg)
}

type Page struct {
	PageSize   int `json:"pageSize"`
	PageNumber int `json:"page"`
}

// NewPage creates a new Page with the given pageNumber and pageSize.
func NewPage(pageNumber int, pageSize int) Page {
	return Page{
		PageSize:   pageSize,
		PageNumber: pageNumber,
	}
}

// NewPageFromRef creates a new Page from pointers to pageNumber and pageSize.
// Useful for handling query parameters.
func NewPageFromRef(pageNumber *int, pageSize *int) Page {
	pn := 0
	ps := 0

	if pageNumber != nil {
		pn = *pageNumber
	}

	if pageSize != nil {
		ps = *pageSize
	}

	return NewPage(pn, ps)
}

func (p Page) Offset() int {
	return p.PageSize * (p.PageNumber - 1)
}

func (p Page) Limit() int {
	return p.PageSize
}

func (p Page) Validate() error {
	if p.PageSize < 0 {
		return &InvalidError{p, "pagesize cannot be negative"}
	}

	if p.PageSize > MaxPageSize {
		return &InvalidError{p, fmt.Sprintf("pagesize cannot be greater than %d", MaxPageSize)}
	}

	if p.PageNumber < 1 {
		return &InvalidError{p, "page has to be at least 1"}
	}

	// The offset is computed as PageSize * (PageNumber - 1); reject
	// combinations that would overflow int and wrap the OFFSET negative.
	// Written as PageNumber-1 > MaxInt/PageSize because MaxInt/PageSize+1
	// itself overflows when PageSize is 1.
	if p.PageSize > 0 && p.PageNumber-1 > math.MaxInt/p.PageSize {
		return &InvalidError{p, "page and pagesize combination is too large"}
	}

	return nil
}

func (p Page) IsZero() bool {
	return p.PageSize == 0 && p.PageNumber == 0
}
