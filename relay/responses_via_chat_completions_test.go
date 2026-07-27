package relay

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel/deepseek"
	"github.com/QuantumNous/new-api/relay/channel/moonshot"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type responsesViaChatUpstreamRequest struct {
	path    string
	request dto.GeneralOpenAIRequest
	err     error
}

func TestResponsesViaChatCompletionsNonStreamDeepSeek(t *testing.T) {
	service.InitHttpClient()
	upstreamRequests := make(chan responsesViaChatUpstreamRequest, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request dto.GeneralOpenAIRequest
		err := common.DecodeJson(r.Body, &request)
		upstreamRequests <- responsesViaChatUpstreamRequest{path: r.URL.Path, request: request, err: err}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl_1",
			"object":"chat.completion",
			"created":1710000000,
			"model":"deepseek-v4-flash",
			"choices":[{"index":0,"message":{"role":"assistant","content":"你好"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":2,"completion_tokens":3,"total_tokens":5}
		}`))
	}))
	defer upstream.Close()

	request := &dto.OpenAIResponsesRequest{
		Model:           "deepseek-v4-flash",
		Input:           []byte(`"你好"`),
		MaxOutputTokens: common.GetPointer[uint](64),
	}
	info := newResponsesViaChatTestInfo(upstream.URL, request.Model, false)
	c, recorder := newResponsesViaChatTestContext(request)
	adaptor := &deepseek.Adaptor{}

	usage, relayErr := responsesViaChatCompletions(c, info, adaptor, request)

	require.Nil(t, relayErr)
	require.Equal(t, 5, usage.TotalTokens)
	upstreamRequest := <-upstreamRequests
	require.NoError(t, upstreamRequest.err)
	require.Equal(t, "/v1/chat/completions", upstreamRequest.path)
	require.Equal(t, request.Model, upstreamRequest.request.Model)
	require.Len(t, upstreamRequest.request.Messages, 1)
	require.Equal(t, "user", upstreamRequest.request.Messages[0].Role)
	require.Equal(t, "你好", upstreamRequest.request.Messages[0].StringContent())
	require.Equal(t, types.RelayFormatOpenAI, info.FinalRequestRelayFormat)
	require.Equal(t, relayconstant.RelayModeResponses, info.RelayMode)
	require.Equal(t, types.RelayFormat(types.RelayFormatOpenAIResponses), info.RelayFormat)
	require.Equal(t, "/v1/responses", info.RequestURLPath)
	require.Contains(t, recorder.Body.String(), `"object":"response"`)
	require.Contains(t, recorder.Body.String(), `"text":"你好"`)
}

func TestResponsesViaChatCompletionsStreamMoonshot(t *testing.T) {
	service.InitHttpClient()
	oldStreamingTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() {
		constant.StreamingTimeout = oldStreamingTimeout
	})
	upstreamRequests := make(chan responsesViaChatUpstreamRequest, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request dto.GeneralOpenAIRequest
		err := common.DecodeJson(r.Body, &request)
		upstreamRequests <- responsesViaChatUpstreamRequest{path: r.URL.Path, request: request, err: err}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, strings.Join([]string{
			`data: {"id":"chatcmpl_2","object":"chat.completion.chunk","created":1710000000,"model":"kimi-k2.6","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl_2","object":"chat.completion.chunk","created":1710000000,"model":"kimi-k2.6","choices":[{"index":0,"delta":{"content":"流式内容"},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl_2","object":"chat.completion.chunk","created":1710000000,"model":"kimi-k2.6","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
			`data: {"id":"chatcmpl_2","object":"chat.completion.chunk","created":1710000000,"model":"kimi-k2.6","choices":[],"usage":{"prompt_tokens":4,"completion_tokens":5,"total_tokens":9}}`,
			`data: [DONE]`,
			``,
		}, "\n"))
	}))
	defer upstream.Close()

	temperature := 0.7
	request := &dto.OpenAIResponsesRequest{
		Model:       "kimi-k2.6",
		Input:       []byte(`"测试流式输出"`),
		Stream:      common.GetPointer(true),
		Temperature: &temperature,
	}
	info := newResponsesViaChatTestInfo(upstream.URL, request.Model, true)
	info.ChannelType = constant.ChannelTypeMoonshot
	c, recorder := newResponsesViaChatTestContext(request)
	adaptor := &moonshot.Adaptor{}

	usage, relayErr := responsesViaChatCompletions(c, info, adaptor, request)

	require.Nil(t, relayErr)
	require.Equal(t, 9, usage.TotalTokens)
	upstreamRequest := <-upstreamRequests
	require.NoError(t, upstreamRequest.err)
	require.Equal(t, "/v1/chat/completions", upstreamRequest.path)
	require.NotNil(t, upstreamRequest.request.Stream)
	require.True(t, *upstreamRequest.request.Stream)
	require.NotNil(t, upstreamRequest.request.Temperature)
	require.Equal(t, 1.0, *upstreamRequest.request.Temperature)
	require.Equal(t, "text/event-stream", recorder.Header().Get("Content-Type"))
	require.Contains(t, recorder.Body.String(), `event: response.output_text.delta`)
	require.Contains(t, recorder.Body.String(), `"delta":"流式内容"`)
	require.Contains(t, recorder.Body.String(), `event: response.completed`)
}

func newResponsesViaChatTestInfo(baseURL string, model string, stream bool) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		IsStream:               stream,
		RelayMode:              relayconstant.RelayModeResponses,
		RelayFormat:            types.RelayFormatOpenAIResponses,
		OriginModelName:        model,
		RequestURLPath:         "/v1/responses",
		RequestConversionChain: []types.RelayFormat{types.RelayFormatOpenAIResponses},
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeDeepSeek,
			ChannelBaseUrl:    baseURL,
			ApiKey:            "test-key",
			UpstreamModelName: model,
		},
	}
}

func newResponsesViaChatTestContext(request *dto.OpenAIResponsesRequest) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(string(constant.ContextKeyOriginalModel), request.Model)
	return c, recorder
}
