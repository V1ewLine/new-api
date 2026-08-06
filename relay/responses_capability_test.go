package relay

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveResponsesUpstreamModeHonorsManualMode(t *testing.T) {
	tests := []struct {
		configured dto.ResponsesUpstreamMode
		expected   dto.ResponsesUpstreamMode
	}{
		{configured: dto.ResponsesUpstreamModeNative, expected: dto.ResponsesUpstreamModeNative},
		{configured: dto.ResponsesUpstreamModeChatCompletions, expected: dto.ResponsesUpstreamModeChatCompletions},
	}

	for _, test := range tests {
		info := &relaycommon.RelayInfo{
			ChannelMeta: &relaycommon.ChannelMeta{
				ChannelSetting: dto.ChannelSettings{
					ResponsesUpstreamMode: test.configured,
				},
			},
		}
		assert.Equal(t, test.expected, ResolveResponsesUpstreamMode(nil, info))
	}
}

func TestDetectResponsesCapabilityUsesObservedUpstreamProtocol(t *testing.T) {
	service.InitHttpClient()

	t.Run("native Responses", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v1/responses", r.URL.Path)
			var request dto.OpenAIResponsesRequest
			assert.NoError(t, common.DecodeJson(r.Body, &request))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"id":"resp_native",
				"object":"response",
				"created_at":1710000000,
				"status":"completed",
				"model":"native-model",
				"output":[],
				"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}
			}`))
		}))
		defer upstream.Close()

		c, info := newResponsesCapabilityTestContext(
			upstream.URL,
			constant.ChannelTypeOpenAI,
			constant.APITypeOpenAI,
			"native-model",
		)
		result := detectResponsesCapability(c, info, false)

		require.NoError(t, result.err)
		assert.Equal(t, model.ResponsesCapabilityModeNative, result.mode)
	})

	t.Run("Chat Completions compatibility", func(t *testing.T) {
		var requestCount atomic.Int32
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount.Add(1)
			if r.URL.Path == "/responses" {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":{"message":"Responses endpoint is not supported"}}`))
				return
			}
			assert.Equal(t, "/v1/chat/completions", r.URL.Path)
			var request dto.GeneralOpenAIRequest
			assert.NoError(t, common.DecodeJson(r.Body, &request))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"id":"chatcmpl_detected",
				"object":"chat.completion",
				"created":1710000000,
				"model":"deepseek-model",
				"choices":[{"index":0,"message":{"role":"assistant","content":"OK"},"finish_reason":"stop"}],
				"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}
			}`))
		}))
		defer upstream.Close()

		c, info := newResponsesCapabilityTestContext(
			upstream.URL,
			constant.ChannelTypeDeepSeek,
			constant.APITypeDeepSeek,
			"deepseek-model",
		)
		result := detectResponsesCapability(c, info, false)

		require.NoError(t, result.err)
		assert.Equal(t, model.ResponsesCapabilityModeChatCompletions, result.mode)
		assert.Equal(t, int32(2), requestCount.Load())
	})

	t.Run("native Responses text compatibility", func(t *testing.T) {
		var requestCount atomic.Int32
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount.Add(1)
			assert.Equal(t, "/v1/responses", r.URL.Path)
			var request dto.OpenAIResponsesRequest
			require.NoError(t, common.DecodeJson(r.Body, &request))
			contentType := responsesInputContentType(t, request.Input)
			if contentType == "input_text" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"detail":[{"loc":["body","messages",0,"content",0,"type"],"msg":"Input should be 'text'","type":"literal_error","input":"input_text"}],"model":"ChatCompletionRequest"}`))
				return
			}
			assert.Equal(t, "text", contentType)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"id":"resp_native_compat",
				"object":"response",
				"created_at":1710000000,
				"status":"completed",
				"model":"sglang-model",
				"output":[],
				"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}
			}`))
		}))
		defer upstream.Close()

		c, info := newResponsesCapabilityTestContext(
			upstream.URL,
			constant.ChannelTypeOpenAI,
			constant.APITypeOpenAI,
			"sglang-model",
		)
		result := detectResponsesCapability(c, info, false)

		require.NoError(t, result.err)
		assert.Equal(t, model.ResponsesCapabilityModeNativeTextCompat, result.mode)
		assert.Equal(t, int32(2), requestCount.Load())
	})

	t.Run("Chat fallback after both native content formats fail", func(t *testing.T) {
		var requestCount atomic.Int32
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount.Add(1)
			if r.URL.Path == "/v1/chat/completions" {
				var request dto.GeneralOpenAIRequest
				require.NoError(t, common.DecodeJson(r.Body, &request))
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{
					"id":"chatcmpl_after_native_failures",
					"object":"chat.completion",
					"created":1710000000,
					"model":"mixed-compat-model",
					"choices":[{"index":0,"message":{"role":"assistant","content":"OK"},"finish_reason":"stop"}],
					"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}
				}`))
				return
			}

			assert.Equal(t, "/v1/responses", r.URL.Path)
			var request dto.OpenAIResponsesRequest
			require.NoError(t, common.DecodeJson(r.Body, &request))
			if responsesInputContentType(t, request.Input) == "input_text" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"ChatCompletionRequest input_text Input should be 'text' literal_error"}`))
				return
			}
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(`{"error":"legacy native Responses schema rejected text content"}`))
		}))
		defer upstream.Close()

		c, info := newResponsesCapabilityTestContext(
			upstream.URL,
			constant.ChannelTypeOpenAI,
			constant.APITypeOpenAI,
			"mixed-compat-model",
		)
		result := detectResponsesCapability(c, info, false)

		require.NoError(t, result.err)
		assert.Equal(t, model.ResponsesCapabilityModeChatCompletions, result.mode)
		assert.Equal(t, int32(3), requestCount.Load())
	})
}

func TestNormalizeResponsesTextPartsForNativeCompat(t *testing.T) {
	original := []byte(`[
		{"role":"user","content":[{"type":"input_text","text":"hello"}]},
		{"type":"input_text","text":"standalone"},
		{"role":"assistant","content":[{"type":"output_text","text":"previous answer"}]},
		{"type":"function_call_output","output":{"type":"input_text","text":"do not rewrite tool payload"}}
	]`)
	originalCopy := append([]byte(nil), original...)

	normalized, changed, err := normalizeResponsesTextPartsForNativeCompat(original)

	require.NoError(t, err)
	require.True(t, changed)
	var value []map[string]any
	require.NoError(t, common.Unmarshal(normalized, &value))
	require.Len(t, value, 4)
	content, ok := value[0]["content"].([]any)
	require.True(t, ok)
	require.Len(t, content, 1)
	contentPart, ok := content[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "text", contentPart["type"])
	assert.Equal(t, "text", value[1]["type"])
	assistantContent, ok := value[2]["content"].([]any)
	require.True(t, ok)
	require.Len(t, assistantContent, 1)
	assistantPart, ok := assistantContent[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "text", assistantPart["type"])
	toolOutput, ok := value[3]["output"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "input_text", toolOutput["type"])

	assert.Equal(t, originalCopy, original)
}

func TestNextResponsesRuntimeModeOnlyCorrectsProtocolFailures(t *testing.T) {
	inputTextError := `43 validation errors for ChatCompletionRequest: Input should be 'text'; input=input_text; type=literal_error`
	tests := []struct {
		name       string
		current    model.ResponsesCapabilityMode
		statusCode int
		body       string
		expected   model.ResponsesCapabilityMode
		correct    bool
	}{
		{
			name:       "normalizes native input_text validation failure",
			current:    model.ResponsesCapabilityModeNative,
			statusCode: http.StatusBadRequest,
			body:       inputTextError,
			expected:   model.ResponsesCapabilityModeNativeTextCompat,
			correct:    true,
		},
		{
			name:       "falls back to Chat when Responses route disappears",
			current:    model.ResponsesCapabilityModeNativeTextCompat,
			statusCode: http.StatusNotFound,
			body:       "not found",
			expected:   model.ResponsesCapabilityModeChatCompletions,
			correct:    true,
		},
		{
			name:       "does not reinterpret rate limiting",
			current:    model.ResponsesCapabilityModeNative,
			statusCode: http.StatusTooManyRequests,
			body:       inputTextError,
			expected:   model.ResponsesCapabilityModeNative,
			correct:    false,
		},
		{
			name:       "does not loop after Chat compatibility",
			current:    model.ResponsesCapabilityModeChatCompletions,
			statusCode: http.StatusNotFound,
			body:       "not found",
			expected:   model.ResponsesCapabilityModeChatCompletions,
			correct:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mode, correct := nextResponsesRuntimeMode(test.current, test.statusCode, test.body)
			assert.Equal(t, test.expected, mode)
			assert.Equal(t, test.correct, correct)
		})
	}
}

func TestDetectResponsesCapabilityDoesNotProbeChatAfterAmbiguousNativeFailure(t *testing.T) {
	service.InitHttpClient()
	var requestCount atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		assert.Equal(t, "/v1/responses", r.URL.Path)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer upstream.Close()

	c, info := newResponsesCapabilityTestContext(
		upstream.URL,
		constant.ChannelTypeOpenAI,
		constant.APITypeOpenAI,
		"unauthorized-model",
	)
	result := detectResponsesCapability(c, info, false)

	require.Error(t, result.err)
	assert.Equal(t, model.ResponsesCapabilityModeUnknown, result.mode)
	assert.Equal(t, int32(1), requestCount.Load())
}

func TestDetectResponsesCapabilitySeparatesStreamingSupport(t *testing.T) {
	oldStreamingTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() {
		constant.StreamingTimeout = oldStreamingTimeout
	})

	service.InitHttpClient()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request dto.GeneralOpenAIRequest
		assert.NoError(t, common.DecodeJson(r.Body, &request))
		if !assert.NotNil(t, request.Stream) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		assert.True(t, *request.Stream)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, strings.Join([]string{
			`data: {"id":"chatcmpl_stream","object":"chat.completion.chunk","created":1710000000,"model":"deepseek-model","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl_stream","object":"chat.completion.chunk","created":1710000000,"model":"deepseek-model","choices":[{"index":0,"delta":{"content":"OK"},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl_stream","object":"chat.completion.chunk","created":1710000000,"model":"deepseek-model","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
			`data: [DONE]`,
			``,
		}, "\n"))
	}))
	defer upstream.Close()

	c, info := newResponsesCapabilityTestContext(
		upstream.URL,
		constant.ChannelTypeDeepSeek,
		constant.APITypeDeepSeek,
		"deepseek-model",
	)
	result := detectResponsesCapability(c, info, true)

	require.NoError(t, result.err)
	assert.Equal(t, model.ResponsesCapabilityModeChatCompletions, result.mode)
}

func newResponsesCapabilityTestContext(
	baseURL string,
	channelType int,
	apiType int,
	modelName string,
) (*gin.Context, *relaycommon.RelayInfo) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	c.Request.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeResponses,
		RelayFormat:     types.RelayFormatOpenAIResponses,
		OriginModelName: modelName,
		RequestURLPath:  "/v1/responses",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       channelType,
			ChannelId:         1,
			ChannelBaseUrl:    baseURL,
			ApiType:           apiType,
			ApiKey:            "test-key",
			UpstreamModelName: modelName,
		},
	}
	return c, info
}

func responsesInputContentType(t *testing.T, input []byte) string {
	t.Helper()
	var items []map[string]any
	require.NoError(t, common.Unmarshal(input, &items))
	require.NotEmpty(t, items)
	content, ok := items[0]["content"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, content)
	part, ok := content[0].(map[string]any)
	require.True(t, ok)
	contentType, ok := part["type"].(string)
	require.True(t, ok)
	return contentType
}
