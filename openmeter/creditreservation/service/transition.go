package service

import (
	"fmt"
	"strings"

	"github.com/openmeterio/openmeter/openmeter/creditreservation"
)

func validateReleaseEvidence(state creditreservation.ReservationState, evidence creditreservation.Evidence) error {
	switch state {
	case creditreservation.ReservationStateActive:
		if evidence.Kind == "" || evidence.Kind == creditreservation.EvidenceNotSent {
			return nil
		}
	case creditreservation.ReservationStateExecuting, creditreservation.ReservationStateUnknown:
		if evidence.Kind == creditreservation.EvidenceProviderConfirmedNotExecuted && strings.TrimSpace(evidence.Reference) != "" {
			return nil
		}
	}
	return fmt.Errorf("%w: release from %s", creditreservation.ErrTransitionEvidenceRequired, state)
}
