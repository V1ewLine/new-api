package dto

import (
	"testing"

	"github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIResponsesCreatedAtAcceptsIntegralTimestampForms(t *testing.T) {
	testCases := []struct {
		name string
		body string
	}{
		{
			name: "integer",
			body: `{"id":"resp_1","created_at":1785149227}`,
		},
		{
			name: "integral decimal",
			body: `{"id":"resp_1","created_at":1785149227.0}`,
		},
		{
			name: "multiple zero decimals",
			body: `{"id":"resp_1","created_at":1785149227.000}`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var response OpenAIResponsesResponse
			require.NoError(t, kitutil.Unmarshal([]byte(testCase.body), &response))
			assert.Equal(t, int64(1785149227), int64(response.CreatedAt))

			encoded, err := kitutil.Marshal(response)
			require.NoError(t, err)
			assert.Contains(t, string(encoded), `"created_at":1785149227`)
			assert.NotContains(t, string(encoded), `"created_at":1785149227.0`)
		})
	}
}

func TestResponsesStreamCreatedAtAcceptsIntegralDecimal(t *testing.T) {
	var event ResponsesStreamResponse
	require.NoError(
		t,
		kitutil.Unmarshal(
			[]byte(`{"type":"response.created","response":{"id":"resp_1","created_at":1785149227.0}}`),
			&event,
		),
	)
	require.NotNil(t, event.Response)
	assert.Equal(t, int64(1785149227), int64(event.Response.CreatedAt))
}

func TestOpenAIResponsesCreatedAtRejectsFractionalTimestamp(t *testing.T) {
	var response OpenAIResponsesResponse
	err := kitutil.Unmarshal([]byte(`{"id":"resp_1","created_at":1785149227.5}`), &response)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fractional value is not supported")
}
