package clusterstatus

import (
	"context"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHistoryRangeUsesStartInclusiveEndExclusiveAndStableCursor(t *testing.T) {
	setupClusterServiceTestDB(t)
	cluster := &model.Cluster{
		ModelID:           1,
		ModelNameSnapshot: "model-a",
		Name:              "cluster-a",
		Enabled:           true,
	}
	require.NoError(t, model.CreateCluster(cluster))
	for index, collectedAt := range []int64{99, 100, 100, 199, 200} {
		require.NoError(t, model.DB.Create(&model.ClusterTelemetryHistory{
			ClusterID:    cluster.ID,
			Status:       model.ClusterTelemetrySampleError,
			HealthStatus: model.ClusterHealthOffline,
			ErrorCode:    "TEST",
			CollectedAt:  collectedAt,
			CreatedAt:    int64(index + 1),
		}).Error)
	}

	firstPage, err := model.ListClusterTelemetryHistoryBatch(
		context.Background(),
		model.ClusterTelemetryHistoryFilter{
			ClusterIDs:    []int64{cluster.ID},
			FromInclusive: 100,
			ToExclusive:   200,
			Limit:         2,
		},
	)
	require.NoError(t, err)
	require.Len(t, firstPage, 2)
	assert.Equal(t, int64(100), firstPage[0].CollectedAt)
	assert.Equal(t, int64(100), firstPage[1].CollectedAt)

	secondPage, err := model.ListClusterTelemetryHistoryBatch(
		context.Background(),
		model.ClusterTelemetryHistoryFilter{
			ClusterIDs:       []int64{cluster.ID},
			FromInclusive:    100,
			ToExclusive:      200,
			AfterCollectedAt: firstPage[1].CollectedAt,
			AfterID:          firstPage[1].ID,
			Limit:            2,
		},
	)
	require.NoError(t, err)
	require.Len(t, secondPage, 1)
	assert.Equal(t, int64(199), secondPage[0].CollectedAt)
}

func TestHistoryCleanupDeletesOnlyExpiredRowsInBatches(t *testing.T) {
	setupClusterServiceTestDB(t)
	for _, collectedAt := range []int64{100, 200, 300} {
		require.NoError(t, model.DB.Create(&model.ClusterTelemetryHistory{
			ClusterID:    1,
			Status:       model.ClusterTelemetrySampleError,
			HealthStatus: model.ClusterHealthOffline,
			ErrorCode:    "TEST",
			CollectedAt:  collectedAt,
			CreatedAt:    collectedAt,
		}).Error)
	}

	deleted, err := model.DeleteClusterTelemetryHistoryBefore(250, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)
	deleted, err = model.DeleteClusterTelemetryHistoryBefore(250, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	var rows []model.ClusterTelemetryHistory
	require.NoError(t, model.DB.Order("collected_at ASC").Find(&rows).Error)
	require.Len(t, rows, 1)
	assert.Equal(t, int64(300), rows[0].CollectedAt)
}
