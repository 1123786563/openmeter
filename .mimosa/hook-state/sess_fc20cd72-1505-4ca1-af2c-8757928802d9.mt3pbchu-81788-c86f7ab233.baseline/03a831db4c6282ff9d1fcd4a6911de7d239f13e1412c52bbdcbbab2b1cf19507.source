// Hand-written wire tests for the generated credit reservation client.
package openmeter_test

import (
	"net/http"
	"testing"
)

func TestClientInitializesCreditReservationsService(t *testing.T) {
	recorder := &requestRecorder{}
	client := newTestClient(t, recorder.handler(http.StatusOK, `{"id":"reservation-1"}`))

	if client.CreditReservations == nil {
		t.Fatal("CreditReservations service is not initialized")
	}
}
