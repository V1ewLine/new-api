package router

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestUsageLogExportRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetApiRouter(engine)

	routes := make(map[string]struct{}, len(engine.Routes()))
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}

	_, hasAdminExport := routes[http.MethodGet+" /api/log/export"]
	_, hasSelfExport := routes[http.MethodGet+" /api/log/self/export"]
	assert.True(t, hasAdminExport)
	assert.True(t, hasSelfExport)
}
