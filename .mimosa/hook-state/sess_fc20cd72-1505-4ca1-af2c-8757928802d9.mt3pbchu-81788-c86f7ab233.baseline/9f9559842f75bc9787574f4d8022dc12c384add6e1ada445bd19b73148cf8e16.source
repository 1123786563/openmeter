package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/creditreservation"
)

func TestMatchesReverseIdentityRejectsAChangedReplay(t *testing.T) {
	stored := creditreservation.CommandIdentity{IdempotencyKey: "reverse-1", PayloadHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	changed := creditreservation.CommandIdentity{IdempotencyKey: "reverse-2", PayloadHash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}

	require.ErrorIs(t, matchesReverseIdentity(stored, changed), creditreservation.ErrIdempotencyConflict)
	require.NoError(t, matchesReverseIdentity(stored, stored))
}
