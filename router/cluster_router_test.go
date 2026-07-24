package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClusterMutationRoutesRequireAdminAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetApiRouter(engine)

	routes := make(map[string]struct{}, len(engine.Routes()))
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}
	testCases := []struct {
		method string
		path   string
		route  string
	}{
		{method: http.MethodGet, path: "/api/clusters/settings", route: "/api/clusters/settings"},
		{method: http.MethodGet, path: "/api/clusters/export/latest?scope=models&format=csv", route: "/api/clusters/export/latest"},
		{method: http.MethodGet, path: "/api/clusters/export/history?scope=all", route: "/api/clusters/export/history"},
		{method: http.MethodGet, path: "/api/clusters/1/telemetry/trends?start_at=2026-07-25T00:00:00Z&end_at=2026-07-25T01:00:00Z", route: "/api/clusters/:clusterId/telemetry/trends"},
		{method: http.MethodDelete, path: "/api/clusters/1", route: "/api/clusters/:clusterId"},
		{method: http.MethodPost, path: "/api/clusters/1/credential/rotate", route: "/api/clusters/:clusterId/credential/rotate"},
		{method: http.MethodPost, path: "/api/clusters/1/credential/verify", route: "/api/clusters/:clusterId/credential/verify"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.method+" "+testCase.route, func(t *testing.T) {
			_, registered := routes[testCase.method+" "+testCase.route]
			require.True(t, registered)

			request := httptest.NewRequest(testCase.method, testCase.path, nil)
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)

			assert.Equal(t, http.StatusUnauthorized, response.Code)
			assert.Contains(t, response.Body.String(), "AUTH_UNAUTHORIZED")
		})
	}
}
