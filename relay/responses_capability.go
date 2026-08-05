package relay

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

const (
	responsesCapabilityUnknownRetry = 6 * time.Hour
	responsesCapabilityCacheTTL     = time.Minute
	responsesCapabilityProbeTimeout = 20 * time.Second
)

type responsesCapabilityCacheEntry struct {
	mode       model.ResponsesCapabilityMode
	detectedAt int64
	expiresAt  time.Time
}

type responsesCapabilityProbeResult struct {
	mode                  model.ResponsesCapabilityMode
	err                   error
	statusCode            int
	route                 bool
	inputTextIncompatible bool
}

var responsesCapabilityFlights singleflight.Group
var responsesCapabilityCache sync.Map

func responsesCapabilityKey(channelId int, modelName string, isStream bool) string {
	return fmt.Sprintf("%d:%s:%t", channelId, strings.TrimSpace(modelName), isStream)
}

func responsesCapabilityLogContext(c *gin.Context) context.Context {
	if c != nil && c.Request != nil {
		return c.Request.Context()
	}
	return context.Background()
}

func InvalidateResponsesCapabilityCache(channelId int) {
	prefix := fmt.Sprintf("%d:", channelId)
	responsesCapabilityCache.Range(func(key, _ any) bool {
		keyString, ok := key.(string)
		if ok && strings.HasPrefix(keyString, prefix) {
			responsesCapabilityCache.Delete(key)
		}
		return true
	})
}

func rememberResponsesCapabilityMode(
	c *gin.Context,
	info *relaycommon.RelayInfo,
	mode model.ResponsesCapabilityMode,
	lastError string,
) {
	if info == nil || info.ChannelMeta == nil || info.ChannelId <= 0 {
		return
	}
	modelName := strings.TrimSpace(info.UpstreamModelName)
	if modelName == "" {
		return
	}
	if err := model.SaveChannelResponsesCapabilityMode(info.ChannelId, modelName, info.IsStream, mode, lastError); err != nil {
		logger.LogWarn(responsesCapabilityLogContext(c), fmt.Sprintf(
			"failed to save corrected Responses capability: channel_id=%d model=%s error=%v",
			info.ChannelId,
			modelName,
			err,
		))
		return
	}
	responsesCapabilityCache.Store(
		responsesCapabilityKey(info.ChannelId, modelName, info.IsStream),
		responsesCapabilityCacheEntry{
			mode:       mode,
			detectedAt: time.Now().Unix(),
			expiresAt:  time.Now().Add(responsesCapabilityCacheTTL),
		},
	)
}

func ResolveResponsesUpstreamMode(
	c *gin.Context,
	info *relaycommon.RelayInfo,
) dto.ResponsesUpstreamMode {
	if ResolveResponsesCapabilityMode(c, info) == model.ResponsesCapabilityModeChatCompletions {
		return dto.ResponsesUpstreamModeChatCompletions
	}
	return dto.ResponsesUpstreamModeNative
}

func ResolveResponsesCapabilityMode(
	c *gin.Context,
	info *relaycommon.RelayInfo,
) model.ResponsesCapabilityMode {
	if info == nil || info.ChannelMeta == nil {
		return model.ResponsesCapabilityModeNative
	}
	configuredMode := info.ChannelSetting.ResponsesUpstreamMode.Normalize()
	if configuredMode == dto.ResponsesUpstreamModeNative {
		return model.ResponsesCapabilityModeNative
	}
	if configuredMode == dto.ResponsesUpstreamModeChatCompletions {
		return model.ResponsesCapabilityModeChatCompletions
	}

	modelName := strings.TrimSpace(info.UpstreamModelName)
	if info.ChannelId <= 0 || modelName == "" {
		return model.ResponsesCapabilityModeNative
	}
	key := responsesCapabilityKey(info.ChannelId, modelName, info.IsStream)
	now := time.Now()
	if cached, ok := responsesCapabilityCache.Load(key); ok {
		entry := cached.(responsesCapabilityCacheEntry)
		if now.Before(entry.expiresAt) {
			return runtimeResponsesCapabilityMode(entry.mode)
		}
		responsesCapabilityCache.Delete(key)
	}

	capability, err := model.GetChannelResponsesCapability(info.ChannelId, modelName)
	if err == nil {
		mode, detectedAt := capabilityMode(capability, info.IsStream)
		if mode != model.ResponsesCapabilityModeUnknown ||
			detectedAt >= now.Add(-responsesCapabilityUnknownRetry).Unix() {
			responsesCapabilityCache.Store(key, responsesCapabilityCacheEntry{
				mode:       mode,
				detectedAt: detectedAt,
				expiresAt:  now.Add(responsesCapabilityCacheTTL),
			})
			return runtimeResponsesCapabilityMode(mode)
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.LogWarn(responsesCapabilityLogContext(c), fmt.Sprintf(
			"failed to read Responses capability: channel_id=%d model=%s error=%v",
			info.ChannelId,
			modelName,
			err,
		))
		return model.ResponsesCapabilityModeNative
	}

	value, _, _ := responsesCapabilityFlights.Do(key, func() (any, error) {
		if capability, lookupErr := model.GetChannelResponsesCapability(info.ChannelId, modelName); lookupErr == nil {
			mode, detectedAt := capabilityMode(capability, info.IsStream)
			if mode != model.ResponsesCapabilityModeUnknown ||
				detectedAt >= time.Now().Add(-responsesCapabilityUnknownRetry).Unix() {
				return responsesCapabilityCacheEntry{
					mode:       mode,
					detectedAt: detectedAt,
					expiresAt:  time.Now().Add(responsesCapabilityCacheTTL),
				}, nil
			}
		}

		result := detectResponsesCapability(c, info, info.IsStream)
		lastError := ""
		if result.err != nil {
			lastError = result.err.Error()
		}
		if saveErr := model.SaveChannelResponsesCapabilityMode(
			info.ChannelId,
			modelName,
			info.IsStream,
			result.mode,
			lastError,
		); saveErr != nil {
			logger.LogWarn(responsesCapabilityLogContext(c), fmt.Sprintf(
				"failed to save Responses capability: channel_id=%d model=%s error=%v",
				info.ChannelId,
				modelName,
				saveErr,
			))
		}
		return responsesCapabilityCacheEntry{
			mode:       result.mode,
			detectedAt: time.Now().Unix(),
			expiresAt:  time.Now().Add(responsesCapabilityCacheTTL),
		}, nil
	})
	entry := value.(responsesCapabilityCacheEntry)
	responsesCapabilityCache.Store(key, entry)
	if entry.mode == model.ResponsesCapabilityModeUnknown {
		logger.LogWarn(responsesCapabilityLogContext(c), fmt.Sprintf(
			"Responses capability remains unknown; using native upstream mode: channel_id=%d model=%s stream=%t",
			info.ChannelId,
			modelName,
			info.IsStream,
		))
	}
	return runtimeResponsesCapabilityMode(entry.mode)
}

func DetectResponsesCapabilities(
	c *gin.Context,
	info *relaycommon.RelayInfo,
) (*model.ChannelResponsesCapability, error) {
	if info == nil || info.ChannelMeta == nil {
		return nil, errors.New("channel metadata is required")
	}
	modelName := strings.TrimSpace(info.UpstreamModelName)
	if info.ChannelId <= 0 || modelName == "" {
		return nil, errors.New("channel id and upstream model are required")
	}

	var detectionErrors []string
	for _, isStream := range []bool{false, true} {
		key := responsesCapabilityKey(info.ChannelId, modelName, isStream)
		responsesCapabilityFlights.Forget(key)
		result := detectResponsesCapability(c, info, isStream)
		lastError := ""
		if result.err != nil {
			lastError = result.err.Error()
			detectionErrors = append(detectionErrors, fmt.Sprintf("stream=%t: %s", isStream, lastError))
		}
		if err := model.SaveChannelResponsesCapabilityMode(
			info.ChannelId,
			modelName,
			isStream,
			result.mode,
			lastError,
		); err != nil {
			return nil, err
		}
		responsesCapabilityCache.Store(key, responsesCapabilityCacheEntry{
			mode:       result.mode,
			detectedAt: time.Now().Unix(),
			expiresAt:  time.Now().Add(responsesCapabilityCacheTTL),
		})
	}

	capability, err := model.GetChannelResponsesCapability(info.ChannelId, modelName)
	if err != nil {
		return nil, err
	}
	if len(detectionErrors) > 0 {
		return capability, errors.New(strings.Join(detectionErrors, "; "))
	}
	return capability, nil
}

func capabilityMode(
	capability *model.ChannelResponsesCapability,
	isStream bool,
) (model.ResponsesCapabilityMode, int64) {
	if capability == nil {
		return model.ResponsesCapabilityModeUnknown, 0
	}
	if isStream {
		return capability.StreamMode, capability.StreamDetectedAt
	}
	return capability.NonStreamMode, capability.NonStreamDetectedAt
}

func runtimeResponsesCapabilityMode(mode model.ResponsesCapabilityMode) model.ResponsesCapabilityMode {
	switch mode {
	case model.ResponsesCapabilityModeNative,
		model.ResponsesCapabilityModeNativeTextCompat,
		model.ResponsesCapabilityModeChatCompletions:
		return mode
	default:
		return model.ResponsesCapabilityModeNative
	}
}

func detectResponsesCapability(
	c *gin.Context,
	info *relaycommon.RelayInfo,
	isStream bool,
) responsesCapabilityProbeResult {
	nativeResult := probeResponsesUpstream(c, info, isStream, model.ResponsesCapabilityModeNative)
	if nativeResult.mode == model.ResponsesCapabilityModeNative {
		return nativeResult
	}
	var nativeCompatResult *responsesCapabilityProbeResult
	if nativeResult.inputTextIncompatible {
		compatResult := probeResponsesUpstream(c, info, isStream, model.ResponsesCapabilityModeNativeTextCompat)
		nativeCompatResult = &compatResult
		if compatResult.mode == model.ResponsesCapabilityModeNativeTextCompat {
			return compatResult
		}
		canTryChat := compatResult.route ||
			compatResult.statusCode == http.StatusBadRequest ||
			compatResult.statusCode == http.StatusUnprocessableEntity
		if !canTryChat {
			return compatResult
		}
		nativeResult.route = true
	}
	if !nativeResult.route {
		if nativeResult.err != nil {
			return nativeResult
		}
		return responsesCapabilityProbeResult{
			mode: model.ResponsesCapabilityModeUnknown,
			err:  errors.New("native Responses capability could not be determined safely"),
		}
	}

	chatResult := probeResponsesUpstream(c, info, isStream, model.ResponsesCapabilityModeChatCompletions)
	if chatResult.mode == model.ResponsesCapabilityModeChatCompletions {
		return chatResult
	}

	errorParts := make([]string, 0, 3)
	if nativeResult.err != nil {
		errorParts = append(errorParts, "native: "+nativeResult.err.Error())
	}
	if nativeCompatResult != nil && nativeCompatResult.err != nil {
		errorParts = append(errorParts, "native_text_compat: "+nativeCompatResult.err.Error())
	}
	if chatResult.err != nil {
		errorParts = append(errorParts, "chat_completions: "+chatResult.err.Error())
	}
	if len(errorParts) == 0 {
		errorParts = append(errorParts, "unable to determine upstream Responses capability")
	}
	return responsesCapabilityProbeResult{
		mode: model.ResponsesCapabilityModeUnknown,
		err:  errors.New(strings.Join(errorParts, "; ")),
	}
}

func probeResponsesUpstream(
	sourceContext *gin.Context,
	sourceInfo *relaycommon.RelayInfo,
	isStream bool,
	mode model.ResponsesCapabilityMode,
) responsesCapabilityProbeResult {
	probeRequest := &dto.OpenAIResponsesRequest{
		Model:           sourceInfo.UpstreamModelName,
		Input:           []byte(`[{"role":"user","content":[{"type":"input_text","text":"Reply with OK"}]}]`),
		MaxOutputTokens: common.GetPointer(uint(16)),
		Stream:          common.GetPointer(isStream),
	}
	probeInfo := cloneResponsesProbeInfo(sourceInfo, probeRequest, isStream, mode)
	adaptor := GetAdaptor(probeInfo.ApiType)
	if adaptor == nil {
		return responsesCapabilityProbeResult{
			mode: model.ResponsesCapabilityModeUnknown,
			err:  fmt.Errorf("invalid api type: %d", probeInfo.ApiType),
		}
	}
	adaptor.Init(&probeInfo)

	recorder := httptest.NewRecorder()
	parentContext := context.Background()
	if sourceContext != nil && sourceContext.Request != nil {
		parentContext = sourceContext.Request.Context()
	}
	probeContext, cancel := context.WithTimeout(parentContext, responsesCapabilityProbeTimeout)
	defer cancel()
	probeGin, _ := gin.CreateTestContext(recorder)
	if sourceContext != nil {
		copied := sourceContext.Copy()
		for key, value := range copied.Keys {
			probeGin.Set(key, value)
		}
	}
	path := "/v1/responses"
	if mode == model.ResponsesCapabilityModeChatCompletions {
		path = "/v1/chat/completions"
	}
	probeGin.Request = httptest.NewRequestWithContext(probeContext, http.MethodPost, path, nil)
	probeGin.Request.Header.Set("Content-Type", "application/json")

	if mode == model.ResponsesCapabilityModeNative {
		nativeURL, nativeErr := adaptor.GetRequestURL(&probeInfo)
		chatInfo := cloneResponsesProbeInfo(sourceInfo, probeRequest, isStream, model.ResponsesCapabilityModeChatCompletions)
		chatAdaptor := GetAdaptor(chatInfo.ApiType)
		if nativeErr == nil && chatAdaptor != nil {
			chatAdaptor.Init(&chatInfo)
			if chatURL, chatErr := chatAdaptor.GetRequestURL(&chatInfo); chatErr == nil && nativeURL == chatURL {
				return responsesCapabilityProbeResult{
					mode:  model.ResponsesCapabilityModeUnknown,
					err:   errors.New("adapter resolves Responses and Chat Completions to the same upstream route"),
					route: true,
				}
			}
		}
	}

	var convertedRequest any
	var err error
	if mode == model.ResponsesCapabilityModeChatCompletions {
		result, convertErr := service.ConvertRequest(probeGin, &probeInfo, types.RelayFormatOpenAI, probeRequest)
		if convertErr != nil {
			err = convertErr
		} else {
			chatRequest, ok := result.Value.(*dto.GeneralOpenAIRequest)
			if !ok {
				err = fmt.Errorf("expected OpenAI chat completions request, got %T", result.Value)
			} else {
				convertedRequest, err = adaptor.ConvertOpenAIRequest(probeGin, &probeInfo, chatRequest)
			}
		}
	} else {
		nativeRequest := *probeRequest
		if mode == model.ResponsesCapabilityModeNativeTextCompat {
			nativeRequest.Input, _, err = normalizeResponsesTextPartsForNativeCompat(nativeRequest.Input)
		}
		if err == nil {
			convertedRequest, err = adaptor.ConvertOpenAIResponsesRequest(probeGin, &probeInfo, nativeRequest)
		}
	}
	if err != nil {
		return responsesCapabilityProbeResult{
			mode:  model.ResponsesCapabilityModeUnknown,
			err:   err,
			route: isExplicitResponsesUnsupportedError(err.Error()),
		}
	}

	jsonData, err := common.Marshal(convertedRequest)
	if err != nil {
		return responsesCapabilityProbeResult{mode: model.ResponsesCapabilityModeUnknown, err: err}
	}
	jsonData, err = relaycommon.RemoveDisabledFields(jsonData, probeInfo.ChannelOtherSettings, false)
	if err != nil {
		return responsesCapabilityProbeResult{mode: model.ResponsesCapabilityModeUnknown, err: err}
	}
	if len(probeInfo.ParamOverride) > 0 {
		jsonData, err = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, &probeInfo)
		if err != nil {
			return responsesCapabilityProbeResult{mode: model.ResponsesCapabilityModeUnknown, err: err}
		}
	}
	body, size, closer, err := relaycommon.NewOutboundJSONBody(jsonData)
	if err != nil {
		return responsesCapabilityProbeResult{mode: model.ResponsesCapabilityModeUnknown, err: err}
	}
	defer closer.Close()
	probeInfo.UpstreamRequestBodySize = size

	response, err := adaptor.DoRequest(probeGin, &probeInfo, body)
	if err != nil {
		return responsesCapabilityProbeResult{mode: model.ResponsesCapabilityModeUnknown, err: err}
	}
	httpResponse, ok := response.(*http.Response)
	if !ok || httpResponse == nil {
		return responsesCapabilityProbeResult{
			mode: model.ResponsesCapabilityModeUnknown,
			err:  fmt.Errorf("expected HTTP response, got %T", response),
		}
	}
	if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusMultipleChoices {
		errorBody, _ := io.ReadAll(io.LimitReader(httpResponse.Body, 64*1024))
		_ = httpResponse.Body.Close()
		errorText := strings.TrimSpace(string(errorBody))
		inputTextIncompatible := isResponsesInputTextCompatibilityError(httpResponse.StatusCode, errorText)
		errorMessage := fmt.Sprintf("upstream returned HTTP %d", httpResponse.StatusCode)
		if inputTextIncompatible {
			errorMessage += ": Responses input_text is incompatible with the upstream ChatCompletionRequest"
		}
		return responsesCapabilityProbeResult{
			mode:                  model.ResponsesCapabilityModeUnknown,
			err:                   errors.New(errorMessage),
			statusCode:            httpResponse.StatusCode,
			route:                 isExplicitResponsesUnsupportedStatus(httpResponse.StatusCode, errorText),
			inputTextIncompatible: inputTextIncompatible,
		}
	}

	_, responseErr := adaptor.DoResponse(probeGin, httpResponse, &probeInfo)
	if responseErr != nil {
		return responsesCapabilityProbeResult{
			mode: model.ResponsesCapabilityModeUnknown,
			err:  responseErr,
		}
	}
	responseBody := recorder.Body.String()
	if mode == model.ResponsesCapabilityModeChatCompletions {
		if !validChatProbeResponse(responseBody, isStream) {
			return responsesCapabilityProbeResult{
				mode: model.ResponsesCapabilityModeUnknown,
				err:  errors.New("upstream did not return a valid Chat Completions response"),
			}
		}
		return responsesCapabilityProbeResult{mode: model.ResponsesCapabilityModeChatCompletions}
	}
	if !validNativeResponsesProbeResponse(responseBody, isStream) {
		return responsesCapabilityProbeResult{
			mode:  model.ResponsesCapabilityModeUnknown,
			err:   errors.New("upstream did not return a valid Responses response"),
			route: true,
		}
	}
	return responsesCapabilityProbeResult{mode: mode}
}

func cloneResponsesProbeInfo(
	source *relaycommon.RelayInfo,
	request *dto.OpenAIResponsesRequest,
	isStream bool,
	mode model.ResponsesCapabilityMode,
) relaycommon.RelayInfo {
	cloned := *source
	if source.ChannelMeta != nil {
		channelMeta := *source.ChannelMeta
		cloned.ChannelMeta = &channelMeta
	}
	cloned.Request = request
	cloned.IsStream = isStream
	cloned.StartTime = time.Now()
	cloned.FirstResponseTime = cloned.StartTime.Add(-time.Second)
	cloned.ResponsesUsageInfo = &relaycommon.ResponsesUsageInfo{
		BuiltInTools: make(map[string]*relaycommon.BuildInToolInfo),
	}
	cloned.RequestConversionChain = []types.RelayFormat{types.RelayFormatOpenAIResponses}
	cloned.FinalRequestRelayFormat = ""
	if mode == model.ResponsesCapabilityModeChatCompletions {
		cloned.RelayMode = relayconstant.RelayModeChatCompletions
		cloned.RelayFormat = types.RelayFormatOpenAI
		cloned.RequestURLPath = "/v1/chat/completions"
	} else {
		cloned.RelayMode = relayconstant.RelayModeResponses
		cloned.RelayFormat = types.RelayFormatOpenAIResponses
		cloned.RequestURLPath = "/v1/responses"
	}
	return cloned
}

func isResponsesInputTextCompatibilityError(statusCode int, responseBody string) bool {
	if statusCode != http.StatusBadRequest && statusCode != http.StatusUnprocessableEntity {
		return false
	}
	lower := strings.ToLower(responseBody)
	if !strings.Contains(lower, "input_text") || !strings.Contains(lower, "chatcompletionrequest") {
		return false
	}
	return strings.Contains(lower, "input should be 'text'") ||
		strings.Contains(lower, `input should be \"text\"`) ||
		strings.Contains(lower, "literal_error")
}

func nextResponsesRuntimeMode(
	current model.ResponsesCapabilityMode,
	statusCode int,
	responseBody string,
) (model.ResponsesCapabilityMode, bool) {
	if current == model.ResponsesCapabilityModeNative &&
		isResponsesInputTextCompatibilityError(statusCode, responseBody) {
		return model.ResponsesCapabilityModeNativeTextCompat, true
	}
	if current != model.ResponsesCapabilityModeChatCompletions &&
		isExplicitResponsesUnsupportedStatus(statusCode, responseBody) {
		return model.ResponsesCapabilityModeChatCompletions, true
	}
	return current, false
}

func isExplicitResponsesUnsupportedStatus(statusCode int, responseBody string) bool {
	if statusCode == http.StatusNotFound ||
		statusCode == http.StatusMethodNotAllowed ||
		statusCode == http.StatusNotImplemented {
		return true
	}
	if statusCode != http.StatusBadRequest {
		return false
	}
	return isExplicitResponsesUnsupportedError(responseBody)
}

func isExplicitResponsesUnsupportedError(message string) bool {
	lower := strings.ToLower(message)
	for _, marker := range []string{
		"not implemented",
		"unsupported endpoint",
		"unsupported route",
		"unknown endpoint",
		"unknown route",
		"invalid url",
		"no route",
		"method not allowed",
		"stream is not supported",
		"streaming is not supported",
		"does not support stream",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func validNativeResponsesProbeResponse(responseBody string, isStream bool) bool {
	if isStream {
		return strings.Contains(responseBody, `"type":"response.`) ||
			strings.Contains(responseBody, `"type": "response.`)
	}
	var response dto.OpenAIResponsesResponse
	if err := common.Unmarshal([]byte(responseBody), &response); err != nil {
		return false
	}
	return response.Object == "response" && response.ID != ""
}

func validChatProbeResponse(responseBody string, isStream bool) bool {
	if isStream {
		return strings.Contains(responseBody, `"choices"`)
	}
	var response dto.OpenAITextResponse
	if err := common.Unmarshal([]byte(responseBody), &response); err != nil {
		return false
	}
	return response.Id != "" && len(response.Choices) > 0
}
