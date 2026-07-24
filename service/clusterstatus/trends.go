package clusterstatus

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const (
	defaultTrendMaxPoints = 720
	maxTrendPoints        = 2000
)

var trendBucketSteps = []int64{
	1, 2, 5, 10, 15, 30,
	60, 120, 300, 600, 900, 1800,
	3600, 7200, 10800, 21600, 43200,
	86400, 172800, 604800,
}

type TelemetryTrendInput struct {
	ClusterID int64
	StartAt   time.Time
	EndAt     time.Time
	MaxPoints int
}

type TelemetryTrendResponse struct {
	StartAt       string                `json:"start_at"`
	EndAt         string                `json:"end_at"`
	AvailableFrom int64                 `json:"available_from"`
	RetentionDays int                   `json:"retention_days"`
	BucketSeconds int64                 `json:"bucket_seconds"`
	SampleCount   int64                 `json:"sample_count"`
	Points        []TelemetryTrendPoint `json:"points"`
}

type TelemetryTrendPoint struct {
	Timestamp                  int64                    `json:"timestamp"`
	SampledAt                  int64                    `json:"sampled_at,omitempty"`
	SampleCount                int64                    `json:"sample_count"`
	SuccessCount               int64                    `json:"success_count"`
	PollSuccessPercent         float64                  `json:"poll_success_percent"`
	EngineAvailabilityPercent  *float64                 `json:"engine_availability_percent,omitempty"`
	MachineAvailabilityPercent *float64                 `json:"machine_availability_percent,omitempty"`
	RunningRequests            *float64                 `json:"running_requests,omitempty"`
	WaitingRequests            *float64                 `json:"waiting_requests,omitempty"`
	TokenUsage                 *float64                 `json:"token_usage,omitempty"`
	Throughput                 *float64                 `json:"throughput,omitempty"`
	CacheUsage                 *float64                 `json:"cache_usage,omitempty"`
	GPUBoardPowerWatts         *float64                 `json:"gpu_board_power_watts,omitempty"`
	CPUUtilizationPercent      *float64                 `json:"cpu_utilization_percent,omitempty"`
	MemoryUtilizationPercent   *float64                 `json:"memory_utilization_percent,omitempty"`
	GPUs                       []TelemetryTrendGPUPoint `json:"gpus"`
}

type TelemetryTrendGPUPoint struct {
	ID         string   `json:"id"`
	Index      int      `json:"index"`
	Name       string   `json:"name"`
	PowerWatts *float64 `json:"power_watts,omitempty"`
}

func (service *Service) GetTelemetryTrends(ctx context.Context, input TelemetryTrendInput) (*TelemetryTrendResponse, error) {
	cluster, err := model.GetClusterByID(input.ClusterID)
	if err != nil {
		return nil, err
	}
	if cluster == nil {
		return nil, ErrClusterNotFound
	}

	retentionDays := common.GetClusterTelemetryRetentionDays()
	if input.StartAt.IsZero() || input.EndAt.IsZero() || !input.StartAt.Before(input.EndAt) {
		return nil, ErrClusterTrendInvalid
	}
	if input.EndAt.Sub(input.StartAt) > time.Duration(retentionDays)*24*time.Hour {
		return nil, ErrClusterTrendInvalid
	}
	if input.MaxPoints == 0 {
		input.MaxPoints = defaultTrendMaxPoints
	}
	if input.MaxPoints < 1 || input.MaxPoints > maxTrendPoints {
		return nil, ErrClusterTrendInvalid
	}

	startAt := input.StartAt.UTC()
	endAt := input.EndAt.UTC()
	bucketSeconds := selectTrendBucketSeconds(startAt.Unix(), endAt.Unix(), input.MaxPoints)
	buckets, err := model.ListClusterTelemetryHistoryBuckets(
		ctx,
		input.ClusterID,
		startAt.Unix(),
		endAt.Unix(),
		bucketSeconds,
		input.MaxPoints,
	)
	if err != nil {
		return nil, err
	}

	successIDs := make([]int64, 0, len(buckets))
	var sampleCount int64
	for _, bucket := range buckets {
		sampleCount += bucket.SampleCount
		if bucket.LatestSuccessID > 0 {
			successIDs = append(successIDs, bucket.LatestSuccessID)
		}
	}
	historyRows, err := model.ListClusterTelemetryHistoryByIDs(ctx, successIDs)
	if err != nil {
		return nil, err
	}
	historyByID := make(map[int64]*model.ClusterTelemetryHistory, len(historyRows))
	for _, row := range historyRows {
		historyByID[row.ID] = row
	}

	points := make([]TelemetryTrendPoint, 0, len(buckets))
	for _, bucket := range buckets {
		timestamp := bucket.BucketStart
		if timestamp < startAt.Unix() {
			timestamp = startAt.Unix()
		}
		point := TelemetryTrendPoint{
			Timestamp:          timestamp,
			SampleCount:        bucket.SampleCount,
			SuccessCount:       bucket.SuccessCount,
			PollSuccessPercent: float64(bucket.SuccessCount) / float64(bucket.SampleCount) * 100,
			GPUs:               []TelemetryTrendGPUPoint{},
		}
		row := historyByID[bucket.LatestSuccessID]
		if row != nil && row.NormalizedPayload != "" {
			var telemetry NormalizedTelemetry
			if common.UnmarshalJsonStr(row.NormalizedPayload, &telemetry) == nil {
				point.SampledAt = row.CollectedAt
				point.EngineAvailabilityPercent = boolPercent(telemetry.Engine.Up)
				point.MachineAvailabilityPercent = boolPercent(telemetry.Machine.Up)
				point.RunningRequests = finiteMetric(telemetry.Engine.RunningRequests)
				point.WaitingRequests = finiteMetric(telemetry.Engine.WaitingRequests)
				point.TokenUsage = finiteMetric(telemetry.Engine.TokenUsage)
				point.Throughput = finiteMetric(telemetry.Engine.Throughput)
				point.CacheUsage = finiteMetric(telemetry.Engine.CacheUsage)
				point.GPUBoardPowerWatts = finiteMetric(telemetry.Machine.GPU.PowerTotalWatts)
				point.CPUUtilizationPercent = finiteMetric(telemetry.Machine.System.CPUUtilizationPercent)
				point.MemoryUtilizationPercent = finiteMetric(telemetry.Machine.System.MemoryUtilizationPercent)
				for _, device := range telemetry.Machine.GPU.Devices {
					name := strings.TrimSpace(device.Name)
					if name == "" {
						name = fmt.Sprintf("GPU %d", device.Index)
					}
					id := strings.TrimSpace(device.UUID)
					if id == "" {
						id = fmt.Sprintf("gpu-%d-%s", device.Index, name)
					}
					point.GPUs = append(point.GPUs, TelemetryTrendGPUPoint{
						ID:         id,
						Index:      device.Index,
						Name:       name,
						PowerWatts: finiteMetric(device.PowerWatts),
					})
				}
			}
		}
		points = append(points, point)
	}

	availableFrom, err := model.GetClusterTelemetryHistoryAvailableFromByClusterID(input.ClusterID)
	if err != nil {
		return nil, err
	}
	return &TelemetryTrendResponse{
		StartAt:       startAt.Format(time.RFC3339),
		EndAt:         endAt.Format(time.RFC3339),
		AvailableFrom: availableFrom,
		RetentionDays: retentionDays,
		BucketSeconds: bucketSeconds,
		SampleCount:   sampleCount,
		Points:        points,
	}, nil
}

func selectTrendBucketSeconds(startAt int64, endAt int64, maxPoints int) int64 {
	for _, step := range trendBucketSteps {
		firstBucket := startAt / step
		lastBucket := (endAt - 1) / step
		if lastBucket-firstBucket+1 <= int64(maxPoints) {
			return step
		}
	}
	duration := endAt - startAt
	return int64(math.Ceil(float64(duration)/float64(maxPoints)/86400)) * 86400
}

func boolPercent(value bool) *float64 {
	percent := float64(0)
	if value {
		percent = 100
	}
	return &percent
}

func finiteMetric(value *float64) *float64 {
	if value == nil || math.IsNaN(*value) || math.IsInf(*value, 0) {
		return nil
	}
	metric := *value
	return &metric
}
