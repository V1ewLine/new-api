package clusterstatus

import (
	"context"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTelemetryTrendsDownsamplesAndUsesLatestSuccessfulSample(t *testing.T) {
	setupClusterServiceTestDB(t)
	previousRetention := common.ClusterTelemetryRetentionDays
	common.ClusterTelemetryRetentionDays = 7
	t.Cleanup(func() {
		common.ClusterTelemetryRetentionDays = previousRetention
	})

	cluster := &model.Cluster{
		ModelID:              1,
		ModelNameSnapshot:    "model-a",
		Name:                 "cluster-a",
		LinkSecretCiphertext: "ciphertext",
		Enabled:              true,
	}
	require.NoError(t, model.DB.Create(cluster).Error)

	startUnix := int64(1_700_000_000)
	createTrendSample(t, cluster.ID, startUnix+1, ClusterTelemetrySampleFixture{
		RunningRequests: 1,
		GPUPowerWatts:   100,
	})
	createTrendErrorSample(t, cluster.ID, startUnix+3)
	createTrendSample(t, cluster.ID, startUnix+4, ClusterTelemetrySampleFixture{
		RunningRequests:     4,
		WaitingRequests:     2,
		TokenUsage:          120,
		Throughput:          30,
		CacheUsage:          80,
		GPUPowerWatts:       240,
		CPUPercent:          55,
		MemoryPercent:       65,
		GPUDevicePower:      120,
		EngineAvailable:     true,
		MachineAvailable:    true,
		IncludeAvailability: true,
	})
	createTrendErrorSample(t, cluster.ID, startUnix+6)
	createTrendSample(t, cluster.ID, startUnix+11, ClusterTelemetrySampleFixture{
		RunningRequests: 9,
		GPUPowerWatts:   260,
	})

	response, err := testService(t, failingAgentClient{}).GetTelemetryTrends(
		context.Background(),
		TelemetryTrendInput{
			ClusterID: cluster.ID,
			StartAt:   time.Unix(startUnix, 0),
			EndAt:     time.Unix(startUnix+20, 0),
			MaxPoints: 4,
		},
	)

	require.NoError(t, err)
	assert.Equal(t, int64(5), response.BucketSeconds)
	assert.Equal(t, int64(5), response.SampleCount)
	assert.Equal(t, startUnix+1, response.AvailableFrom)
	require.Len(t, response.Points, 3)

	first := response.Points[0]
	assert.Equal(t, int64(3), first.SampleCount)
	assert.Equal(t, int64(2), first.SuccessCount)
	assert.InDelta(t, 66.67, first.PollSuccessPercent, 0.01)
	require.NotNil(t, first.RunningRequests)
	assert.Equal(t, float64(4), *first.RunningRequests)
	require.NotNil(t, first.GPUBoardPowerWatts)
	assert.Equal(t, float64(240), *first.GPUBoardPowerWatts)
	require.NotNil(t, first.EngineAvailabilityPercent)
	assert.Equal(t, float64(100), *first.EngineAvailabilityPercent)
	require.Len(t, first.GPUs, 1)
	assert.Equal(t, "GPU-uuid-0", first.GPUs[0].ID)
	require.NotNil(t, first.GPUs[0].PowerWatts)
	assert.Equal(t, float64(120), *first.GPUs[0].PowerWatts)

	assert.Zero(t, response.Points[1].SuccessCount)
	assert.Nil(t, response.Points[1].RunningRequests)
	require.NotNil(t, response.Points[2].RunningRequests)
	assert.Equal(t, float64(9), *response.Points[2].RunningRequests)
}

func TestGetTelemetryTrendsValidatesClusterRangeAndPointLimit(t *testing.T) {
	setupClusterServiceTestDB(t)
	service := testService(t, failingAgentClient{})
	startAt := time.Unix(1_700_000_000, 0)

	_, err := service.GetTelemetryTrends(context.Background(), TelemetryTrendInput{
		ClusterID: 999,
		StartAt:   startAt,
		EndAt:     startAt.Add(time.Hour),
		MaxPoints: 100,
	})
	require.ErrorIs(t, err, ErrClusterNotFound)

	cluster := &model.Cluster{
		ModelID:              1,
		ModelNameSnapshot:    "model-a",
		Name:                 "cluster-a",
		LinkSecretCiphertext: "ciphertext",
		Enabled:              true,
	}
	require.NoError(t, model.DB.Create(cluster).Error)
	testCases := []TelemetryTrendInput{
		{ClusterID: cluster.ID, StartAt: startAt, EndAt: startAt, MaxPoints: 100},
		{ClusterID: cluster.ID, StartAt: startAt, EndAt: startAt.Add(8 * 24 * time.Hour), MaxPoints: 100},
		{ClusterID: cluster.ID, StartAt: startAt, EndAt: startAt.Add(time.Hour), MaxPoints: maxTrendPoints + 1},
	}
	for _, input := range testCases {
		_, err = service.GetTelemetryTrends(context.Background(), input)
		require.ErrorIs(t, err, ErrClusterTrendInvalid)
	}
}

type ClusterTelemetrySampleFixture struct {
	RunningRequests     float64
	WaitingRequests     float64
	TokenUsage          float64
	Throughput          float64
	CacheUsage          float64
	GPUPowerWatts       float64
	CPUPercent          float64
	MemoryPercent       float64
	GPUDevicePower      float64
	EngineAvailable     bool
	MachineAvailable    bool
	IncludeAvailability bool
}

func createTrendSample(t *testing.T, clusterID int64, collectedAt int64, fixture ClusterTelemetrySampleFixture) {
	t.Helper()
	engineAvailable := fixture.EngineAvailable
	machineAvailable := fixture.MachineAvailable
	if !fixture.IncludeAvailability {
		engineAvailable = true
		machineAvailable = true
	}
	telemetry := NormalizedTelemetry{
		SchemaVersion: "1.0",
		CollectionID:  "collection",
		Engine: TelemetryEngine{
			Up:              engineAvailable,
			RunningRequests: &fixture.RunningRequests,
			WaitingRequests: &fixture.WaitingRequests,
			TokenUsage:      &fixture.TokenUsage,
			Throughput:      &fixture.Throughput,
			CacheUsage:      &fixture.CacheUsage,
		},
		Machine: TelemetryMachine{
			Up: machineAvailable,
			GPU: TelemetryGPU{
				PowerTotalWatts: &fixture.GPUPowerWatts,
				Devices: []TelemetryGPUDevice{
					{
						Index:      0,
						UUID:       "GPU-uuid-0",
						Name:       "GPU A",
						PowerWatts: &fixture.GPUDevicePower,
					},
				},
			},
			System: TelemetrySystem{
				CPUUtilizationPercent:    &fixture.CPUPercent,
				MemoryUtilizationPercent: &fixture.MemoryPercent,
			},
		},
	}
	payload, err := common.Marshal(telemetry)
	require.NoError(t, err)
	collectionID := "collection-" + time.Unix(collectedAt, 0).Format("150405")
	require.NoError(t, model.DB.Create(&model.ClusterTelemetryHistory{
		ClusterID:         clusterID,
		CollectionID:      &collectionID,
		Status:            model.ClusterTelemetrySampleSuccess,
		HealthStatus:      model.ClusterHealthOnline,
		SchemaVersion:     "1.0",
		NormalizedPayload: string(payload),
		CollectedAt:       collectedAt,
		CreatedAt:         collectedAt,
	}).Error)
}

func createTrendErrorSample(t *testing.T, clusterID int64, collectedAt int64) {
	t.Helper()
	require.NoError(t, model.DB.Create(&model.ClusterTelemetryHistory{
		ClusterID:    clusterID,
		Status:       model.ClusterTelemetrySampleError,
		HealthStatus: model.ClusterHealthOffline,
		ErrorCode:    "AGENT_UNREACHABLE",
		CollectedAt:  collectedAt,
		CreatedAt:    collectedAt,
	}).Error)
}
