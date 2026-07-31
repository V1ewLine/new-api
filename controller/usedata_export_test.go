package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service/dashboardexport"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelfModelAnalyticsExportRejectsMissingAuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handlers := []struct {
		name    string
		handler gin.HandlerFunc
	}{
		{
			name:    "model options",
			handler: GetSelfModelAnalyticsExportOptions,
		},
		{
			name:    "CSV export",
			handler: ExportSelfModelAnalytics,
		},
	}

	for _, test := range handlers {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			request, err := http.NewRequest(
				http.MethodGet,
				"/?start_timestamp=1785081600&end_timestamp=1785085200&granularity=hour&timezone=UTC",
				nil,
			)
			require.NoError(t, err)
			ctx.Request = request

			test.handler(ctx)

			var response struct {
				Success bool   `json:"success"`
				Message string `json:"message"`
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
			assert.False(t, response.Success)
			assert.Equal(
				t,
				dashboardexport.ErrInvalidInput.Error(),
				response.Message,
			)
		})
	}
}
