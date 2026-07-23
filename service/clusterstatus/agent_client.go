package clusterstatus

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
)

type AgentClientError struct {
	Code string
}

func (err *AgentClientError) Error() string {
	if err == nil || err.Code == "" {
		return "cluster Agent request failed"
	}
	return "cluster Agent request failed: " + err.Code
}

type HTTPAgentClient struct {
	client       *http.Client
	maxBodyBytes int64
}

func NewHTTPAgentClient(client *http.Client, maxBodyBytes int64) *HTTPAgentClient {
	if maxBodyBytes <= 0 {
		maxBodyBytes = 2 * 1024 * 1024
	}
	return &HTTPAgentClient{client: client, maxBodyBytes: maxBodyBytes}
}

func (client *HTTPAgentClient) Fetch(ctx context.Context, connection ResolvedAgentConnection) ([]byte, error) {
	if client == nil || client.client == nil {
		return nil, &AgentClientError{Code: "AGENT_CLIENT_UNAVAILABLE"}
	}
	endpoint := strings.TrimRight(connection.BaseURL, "/") + "/v1/telemetry"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, &AgentClientError{Code: "AGENT_REQUEST_INVALID"}
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+connection.BearerToken)

	response, err := client.client.Do(request)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, &AgentClientError{Code: "AGENT_TIMEOUT"}
		}
		if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			return nil, context.Canceled
		}
		return nil, &AgentClientError{Code: "AGENT_UNREACHABLE"}
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, &AgentClientError{Code: fmt.Sprintf("AGENT_HTTP_%d", response.StatusCode)}
	}
	contentType := strings.TrimSpace(response.Header.Get("Content-Type"))
	if contentType == "" {
		return nil, &AgentClientError{Code: "AGENT_CONTENT_TYPE_MISSING"}
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "application/json" {
		return nil, &AgentClientError{Code: "AGENT_CONTENT_TYPE_INVALID"}
	}

	limited := io.LimitReader(response.Body, client.maxBodyBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, &AgentClientError{Code: "AGENT_BODY_READ_FAILED"}
	}
	if int64(len(body)) > client.maxBodyBytes {
		return nil, &AgentClientError{Code: "AGENT_BODY_TOO_LARGE"}
	}
	return body, nil
}

func agentErrorCode(err error) string {
	var clientErr *AgentClientError
	if errors.As(err, &clientErr) && clientErr.Code != "" {
		return clientErr.Code
	}
	if errors.Is(err, context.Canceled) {
		return "AGENT_REQUEST_CANCELED"
	}
	return "AGENT_REQUEST_FAILED"
}
