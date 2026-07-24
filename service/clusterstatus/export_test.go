package clusterstatus

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func saveLatestTelemetryFixture(t *testing.T, cluster *model.Cluster, gpuCount int) {
	t.Helper()
	telemetry, err := (SchemaV1Adapter{}).Adapt(
		telemetryFixture(t, "ok", cluster.ModelNameSnapshot, gpuCount),
		cluster.ModelNameSnapshot,
	)
	require.NoError(t, err)
	normalized, err := common.Marshal(telemetry)
	require.NoError(t, err)
	require.NoError(t, model.DB.Create(&model.ClusterTelemetryLatest{
		ClusterID:         cluster.ID,
		SchemaVersion:     telemetry.SchemaVersion,
		CollectionID:      telemetry.CollectionID,
		RawPayload:        `{"private_raw_marker":"must-not-export"}`,
		NormalizedPayload: string(normalized),
		CollectedAt:       1784808000,
	}).Error)
}

func saveHistoryTelemetryFixture(t *testing.T, cluster *model.Cluster, collectionID string, collectedAt int64, gpuCount int) {
	t.Helper()
	telemetry, err := (SchemaV1Adapter{}).Adapt(
		telemetryFixture(t, "ok", cluster.ModelNameSnapshot, gpuCount),
		cluster.ModelNameSnapshot,
	)
	require.NoError(t, err)
	telemetry.CollectionID = collectionID
	telemetry.CollectedAt = time.Unix(collectedAt, 0).UTC().Format(time.RFC3339)
	normalized, err := common.Marshal(telemetry)
	require.NoError(t, err)
	require.NoError(t, model.DB.Create(&model.ClusterTelemetryHistory{
		ClusterID:         cluster.ID,
		CollectionID:      &collectionID,
		Status:            model.ClusterTelemetrySampleSuccess,
		HealthStatus:      model.ClusterHealthOnline,
		SchemaVersion:     telemetry.SchemaVersion,
		NormalizedPayload: string(normalized),
		CollectedAt:       collectedAt,
		CreatedAt:         collectedAt + 1,
	}).Error)
}

func readCSVExport(t *testing.T, prepared *PreparedLatestExport) [][]string {
	t.Helper()
	var buffer bytes.Buffer
	require.NoError(t, prepared.WriteTo(&buffer))
	payload := bytes.TrimPrefix(buffer.Bytes(), []byte{0xEF, 0xBB, 0xBF})
	rows, err := csv.NewReader(bytes.NewReader(payload)).ReadAll()
	require.NoError(t, err)
	return rows
}

func TestLatestModelExportUsesAllMatchingClusters(t *testing.T) {
	setupClusterServiceTestDB(t)
	service := testService(t, failingAgentClient{})
	modelA := createTestModel(t, "model-a", 1)
	modelB := createTestModel(t, "model-b", 1)
	clusters := []*model.Cluster{
		{ModelID: modelA.Id, ModelNameSnapshot: modelA.ModelName, Name: "east-a", Enabled: true, HealthStatus: model.ClusterHealthOnline, CredentialStatus: model.ClusterCredentialActive},
		{ModelID: modelA.Id, ModelNameSnapshot: modelA.ModelName, Name: "east-b", Enabled: true, HealthStatus: model.ClusterHealthOnline, CredentialStatus: model.ClusterCredentialActive},
		{ModelID: modelB.Id, ModelNameSnapshot: modelB.ModelName, Name: "east-c", Enabled: true, HealthStatus: model.ClusterHealthOnline, CredentialStatus: model.ClusterCredentialActive},
		{ModelID: modelA.Id, ModelNameSnapshot: modelA.ModelName, Name: "west-a", Enabled: true, HealthStatus: model.ClusterHealthOnline, CredentialStatus: model.ClusterCredentialActive},
	}
	for _, cluster := range clusters {
		require.NoError(t, model.CreateCluster(cluster))
	}

	prepared, err := service.PrepareLatestExport(LatestExportInput{
		Scope:   "models",
		Format:  "csv",
		Search:  "east",
		ModelID: modelA.Id,
		Health:  model.ClusterHealthOnline,
	})

	require.NoError(t, err)
	assert.Equal(t, 2, prepared.ClusterCount)
	rows := readCSVExport(t, prepared)
	require.Len(t, rows, 2)
	assert.Equal(t, "model-a", rows[1][2])
	assert.Equal(t, "2", rows[1][6])
}

func TestLatestClusterCSVLeavesMissingMetricsBlankAndEscapesFormulas(t *testing.T) {
	setupClusterServiceTestDB(t)
	service := testService(t, failingAgentClient{})
	linkedModel := createTestModel(t, "model-a", 1)
	cluster := &model.Cluster{
		ModelID:              linkedModel.Id,
		ModelNameSnapshot:    linkedModel.ModelName,
		Name:                 "=HYPERLINK(\"https://example.invalid\")",
		LinkSecretCiphertext: "encrypted-secret",
		Enabled:              true,
		HealthStatus:         model.ClusterHealthUnknown,
		CredentialStatus:     model.ClusterCredentialPending,
	}
	require.NoError(t, model.CreateCluster(cluster))

	prepared, err := service.PrepareLatestExport(LatestExportInput{
		Scope:   "clusters",
		Format:  "csv",
		ModelID: linkedModel.Id,
	})
	require.NoError(t, err)
	rows := readCSVExport(t, prepared)
	require.Len(t, rows, 2)

	values := make(map[string]string, len(rows[0]))
	for index, header := range rows[0] {
		values[header] = rows[1][index]
	}
	assert.True(t, strings.HasPrefix(values["cluster_name"], "'="))
	assert.Empty(t, values["collection_id"])
	assert.Empty(t, values["running_requests"])
	assert.Empty(t, values["requests_value"])
	assert.Empty(t, values["gpu_count"])
}

func TestLatestZIPExportContainsNormalizedFilesWithoutSecrets(t *testing.T) {
	setupClusterServiceTestDB(t)
	service := testService(t, failingAgentClient{})
	linkedModel := createTestModel(t, "model-a", 1)
	cluster := &model.Cluster{
		ModelID:              linkedModel.Id,
		ModelNameSnapshot:    linkedModel.ModelName,
		Name:                 "cluster-a",
		LinkSecretCiphertext: "top-secret-ciphertext",
		Enabled:              true,
		HealthStatus:         model.ClusterHealthOnline,
		CredentialStatus:     model.ClusterCredentialActive,
		LastFailurePayload:   "private-diagnostic-payload",
	}
	require.NoError(t, model.CreateCluster(cluster))
	saveLatestTelemetryFixture(t, cluster, 2)

	prepared, err := service.PrepareLatestExport(LatestExportInput{
		Scope:  "all",
		Format: "zip",
	})
	require.NoError(t, err)
	var buffer bytes.Buffer
	require.NoError(t, prepared.WriteTo(&buffer))

	reader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	require.NoError(t, err)
	fileNames := make([]string, 0, len(reader.File))
	var combined strings.Builder
	for _, file := range reader.File {
		fileNames = append(fileNames, file.Name)
		entry, openErr := file.Open()
		require.NoError(t, openErr)
		payload, readErr := io.ReadAll(entry)
		require.NoError(t, readErr)
		require.NoError(t, entry.Close())
		combined.Write(payload)
	}
	assert.ElementsMatch(t, []string{
		"manifest.json",
		"models.csv",
		"clusters.csv",
		"gpu_devices.csv",
		"engine_loads.csv",
		"normalized_telemetry.jsonl",
	}, fileNames)
	assert.NotContains(t, combined.String(), "top-secret-ciphertext")
	assert.NotContains(t, combined.String(), "private-diagnostic-payload")
	assert.NotContains(t, combined.String(), "must-not-export")
	assert.Contains(t, combined.String(), "gpu-a")
	assert.Contains(t, combined.String(), "gpu-b")
}

func TestLatestJSONExportLabelsLegacyMetricSemanticsAsUnknown(t *testing.T) {
	setupClusterServiceTestDB(t)
	service := testService(t, failingAgentClient{})
	linkedModel := createTestModel(t, "model-a", 1)
	cluster := &model.Cluster{
		ModelID:           linkedModel.Id,
		ModelNameSnapshot: linkedModel.ModelName,
		Name:              "cluster-a",
		Enabled:           true,
	}
	require.NoError(t, model.CreateCluster(cluster))
	requests := 12.0
	tokens := 34.0
	normalized, err := common.Marshal(NormalizedTelemetry{
		SchemaVersion: "1.0",
		CollectionID:  "legacy-collection",
		Metrics: TelemetryAggregateMetrics{
			Requests: &requests,
			Tokens:   &tokens,
		},
	})
	require.NoError(t, err)
	require.NoError(t, model.DB.Create(&model.ClusterTelemetryLatest{
		ClusterID:         cluster.ID,
		SchemaVersion:     "1.0",
		CollectionID:      "legacy-collection",
		RawPayload:        "{}",
		NormalizedPayload: string(normalized),
	}).Error)

	prepared, err := service.PrepareLatestExport(LatestExportInput{
		Scope:     "cluster",
		Format:    "json",
		ClusterID: cluster.ID,
	})
	require.NoError(t, err)
	var buffer bytes.Buffer
	require.NoError(t, prepared.WriteTo(&buffer))
	var exported latestExportData
	require.NoError(t, common.Unmarshal(buffer.Bytes(), &exported))
	require.Len(t, exported.Clusters, 1)
	require.NotNil(t, exported.Clusters[0].Telemetry)
	assert.Equal(t, "unknown", exported.Clusters[0].Telemetry.Metrics.RequestsSemantics)
	assert.Equal(t, "unknown", exported.Clusters[0].Telemetry.Metrics.TokensSemantics)
}

func TestHistoryZIPExportUsesExactWindowAndExcludesSecrets(t *testing.T) {
	setupClusterServiceTestDB(t)
	service := testService(t, failingAgentClient{})
	linkedModel := createTestModel(t, "model-a", 1)
	cluster := &model.Cluster{
		ModelID:              linkedModel.Id,
		ModelNameSnapshot:    linkedModel.ModelName,
		Name:                 "cluster-a",
		LinkSecretCiphertext: "top-secret-ciphertext",
		Enabled:              true,
		HealthStatus:         model.ClusterHealthOnline,
		CredentialStatus:     model.ClusterCredentialActive,
		LastFailurePayload:   "private-diagnostic-payload",
	}
	require.NoError(t, model.CreateCluster(cluster))
	saveHistoryTelemetryFixture(t, cluster, "inside-start", 100, 2)
	require.NoError(t, model.DB.Create(&model.ClusterTelemetryHistory{
		ClusterID:    cluster.ID,
		Status:       model.ClusterTelemetrySampleError,
		HealthStatus: model.ClusterHealthOffline,
		ErrorCode:    "AGENT_UNREACHABLE",
		CollectedAt:  150,
		CreatedAt:    151,
	}).Error)
	saveHistoryTelemetryFixture(t, cluster, "excluded-end", 200, 1)

	prepared, err := service.PrepareHistoryExport(context.Background(), HistoryExportInput{
		Scope:     "cluster",
		ClusterID: cluster.ID,
		StartAt:   time.Unix(100, 0),
		EndAt:     time.Unix(200, 0),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), prepared.SampleCount)

	var buffer bytes.Buffer
	require.NoError(t, prepared.WriteTo(&buffer))
	reader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	require.NoError(t, err)
	fileNames := make([]string, 0, len(reader.File))
	var combined strings.Builder
	var telemetryRows [][]string
	for _, file := range reader.File {
		fileNames = append(fileNames, file.Name)
		entry, openErr := file.Open()
		require.NoError(t, openErr)
		payload, readErr := io.ReadAll(entry)
		require.NoError(t, readErr)
		require.NoError(t, entry.Close())
		combined.Write(payload)
		if file.Name == "telemetry_history.csv" {
			telemetryRows, readErr = csv.NewReader(
				bytes.NewReader(bytes.TrimPrefix(payload, []byte{0xEF, 0xBB, 0xBF})),
			).ReadAll()
			require.NoError(t, readErr)
		}
	}
	assert.ElementsMatch(t, []string{
		"manifest.json",
		"clusters.csv",
		"telemetry_history.csv",
		"gpu_device_history.csv",
		"engine_load_history.csv",
		"normalized_telemetry_history.jsonl",
	}, fileNames)
	assert.Contains(t, combined.String(), "inside-start")
	assert.Contains(t, combined.String(), "AGENT_UNREACHABLE")
	assert.NotContains(t, combined.String(), "excluded-end")
	assert.NotContains(t, combined.String(), "top-secret-ciphertext")
	assert.NotContains(t, combined.String(), "private-diagnostic-payload")
	require.Len(t, telemetryRows, 3)
}

func TestHistoryExportRejectsRangeLongerThanRetention(t *testing.T) {
	setupClusterServiceTestDB(t)
	service := testService(t, failingAgentClient{})
	previousRetention := common.ClusterTelemetryRetentionDays
	common.ClusterTelemetryRetentionDays = 7
	t.Cleanup(func() {
		common.ClusterTelemetryRetentionDays = previousRetention
	})

	_, err := service.PrepareHistoryExport(context.Background(), HistoryExportInput{
		Scope:   "all",
		StartAt: time.Unix(0, 0),
		EndAt:   time.Unix(8*24*60*60, 0),
	})

	require.ErrorIs(t, err, ErrClusterExportInvalid)
}
