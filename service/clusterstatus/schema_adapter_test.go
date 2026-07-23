package clusterstatus

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func telemetryFixture(t *testing.T, status string, modelName string, gpuCount int) []byte {
	t.Helper()
	gpus := make([]map[string]any, 0, gpuCount)
	for index := 0; index < gpuCount; index++ {
		gpus = append(gpus, map[string]any{
			"index":                      index,
			"uuid":                       "gpu-" + string(rune('a'+index)),
			"name":                       "NVIDIA GPU",
			"power_watts":                250 + index,
			"temperature_celsius":        60 + index,
			"utilization_percent":        70 + index,
			"memory_utilization_percent": 50 + index,
			"memory_used_bytes":          1024 * (index + 1),
			"memory_total_bytes":         8192,
			"sm_clock_mhz":               1500,
		})
	}
	payload, err := common.Marshal(map[string]any{
		"schema_version":         "1.0",
		"status":                 status,
		"collection_id":          "collection-1",
		"collection_finished_at": "2026-07-23T12:00:00Z",
		"identity": map[string]any{
			"node_id":   "node-a",
			"engine_id": "engine-a",
			"model":     modelName,
		},
		"engine": map[string]any{
			"up":                  true,
			"version":             "0.5.15",
			"observed_at":         "2026-07-23T12:00:00Z",
			"request_duration_ms": 12.5,
			"loads": []map[string]any{{
				"num_running_reqs": 2,
				"num_waiting_reqs": 1,
				"num_used_tokens":  120,
				"gen_throughput":   42.5,
				"cache_usage":      80,
				"total_requests":   500,
				"total_tokens":     8000,
			}},
		},
		"machine": map[string]any{
			"up": true,
			"nearest": map[string]any{
				"sampled_at": "2026-07-23T12:00:00Z",
				"collectors": map[string]any{
					"nvidia": map[string]any{
						"status": "ok",
						"data": map[string]any{
							"gpu_count":             gpuCount,
							"gpu_power_total_watts": gpuCount * 250,
							"driver_version":        "580.65",
							"gpus":                  gpus,
						},
					},
					"system": map[string]any{
						"status": "ok",
						"data": map[string]any{
							"cpu_utilization_percent":    25,
							"cpu_count":                  64,
							"memory_used_bytes":          4096,
							"memory_total_bytes":         16384,
							"memory_utilization_percent": 25,
							"load_average": map[string]any{
								"1m": 1.0, "5m": 2.0, "15m": 3.0,
							},
						},
					},
				},
			},
			"window": map[string]any{"window_complete": true},
		},
		"alignment": map[string]any{
			"method":  "nearest_timestamp",
			"quality": "good",
			"skew_ms": 12,
		},
	})
	require.NoError(t, err)
	return payload
}

func TestSchemaV1AdapterNormalizesDynamicGPUCounts(t *testing.T) {
	for _, gpuCount := range []int{0, 1, 2, 8} {
		t.Run(string(rune('0'+gpuCount))+"_gpus", func(t *testing.T) {
			telemetry, err := (SchemaV1Adapter{}).Adapt(
				telemetryFixture(t, "ok", "model-a", gpuCount),
				"model-a",
			)

			require.NoError(t, err)
			assert.Equal(t, gpuCount, telemetry.Machine.GPU.Count)
			assert.Len(t, telemetry.Machine.GPU.Devices, gpuCount)
			assert.Equal(t, float64(500), *telemetry.Metrics.Requests)
			assert.Equal(t, float64(8000), *telemetry.Metrics.Tokens)
		})
	}
}

func TestSchemaV1AdapterMarksModelMismatchWithoutChangingIdentity(t *testing.T) {
	telemetry, err := (SchemaV1Adapter{}).Adapt(
		telemetryFixture(t, "ok", "agent-model", 1),
		"linked-model",
	)

	require.NoError(t, err)
	assert.True(t, telemetry.ModelMismatch)
	assert.Equal(t, "agent-model", telemetry.Identity.Model)
	assert.Equal(
		t,
		model.ClusterHealthAbnormal,
		NewDefaultHealthEvaluator(3).Evaluate(telemetry),
	)
}

func TestSchemaV1AdapterRejectsUnknownSchema(t *testing.T) {
	raw, err := common.Marshal(map[string]any{
		"schema_version": "2.0",
		"collection_id":  "collection-2",
	})
	require.NoError(t, err)

	_, err = (SchemaV1Adapter{}).Adapt(raw, "model-a")

	var schemaErr *SchemaAdapterError
	require.ErrorAs(t, err, &schemaErr)
	assert.Equal(t, "AGENT_SCHEMA_UNSUPPORTED", schemaErr.Code)
}

func TestHealthEvaluatorMapsPartialAndFailures(t *testing.T) {
	telemetry, err := (SchemaV1Adapter{}).Adapt(
		telemetryFixture(t, "partial", "model-a", 1),
		"model-a",
	)
	require.NoError(t, err)
	evaluator := NewDefaultHealthEvaluator(3)

	assert.Equal(t, model.ClusterHealthPartial, evaluator.Evaluate(telemetry))
	assert.Equal(t, model.ClusterHealthAbnormal, evaluator.FailureStatus(2))
	assert.Equal(t, model.ClusterHealthOffline, evaluator.FailureStatus(3))
}
