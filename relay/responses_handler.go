package relay

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	appconstant "github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func ResponsesHelper(c *gin.Context, info *relaycommon.RelayInfo) (newAPIError *types.NewAPIError) {
	info.InitChannelMeta(c)
	if info.RelayMode == relayconstant.RelayModeResponsesCompact {
		switch info.ApiType {
		case appconstant.APITypeOpenAI, appconstant.APITypeCodex:
		default:
			return types.NewErrorWithStatusCode(
				fmt.Errorf("unsupported endpoint %q for api type %d", "/v1/responses/compact", info.ApiType),
				types.ErrorCodeInvalidRequest,
				http.StatusBadRequest,
				types.ErrOptionWithSkipRetry(),
			)
		}
	}

	var responsesReq *dto.OpenAIResponsesRequest
	switch req := info.Request.(type) {
	case *dto.OpenAIResponsesRequest:
		responsesReq = req
	case *dto.OpenAIResponsesCompactionRequest:
		// Only fields documented for POST /v1/responses/compact are forwarded:
		// model, input, instructions, previous_response_id, prompt_cache_key,
		// prompt_cache_options, prompt_cache_retention, service_tier.
		// Undocumented Codex-parity fields (tools, reasoning, text) are parsed
		// for client compatibility but intentionally not sent upstream.
		responsesReq = &dto.OpenAIResponsesRequest{
			Model:                req.Model,
			Input:                req.Input,
			Instructions:         req.Instructions,
			PreviousResponseID:   req.PreviousResponseID,
			ParallelToolCalls:    req.ParallelToolCalls,
			ServiceTier:          req.ServiceTier,
			PromptCacheKey:       req.PromptCacheKey,
			PromptCacheOptions:   req.PromptCacheOptions,
			PromptCacheRetention: req.PromptCacheRetention,
		}
	default:
		return types.NewErrorWithStatusCode(
			fmt.Errorf("invalid request type, expected dto.OpenAIResponsesRequest or dto.OpenAIResponsesCompactionRequest, got %T", info.Request),
			types.ErrorCodeInvalidRequest,
			http.StatusBadRequest,
			types.ErrOptionWithSkipRetry(),
		)
	}

	request, err := common.DeepCopy(responsesReq)
	if err != nil {
		return types.NewError(fmt.Errorf("failed to copy request to GeneralOpenAIRequest: %w", err), types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
	}

	err = helper.ModelMappedHelper(c, info, request)
	if err != nil {
		return types.NewError(err, types.ErrorCodeChannelModelMappedError, types.ErrOptionWithSkipRetry())
	}

	adaptor := GetAdaptor(info.ApiType)
	if adaptor == nil {
		return types.NewError(fmt.Errorf("invalid api type: %d", info.ApiType), types.ErrorCodeInvalidApiType, types.ErrOptionWithSkipRetry())
	}
	adaptor.Init(info)
	runtimeMode := model.ResponsesCapabilityModeNative
	autoMode := false
	if info.RelayMode == relayconstant.RelayModeResponses {
		runtimeMode = ResolveResponsesCapabilityMode(c, info)
		autoMode = info.ChannelSetting.ResponsesUpstreamMode.Normalize() == dto.ResponsesUpstreamModeAuto
	}
	correctedMode := false
	statusCodeMappingStr := c.GetString("status_code_mapping")

	for {
		if info.RelayMode == relayconstant.RelayModeResponses &&
			runtimeMode == model.ResponsesCapabilityModeChatCompletions {
			chatAdaptor := adaptor
			if correctedMode {
				chatAdaptor = GetAdaptor(info.ApiType)
				if chatAdaptor == nil {
					return types.NewError(fmt.Errorf("invalid api type: %d", info.ApiType), types.ErrorCodeInvalidApiType, types.ErrOptionWithSkipRetry())
				}
				chatAdaptor.Init(info)
			}
			usage, relayError := responsesViaChatCompletions(c, info, chatAdaptor, request)
			if relayError != nil {
				return relayError
			}
			if correctedMode {
				rememberResponsesCapabilityMode(c, info, runtimeMode, "")
			}
			return finalizeResponsesUsage(c, info, usage)
		}

		var requestBody io.Reader
		var closeRequestBody func()
		passThrough := runtimeMode == model.ResponsesCapabilityModeNative &&
			(model_setting.GetGlobalSettings().PassThroughRequestEnabled || info.ChannelSetting.PassThroughBodyEnabled)
		if passThrough {
			storage, storageErr := common.GetBodyStorage(c)
			if storageErr != nil {
				return types.NewError(storageErr, types.ErrorCodeReadRequestBodyFailed, types.ErrOptionWithSkipRetry())
			}
			requestBody = common.ReaderOnly(storage)
		} else {
			upstreamRequest := *request
			if runtimeMode == model.ResponsesCapabilityModeNativeTextCompat {
				upstreamRequest.Input, _, err = normalizeResponsesTextPartsForNativeCompat(upstreamRequest.Input)
				if err != nil {
					return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
				}
			}

			convertedRequest, convertErr := adaptor.ConvertOpenAIResponsesRequest(c, info, upstreamRequest)
			if convertErr != nil {
				return types.NewError(convertErr, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
			}
			relaycommon.AppendRequestConversionFromRequest(info, convertedRequest)
			jsonData, marshalErr := common.Marshal(convertedRequest)
			if marshalErr != nil {
				return types.NewError(marshalErr, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
			}

			jsonData, convertErr = relaycommon.RemoveDisabledFields(jsonData, info.ChannelOtherSettings, info.ChannelSetting.PassThroughBodyEnabled)
			if convertErr != nil {
				return types.NewError(convertErr, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
			}
			if len(info.ParamOverride) > 0 {
				jsonData, convertErr = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, info)
				if convertErr != nil {
					return newAPIErrorFromParamOverride(convertErr)
				}
			}

			logger.LogDebug(c, "requestBody: %s", jsonData)
			body, size, closer, bodyErr := relaycommon.NewOutboundJSONBody(jsonData)
			if bodyErr != nil {
				return types.NewError(bodyErr, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
			}
			info.UpstreamRequestBodySize = size
			requestBody = body
			closeRequestBody = func() {
				_ = closer.Close()
			}
		}

		resp, requestErr := adaptor.DoRequest(c, info, requestBody)
		if closeRequestBody != nil {
			closeRequestBody()
		}
		if requestErr != nil {
			return types.NewOpenAIError(requestErr, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
		}
		httpResp, ok := resp.(*http.Response)
		if !ok || httpResp == nil {
			return types.NewOpenAIError(
				fmt.Errorf("expected HTTP response, got %T", resp),
				types.ErrorCodeBadResponse,
				http.StatusInternalServerError,
			)
		}

		if httpResp.StatusCode != http.StatusOK {
			if autoMode && info.RelayMode == relayconstant.RelayModeResponses {
				errorBody, readErr := io.ReadAll(io.LimitReader(httpResp.Body, 64*1024))
				_ = httpResp.Body.Close()
				httpResp.Body = io.NopCloser(bytes.NewReader(errorBody))
				if readErr == nil {
					errorText := string(errorBody)
					nextMode, shouldCorrect := nextResponsesRuntimeMode(runtimeMode, httpResp.StatusCode, errorText)
					if shouldCorrect {
						_ = httpResp.Body.Close()
						logger.LogWarn(c, fmt.Sprintf(
							"correcting Responses upstream mode after validation failure: channel_id=%d model=%s stream=%t old_mode=%s new_mode=%s",
							info.ChannelId,
							info.UpstreamModelName,
							info.IsStream,
							runtimeMode,
							nextMode,
						))
						correctedMode = true
						runtimeMode = nextMode
						continue
					}
				}
			}

			relayError := service.RelayErrorHandler(c.Request.Context(), httpResp, false)
			service.ResetStatusCode(relayError, statusCodeMappingStr)
			return relayError
		}

		usage, relayError := adaptor.DoResponse(c, httpResp, info)
		if relayError != nil {
			service.ResetStatusCode(relayError, statusCodeMappingStr)
			return relayError
		}
		if correctedMode {
			rememberResponsesCapabilityMode(c, info, runtimeMode, "")
		}
		return finalizeResponsesUsage(c, info, usage.(*dto.Usage))
	}
}

func finalizeResponsesUsage(c *gin.Context, info *relaycommon.RelayInfo, usageDto *dto.Usage) *types.NewAPIError {
	if info.RelayMode == relayconstant.RelayModeResponsesCompact {
		originModelName := info.OriginModelName
		originPriceData := info.PriceData

		_, err := helper.ModelPriceHelper(c, info, info.GetEstimatePromptTokens(), &types.TokenCountMeta{})
		if err != nil {
			info.OriginModelName = originModelName
			info.PriceData = originPriceData
			return types.NewError(err, types.ErrorCodeModelPriceError, types.ErrOptionWithSkipRetry(), types.ErrOptionWithStatusCode(http.StatusBadRequest))
		}
		service.PostTextConsumeQuota(c, info, usageDto, nil)

		info.OriginModelName = originModelName
		info.PriceData = originPriceData
		return nil
	}

	if strings.HasPrefix(info.OriginModelName, "gpt-4o-audio") {
		service.PostAudioConsumeQuota(c, info, usageDto, "")
	} else {
		service.PostTextConsumeQuota(c, info, usageDto, nil)
	}
	return nil
}
