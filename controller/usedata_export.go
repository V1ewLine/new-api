package controller

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service/dashboardexport"

	"github.com/gin-gonic/gin"
)

func GetModelAnalyticsExportOptions(c *gin.Context) {
	getModelAnalyticsExportOptions(c, false)
}

func GetSelfModelAnalyticsExportOptions(c *gin.Context) {
	getModelAnalyticsExportOptions(c, true)
}

func ExportModelAnalytics(c *gin.Context) {
	exportModelAnalytics(c, false)
}

func ExportSelfModelAnalytics(c *gin.Context) {
	exportModelAnalytics(c, true)
}

func getModelAnalyticsExportOptions(c *gin.Context, selfOnly bool) {
	startTimestamp, endTimestamp, ok := parseModelAnalyticsExportTimeRange(c)
	if !ok {
		return
	}
	input := dashboardexport.ModelOptionInput{
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
		Granularity: dashboardexport.Granularity(
			strings.TrimSpace(c.DefaultQuery("granularity", "hour")),
		),
		Timezone: strings.TrimSpace(c.DefaultQuery("timezone", "UTC")),
	}
	if selfOnly {
		input.UserID = c.GetInt("id")
		if input.UserID <= 0 {
			common.ApiErrorMsg(c, dashboardexport.ErrInvalidInput.Error())
			return
		}
	} else {
		input.Username = strings.TrimSpace(c.Query("username"))
	}

	options, err := dashboardexport.ListModelOptions(c.Request.Context(), input)
	if err != nil {
		writeModelAnalyticsExportError(c, err)
		return
	}
	common.ApiSuccess(c, options)
}

func exportModelAnalytics(c *gin.Context, selfOnly bool) {
	if format := strings.TrimSpace(c.DefaultQuery("format", "csv")); format != "csv" {
		common.ApiErrorMsg(c, dashboardexport.ErrInvalidInput.Error())
		return
	}
	startTimestamp, endTimestamp, ok := parseModelAnalyticsExportTimeRange(c)
	if !ok {
		return
	}
	input := dashboardexport.Input{
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
		Granularity: dashboardexport.Granularity(
			strings.TrimSpace(c.DefaultQuery("granularity", "hour")),
		),
		Timezone:  strings.TrimSpace(c.DefaultQuery("timezone", "UTC")),
		ModelName: strings.TrimSpace(c.Query("model_name")),
	}
	if selfOnly {
		input.UserID = c.GetInt("id")
		if input.UserID <= 0 {
			common.ApiErrorMsg(c, dashboardexport.ErrInvalidInput.Error())
			return
		}
	} else {
		input.Username = strings.TrimSpace(c.Query("username"))
	}

	prepared, err := dashboardexport.Prepare(c.Request.Context(), input)
	if err != nil {
		writeModelAnalyticsExportError(c, err)
		return
	}

	c.Header("Content-Type", prepared.ContentType)
	c.Header(
		"Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s"`, prepared.Filename),
	)
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	if err := prepared.WriteTo(c.Writer); err != nil {
		common.SysError("write model analytics export failed: " + err.Error())
		return
	}

	if !selfOnly {
		modelName := prepared.ModelName
		if modelName == "" {
			modelName = "all"
		}
		recordManageAudit(c, "dashboard.model_analytics.export", map[string]interface{}{
			"model":       modelName,
			"granularity": prepared.Granularity,
			"row_count":   prepared.RowCount,
			"model_count": prepared.ModelCount,
			"start_at":    prepared.EffectiveStart.Format(time.RFC3339),
			"end_at":      prepared.EffectiveEnd.Format(time.RFC3339),
		})
	}
}

func parseModelAnalyticsExportTimeRange(c *gin.Context) (int64, int64, bool) {
	startTimestamp, err := strconv.ParseInt(
		strings.TrimSpace(c.Query("start_timestamp")),
		10,
		64,
	)
	if err != nil || startTimestamp <= 0 {
		common.ApiErrorMsg(c, dashboardexport.ErrInvalidInput.Error())
		return 0, 0, false
	}
	endTimestamp, err := strconv.ParseInt(
		strings.TrimSpace(c.Query("end_timestamp")),
		10,
		64,
	)
	if err != nil || endTimestamp <= startTimestamp {
		common.ApiErrorMsg(c, dashboardexport.ErrInvalidInput.Error())
		return 0, 0, false
	}
	return startTimestamp, endTimestamp, true
}

func writeModelAnalyticsExportError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, dashboardexport.ErrInvalidInput),
		errors.Is(err, dashboardexport.ErrRangeTooLarge),
		errors.Is(err, dashboardexport.ErrTooManyRows),
		errors.Is(err, dashboardexport.ErrNoData):
		common.ApiErrorMsg(c, err.Error())
	default:
		common.ApiError(c, err)
	}
}
