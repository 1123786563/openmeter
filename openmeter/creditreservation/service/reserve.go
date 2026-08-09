package service

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/creditreservation"
)

type fundingSplit struct {
	prepaid    int64
	enterprise int64
}

// splitFunding is the strict prepaid boundary used by Reserve. A nil limit is
// not a zero-value convenience: it explicitly prohibits receivables.
func splitFunding(prepaid int64, enterpriseLimit *int64, required int64) (fundingSplit, error) {
	if prepaid < 0 || required < 0 {
		return fundingSplit{}, fmt.Errorf("%w: negative balance or hold", creditreservation.ErrInsufficientFunds)
	}

	split := fundingSplit{prepaid: min(prepaid, required)}
	remaining := required - split.prepaid
	if remaining == 0 {
		return split, nil
	}
	if enterpriseLimit == nil || *enterpriseLimit < remaining {
		return fundingSplit{}, creditreservation.ErrInsufficientFunds
	}

	split.enterprise = remaining
	return split, nil
}
