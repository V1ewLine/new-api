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

func setupClusterServiceTestDB(t *testing.T) {
	t.Helper()
	previousDB := model.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Model{}, &model.Cluster{}, &model.ClusterTelemetryLatest{}))
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
		EmptyHistoryRepository{},
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

func TestCreateClusterRejectsMissingAndDisabledModels(t *testing.T) {
	setupClusterServiceTestDB(t)
	service := testService(t, failingAgentClient{})

	_, err := service.CreateCluster(context.Background(), CreateClusterInput{
		ModelID:          999,
		Name:             "missing-model-cluster",
		AgentAddress:     "https://agent.example:9443",
		AgentBearerToken: "agent-token",
	})
	require.ErrorIs(t, err, ErrClusterModelNotFound)

	disabledModel := createTestModel(t, "disabled-model", 0)
	_, err = service.CreateCluster(context.Background(), CreateClusterInput{
		ModelID:          disabledModel.Id,
		Name:             "disabled-model-cluster",
		AgentAddress:     "https://agent.example:9443",
		AgentBearerToken: "agent-token",
	})
	require.ErrorIs(t, err, ErrClusterModelDisabled)
}

func TestCreateClusterResponseNeverContainsSecret(t *testing.T) {
	setupClusterServiceTestDB(t)
	service := testService(t, failingAgentClient{})
	linkedModel := createTestModel(t, "model-a", 1)

	response, err := service.CreateCluster(context.Background(), CreateClusterInput{
		ModelID:          linkedModel.Id,
		Name:             "cluster-a",
		AgentAddress:     "https://agent.example:9443",
		AgentBearerToken: "sensitive-agent-token",
	})
	require.NoError(t, err)
	assert.True(t, response.HasLinkSecret)

	payload, err := common.Marshal(response)
	require.NoError(t, err)
	assert.NotContains(t, string(payload), "sensitive-agent-token")
	assert.NotContains(t, string(payload), "LinkSecretCiphertext")

	storedCluster, err := model.GetClusterByID(response.ID)
	require.NoError(t, err)
	require.NotNil(t, storedCluster)
	assert.NotContains(t, storedCluster.LinkSecretCiphertext, "sensitive-agent-token")
}

func TestOverviewCombinesSearchStatusAndModelPagination(t *testing.T) {
	setupClusterServiceTestDB(t)
	service := testService(t, failingAgentClient{})
	modelA := createTestModel(t, "model-a", 1)
	modelB := createTestModel(t, "model-b", 1)
	clusters := []*model.Cluster{
		{ModelID: modelA.Id, ModelNameSnapshot: modelA.ModelName, Name: "east-online", Enabled: true, HealthStatus: model.ClusterHealthOnline},
		{ModelID: modelA.Id, ModelNameSnapshot: modelA.ModelName, Name: "west-offline", Enabled: true, HealthStatus: model.ClusterHealthOffline},
		{ModelID: modelB.Id, ModelNameSnapshot: modelB.ModelName, Name: "east-other", Enabled: true, HealthStatus: model.ClusterHealthOnline},
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
	assert.Equal(t, 1, response.Overview.TotalClusters)
	assert.Equal(t, 1, response.Pagination.Total)
	require.Len(t, response.ModelGroups, 1)
	require.Len(t, response.ModelGroups[0].Models, 1)
	assert.Equal(t, modelA.Id, response.ModelGroups[0].Models[0].ModelID)
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
}
