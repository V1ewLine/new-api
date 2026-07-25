package clusterstatus

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type failingAgentClient struct{}

func (failingAgentClient) Fetch(_ context.Context, _ ResolvedAgentConnection) ([]byte, error) {
	return nil, &AgentClientError{Code: "AGENT_UNREACHABLE"}
}

type successfulAgentClient struct {
	payload []byte
}

func (client successfulAgentClient) Fetch(_ context.Context, _ ResolvedAgentConnection) ([]byte, error) {
	return client.payload, nil
}

func setupClusterServiceTestDB(t *testing.T) {
	t.Helper()
	previousDB := model.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.Model{},
		&model.Cluster{},
		&model.ClusterTelemetryLatest{},
		&model.ClusterTelemetryHistory{},
	))
	model.DB = db
	t.Cleanup(func() {
		model.DB = previousDB
	})
}

func testService(t *testing.T, client TelemetryAgentClient) *Service {
	t.Helper()
	protector, err := NewAESGCMSecretProtector("test-crypto-secret")
	require.NoError(t, err)
	return NewService(
		TemporaryLinkResolver{},
		protector,
		client,
		SchemaV1Adapter{},
		NewDefaultHealthEvaluator(2),
		GORMHistoryRepository{},
		func(string) error { return nil },
		PollConfig{
			Interval:         time.Second,
			RequestTimeout:   100 * time.Millisecond,
			MaxConcurrency:   1,
			FailureThreshold: 2,
			MaxBodyBytes:     1024,
			LeaseTTL:         time.Second,
			MaxBackoff:       8 * time.Second,
		},
	)
}

func createTestModel(t *testing.T, name string, status int) *model.Model {
	t.Helper()
	linkedModel := &model.Model{ModelName: name, Status: 1}
	require.NoError(t, model.DB.Create(linkedModel).Error)
	if status != 1 {
		require.NoError(t, model.DB.Model(linkedModel).Update("status", status).Error)
		linkedModel.Status = status
	}
	return linkedModel
}

func TestInitializeClusterCredentialStatusesMigratesLegacyClusters(t *testing.T) {
	setupClusterServiceTestDB(t)
	verifiedCluster := &model.Cluster{
		ModelID:              1,
		ModelNameSnapshot:    "model-a",
		Name:                 "verified-legacy-cluster",
		LinkSecretCiphertext: "ciphertext-a",
		Enabled:              true,
		CreatedAt:            100,
	}
	pendingCluster := &model.Cluster{
		ModelID:              1,
		ModelNameSnapshot:    "model-a",
		Name:                 "pending-legacy-cluster",
		LinkSecretCiphertext: "ciphertext-b",
		Enabled:              true,
		CreatedAt:            300,
	}
	require.NoError(t, model.DB.Create(verifiedCluster).Error)
	require.NoError(t, model.DB.Create(pendingCluster).Error)
	require.NoError(t, model.DB.Model(&model.Cluster{}).
		Where("id = ?", verifiedCluster.ID).
		Updates(map[string]any{
			"credential_status":      "",
			"credential_version":     0,
			"credential_issued_at":   0,
			"credential_verified_at": 0,
			"last_success_at":        200,
			"health_status":          model.ClusterHealthOnline,
		}).Error)
	require.NoError(t, model.DB.Model(&model.Cluster{}).
		Where("id = ?", pendingCluster.ID).
		Updates(map[string]any{
			"credential_status":      "",
			"credential_version":     0,
			"credential_issued_at":   0,
			"credential_verified_at": 0,
			"last_success_at":        0,
			"health_status":          model.ClusterHealthOffline,
		}).Error)

	require.NoError(t, model.InitializeClusterCredentialStatuses())
	require.NoError(t, model.InitializeClusterCredentialStatuses())

	var migratedVerified model.Cluster
	require.NoError(t, model.DB.First(&migratedVerified, verifiedCluster.ID).Error)
	assert.Equal(t, model.ClusterCredentialActive, migratedVerified.CredentialStatus)
	assert.Equal(t, 1, migratedVerified.CredentialVersion)
	assert.Equal(t, int64(100), migratedVerified.CredentialIssuedAt)
	assert.Equal(t, int64(200), migratedVerified.CredentialVerifiedAt)
	assert.Equal(t, model.ClusterHealthOnline, migratedVerified.HealthStatus)

	var migratedPending model.Cluster
	require.NoError(t, model.DB.First(&migratedPending, pendingCluster.ID).Error)
	assert.Equal(t, model.ClusterCredentialPending, migratedPending.CredentialStatus)
	assert.Equal(t, 1, migratedPending.CredentialVersion)
	assert.Equal(t, int64(300), migratedPending.CredentialIssuedAt)
	assert.Zero(t, migratedPending.CredentialVerifiedAt)
	assert.Equal(t, model.ClusterHealthUnknown, migratedPending.HealthStatus)
}

func TestCreateClusterRejectsMissingAndDisabledModels(t *testing.T) {
	setupClusterServiceTestDB(t)
	service := testService(t, failingAgentClient{})

	_, err := service.CreateCluster(context.Background(), CreateClusterInput{
		ModelID:      999,
		Name:         "missing-model-cluster",
		AgentAddress: "https://agent.example:9443",
	})
	require.ErrorIs(t, err, ErrClusterModelNotFound)

	disabledModel := createTestModel(t, "disabled-model", 0)
	_, err = service.CreateCluster(context.Background(), CreateClusterInput{
		ModelID:      disabledModel.Id,
		Name:         "disabled-model-cluster",
		AgentAddress: "https://agent.example:9443",
	})
	require.ErrorIs(t, err, ErrClusterModelDisabled)
}

func TestCreateClusterIssuesTokenOnceAndStoresOnlyCiphertext(t *testing.T) {
	setupClusterServiceTestDB(t)
	service := testService(t, failingAgentClient{})
	linkedModel := createTestModel(t, "model-a", 1)

	response, err := service.CreateCluster(context.Background(), CreateClusterInput{
		ModelID:      linkedModel.Id,
		Name:         "cluster-a",
		AgentAddress: "https://agent.example:9443",
	})
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(response.BootstrapToken, agentBearerTokenPrefix))
	assert.True(t, response.Cluster.HasLinkSecret)
	assert.Equal(t, model.ClusterCredentialPending, response.Cluster.CredentialStatus)
	assert.Equal(t, 1, response.Cluster.CredentialVersion)

	payload, err := common.Marshal(response.Cluster)
	require.NoError(t, err)
	assert.NotContains(t, string(payload), response.BootstrapToken)
	assert.NotContains(t, string(payload), "LinkSecretCiphertext")

	storedCluster, err := model.GetClusterByID(response.Cluster.ID)
	require.NoError(t, err)
	require.NotNil(t, storedCluster)
	assert.NotContains(t, storedCluster.LinkSecretCiphertext, response.BootstrapToken)

	fetched, err := service.GetCluster(response.Cluster.ID)
	require.NoError(t, err)
	fetchedPayload, err := common.Marshal(fetched)
	require.NoError(t, err)
	assert.NotContains(t, string(fetchedPayload), response.BootstrapToken)
}

func TestRotateCredentialReplacesTokenAndReturnsPendingCredential(t *testing.T) {
	setupClusterServiceTestDB(t)
	service := testService(t, failingAgentClient{})
	linkedModel := createTestModel(t, "model-a", 1)
	created, err := service.CreateCluster(context.Background(), CreateClusterInput{
		ModelID:      linkedModel.Id,
		Name:         "cluster-a",
		AgentAddress: "https://agent.example:9443",
	})
	require.NoError(t, err)

	rotated, err := service.RotateCredential(context.Background(), created.Cluster.ID)

	require.NoError(t, err)
	assert.NotEqual(t, created.BootstrapToken, rotated.BootstrapToken)
	assert.Equal(t, model.ClusterCredentialPending, rotated.Cluster.CredentialStatus)
	assert.Equal(t, 2, rotated.Cluster.CredentialVersion)
	stored, err := model.GetClusterByID(created.Cluster.ID)
	require.NoError(t, err)
	require.NotNil(t, stored)
	linkSecret, err := service.protector.Unprotect(stored.LinkSecretCiphertext)
	require.NoError(t, err)
	connection, err := service.resolver.Resolve(context.Background(), linkSecret)
	require.NoError(t, err)
	assert.Equal(t, rotated.BootstrapToken, connection.BearerToken)
}

func TestVerifyCredentialReturnsFailureCodeWithoutExposingToken(t *testing.T) {
	setupClusterServiceTestDB(t)
	service := testService(t, failingAgentClient{})
	linkedModel := createTestModel(t, "model-a", 1)
	created, err := service.CreateCluster(context.Background(), CreateClusterInput{
		ModelID:      linkedModel.Id,
		Name:         "cluster-a",
		AgentAddress: "https://agent.example:9443",
	})
	require.NoError(t, err)

	verification, err := service.VerifyCredential(context.Background(), created.Cluster.ID)

	require.NoError(t, err)
	assert.False(t, verification.Verified)
	assert.Equal(t, "AGENT_UNREACHABLE", verification.ErrorCode)
	assert.Equal(t, model.ClusterCredentialPending, verification.Cluster.CredentialStatus)
	assert.Equal(t, model.ClusterHealthUnknown, verification.Cluster.HealthStatus)
	payload, err := common.Marshal(verification)
	require.NoError(t, err)
	assert.NotContains(t, string(payload), created.BootstrapToken)
}

func TestVerifyCredentialActivatesCredentialAfterSuccessfulTelemetry(t *testing.T) {
	setupClusterServiceTestDB(t)
	service := testService(t, successfulAgentClient{
		payload: telemetryFixture(t, "ok", "model-a", 1),
	})
	linkedModel := createTestModel(t, "model-a", 1)
	created, err := service.CreateCluster(context.Background(), CreateClusterInput{
		ModelID:      linkedModel.Id,
		Name:         "cluster-a",
		AgentAddress: "https://agent.example:9443",
	})
	require.NoError(t, err)

	verification, err := service.VerifyCredential(context.Background(), created.Cluster.ID)

	require.NoError(t, err)
	assert.True(t, verification.Verified)
	assert.Empty(t, verification.ErrorCode)
	assert.Equal(t, model.ClusterCredentialActive, verification.Cluster.CredentialStatus)
	assert.Greater(t, verification.Cluster.CredentialVerifiedAt, int64(0))
	var historyRows []model.ClusterTelemetryHistory
	require.NoError(t, model.DB.Where("cluster_id = ?", created.Cluster.ID).Find(&historyRows).Error)
	require.Len(t, historyRows, 1)
	assert.Equal(t, model.ClusterTelemetrySampleSuccess, historyRows[0].Status)
	assert.NotEmpty(t, historyRows[0].NormalizedPayload)
}

func TestDeleteClusterRemovesConfigurationLatestAndHistoryTelemetry(t *testing.T) {
	setupClusterServiceTestDB(t)
	service := testService(t, failingAgentClient{})
	linkedModel := createTestModel(t, "model-a", 1)
	cluster := &model.Cluster{
		ModelID:              linkedModel.Id,
		ModelNameSnapshot:    linkedModel.ModelName,
		Name:                 "cluster-a",
		LinkSecretCiphertext: "encrypted-secret",
		Enabled:              true,
	}
	require.NoError(t, model.CreateCluster(cluster))
	require.NoError(t, model.DB.Create(&model.ClusterTelemetryLatest{
		ClusterID:         cluster.ID,
		SchemaVersion:     "1.0",
		CollectionID:      "collection-a",
		RawPayload:        "{}",
		NormalizedPayload: "{}",
	}).Error)
	collectionID := "history-collection-a"
	require.NoError(t, model.DB.Create(&model.ClusterTelemetryHistory{
		ClusterID:         cluster.ID,
		CollectionID:      &collectionID,
		Status:            model.ClusterTelemetrySampleSuccess,
		HealthStatus:      model.ClusterHealthOnline,
		SchemaVersion:     "1.0",
		NormalizedPayload: "{}",
		CollectedAt:       100,
		CreatedAt:         100,
	}).Error)

	require.NoError(t, service.DeleteCluster(cluster.ID))

	storedCluster, err := model.GetClusterByID(cluster.ID)
	require.NoError(t, err)
	assert.Nil(t, storedCluster)
	telemetry, err := model.GetLatestClusterTelemetry(cluster.ID)
	require.NoError(t, err)
	assert.Nil(t, telemetry)
	var historyCount int64
	require.NoError(t, model.DB.Model(&model.ClusterTelemetryHistory{}).
		Where("cluster_id = ?", cluster.ID).
		Count(&historyCount).Error)
	assert.Zero(t, historyCount)
	require.ErrorIs(t, service.DeleteCluster(cluster.ID), ErrClusterNotFound)
}

func TestOverviewCombinesSearchStatusAndModelPagination(t *testing.T) {
	setupClusterServiceTestDB(t)
	service := testService(t, failingAgentClient{})
	modelA := createTestModel(t, "model-a", 1)
	modelB := createTestModel(t, "model-b", 1)
	clusters := []*model.Cluster{
		{ModelID: modelA.Id, ModelNameSnapshot: modelA.ModelName, Name: "east-online", Enabled: true, HealthStatus: model.ClusterHealthOnline, CredentialStatus: model.ClusterCredentialActive},
		{ModelID: modelA.Id, ModelNameSnapshot: modelA.ModelName, Name: "west-offline", Enabled: true, HealthStatus: model.ClusterHealthOffline, CredentialStatus: model.ClusterCredentialActive},
		{ModelID: modelB.Id, ModelNameSnapshot: modelB.ModelName, Name: "east-other", Enabled: true, HealthStatus: model.ClusterHealthOnline, CredentialStatus: model.ClusterCredentialActive},
	}
	for _, cluster := range clusters {
		require.NoError(t, model.CreateCluster(cluster))
	}

	response, err := service.GetOverview(
		"east",
		modelA.Id,
		model.ClusterHealthOnline,
		1,
		1,
	)

	require.NoError(t, err)
	assert.Equal(t, 3, response.Overview.TotalClusters)
	assert.Equal(t, 1, response.Pagination.Total)
	require.Len(t, response.ModelGroups, 1)
	require.Len(t, response.ModelGroups[0].Models, 1)
	assert.Equal(t, modelA.Id, response.ModelGroups[0].Models[0].ModelID)
}

func TestOverviewCurrentLoadIsGlobalFreshAndUsesEngineMetrics(t *testing.T) {
	setupClusterServiceTestDB(t)
	service := testService(t, failingAgentClient{})
	modelA := createTestModel(t, "model-a", 1)
	modelB := createTestModel(t, "model-b", 1)
	now := common.GetTimestamp()
	clusters := []*model.Cluster{
		{
			ModelID: modelA.Id, ModelNameSnapshot: modelA.ModelName, Name: "fresh-a",
			Enabled: true, HealthStatus: model.ClusterHealthOnline,
			CredentialStatus: model.ClusterCredentialActive, LastSuccessAt: now,
		},
		{
			ModelID: modelB.Id, ModelNameSnapshot: modelB.ModelName, Name: "fresh-b",
			Enabled: true, HealthStatus: model.ClusterHealthOnline,
			CredentialStatus: model.ClusterCredentialActive, LastSuccessAt: now,
		},
		{
			ModelID: modelA.Id, ModelNameSnapshot: modelA.ModelName, Name: "stale",
			Enabled: true, HealthStatus: model.ClusterHealthOnline,
			CredentialStatus: model.ClusterCredentialActive, LastSuccessAt: now - 60,
		},
		{
			ModelID: modelA.Id, ModelNameSnapshot: modelA.ModelName, Name: "pending",
			Enabled: true, HealthStatus: model.ClusterHealthUnknown,
			CredentialStatus: model.ClusterCredentialPending, LastSuccessAt: now,
		},
	}
	for _, cluster := range clusters {
		require.NoError(t, model.CreateCluster(cluster))
	}
	createLatestTelemetry(t, clusters[0], 2, 1, 30, 9000, 8000)
	createLatestTelemetry(t, clusters[1], 4, 3, 70, 7000, 6000)
	createLatestTelemetry(t, clusters[2], 100, 100, 1000, 1, 1)
	createLatestTelemetry(t, clusters[3], 100, 100, 1000, 1, 1)

	response, err := service.GetOverview("fresh-a", modelA.Id, model.ClusterHealthOnline, 1, 10)

	require.NoError(t, err)
	assert.Equal(t, 3, response.Overview.MonitoredClusters)
	assert.Equal(t, 2, response.Overview.RequestsReportingClusters)
	assert.Equal(t, 2, response.Overview.TokensReportingClusters)
	assert.Equal(t, float64(10), response.Overview.CurrentRequests)
	assert.Equal(t, float64(100), response.Overview.CurrentTokenUsage)
	assert.Equal(t, response.Overview.CurrentRequests, response.Overview.TotalRequests)
	require.Len(t, response.ModelGroups, 1)
	modelSummary := response.ModelGroups[0].Models[0]
	assert.Equal(t, float64(3), modelSummary.CurrentRequests)
	assert.Equal(t, float64(30), modelSummary.CurrentTokenUsage)
	assert.Equal(t, 1, modelSummary.MonitoredClusters)
}

func TestModelDetailCurrentLoadUsesOnlySelectedModelClusters(t *testing.T) {
	setupClusterServiceTestDB(t)
	service := testService(t, failingAgentClient{})
	modelA := createTestModel(t, "model-a", 1)
	modelB := createTestModel(t, "model-b", 1)
	now := common.GetTimestamp()
	clusterA := &model.Cluster{
		ModelID: modelA.Id, ModelNameSnapshot: modelA.ModelName, Name: "a",
		Enabled: true, HealthStatus: model.ClusterHealthOnline,
		CredentialStatus: model.ClusterCredentialActive, LastSuccessAt: now,
	}
	clusterB := &model.Cluster{
		ModelID: modelB.Id, ModelNameSnapshot: modelB.ModelName, Name: "b",
		Enabled: true, HealthStatus: model.ClusterHealthOnline,
		CredentialStatus: model.ClusterCredentialActive, LastSuccessAt: now,
	}
	require.NoError(t, model.CreateCluster(clusterA))
	require.NoError(t, model.CreateCluster(clusterB))
	createLatestTelemetry(t, clusterA, 5, 2, 40, 5000, 4000)
	createLatestTelemetry(t, clusterB, 20, 10, 200, 1, 1)

	response, err := service.GetModelDetail(modelA.Id)

	require.NoError(t, err)
	assert.Equal(t, float64(7), response.Summary.CurrentRequests)
	assert.Equal(t, float64(40), response.Summary.CurrentTokenUsage)
	assert.Equal(t, 1, response.Summary.MonitoredClusters)
	assert.Equal(t, 1, response.Summary.RequestsReportingClusters)
	assert.Equal(t, 1, response.Summary.TokensReportingClusters)
}

func createLatestTelemetry(
	t *testing.T,
	cluster *model.Cluster,
	running float64,
	waiting float64,
	tokenUsage float64,
	legacyRequests float64,
	legacyTokens float64,
) {
	t.Helper()
	telemetry := NormalizedTelemetry{
		Engine: TelemetryEngine{
			RunningRequests: &running,
			WaitingRequests: &waiting,
			TokenUsage:      &tokenUsage,
		},
		Metrics: TelemetryAggregateMetrics{
			Requests: &legacyRequests,
			Tokens:   &legacyTokens,
		},
	}
	payload, err := common.Marshal(telemetry)
	require.NoError(t, err)
	require.NoError(t, model.DB.Create(&model.ClusterTelemetryLatest{
		ClusterID:         cluster.ID,
		SchemaVersion:     "1.0",
		CollectionID:      cluster.Name,
		NormalizedPayload: string(payload),
		CollectedAt:       cluster.LastSuccessAt,
		UpdatedAt:         cluster.LastSuccessAt,
	}).Error)
}

func TestOverviewDoesNotTreatPendingCredentialAsOperationalAlert(t *testing.T) {
	setupClusterServiceTestDB(t)
	service := testService(t, failingAgentClient{})
	linkedModel := createTestModel(t, "model-a", 1)
	cluster := &model.Cluster{
		ModelID:           linkedModel.Id,
		ModelNameSnapshot: linkedModel.ModelName,
		Name:              "pending-cluster",
		Enabled:           true,
		HealthStatus:      model.ClusterHealthOffline,
		CredentialStatus:  model.ClusterCredentialPending,
		LastErrorCode:     "AGENT_HTTP_401",
	}
	require.NoError(t, model.CreateCluster(cluster))

	response, err := service.GetOverview("", 0, "", 1, 10)

	require.NoError(t, err)
	assert.Equal(t, 1, response.Overview.TotalClusters)
	assert.Equal(t, 0, response.Overview.AbnormalClusters)
	assert.Empty(t, response.Alerts)
	require.Len(t, response.ModelGroups, 1)
	require.Len(t, response.ModelGroups[0].Models, 1)
	assert.Equal(t, model.ClusterHealthUnknown, response.ModelGroups[0].Models[0].HealthStatus)
}

func TestOverviewAlertIncludesPollingFreshnessAndFailureCount(t *testing.T) {
	setupClusterServiceTestDB(t)
	service := testService(t, failingAgentClient{})
	linkedModel := createTestModel(t, "model-a", 1)
	cluster := &model.Cluster{
		ModelID:             linkedModel.Id,
		ModelNameSnapshot:   linkedModel.ModelName,
		Name:                "failed-cluster",
		Enabled:             true,
		HealthStatus:        model.ClusterHealthOffline,
		CredentialStatus:    model.ClusterCredentialActive,
		LastErrorCode:       "AGENT_TIMEOUT",
		LastPolledAt:        300,
		LastSuccessAt:       200,
		ConsecutiveFailures: 3,
	}
	require.NoError(t, model.CreateCluster(cluster))

	response, err := service.GetOverview("", 0, "", 1, 10)

	require.NoError(t, err)
	require.Len(t, response.Alerts, 1)
	assert.Equal(t, int64(300), response.Alerts[0].LastPolledAt)
	assert.Equal(t, int64(200), response.Alerts[0].LastSuccessAt)
	assert.Equal(t, 3, response.Alerts[0].ConsecutiveFailures)
}

func TestPollClusterPersistsConsecutiveFailuresAndBackoff(t *testing.T) {
	setupClusterServiceTestDB(t)
	service := testService(t, failingAgentClient{})
	linkedModel := createTestModel(t, "model-a", 1)
	protector := service.protector
	linkSecret := testLinkSecret(t, "https://agent.example", "agent-token")
	ciphertext, err := protector.Protect(linkSecret)
	require.NoError(t, err)
	cluster := &model.Cluster{
		ModelID:              linkedModel.Id,
		ModelNameSnapshot:    linkedModel.ModelName,
		Name:                 "cluster-a",
		LinkSecretCiphertext: ciphertext,
		Enabled:              true,
		CredentialStatus:     model.ClusterCredentialActive,
	}
	require.NoError(t, model.CreateCluster(cluster))

	firstErr := service.PollCluster(context.Background(), cluster.ID, "runner-1", true)
	var firstFailure *PollFailureError
	require.ErrorAs(t, firstErr, &firstFailure)
	assert.Equal(t, "AGENT_UNREACHABLE", firstFailure.Code)
	first, err := model.GetClusterByID(cluster.ID)
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.Equal(t, 1, first.ConsecutiveFailures)
	assert.Equal(t, model.ClusterHealthAbnormal, first.HealthStatus)

	secondErr := service.PollCluster(context.Background(), cluster.ID, "runner-2", true)
	var secondFailure *PollFailureError
	require.True(t, errors.As(secondErr, &secondFailure))
	second, err := model.GetClusterByID(cluster.ID)
	require.NoError(t, err)
	require.NotNil(t, second)
	assert.Equal(t, 2, second.ConsecutiveFailures)
	assert.Equal(t, model.ClusterHealthOffline, second.HealthStatus)
	assert.GreaterOrEqual(t, second.NextPollAt, first.NextPollAt)
	assert.False(t, strings.Contains(secondErr.Error(), "agent-token"))
	var historyRows []model.ClusterTelemetryHistory
	require.NoError(t, model.DB.Where("cluster_id = ?", cluster.ID).
		Order("id ASC").
		Find(&historyRows).Error)
	require.Len(t, historyRows, 2)
	assert.Equal(t, model.ClusterTelemetrySampleError, historyRows[0].Status)
	assert.Equal(t, "AGENT_UNREACHABLE", historyRows[0].ErrorCode)
	assert.Empty(t, historyRows[0].NormalizedPayload)
	assert.Equal(t, model.ClusterHealthOffline, historyRows[1].HealthStatus)
}
