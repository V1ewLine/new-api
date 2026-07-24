package clusterstatus

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const supportedAgentSchemaVersion = "1.0"

type SchemaAdapterError struct {
	Code string
}

func (err *SchemaAdapterError) Error() string {
	if err == nil || err.Code == "" {
		return "cluster telemetry schema error"
	}
	return "cluster telemetry schema error: " + err.Code
}

type agentEnvelope struct {
	SchemaVersion        string         `json:"schema_version"`
	Status               string         `json:"status"`
	CollectionID         string         `json:"collection_id"`
	CollectionFinishedAt string         `json:"collection_finished_at"`
	Identity             agentIdentity  `json:"identity"`
	Engine               agentEngine    `json:"engine"`
	Machine              agentMachine   `json:"machine"`
	Alignment            agentAlignment `json:"alignment"`
}

type agentIdentity struct {
	NodeID   string `json:"node_id"`
	EngineID string `json:"engine_id"`
	Model    string `json:"model"`
}

type agentError struct {
	Code string `json:"code"`
}

type agentEngine struct {
	Up                bool             `json:"up"`
	Version           string           `json:"version"`
	ObservedAt        string           `json:"observed_at"`
	RequestDurationMS float64          `json:"request_duration_ms"`
	Loads             []map[string]any `json:"loads"`
	Error             *agentError      `json:"error"`
}

type agentMachine struct {
	Up      bool                 `json:"up"`
	Nearest *agentMachineNearest `json:"nearest"`
	Window  agentMachineWindow   `json:"window"`
}

type agentMachineNearest struct {
	SampledAt  string                           `json:"sampled_at"`
	Collectors map[string]agentCollectorReading `json:"collectors"`
}

type agentCollectorReading struct {
	Status string         `json:"status"`
	Data   map[string]any `json:"data"`
	Error  *agentError    `json:"error"`
}

type agentMachineWindow struct {
	WindowComplete bool                      `json:"window_complete"`
	Collectors     map[string]map[string]any `json:"collectors"`
}

type agentAlignment struct {
	Method  string   `json:"method"`
	Quality string   `json:"quality"`
	SkewMS  *float64 `json:"skew_ms"`
}

type SchemaV1Adapter struct{}

func (SchemaV1Adapter) Adapt(raw []byte, expectedModelName string) (*NormalizedTelemetry, error) {
	if len(raw) == 0 {
		return nil, &SchemaAdapterError{Code: "AGENT_SCHEMA_EMPTY"}
	}
	var envelope agentEnvelope
	if err := common.Unmarshal(raw, &envelope); err != nil {
		return nil, &SchemaAdapterError{Code: "AGENT_SCHEMA_INVALID_JSON"}
	}
	if envelope.SchemaVersion != supportedAgentSchemaVersion {
		return nil, &SchemaAdapterError{Code: "AGENT_SCHEMA_UNSUPPORTED"}
	}
	if strings.TrimSpace(envelope.CollectionID) == "" {
		return nil, &SchemaAdapterError{Code: "AGENT_SCHEMA_INVALID"}
	}

	telemetry := &NormalizedTelemetry{
		SchemaVersion: envelope.SchemaVersion,
		CollectionID:  envelope.CollectionID,
		Status:        envelope.Status,
		CollectedAt:   envelope.CollectionFinishedAt,
		Identity: TelemetryIdentity{
			NodeID:   envelope.Identity.NodeID,
			EngineID: envelope.Identity.EngineID,
			Model:    envelope.Identity.Model,
		},
		Engine: TelemetryEngine{
			Up:                envelope.Engine.Up,
			Version:           envelope.Engine.Version,
			ObservedAt:        envelope.Engine.ObservedAt,
			RequestDurationMS: envelope.Engine.RequestDurationMS,
			Loads:             envelope.Engine.Loads,
		},
		Machine: TelemetryMachine{
			Up:             envelope.Machine.Up,
			WindowComplete: envelope.Machine.Window.WindowComplete,
			GPU:            TelemetryGPU{Devices: []TelemetryGPUDevice{}},
			System: TelemetrySystem{
				LoadAverage: map[string]*float64{},
			},
		},
		Alignment: TelemetryAlignment{
			Method:  envelope.Alignment.Method,
			Quality: envelope.Alignment.Quality,
			SkewMS:  envelope.Alignment.SkewMS,
		},
	}
	if envelope.Engine.Error != nil {
		telemetry.Engine.ErrorCode = envelope.Engine.Error.Code
	}
	expectedModelName = strings.TrimSpace(expectedModelName)
	agentModelName := strings.TrimSpace(envelope.Identity.Model)
	telemetry.ModelMismatch = expectedModelName != "" && agentModelName != "" && !strings.EqualFold(expectedModelName, agentModelName)

	telemetry.Engine.RunningRequests = sumLoadMetric(envelope.Engine.Loads,
		"num_running_reqs", "running_requests", "num_running_requests")
	telemetry.Engine.WaitingRequests = sumLoadMetric(envelope.Engine.Loads,
		"num_waiting_reqs", "waiting_requests", "num_waiting_requests")
	telemetry.Engine.TokenUsage = sumLoadMetric(envelope.Engine.Loads,
		"num_used_tokens", "token_usage", "used_tokens")
	telemetry.Engine.Throughput = sumLoadMetric(envelope.Engine.Loads,
		"gen_throughput", "throughput", "token_throughput")
	telemetry.Engine.CacheUsage = averageLoadMetric(envelope.Engine.Loads,
		"cache_usage", "cache_usage_percent", "gpu_cache_usage")
	telemetry.Metrics.Requests = sumLoadMetric(envelope.Engine.Loads,
		"total_requests", "request_count", "requests")
	if telemetry.Metrics.Requests != nil {
		telemetry.Metrics.RequestsSemantics = "cumulative"
	}
	telemetry.Metrics.Tokens = sumLoadMetric(envelope.Engine.Loads,
		"total_tokens", "token_count", "tokens")
	if telemetry.Metrics.Tokens != nil {
		telemetry.Metrics.TokensSemantics = "cumulative"
	}
	if telemetry.Metrics.Requests == nil {
		runningRequests := sumLoadMetric(envelope.Engine.Loads,
			"num_running_reqs", "running_requests", "num_running_requests")
		waitingRequests := sumLoadMetric(envelope.Engine.Loads,
			"num_waiting_reqs", "waiting_requests", "num_waiting_requests")
		if runningRequests != nil || waitingRequests != nil {
			var currentRequests float64
			if runningRequests != nil {
				currentRequests += *runningRequests
			}
			if waitingRequests != nil {
				currentRequests += *waitingRequests
			}
			telemetry.Metrics.Requests = &currentRequests
			telemetry.Metrics.RequestsSemantics = "current_inflight"
		}
	}
	if telemetry.Metrics.Tokens == nil {
		telemetry.Metrics.Tokens = sumLoadMetric(envelope.Engine.Loads,
			"num_total_tokens", "num_used_tokens", "token_usage", "used_tokens")
		if telemetry.Metrics.Tokens != nil {
			telemetry.Metrics.TokensSemantics = "current_usage"
		}
	}

	if envelope.Machine.Nearest != nil {
		telemetry.Machine.SampledAt = envelope.Machine.Nearest.SampledAt
		if reading, ok := envelope.Machine.Nearest.Collectors["nvidia"]; ok && reading.Status == "ok" {
			telemetry.Machine.GPU = adaptNvidiaReading(reading.Data)
		}
		if reading, ok := envelope.Machine.Nearest.Collectors["system"]; ok && reading.Status == "ok" {
			telemetry.Machine.System = adaptSystemReading(reading.Data)
		}
	}

	return telemetry, nil
}

func adaptNvidiaReading(data map[string]any) TelemetryGPU {
	gpu := TelemetryGPU{
		Available:       true,
		Count:           int(numberValue(data["gpu_count"])),
		PowerTotalWatts: numberPointer(data["gpu_power_total_watts"]),
		DriverVersion:   textValue(data["driver_version"]),
		Devices:         []TelemetryGPUDevice{},
	}
	rawDevices, ok := data["gpus"].([]any)
	if !ok {
		return gpu
	}
	for _, rawDevice := range rawDevices {
		deviceData, ok := rawDevice.(map[string]any)
		if !ok {
			continue
		}
		gpu.Devices = append(gpu.Devices, TelemetryGPUDevice{
			Index:                    int(numberValue(deviceData["index"])),
			UUID:                     textValue(deviceData["uuid"]),
			Name:                     textValue(deviceData["name"]),
			PowerWatts:               numberPointer(deviceData["power_watts"]),
			TemperatureCelsius:       numberPointer(deviceData["temperature_celsius"]),
			UtilizationPercent:       numberPointer(deviceData["utilization_percent"]),
			MemoryUtilizationPercent: numberPointer(deviceData["memory_utilization_percent"]),
			MemoryUsedBytes:          numberPointer(deviceData["memory_used_bytes"]),
			MemoryTotalBytes:         numberPointer(deviceData["memory_total_bytes"]),
			SMClockMHz:               numberPointer(deviceData["sm_clock_mhz"]),
		})
	}
	if gpu.Count == 0 {
		gpu.Count = len(gpu.Devices)
	}
	return gpu
}

func adaptSystemReading(data map[string]any) TelemetrySystem {
	system := TelemetrySystem{
		Available:                true,
		CPUUtilizationPercent:    numberPointer(data["cpu_utilization_percent"]),
		CPUCount:                 numberPointer(data["cpu_count"]),
		MemoryUsedBytes:          numberPointer(data["memory_used_bytes"]),
		MemoryAvailableBytes:     numberPointer(data["memory_available_bytes"]),
		MemoryTotalBytes:         numberPointer(data["memory_total_bytes"]),
		MemoryUtilizationPercent: numberPointer(data["memory_utilization_percent"]),
		LoadAverage:              map[string]*float64{},
	}
	if loadAverage, ok := data["load_average"].(map[string]any); ok {
		for _, window := range []string{"1m", "5m", "15m"} {
			system.LoadAverage[window] = numberPointer(loadAverage[window])
		}
	}
	return system
}

func sumLoadMetric(loads []map[string]any, keys ...string) *float64 {
	var total float64
	found := false
	for _, load := range loads {
		for _, key := range keys {
			value := numberPointer(load[key])
			if value == nil {
				continue
			}
			total += *value
			found = true
			break
		}
	}
	if !found {
		return nil
	}
	return &total
}

func averageLoadMetric(loads []map[string]any, keys ...string) *float64 {
	var total float64
	count := 0
	for _, load := range loads {
		for _, key := range keys {
			value := numberPointer(load[key])
			if value == nil {
				continue
			}
			total += *value
			count++
			break
		}
	}
	if count == 0 {
		return nil
	}
	average := total / float64(count)
	return &average
}

func numberPointer(value any) *float64 {
	switch typed := value.(type) {
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) {
			return nil
		}
		return &typed
	case float32:
		converted := float64(typed)
		return &converted
	case int:
		converted := float64(typed)
		return &converted
	case int64:
		converted := float64(typed)
		return &converted
	case string:
		converted, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil || math.IsNaN(converted) || math.IsInf(converted, 0) {
			return nil
		}
		return &converted
	default:
		return nil
	}
}

func numberValue(value any) float64 {
	number := numberPointer(value)
	if number == nil {
		return 0
	}
	return *number
}

func textValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return ""
	}
}

func schemaErrorCode(err error) string {
	var schemaErr *SchemaAdapterError
	if errors.As(err, &schemaErr) && schemaErr.Code != "" {
		return schemaErr.Code
	}
	return "AGENT_SCHEMA_INVALID"
}
