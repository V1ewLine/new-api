package controller

import (
	"errors"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/clusterstatus"

	"github.com/gin-gonic/gin"
)

type createClusterRequest struct {
	ModelID          int    `json:"model_id"`
	Name             string `json:"name"`
	AgentAddress     string `json:"agent_address"`
	AgentBearerToken string `json:"agent_bearer_token"`
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
		ModelID:          request.ModelID,
		Name:             request.Name,
		AgentAddress:     request.AgentAddress,
		AgentBearerToken: request.AgentBearerToken,
	})
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
		"available": false,
		"items":     history,
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
		common.ApiErrorMsg(c, "invalid cluster Agent address or Bearer Token")
	default:
		var pollFailure *clusterstatus.PollFailureError
		if errors.As(err, &pollFailure) {
			common.ApiErrorMsg(c, "cluster telemetry refresh failed: "+pollFailure.Code)
			return
		}
		common.ApiError(c, err)
	}
}
