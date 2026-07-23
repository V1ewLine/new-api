package clusterstatus

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(request *http.Request) (*http.Response, error)

func (roundTrip roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

func TestHTTPAgentClientFetchesTelemetryWithBearerAuthentication(t *testing.T) {
	var receivedAuthorization string
	httpClient := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		receivedAuthorization = request.Header.Get("Authorization")
		assert.Equal(t, "/v1/telemetry", request.URL.Path)
		assert.Equal(t, "application/json", request.Header.Get("Accept"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"application/json; charset=utf-8"},
			},
			Body: io.NopCloser(strings.NewReader(`{"schema_version":"1.0"}`)),
		}, nil
	})}

	body, err := NewHTTPAgentClient(httpClient, 1024).Fetch(
		context.Background(),
		ResolvedAgentConnection{BaseURL: "https://agent.example", BearerToken: "agent-token"},
	)

	require.NoError(t, err)
	assert.JSONEq(t, `{"schema_version":"1.0"}`, string(body))
	assert.Equal(t, "Bearer agent-token", receivedAuthorization)
}

func TestHTTPAgentClientReturnsStableTimeoutError(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		<-request.Context().Done()
		return nil, request.Context().Err()
	})}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	t.Cleanup(cancel)

	_, err := NewHTTPAgentClient(httpClient, 1024).Fetch(
		ctx,
		ResolvedAgentConnection{BaseURL: "https://agent.example", BearerToken: "agent-token"},
	)

	var clientErr *AgentClientError
	require.ErrorAs(t, err, &clientErr)
	assert.Equal(t, "AGENT_TIMEOUT", clientErr.Code)
	assert.NotContains(t, err.Error(), "agent-token")
}

func TestHTTPAgentClientRejectsOversizedResponse(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: io.NopCloser(strings.NewReader(`{"payload":"too-large"}`)),
		}, nil
	})}

	_, err := NewHTTPAgentClient(httpClient, 8).Fetch(
		context.Background(),
		ResolvedAgentConnection{BaseURL: "https://agent.example", BearerToken: "agent-token"},
	)

	var clientErr *AgentClientError
	require.True(t, errors.As(err, &clientErr))
	assert.Equal(t, "AGENT_BODY_TOO_LARGE", clientErr.Code)
}
