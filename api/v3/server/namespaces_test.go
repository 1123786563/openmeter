package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	namespaceshandler "github.com/openmeterio/openmeter/api/v3/handlers/namespaces"
	"github.com/openmeterio/openmeter/app/config"
)

func TestListNamespacesRoute(t *testing.T) {
	s := Server{
		Config: &Config{},
		namespacesHandler: namespaceshandler.New(config.NamespaceConfiguration{
			Default:   "default",
			Allowlist: []string{"tenant-a", "tenant-b"},
		}),
	}

	rec := httptest.NewRecorder()
	s.ListNamespaces(rec, httptest.NewRequest(http.MethodGet, "/api/v3/openmeter/namespaces", nil))

	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Default    string   `json:"default"`
		Namespaces []string `json:"namespaces"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "default", body.Default)
	require.Equal(t, []string{"default", "tenant-a", "tenant-b"}, body.Namespaces)
}

func TestNamespacesHandlerIsRequired(t *testing.T) {
	err := (&Config{}).Validate()
	require.ErrorContains(t, err, "namespaces handler is required")
}
