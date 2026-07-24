package controller

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/clusterstatus"

	"github.com/gin-gonic/gin"
)

type createClusterRequest struct {
	ModelID      int    `json:"model_id"`
	Name         string `json:"name"`
	AgentAddress string `json:"agent_address"`
}

func GetClusterSettings(c *gin.Context) {
	availableFrom, err := model.GetClusterTelemetryHistoryAvailableFrom()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"refresh_interval_seconds": common.GetClusterStatusRefreshIntervalSeconds(),
		"retention_days":           common.GetClusterTelemetryRetentionDays(),
		"history_available_from":   availableFrom,
	})
}

func GetClusterOverview(c *gin.Context) {
	service, err := clusterstatus.DefaultService()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo := common.GetPageQuery(c)
	modelID := 0
	if rawModelID := strings.TrimSpace(c.Query("model_id")); rawModelID != "" {
		modelID, err = strconv.Atoi(rawModelID)
		if err != nil || modelID <= 0 {
			common.ApiErrorMsg(c, "invalid model_id")
			return
		}
	}
	health := model.ClusterHealthStatus(strings.TrimSpace(c.Query("status")))
	switch health {
	case "", model.ClusterHealthUnknown, model.ClusterHealthOnline, model.ClusterHealthPartial, model.ClusterHealthAbnormal, model.ClusterHealthOffline:
	default:
		common.ApiErrorMsg(c, "invalid cluster status")
		return
	}
	response, err := service.GetOverview(
		c.Query("search"),
		modelID,
		health,
		pageInfo.GetPage(),
		pageInfo.GetPageSize(),
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, response)
}

func GetClusterModelOptions(c *gin.Context) {
	service, err := clusterstatus.DefaultService()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	options, err := service.ListModelOptions()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, options)
}

func ExportClusterLatestTelemetry(c *gin.Context) {
	service, err := clusterstatus.DefaultService()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	input := clusterstatus.LatestExportInput{
		Scope:  c.Query("scope"),
		Format: c.Query("format"),
		Search: c.Query("search"),
	}
	if rawModelID := strings.TrimSpace(c.Query("model_id")); rawModelID != "" {
		input.ModelID, err = strconv.Atoi(rawModelID)
		if err != nil || input.ModelID <= 0 {
			common.ApiErrorMsg(c, "invalid model_id")
			return
		}
	}
	if rawClusterID := strings.TrimSpace(c.Query("cluster_id")); rawClusterID != "" {
		input.ClusterID, err = strconv.ParseInt(rawClusterID, 10, 64)
		if err != nil || input.ClusterID <= 0 {
			common.ApiErrorMsg(c, "invalid cluster_id")
			return
		}
	}
	input.Health = model.ClusterHealthStatus(strings.TrimSpace(c.Query("status")))

	prepared, err := service.PrepareLatestExport(input)
	if err != nil {
		writeClusterServiceError(c, err)
		return
	}
	c.Header("Content-Type", prepared.ContentType)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, prepared.Filename))
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	if err := prepared.WriteTo(c.Writer); err != nil {
		common.SysError("write cluster telemetry export failed: " + err.Error())
		return
	}
	recordManageAudit(c, "cluster.export", map[string]interface{}{
		"scope":  prepared.Scope,
		"format": prepared.Format,
		"count":  prepared.ClusterCount,
	})
}

func ExportClusterTelemetryHistory(c *gin.Context) {
	service, err := clusterstatus.DefaultService()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	startAt, err := time.Parse(time.RFC3339, strings.TrimSpace(c.Query("start_at")))
	if err != nil {
		common.ApiErrorMsg(c, "invalid start_at")
		return
	}
	endAt, err := time.Parse(time.RFC3339, strings.TrimSpace(c.Query("end_at")))
	if err != nil {
		common.ApiErrorMsg(c, "invalid end_at")
		return
	}
	input := clusterstatus.HistoryExportInput{
		Scope:   c.Query("scope"),
		Search:  c.Query("search"),
		Health:  model.ClusterHealthStatus(strings.TrimSpace(c.Query("status"))),
		StartAt: startAt,
		EndAt:   endAt,
	}
	if rawModelID := strings.TrimSpace(c.Query("model_id")); rawModelID != "" {
		input.ModelID, err = strconv.Atoi(rawModelID)
		if err != nil || input.ModelID <= 0 {
			common.ApiErrorMsg(c, "invalid model_id")
			return
		}
	}
	if rawClusterID := strings.TrimSpace(c.Query("cluster_id")); rawClusterID != "" {
		input.ClusterID, err = strconv.ParseInt(rawClusterID, 10, 64)
		if err != nil || input.ClusterID <= 0 {
			common.ApiErrorMsg(c, "invalid cluster_id")
			return
		}
	}

	prepared, err := service.PrepareHistoryExport(c.Request.Context(), input)
	if err != nil {
		writeClusterServiceError(c, err)
		return
	}
	c.Header("Content-Type", prepared.ContentType)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, prepared.Filename))
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	if err := prepared.WriteTo(c.Writer); err != nil {
		common.SysError("write cluster telemetry history export failed: " + err.Error())
		return
	}
	recordManageAudit(c, "cluster.export.history", map[string]interface{}{
		"scope":         prepared.Scope,
		"cluster_count": prepared.ClusterCount,
		"sample_count":  prepared.SampleCount,
		"start_at":      prepared.StartAt.UTC().Format(time.RFC3339),
		"end_at":        prepared.EndAt.UTC().Format(time.RFC3339),
	})
}

func CreateCluster(c *gin.Context) {
	var request createClusterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorMsg(c, "invalid cluster request")
		return
	}
	service, err := clusterstatus.DefaultService()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	response, err := service.CreateCluster(c.Request.Context(), clusterstatus.CreateClusterInput{
		ModelID:      request.ModelID,
		Name:         request.Name,
		AgentAddress: request.AgentAddress,
	})
	if err != nil {
		writeClusterServiceError(c, err)
		return
	}
	disableSecretResponseCaching(c)
	common.ApiSuccess(c, response)
}

func RotateClusterCredential(c *gin.Context) {
	clusterID, ok := requireClusterID(c)
	if !ok {
		return
	}
	service, err := clusterstatus.DefaultService()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	response, err := service.RotateCredential(c.Request.Context(), clusterID)
	if err != nil {
		writeClusterServiceError(c, err)
		return
	}
	disableSecretResponseCaching(c)
	common.ApiSuccess(c, response)
}

func VerifyClusterCredential(c *gin.Context) {
	clusterID, ok := requireClusterID(c)
	if !ok {
		return
	}
	service, err := clusterstatus.DefaultService()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	response, err := service.VerifyCredential(c.Request.Context(), clusterID)
	if err != nil {
		writeClusterServiceError(c, err)
		return
	}
	common.ApiSuccess(c, response)
}

func DeleteCluster(c *gin.Context) {
	clusterID, ok := requireClusterID(c)
	if !ok {
		return
	}
	service, err := clusterstatus.DefaultService()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := service.DeleteCluster(clusterID); err != nil {
		writeClusterServiceError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func GetClusterModelDetail(c *gin.Context) {
	modelID, err := strconv.Atoi(c.Param("modelId"))
	if err != nil || modelID <= 0 {
		common.ApiErrorMsg(c, "invalid model id")
		return
	}
	service, err := clusterstatus.DefaultService()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	response, err := service.GetModelDetail(modelID)
	if err != nil {
		writeClusterServiceError(c, err)
		return
	}
	common.ApiSuccess(c, response)
}

func GetClusterDetail(c *gin.Context) {
	clusterID, ok := requireClusterID(c)
	if !ok {
		return
	}
	service, err := clusterstatus.DefaultService()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	response, err := service.GetCluster(clusterID)
	if err != nil {
		writeClusterServiceError(c, err)
		return
	}
	common.ApiSuccess(c, response)
}

func GetClusterLatestTelemetry(c *gin.Context) {
	clusterID, ok := requireClusterID(c)
	if !ok {
		return
	}
	service, err := clusterstatus.DefaultService()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	telemetry, err := service.GetLatestTelemetry(clusterID)
	if err != nil {
		writeClusterServiceError(c, err)
		return
	}
	common.ApiSuccess(c, telemetry)
}

func RefreshClusterTelemetry(c *gin.Context) {
	clusterID, ok := requireClusterID(c)
	if !ok {
		return
	}
	response, err := clusterstatus.RefreshCluster(c.Request.Context(), clusterID)
	if err != nil {
		writeClusterServiceError(c, err)
		return
	}
	common.ApiSuccess(c, response)
}

func GetClusterTelemetryHistory(c *gin.Context) {
	clusterID, ok := requireClusterID(c)
	if !ok {
		return
	}
	service, err := clusterstatus.DefaultService()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	history, err := service.GetTelemetryHistory(c.Request.Context(), clusterID)
	if err != nil {
		writeClusterServiceError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"available":      true,
		"retention_days": common.GetClusterTelemetryRetentionDays(),
		"items":          history,
	})
}

func requireClusterID(c *gin.Context) (int64, bool) {
	clusterID, err := strconv.ParseInt(c.Param("clusterId"), 10, 64)
	if err != nil || clusterID <= 0 {
		common.ApiErrorMsg(c, "invalid cluster id")
		return 0, false
	}
	return clusterID, true
}

func disableSecretResponseCaching(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
}

func writeClusterServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, clusterstatus.ErrClusterNotFound):
		common.ApiErrorMsg(c, "cluster not found")
	case errors.Is(err, clusterstatus.ErrClusterModelNotFound):
		common.ApiErrorMsg(c, "model not found")
	case errors.Is(err, clusterstatus.ErrClusterModelDisabled):
		common.ApiErrorMsg(c, "model is disabled")
	case errors.Is(err, clusterstatus.ErrClusterPollInProgress):
		common.ApiErrorMsg(c, "cluster refresh is already in progress")
	case errors.Is(err, clusterstatus.ErrInvalidLinkSecret):
		common.ApiErrorMsg(c, "invalid cluster Agent address")
	case errors.Is(err, clusterstatus.ErrClusterCredentialUnavailable):
		common.ApiErrorMsg(c, "cluster credential cannot be rotated; recreate the cluster")
	case errors.Is(err, clusterstatus.ErrClusterExportInvalid):
		common.ApiErrorMsg(c, "invalid cluster export request")
	case errors.Is(err, clusterstatus.ErrClusterExportTooLarge):
		common.ApiErrorMsg(c, "cluster export exceeds the allowed size")
	default:
		var pollFailure *clusterstatus.PollFailureError
		if errors.As(err, &pollFailure) {
			common.ApiErrorMsg(c, "cluster telemetry refresh failed: "+pollFailure.Code)
			return
		}
		common.ApiError(c, err)
	}
}
