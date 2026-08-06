package controller

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
)

type detectChannelResponsesCapabilityRequest struct {
	Model string `json:"model"`
}

func GetChannelResponsesCapabilities(c *gin.Context) {
	channelId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	channel, err := model.GetChannelById(channelId, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	capabilities, err := model.ListChannelResponsesCapabilities(channelId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"configured_mode": channel.GetSetting().ResponsesUpstreamMode.Normalize(),
			"capabilities":    capabilities,
		},
	})
}

func DetectChannelResponsesCapabilities(c *gin.Context) {
	channelId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	request := detectChannelResponsesCapabilityRequest{}
	if err := c.ShouldBindJSON(&request); err != nil && !errors.Is(err, io.EOF) {
		common.ApiError(c, err)
		return
	}
	channel, err := model.GetChannelById(channelId, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	requestedModel := strings.TrimSpace(request.Model)
	if requestedModel == "" && channel.TestModel != nil {
		requestedModel = strings.TrimSpace(*channel.TestModel)
	}
	if requestedModel == "" {
		models := channel.GetModels()
		if len(models) > 0 {
			requestedModel = strings.TrimSpace(models[0])
		}
	}
	if requestedModel == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "a test model is required to detect Responses capability",
		})
		return
	}

	recorder := httptest.NewRecorder()
	detectContext, _ := gin.CreateTestContext(recorder)
	detectContext.Request = httptest.NewRequestWithContext(
		c.Request.Context(),
		http.MethodPost,
		"/v1/responses",
		nil,
	)
	detectContext.Request.Header.Set("Content-Type", "application/json")
	if newAPIError := middleware.SetupContextForSelectedChannel(detectContext, channel, requestedModel); newAPIError != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": newAPIError.Error(),
		})
		return
	}

	probeRequest := &dto.OpenAIResponsesRequest{Model: requestedModel}
	info := relaycommon.GenRelayInfoResponses(detectContext, probeRequest)
	info.InitChannelMeta(detectContext)
	if err := helper.ModelMappedHelper(detectContext, info, probeRequest); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	capability, detectErr := relay.DetectResponsesCapabilities(detectContext, info)
	response := gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"requested_model": requestedModel,
			"upstream_model":  info.UpstreamModelName,
			"configured_mode": channel.GetSetting().ResponsesUpstreamMode.Normalize(),
			"capability":      capability,
		},
	}
	if detectErr != nil {
		response["message"] = detectErr.Error()
	}
	c.JSON(http.StatusOK, response)
}
