package clusterstatus

import (
	"context"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

type GORMHistoryRepository struct{}

func (GORMHistoryRepository) List(ctx context.Context, clusterID int64, from time.Time, to time.Time) ([]NormalizedTelemetry, error) {
	if from.IsZero() {
		from = time.Now().AddDate(0, 0, -common.GetClusterTelemetryRetentionDays())
	}
	if to.IsZero() {
		to = time.Now().Add(time.Second)
	}
	rows, err := model.ListClusterTelemetryHistoryBatch(ctx, model.ClusterTelemetryHistoryFilter{
		ClusterIDs:    []int64{clusterID},
		FromInclusive: from.Unix(),
		ToExclusive:   to.Unix(),
		Limit:         1000,
	})
	if err != nil {
		return nil, err
	}
	history := make([]NormalizedTelemetry, 0, len(rows))
	for _, row := range rows {
		if row.Status != model.ClusterTelemetrySampleSuccess || row.NormalizedPayload == "" {
			continue
		}
		var telemetry NormalizedTelemetry
		if err := common.UnmarshalJsonStr(row.NormalizedPayload, &telemetry); err != nil {
			continue
		}
		history = append(history, telemetry)
	}
	return history, nil
}
