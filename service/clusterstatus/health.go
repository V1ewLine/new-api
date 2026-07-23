package clusterstatus

import (
	"strings"

	"github.com/QuantumNous/new-api/model"
)

type DefaultHealthEvaluator struct {
	failureThreshold int
}

func NewDefaultHealthEvaluator(failureThreshold int) *DefaultHealthEvaluator {
	if failureThreshold <= 0 {
		failureThreshold = 3
	}
	return &DefaultHealthEvaluator{failureThreshold: failureThreshold}
}

func (evaluator *DefaultHealthEvaluator) Evaluate(telemetry *NormalizedTelemetry) model.ClusterHealthStatus {
	if telemetry == nil {
		return model.ClusterHealthUnknown
	}
	if telemetry.ModelMismatch {
		return model.ClusterHealthAbnormal
	}
	if !telemetry.Engine.Up && !telemetry.Machine.Up {
		return model.ClusterHealthOffline
	}
	if !telemetry.Engine.Up || !telemetry.Machine.Up {
		return model.ClusterHealthAbnormal
	}
	if strings.EqualFold(telemetry.Status, "partial") ||
		!strings.EqualFold(telemetry.Alignment.Quality, "good") ||
		!telemetry.Machine.WindowComplete {
		return model.ClusterHealthPartial
	}
	if strings.EqualFold(telemetry.Status, "ok") {
		return model.ClusterHealthOnline
	}
	return model.ClusterHealthUnknown
}

func (evaluator *DefaultHealthEvaluator) FailureStatus(consecutiveFailures int) model.ClusterHealthStatus {
	if consecutiveFailures >= evaluator.failureThreshold {
		return model.ClusterHealthOffline
	}
	return model.ClusterHealthAbnormal
}
