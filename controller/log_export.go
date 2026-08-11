package controller

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service/usagelogexport"

	"github.com/gin-gonic/gin"
)

func ExportAllLogs(c *gin.Context) {
	exportUsageLogs(c, false)
}

func ExportSelfLogs(c *gin.Context) {
	exportUsageLogs(c, true)
}

func exportUsageLogs(c *gin.Context, selfOnly bool) {
	if strings.TrimSpace(c.DefaultQuery("format", "csv")) != "csv" {
		common.ApiErrorMsg(c, usagelogexport.ErrInvalidInput.Error())
		return
	}

	startTimestamp, err := strconv.ParseInt(strings.TrimSpace(c.Query("start_timestamp")), 10, 64)
	if err != nil {
		common.ApiErrorMsg(c, usagelogexport.ErrInvalidInput.Error())
		return
	}
	endTimestamp, err := strconv.ParseInt(strings.TrimSpace(c.Query("end_timestamp")), 10, 64)
	if err != nil {
		common.ApiErrorMsg(c, usagelogexport.ErrInvalidInput.Error())
		return
	}
	logType, err := strconv.Atoi(strings.TrimSpace(c.DefaultQuery("type", "0")))
	if err != nil {
		common.ApiErrorMsg(c, usagelogexport.ErrInvalidInput.Error())
		return
	}
	channel, err := strconv.Atoi(strings.TrimSpace(c.DefaultQuery("channel", "0")))
	if err != nil {
		common.ApiErrorMsg(c, usagelogexport.ErrInvalidInput.Error())
		return
	}

	input := usagelogexport.Input{
		StartTimestamp:    startTimestamp,
		EndTimestamp:      endTimestamp,
		Timezone:          strings.TrimSpace(c.DefaultQuery("timezone", "UTC")),
		LogType:           logType,
		ModelName:         c.Query("model_name"),
		TokenName:         c.Query("token_name"),
		Group:             c.Query("group"),
		RequestID:         c.Query("request_id"),
		UpstreamRequestID: c.Query("upstream_request_id"),
	}
	if selfOnly {
		input.UserID = c.GetInt("id")
		if input.UserID <= 0 {
			common.ApiErrorMsg(c, usagelogexport.ErrInvalidInput.Error())
			return
		}
	} else {
		input.Username = c.Query("username")
		input.Channel = channel
	}

	prepared, err := usagelogexport.Prepare(input)
	if err != nil {
		writeUsageLogExportError(c, err)
		return
	}

	c.Header("Content-Type", prepared.ContentType)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, prepared.Filename))
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	if err := prepared.WriteTo(c.Writer); err != nil {
		common.SysError("write usage log export failed: " + err.Error())
		return
	}

	if !selfOnly {
		recordManageAudit(c, "usage_logs.export", map[string]interface{}{
			"row_count": prepared.RowCount,
			"start_at":  prepared.EffectiveStart.Format(time.RFC3339),
			"end_at":    prepared.EffectiveEnd.Format(time.RFC3339),
		})
	}
}

func writeUsageLogExportError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usagelogexport.ErrInvalidInput),
		errors.Is(err, usagelogexport.ErrTooManyRows),
		errors.Is(err, usagelogexport.ErrNoData):
		common.ApiErrorMsg(c, err.Error())
	default:
		common.ApiError(c, err)
	}
}
