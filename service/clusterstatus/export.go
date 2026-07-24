package clusterstatus

import (
	"archive/zip"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const (
	clusterExportSchemaVersion = "new-api.cluster-export.v1"
	maxExportClusters          = 10000
	maxExportDetailRows        = 100000
)

var (
	ErrClusterExportInvalid  = errors.New("invalid cluster export request")
	ErrClusterExportTooLarge = errors.New("cluster export is too large")
)

type LatestExportInput struct {
	Scope     string
	Format    string
	Search    string
	ModelID   int
	ClusterID int64
	Health    model.ClusterHealthStatus
}

type LatestExportCluster struct {
	ID                  int64                         `json:"id"`
	ModelID             int                           `json:"model_id"`
	ModelName           string                        `json:"model_name"`
	ModelAvailable      bool                          `json:"model_available"`
	Name                string                        `json:"name"`
	Enabled             bool                          `json:"enabled"`
	HealthStatus        model.ClusterHealthStatus     `json:"health_status"`
	CredentialStatus    model.ClusterCredentialStatus `json:"credential_status"`
	LastPolledAt        int64                         `json:"last_polled_at"`
	LastSuccessAt       int64                         `json:"last_success_at"`
	ConsecutiveFailures int                           `json:"consecutive_failures"`
	LastErrorCode       string                        `json:"last_error_code,omitempty"`
	CreatedAt           int64                         `json:"created_at"`
	UpdatedAt           int64                         `json:"updated_at"`
	Telemetry           *NormalizedTelemetry          `json:"telemetry,omitempty"`
}

type LatestExportFilters struct {
	Search    string                    `json:"search,omitempty"`
	ModelID   int                       `json:"model_id,omitempty"`
	ClusterID int64                     `json:"cluster_id,omitempty"`
	Status    model.ClusterHealthStatus `json:"status,omitempty"`
}

type latestExportData struct {
	SchemaVersion string                `json:"schema_version"`
	ExportedAt    string                `json:"exported_at"`
	Scope         string                `json:"scope"`
	Filters       LatestExportFilters   `json:"filters"`
	Models        []ModelClusterSummary `json:"models,omitempty"`
	Clusters      []LatestExportCluster `json:"clusters,omitempty"`
}

type PreparedLatestExport struct {
	Filename     string
	ContentType  string
	ClusterCount int
	Scope        string
	Format       string
	data         latestExportData
	format       string
}

func (service *Service) PrepareLatestExport(input LatestExportInput) (*PreparedLatestExport, error) {
	input.Scope = strings.ToLower(strings.TrimSpace(input.Scope))
	input.Format = strings.ToLower(strings.TrimSpace(input.Format))
	input.Search = strings.TrimSpace(input.Search)
	if !validLatestExportCombination(input.Scope, input.Format) {
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
	telemetryMap, err := model.GetLatestClusterTelemetryMap(clusterIDs)
	if err != nil {
		return nil, err
	}
	modelMap, err := loadClusterModels(modelIDs)
	if err != nil {
		return nil, err
	}

	exportClusters := make([]LatestExportCluster, 0, len(clusters))
	detailRows := 0
	for _, cluster := range clusters {
		telemetry := decodeLatestTelemetry(telemetryMap[cluster.ID])
		response := service.clusterResponse(cluster, modelMap[cluster.ModelID], telemetry)
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
			Telemetry:           response.Telemetry,
		})
		if telemetry != nil {
			detailRows += len(telemetry.Machine.GPU.Devices) + len(telemetry.Engine.Loads)
		}
	}
	if detailRows > maxExportDetailRows {
		return nil, ErrClusterExportTooLarge
	}

	modelSummaries := summarizeExportModels(clusters, telemetryMap, modelMap)
	now := time.Now().UTC()
	data := latestExportData{
		SchemaVersion: clusterExportSchemaVersion,
		ExportedAt:    now.Format(time.RFC3339),
		Scope:         input.Scope,
		Filters: LatestExportFilters{
			Search:    input.Search,
			ModelID:   input.ModelID,
			ClusterID: input.ClusterID,
			Status:    input.Health,
		},
	}
	switch input.Scope {
	case "models":
		data.Models = modelSummaries
	case "clusters":
		data.Clusters = exportClusters
	case "cluster":
		data.Models = modelSummaries
		data.Clusters = exportClusters
	default:
		data.Models = modelSummaries
		data.Clusters = exportClusters
	}

	timestamp := now.Format("20060102T150405Z")
	return &PreparedLatestExport{
		Filename:     fmt.Sprintf("cluster-status-%s-%s.%s", input.Scope, timestamp, input.Format),
		ContentType:  exportContentType(input.Format),
		ClusterCount: len(clusters),
		Scope:        input.Scope,
		Format:       input.Format,
		data:         data,
		format:       input.Format,
	}, nil
}

func (prepared *PreparedLatestExport) WriteTo(writer io.Writer) error {
	if prepared == nil {
		return ErrClusterExportInvalid
	}
	switch prepared.format {
	case "csv":
		switch prepared.data.Scope {
		case "models":
			return writeModelsCSV(writer, prepared.data.ExportedAt, prepared.data.Models)
		case "clusters":
			return writeClustersCSV(writer, prepared.data.ExportedAt, prepared.data.Clusters)
		default:
			return ErrClusterExportInvalid
		}
	case "json":
		payload, err := common.Marshal(prepared.data)
		if err != nil {
			return err
		}
		_, err = writer.Write(payload)
		return err
	case "zip":
		return prepared.writeZIP(writer)
	default:
		return ErrClusterExportInvalid
	}
}

func (prepared *PreparedLatestExport) writeZIP(writer io.Writer) error {
	archive := zip.NewWriter(writer)
	files := []struct {
		name  string
		write func(io.Writer) error
	}{
		{
			name: "manifest.json",
			write: func(entry io.Writer) error {
				manifest := map[string]any{
					"schema_version": prepared.data.SchemaVersion,
					"exported_at":    prepared.data.ExportedAt,
					"scope":          prepared.data.Scope,
					"filters":        prepared.data.Filters,
					"cluster_count":  len(prepared.data.Clusters),
					"model_count":    len(prepared.data.Models),
					"files": []string{
						"manifest.json",
						"models.csv",
						"clusters.csv",
						"gpu_devices.csv",
						"engine_loads.csv",
						"normalized_telemetry.jsonl",
					},
				}
				payload, err := common.Marshal(manifest)
				if err != nil {
					return err
				}
				_, err = entry.Write(payload)
				return err
			},
		},
		{name: "models.csv", write: func(entry io.Writer) error {
			return writeModelsCSV(entry, prepared.data.ExportedAt, prepared.data.Models)
		}},
		{name: "clusters.csv", write: func(entry io.Writer) error {
			return writeClustersCSV(entry, prepared.data.ExportedAt, prepared.data.Clusters)
		}},
		{name: "gpu_devices.csv", write: func(entry io.Writer) error {
			return writeGPUDevicesCSV(entry, prepared.data.ExportedAt, prepared.data.Clusters)
		}},
		{name: "engine_loads.csv", write: func(entry io.Writer) error {
			return writeEngineLoadsCSV(entry, prepared.data.ExportedAt, prepared.data.Clusters)
		}},
		{name: "normalized_telemetry.jsonl", write: func(entry io.Writer) error {
			return writeNormalizedTelemetryJSONL(entry, prepared.data.Clusters)
		}},
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

func validLatestExportCombination(scope string, format string) bool {
	switch scope {
	case "models", "clusters":
		return format == "csv" || format == "json"
	case "cluster", "all":
		return format == "zip" || format == "json"
	default:
		return false
	}
}

func exportContentType(format string) string {
	switch format {
	case "csv":
		return "text/csv; charset=utf-8"
	case "zip":
		return "application/zip"
	default:
		return "application/json; charset=utf-8"
	}
}

func summarizeExportModels(
	clusters []*model.Cluster,
	telemetryMap map[int64]*model.ClusterTelemetryLatest,
	modelMap map[int]*model.Model,
) []ModelClusterSummary {
	summaryByModel := make(map[int]*ModelClusterSummary)
	gpuTotalByModel := make(map[int]float64)
	gpuCountByModel := make(map[int]int)
	for _, cluster := range clusters {
		linkedModel := modelMap[cluster.ModelID]
		modelName := cluster.ModelNameSnapshot
		available := false
		icon := ""
		category := "other"
		if linkedModel != nil {
			modelName = linkedModel.ModelName
			available = linkedModel.Status == 1 && !linkedModel.DeletedAt.Valid
			icon = linkedModel.Icon
			category = modelCategory(linkedModel)
		}
		summary := summaryByModel[cluster.ModelID]
		if summary == nil {
			summary = &ModelClusterSummary{
				ModelID:        cluster.ModelID,
				ModelName:      modelName,
				Icon:           icon,
				Type:           category,
				ModelAvailable: available,
				HealthStatus:   model.ClusterHealthUnknown,
			}
			summaryByModel[cluster.ModelID] = summary
		}
		summary.ClusterCount++
		if cluster.CredentialStatus == model.ClusterCredentialActive {
			if cluster.HealthStatus == model.ClusterHealthOnline {
				summary.OnlineCount++
			}
			if cluster.HealthStatus == model.ClusterHealthPartial ||
				cluster.HealthStatus == model.ClusterHealthAbnormal ||
				cluster.HealthStatus == model.ClusterHealthOffline {
				summary.AbnormalCount++
			}
			summary.HealthStatus = mergeHealthStatus(summary.HealthStatus, cluster.HealthStatus)
		}
		telemetry := decodeLatestTelemetry(telemetryMap[cluster.ID])
		if telemetry == nil {
			continue
		}
		if telemetry.Metrics.Requests != nil {
			summary.TotalRequests += *telemetry.Metrics.Requests
			summary.RequestsAvailable = true
		}
		if telemetry.Metrics.Tokens != nil {
			summary.TotalTokens += *telemetry.Metrics.Tokens
			summary.TokensAvailable = true
		}
		for _, device := range telemetry.Machine.GPU.Devices {
			if device.UtilizationPercent != nil {
				gpuTotalByModel[cluster.ModelID] += *device.UtilizationPercent
				gpuCountByModel[cluster.ModelID]++
			}
		}
	}

	summaries := make([]ModelClusterSummary, 0, len(summaryByModel))
	for modelID, summary := range summaryByModel {
		if count := gpuCountByModel[modelID]; count > 0 {
			average := gpuTotalByModel[modelID] / float64(count)
			summary.GPUUtilization = &average
		}
		summaries = append(summaries, *summary)
	}
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Type == summaries[j].Type {
			return strings.ToLower(summaries[i].ModelName) < strings.ToLower(summaries[j].ModelName)
		}
		return modelTypeOrder(summaries[i].Type) < modelTypeOrder(summaries[j].Type)
	})
	return summaries
}

func writeModelsCSV(writer io.Writer, exportedAt string, models []ModelClusterSummary) error {
	rows := [][]string{{
		"exported_at", "model_id", "model_name", "model_type", "model_available",
		"health_status", "cluster_count", "online_count", "abnormal_count",
		"total_requests", "requests_available", "total_tokens", "tokens_available",
		"gpu_utilization_percent",
	}}
	for _, summary := range models {
		rows = append(rows, []string{
			exportedAt,
			strconv.Itoa(summary.ModelID),
			safeCSVText(summary.ModelName),
			safeCSVText(summary.Type),
			strconv.FormatBool(summary.ModelAvailable),
			safeCSVText(string(summary.HealthStatus)),
			strconv.Itoa(summary.ClusterCount),
			strconv.Itoa(summary.OnlineCount),
			strconv.Itoa(summary.AbnormalCount),
			optionalAggregate(summary.TotalRequests, summary.RequestsAvailable),
			strconv.FormatBool(summary.RequestsAvailable),
			optionalAggregate(summary.TotalTokens, summary.TokensAvailable),
			strconv.FormatBool(summary.TokensAvailable),
			optionalFloat(summary.GPUUtilization),
		})
	}
	return writeCSVRows(writer, rows)
}

func writeClustersCSV(writer io.Writer, exportedAt string, clusters []LatestExportCluster) error {
	rows := [][]string{{
		"exported_at", "cluster_id", "cluster_name", "model_id", "model_name",
		"model_available", "enabled", "credential_status", "health_status",
		"last_polled_at", "last_success_at", "consecutive_failures", "last_error_code",
		"collection_id", "collected_at", "telemetry_status", "node_id", "engine_id",
		"reported_model", "model_mismatch", "engine_up", "engine_version",
		"running_requests", "waiting_requests", "token_usage", "throughput",
		"cache_usage", "request_duration_ms", "requests_value", "requests_semantics",
		"tokens_value", "tokens_semantics", "machine_up", "window_complete",
		"gpu_available", "gpu_count", "gpu_power_total_watts",
		"cpu_utilization_percent", "cpu_count", "memory_used_bytes",
		"memory_available_bytes", "memory_total_bytes", "memory_utilization_percent",
		"load_1m", "load_5m", "load_15m", "alignment_method", "alignment_quality",
		"alignment_skew_ms",
	}}
	for _, cluster := range clusters {
		row := []string{
			exportedAt,
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
		}
		if cluster.Telemetry == nil {
			row = append(row, make([]string, 36)...)
			rows = append(rows, row)
			continue
		}
		telemetry := cluster.Telemetry
		row = append(row,
			safeCSVText(telemetry.CollectionID),
			safeCSVText(telemetry.CollectedAt),
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
		rows = append(rows, row)
	}
	return writeCSVRows(writer, rows)
}

func writeGPUDevicesCSV(writer io.Writer, exportedAt string, clusters []LatestExportCluster) error {
	rows := [][]string{{
		"exported_at", "cluster_id", "cluster_name", "model_id", "model_name",
		"collection_id", "device_index", "uuid", "name", "power_watts",
		"temperature_celsius", "utilization_percent", "memory_utilization_percent",
		"memory_used_bytes", "memory_total_bytes", "sm_clock_mhz",
	}}
	for _, cluster := range clusters {
		if cluster.Telemetry == nil {
			continue
		}
		for _, device := range cluster.Telemetry.Machine.GPU.Devices {
			rows = append(rows, []string{
				exportedAt,
				strconv.FormatInt(cluster.ID, 10),
				safeCSVText(cluster.Name),
				strconv.Itoa(cluster.ModelID),
				safeCSVText(cluster.ModelName),
				safeCSVText(cluster.Telemetry.CollectionID),
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
			})
		}
	}
	return writeCSVRows(writer, rows)
}

func writeEngineLoadsCSV(writer io.Writer, exportedAt string, clusters []LatestExportCluster) error {
	rows := [][]string{{
		"exported_at", "cluster_id", "cluster_name", "model_id", "model_name",
		"collection_id", "load_index", "load_json",
	}}
	for _, cluster := range clusters {
		if cluster.Telemetry == nil {
			continue
		}
		for index, load := range cluster.Telemetry.Engine.Loads {
			payload, err := common.Marshal(load)
			if err != nil {
				return err
			}
			rows = append(rows, []string{
				exportedAt,
				strconv.FormatInt(cluster.ID, 10),
				safeCSVText(cluster.Name),
				strconv.Itoa(cluster.ModelID),
				safeCSVText(cluster.ModelName),
				safeCSVText(cluster.Telemetry.CollectionID),
				strconv.Itoa(index),
				safeCSVText(string(payload)),
			})
		}
	}
	return writeCSVRows(writer, rows)
}

func writeNormalizedTelemetryJSONL(writer io.Writer, clusters []LatestExportCluster) error {
	for _, cluster := range clusters {
		if cluster.Telemetry == nil {
			continue
		}
		payload, err := common.Marshal(map[string]any{
			"cluster_id":   cluster.ID,
			"cluster_name": cluster.Name,
			"model_id":     cluster.ModelID,
			"model_name":   cluster.ModelName,
			"telemetry":    cluster.Telemetry,
		})
		if err != nil {
			return err
		}
		if _, err := writer.Write(append(payload, '\n')); err != nil {
			return err
		}
	}
	return nil
}

func writeCSVRows(writer io.Writer, rows [][]string) error {
	if _, err := writer.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return err
	}
	csvWriter := csv.NewWriter(writer)
	csvWriter.WriteAll(rows)
	return csvWriter.Error()
}

func safeCSVText(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed == "" {
		return value
	}
	switch trimmed[0] {
	case '=', '+', '-', '@':
		return "'" + value
	default:
		return value
	}
}

func optionalAggregate(value float64, available bool) string {
	if !available {
		return ""
	}
	return formatFloat(value)
}

func optionalFloat(value *float64) string {
	if value == nil {
		return ""
	}
	return formatFloat(*value)
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func unixTimestamp(value int64) string {
	if value <= 0 {
		return ""
	}
	return time.Unix(value, 0).UTC().Format(time.RFC3339)
}

func metricSemantics(value *float64, semantics string) string {
	if value == nil {
		return "unavailable"
	}
	if strings.TrimSpace(semantics) == "" {
		return "unknown"
	}
	return semantics
}
