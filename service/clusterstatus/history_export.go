package clusterstatus

import (
	"archive/zip"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const (
	clusterHistoryExportSchemaVersion = "new-api.cluster-history-export.v1"
	maxHistoryExportSamples           = 1000000
	historyExportBatchSize            = 2000
)

type HistoryExportInput struct {
	Scope     string
	Search    string
	ModelID   int
	ClusterID int64
	Health    model.ClusterHealthStatus
	StartAt   time.Time
	EndAt     time.Time
}

type PreparedHistoryExport struct {
	Filename     string
	ContentType  string
	ClusterCount int
	SampleCount  int64
	Scope        string
	StartAt      time.Time
	EndAt        time.Time
	context      context.Context
	filters      LatestExportFilters
	clusters     []LatestExportCluster
	clusterIDs   []int64
}

func (service *Service) PrepareHistoryExport(ctx context.Context, input HistoryExportInput) (*PreparedHistoryExport, error) {
	input.Scope = strings.ToLower(strings.TrimSpace(input.Scope))
	input.Search = strings.TrimSpace(input.Search)
	if input.Scope != "all" && input.Scope != "cluster" {
		return nil, ErrClusterExportInvalid
	}
	if input.StartAt.IsZero() || input.EndAt.IsZero() || !input.StartAt.Before(input.EndAt) {
		return nil, ErrClusterExportInvalid
	}
	maxDuration := time.Duration(common.GetClusterTelemetryRetentionDays()) * 24 * time.Hour
	if input.EndAt.Sub(input.StartAt) > maxDuration {
		return nil, ErrClusterExportInvalid
	}
	switch input.Health {
	case "", model.ClusterHealthUnknown, model.ClusterHealthOnline, model.ClusterHealthPartial, model.ClusterHealthAbnormal, model.ClusterHealthOffline:
	default:
		return nil, ErrClusterExportInvalid
	}

	var clusters []*model.Cluster
	var err error
	if input.Scope == "cluster" {
		if input.ClusterID <= 0 {
			return nil, ErrClusterExportInvalid
		}
		cluster, getErr := model.GetClusterByID(input.ClusterID)
		if getErr != nil {
			return nil, getErr
		}
		if cluster == nil {
			return nil, ErrClusterNotFound
		}
		clusters = []*model.Cluster{cluster}
	} else {
		if input.ModelID < 0 {
			return nil, ErrClusterExportInvalid
		}
		clusters, err = model.ListClusters(model.ClusterListFilter{
			Search:  input.Search,
			ModelID: input.ModelID,
			Health:  input.Health,
		})
		if err != nil {
			return nil, err
		}
	}
	if len(clusters) > maxExportClusters {
		return nil, ErrClusterExportTooLarge
	}

	clusterIDs := make([]int64, 0, len(clusters))
	modelIDs := make([]int, 0, len(clusters))
	for _, cluster := range clusters {
		clusterIDs = append(clusterIDs, cluster.ID)
		modelIDs = append(modelIDs, cluster.ModelID)
	}
	var sampleCount int64
	if len(clusterIDs) > 0 {
		sampleCount, err = model.CountClusterTelemetryHistory(
			clusterIDs,
			input.StartAt.Unix(),
			input.EndAt.Unix(),
		)
		if err != nil {
			return nil, err
		}
	}
	if sampleCount > maxHistoryExportSamples {
		return nil, ErrClusterExportTooLarge
	}
	modelMap, err := loadClusterModels(modelIDs)
	if err != nil {
		return nil, err
	}
	exportClusters := make([]LatestExportCluster, 0, len(clusters))
	for _, cluster := range clusters {
		response := service.clusterResponse(cluster, modelMap[cluster.ModelID], nil)
		exportClusters = append(exportClusters, LatestExportCluster{
			ID:                  response.ID,
			ModelID:             response.ModelID,
			ModelName:           response.ModelName,
			ModelAvailable:      response.ModelAvailable,
			Name:                response.Name,
			Enabled:             response.Enabled,
			HealthStatus:        response.HealthStatus,
			CredentialStatus:    response.CredentialStatus,
			LastPolledAt:        response.LastPolledAt,
			LastSuccessAt:       response.LastSuccessAt,
			ConsecutiveFailures: response.ConsecutiveFailures,
			LastErrorCode:       response.LastErrorCode,
			CreatedAt:           response.CreatedAt,
			UpdatedAt:           response.UpdatedAt,
		})
	}

	now := time.Now().UTC()
	return &PreparedHistoryExport{
		Filename:     fmt.Sprintf("cluster-history-%s-%s.zip", input.Scope, now.Format("20060102T150405Z")),
		ContentType:  "application/zip",
		ClusterCount: len(clusters),
		SampleCount:  sampleCount,
		Scope:        input.Scope,
		StartAt:      input.StartAt,
		EndAt:        input.EndAt,
		context:      ctx,
		filters: LatestExportFilters{
			Search:    input.Search,
			ModelID:   input.ModelID,
			ClusterID: input.ClusterID,
			Status:    input.Health,
		},
		clusters:   exportClusters,
		clusterIDs: clusterIDs,
	}, nil
}

func (prepared *PreparedHistoryExport) WriteTo(writer io.Writer) error {
	if prepared == nil {
		return ErrClusterExportInvalid
	}
	archive := zip.NewWriter(writer)
	files := []struct {
		name  string
		write func(io.Writer) error
	}{
		{name: "manifest.json", write: prepared.writeManifest},
		{name: "clusters.csv", write: prepared.writeClustersCSV},
		{name: "telemetry_history.csv", write: prepared.writeTelemetryCSV},
		{name: "gpu_device_history.csv", write: prepared.writeGPUCSV},
		{name: "engine_load_history.csv", write: prepared.writeEngineLoadsCSV},
		{name: "normalized_telemetry_history.jsonl", write: prepared.writeJSONL},
	}
	for _, file := range files {
		entry, err := archive.Create(file.name)
		if err != nil {
			_ = archive.Close()
			return err
		}
		if err := file.write(entry); err != nil {
			_ = archive.Close()
			return err
		}
	}
	return archive.Close()
}

func (prepared *PreparedHistoryExport) writeManifest(writer io.Writer) error {
	payload, err := common.Marshal(map[string]any{
		"schema_version": clusterHistoryExportSchemaVersion,
		"exported_at":    time.Now().UTC().Format(time.RFC3339),
		"scope":          prepared.Scope,
		"filters":        prepared.filters,
		"start_at":       prepared.StartAt.UTC().Format(time.RFC3339),
		"end_at":         prepared.EndAt.UTC().Format(time.RFC3339),
		"time_window":    "[start_at, end_at)",
		"retention_days": common.GetClusterTelemetryRetentionDays(),
		"cluster_count":  prepared.ClusterCount,
		"sample_count":   prepared.SampleCount,
		"files": []string{
			"manifest.json",
			"clusters.csv",
			"telemetry_history.csv",
			"gpu_device_history.csv",
			"engine_load_history.csv",
			"normalized_telemetry_history.jsonl",
		},
	})
	if err != nil {
		return err
	}
	_, err = writer.Write(payload)
	return err
}

func (prepared *PreparedHistoryExport) writeClustersCSV(writer io.Writer) error {
	return writeStreamingCSV(writer, []string{
		"cluster_id", "cluster_name", "model_id", "model_name", "model_available",
		"enabled", "credential_status", "health_status", "last_polled_at",
		"last_success_at", "consecutive_failures", "last_error_code",
	}, func(csvWriter *csv.Writer) error {
		for _, cluster := range prepared.clusters {
			if err := csvWriter.Write([]string{
				strconv.FormatInt(cluster.ID, 10),
				safeCSVText(cluster.Name),
				strconv.Itoa(cluster.ModelID),
				safeCSVText(cluster.ModelName),
				strconv.FormatBool(cluster.ModelAvailable),
				strconv.FormatBool(cluster.Enabled),
				safeCSVText(string(cluster.CredentialStatus)),
				safeCSVText(string(cluster.HealthStatus)),
				unixTimestamp(cluster.LastPolledAt),
				unixTimestamp(cluster.LastSuccessAt),
				strconv.Itoa(cluster.ConsecutiveFailures),
				safeCSVText(cluster.LastErrorCode),
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func (prepared *PreparedHistoryExport) writeTelemetryCSV(writer io.Writer) error {
	return writeStreamingCSV(writer, []string{
		"sample_id", "cluster_id", "cluster_name", "model_id", "model_name",
		"sample_status", "health_status", "error_code", "collection_id",
		"collected_at", "persisted_at", "schema_version", "telemetry_status",
		"node_id", "engine_id", "reported_model", "model_mismatch", "engine_up",
		"engine_version", "running_requests", "waiting_requests", "token_usage",
		"throughput", "cache_usage", "request_duration_ms", "requests_value",
		"requests_semantics", "tokens_value", "tokens_semantics", "machine_up",
		"window_complete", "gpu_available", "gpu_count", "gpu_power_total_watts",
		"cpu_utilization_percent", "cpu_count", "memory_used_bytes",
		"memory_available_bytes", "memory_total_bytes", "memory_utilization_percent",
		"load_1m", "load_5m", "load_15m", "alignment_method",
		"alignment_quality", "alignment_skew_ms",
	}, func(csvWriter *csv.Writer) error {
		return prepared.forEachSample(func(row *model.ClusterTelemetryHistory, cluster LatestExportCluster, telemetry *NormalizedTelemetry) error {
			values := []string{
				strconv.FormatInt(row.ID, 10),
				strconv.FormatInt(cluster.ID, 10),
				safeCSVText(cluster.Name),
				strconv.Itoa(cluster.ModelID),
				safeCSVText(cluster.ModelName),
				safeCSVText(string(row.Status)),
				safeCSVText(string(row.HealthStatus)),
				safeCSVText(row.ErrorCode),
				safeCSVText(historyCollectionID(row)),
				unixTimestamp(row.CollectedAt),
				unixTimestamp(row.CreatedAt),
				safeCSVText(row.SchemaVersion),
			}
			if telemetry == nil {
				values = append(values, make([]string, 34)...)
				return csvWriter.Write(values)
			}
			values = append(values,
				safeCSVText(telemetry.Status),
				safeCSVText(telemetry.Identity.NodeID),
				safeCSVText(telemetry.Identity.EngineID),
				safeCSVText(telemetry.Identity.Model),
				strconv.FormatBool(telemetry.ModelMismatch),
				strconv.FormatBool(telemetry.Engine.Up),
				safeCSVText(telemetry.Engine.Version),
				optionalFloat(telemetry.Engine.RunningRequests),
				optionalFloat(telemetry.Engine.WaitingRequests),
				optionalFloat(telemetry.Engine.TokenUsage),
				optionalFloat(telemetry.Engine.Throughput),
				optionalFloat(telemetry.Engine.CacheUsage),
				formatFloat(telemetry.Engine.RequestDurationMS),
				optionalFloat(telemetry.Metrics.Requests),
				safeCSVText(metricSemantics(telemetry.Metrics.Requests, telemetry.Metrics.RequestsSemantics)),
				optionalFloat(telemetry.Metrics.Tokens),
				safeCSVText(metricSemantics(telemetry.Metrics.Tokens, telemetry.Metrics.TokensSemantics)),
				strconv.FormatBool(telemetry.Machine.Up),
				strconv.FormatBool(telemetry.Machine.WindowComplete),
				strconv.FormatBool(telemetry.Machine.GPU.Available),
				strconv.Itoa(telemetry.Machine.GPU.Count),
				optionalFloat(telemetry.Machine.GPU.PowerTotalWatts),
				optionalFloat(telemetry.Machine.System.CPUUtilizationPercent),
				optionalFloat(telemetry.Machine.System.CPUCount),
				optionalFloat(telemetry.Machine.System.MemoryUsedBytes),
				optionalFloat(telemetry.Machine.System.MemoryAvailableBytes),
				optionalFloat(telemetry.Machine.System.MemoryTotalBytes),
				optionalFloat(telemetry.Machine.System.MemoryUtilizationPercent),
				optionalFloat(telemetry.Machine.System.LoadAverage["1m"]),
				optionalFloat(telemetry.Machine.System.LoadAverage["5m"]),
				optionalFloat(telemetry.Machine.System.LoadAverage["15m"]),
				safeCSVText(telemetry.Alignment.Method),
				safeCSVText(telemetry.Alignment.Quality),
				optionalFloat(telemetry.Alignment.SkewMS),
			)
			return csvWriter.Write(values)
		})
	})
}

func (prepared *PreparedHistoryExport) writeGPUCSV(writer io.Writer) error {
	return writeStreamingCSV(writer, []string{
		"sample_id", "cluster_id", "cluster_name", "model_id", "model_name",
		"collection_id", "collected_at", "device_index", "uuid", "name",
		"power_watts", "temperature_celsius", "utilization_percent",
		"memory_utilization_percent", "memory_used_bytes", "memory_total_bytes",
		"sm_clock_mhz",
	}, func(csvWriter *csv.Writer) error {
		return prepared.forEachSample(func(row *model.ClusterTelemetryHistory, cluster LatestExportCluster, telemetry *NormalizedTelemetry) error {
			if telemetry == nil {
				return nil
			}
			for _, device := range telemetry.Machine.GPU.Devices {
				if err := csvWriter.Write([]string{
					strconv.FormatInt(row.ID, 10),
					strconv.FormatInt(cluster.ID, 10),
					safeCSVText(cluster.Name),
					strconv.Itoa(cluster.ModelID),
					safeCSVText(cluster.ModelName),
					safeCSVText(historyCollectionID(row)),
					unixTimestamp(row.CollectedAt),
					strconv.Itoa(device.Index),
					safeCSVText(device.UUID),
					safeCSVText(device.Name),
					optionalFloat(device.PowerWatts),
					optionalFloat(device.TemperatureCelsius),
					optionalFloat(device.UtilizationPercent),
					optionalFloat(device.MemoryUtilizationPercent),
					optionalFloat(device.MemoryUsedBytes),
					optionalFloat(device.MemoryTotalBytes),
					optionalFloat(device.SMClockMHz),
				}); err != nil {
					return err
				}
			}
			return nil
		})
	})
}

func (prepared *PreparedHistoryExport) writeEngineLoadsCSV(writer io.Writer) error {
	return writeStreamingCSV(writer, []string{
		"sample_id", "cluster_id", "cluster_name", "model_id", "model_name",
		"collection_id", "collected_at", "load_index", "load_json",
	}, func(csvWriter *csv.Writer) error {
		return prepared.forEachSample(func(row *model.ClusterTelemetryHistory, cluster LatestExportCluster, telemetry *NormalizedTelemetry) error {
			if telemetry == nil {
				return nil
			}
			for index, load := range telemetry.Engine.Loads {
				payload, err := common.Marshal(load)
				if err != nil {
					return err
				}
				if err := csvWriter.Write([]string{
					strconv.FormatInt(row.ID, 10),
					strconv.FormatInt(cluster.ID, 10),
					safeCSVText(cluster.Name),
					strconv.Itoa(cluster.ModelID),
					safeCSVText(cluster.ModelName),
					safeCSVText(historyCollectionID(row)),
					unixTimestamp(row.CollectedAt),
					strconv.Itoa(index),
					safeCSVText(string(payload)),
				}); err != nil {
					return err
				}
			}
			return nil
		})
	})
}

func (prepared *PreparedHistoryExport) writeJSONL(writer io.Writer) error {
	return prepared.forEachSample(func(row *model.ClusterTelemetryHistory, cluster LatestExportCluster, telemetry *NormalizedTelemetry) error {
		payload, err := common.Marshal(map[string]any{
			"sample_id":      row.ID,
			"cluster_id":     cluster.ID,
			"cluster_name":   cluster.Name,
			"model_id":       cluster.ModelID,
			"model_name":     cluster.ModelName,
			"sample_status":  row.Status,
			"health_status":  row.HealthStatus,
			"error_code":     row.ErrorCode,
			"collection_id":  row.CollectionID,
			"collected_at":   unixTimestamp(row.CollectedAt),
			"persisted_at":   unixTimestamp(row.CreatedAt),
			"schema_version": row.SchemaVersion,
			"telemetry":      telemetry,
		})
		if err != nil {
			return err
		}
		_, err = writer.Write(append(payload, '\n'))
		return err
	})
}

func (prepared *PreparedHistoryExport) forEachSample(
	visit func(*model.ClusterTelemetryHistory, LatestExportCluster, *NormalizedTelemetry) error,
) error {
	if len(prepared.clusterIDs) == 0 {
		return nil
	}
	clusterMap := make(map[int64]LatestExportCluster, len(prepared.clusters))
	for _, cluster := range prepared.clusters {
		clusterMap[cluster.ID] = cluster
	}
	var afterCollectedAt int64
	var afterID int64
	for {
		if err := prepared.context.Err(); err != nil {
			return err
		}
		rows, err := model.ListClusterTelemetryHistoryBatch(prepared.context, model.ClusterTelemetryHistoryFilter{
			ClusterIDs:       prepared.clusterIDs,
			FromInclusive:    prepared.StartAt.Unix(),
			ToExclusive:      prepared.EndAt.Unix(),
			AfterCollectedAt: afterCollectedAt,
			AfterID:          afterID,
			Limit:            historyExportBatchSize,
		})
		if err != nil {
			return err
		}
		for _, row := range rows {
			var telemetry *NormalizedTelemetry
			if row.Status == model.ClusterTelemetrySampleSuccess && row.NormalizedPayload != "" {
				telemetry = &NormalizedTelemetry{}
				if err := common.UnmarshalJsonStr(row.NormalizedPayload, telemetry); err != nil {
					return err
				}
			}
			cluster, ok := clusterMap[row.ClusterID]
			if !ok {
				continue
			}
			if err := visit(row, cluster, telemetry); err != nil {
				return err
			}
		}
		if len(rows) < historyExportBatchSize {
			return nil
		}
		last := rows[len(rows)-1]
		afterCollectedAt = last.CollectedAt
		afterID = last.ID
	}
}

func writeStreamingCSV(writer io.Writer, header []string, writeRows func(*csv.Writer) error) error {
	if _, err := writer.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return err
	}
	csvWriter := csv.NewWriter(writer)
	if err := csvWriter.Write(header); err != nil {
		return err
	}
	if err := writeRows(csvWriter); err != nil {
		return err
	}
	csvWriter.Flush()
	return csvWriter.Error()
}

func historyCollectionID(row *model.ClusterTelemetryHistory) string {
	if row.CollectionID == nil {
		return ""
	}
	return *row.CollectionID
}
