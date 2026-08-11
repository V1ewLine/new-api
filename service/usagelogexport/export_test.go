package usagelogexport

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupUsageLogExportTestDB(t *testing.T) {
	t.Helper()

	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	previousLogDB := model.LOG_DB
	model.LOG_DB = database
	t.Cleanup(func() {
		model.LOG_DB = previousLogDB
	})
	require.NoError(t, database.AutoMigrate(&model.Log{}))
}

func TestPrepareSelfExportUsesAuthenticatedUserScopeAndSanitizesOther(t *testing.T) {
	setupUsageLogExportTestDB(t)
	createdAt := time.Date(2026, 8, 11, 9, 30, 0, 0, time.UTC).Unix()
	require.NoError(t, model.LOG_DB.Create([]model.Log{
		{
			UserId: 1, CreatedAt: createdAt, Type: model.LogTypeConsume,
			ModelName: "deepseek-v4-flash", PromptTokens: 10, CompletionTokens: 2,
			Other: `{"admin_info":{"use_channel":[3]},"cache_tokens":4}`,
		},
		{
			UserId: 2, CreatedAt: createdAt, Type: model.LogTypeConsume,
			ModelName: "other-user-model",
		},
	}).Error)

	prepared, err := Prepare(Input{
		StartTimestamp: createdAt - 60,
		EndTimestamp:   createdAt + 60,
		Timezone:       "Asia/Shanghai",
		UserID:         1,
	})
	require.NoError(t, err)
	assert.True(t, prepared.SelfOnly)
	assert.Equal(t, 1, prepared.RowCount)

	records := writeAndReadCSV(t, prepared)
	require.Len(t, records, 2)
	assert.NotContains(t, records[0], "username")
	assert.NotContains(t, records[0], "channel_id")
	assert.Equal(t, "deepseek-v4-flash", records[1][7])
	assert.NotContains(t, records[1][17], "admin_info")
	assert.Contains(t, records[1][17], "cache_tokens")
}

func TestPrepareAdminExportAppliesFiltersAndIncludesAdministrativeColumns(t *testing.T) {
	setupUsageLogExportTestDB(t)
	createdAt := time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC).Unix()
	require.NoError(t, model.LOG_DB.Create([]model.Log{
		{
			UserId: 1, Username: "alice", CreatedAt: createdAt,
			Type: model.LogTypeConsume, ModelName: "=formula-model",
			Ip: "127.0.0.1",
		},
		{
			UserId: 2, Username: "bob", CreatedAt: createdAt,
			Type: model.LogTypeError, ModelName: "ignored-model",
		},
	}).Error)

	prepared, err := Prepare(Input{
		StartTimestamp: createdAt - 60,
		EndTimestamp:   createdAt + 60,
		Timezone:       "UTC",
		LogType:        model.LogTypeConsume,
		Username:       "alice",
	})
	require.NoError(t, err)
	assert.False(t, prepared.SelfOnly)
	assert.Equal(t, 1, prepared.RowCount)

	records := writeAndReadCSV(t, prepared)
	require.Len(t, records, 2)
	assert.Contains(t, records[0], "username")
	assert.Contains(t, records[0], "channel_id")
	assert.Equal(t, "'=formula-model", records[1][7])
	assert.Equal(t, "alice", records[1][19])
}

func TestPrepareRejectsInvalidRangeAndEmptyResult(t *testing.T) {
	setupUsageLogExportTestDB(t)

	_, err := Prepare(Input{StartTimestamp: 100, EndTimestamp: 99, Timezone: "UTC"})
	require.ErrorIs(t, err, ErrInvalidInput)

	_, err = Prepare(Input{StartTimestamp: 100, EndTimestamp: 200, Timezone: "UTC"})
	require.ErrorIs(t, err, ErrNoData)
}

func writeAndReadCSV(t *testing.T, prepared *PreparedExport) [][]string {
	t.Helper()
	var output bytes.Buffer
	require.NoError(t, prepared.WriteTo(&output))
	require.True(t, bytes.HasPrefix(output.Bytes(), []byte{0xEF, 0xBB, 0xBF}))
	reader := csv.NewReader(strings.NewReader(string(bytes.TrimPrefix(
		output.Bytes(),
		[]byte{0xEF, 0xBB, 0xBF},
	))))
	records, err := reader.ReadAll()
	require.NoError(t, err)
	return records
}
