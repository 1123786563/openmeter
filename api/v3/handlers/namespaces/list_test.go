package namespaces

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/app/config"
)

func TestListNamespaces(t *testing.T) {
	request := func() *http.Request {
		return httptest.NewRequest(http.MethodGet, "/api/v3/openmeter/namespaces", nil)
	}

	tests := []struct {
		name        string
		config      config.NamespaceConfiguration
		wantDefault string
		wantList    []string
	}{
		{
			name: "empty allowlist lists only the default namespace",
			config: config.NamespaceConfiguration{
				Default: "default",
			},
			wantDefault: "default",
			wantList:    []string{"default"},
		},
		{
			name: "allowlist namespaces are listed with the default",
			config: config.NamespaceConfiguration{
				Default:   "default",
				Allowlist: []string{"tenant-a", "tenant-b"},
			},
			wantDefault: "default",
			wantList:    []string{"default", "tenant-a", "tenant-b"},
		},
		{
			name: "default is always included even when absent from the allowlist",
			config: config.NamespaceConfiguration{
				Default:   "tenant-a",
				Allowlist: []string{"tenant-b", "tenant-c"},
			},
			wantDefault: "tenant-a",
			wantList:    []string{"tenant-a", "tenant-b", "tenant-c"},
		},
		{
			name: "listing is sorted and deduplicated for stable output",
			config: config.NamespaceConfiguration{
				Default:   "zeta",
				Allowlist: []string{"zeta", "alpha", "alpha", "mid"},
			},
			wantDefault: "zeta",
			wantList:    []string{"alpha", "mid", "zeta"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := New(test.config)

			response := httptest.NewRecorder()
			handler.ListNamespaces().ServeHTTP(response, request())

			require.Equal(t, http.StatusOK, response.Code)

			var body struct {
				Default    string   `json:"default"`
				Namespaces []string `json:"namespaces"`
			}
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))

			require.Equal(t, test.wantDefault, body.Default)
			require.Equal(t, test.wantList, body.Namespaces)
		})
	}
}
