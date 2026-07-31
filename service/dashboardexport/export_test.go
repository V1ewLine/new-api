package dashboardexport

import (
	"bytes"
	"context"
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

func setupDashboardExportTestDB(t *testing.T) {
	t.Helper()

	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	previousDB := model.DB
	model.DB = database
	t.Cleanup(func() {
		model.DB = previousDB
	})
	require.NoError(
		t,
		database.Table("quota_data").AutoMigrate(&model.QuotaData{}),
	)
}

func TestPrepareAggregatesHourlyDataByModelAndNaturalDay(t *testing.T) {
	setupDashboardExportTestDB(t)

	firstHour := time.Date(2026, 7, 27, 2, 0, 0, 0, time.UTC).Unix()
	secondHour := time.Date(2026, 7, 27, 3, 0, 0, 0, time.UTC).Unix()
	require.NoError(t, model.DB.Table("quota_data").Create([]model.QuotaData{
		{
			UserID: 1, Username: "alice", ModelName: "glm-4.7-flash",
			CreatedAt: firstHour, Count: 3, TokenUsed: 300, Quota: 30,
		},
		{
			UserID: 2, Username: "bob", ModelName: "glm-4.7-flash",
			CreatedAt: secondHour, Count: 2, TokenUsed: 220, Quota: 22,
		},
		{
			UserID: 1, Username: "alice", ModelName: "deepseek-v4-flash",
			CreatedAt: firstHour, Count: 5, TokenUsed: 700, Quota: 70,
		},
	}).Error)

	prepared, err := Prepare(context.Background(), Input{
		StartTimestamp: time.Date(
			2026, 7, 27, 8, 30, 0, 0, time.FixedZone("UTC+8", 8*3600),
		).Unix(),
		EndTimestamp: time.Date(
			2026, 7, 27, 12, 15, 0, 0, time.FixedZone("UTC+8", 8*3600),
		).Unix(),
		Granularity: GranularityDay,
		Timezone:    "Asia/Shanghai",
	})
	require.NoError(t, err)
	require.Len(t, prepared.rows, 2)

	assert.Equal(t, "deepseek-v4-flash", prepared.rows[0].ModelName)
	assert.Equal(t, int64(5), prepared.rows[0].RequestCount)
	assert.Equal(t, int64(700), prepared.rows[0].TokenUsed)
	assert.Equal(t, "glm-4.7-flash", prepared.rows[1].ModelName)
	assert.Equal(t, int64(5), prepared.rows[1].RequestCount)
	assert.Equal(t, int64(520), prepared.rows[1].TokenUsed)
	assert.Equal(
		t,
		time.Date(2026, 7, 27, 0, 0, 0, 0, prepared.rows[0].BucketStart.Location()),
		prepared.rows[0].BucketStart,
	)
	assert.Equal(t, 2, prepared.ModelCount)
	assert.Equal(t, 2, prepared.RowCount)
}

func TestNormalizeInputRejectsInvalidAndOversizedRanges(t *testing.T) {
	_, err := normalizeInput(Input{
		StartTimestamp: 100,
		EndTimestamp:   100,
		Granularity:    GranularityHour,
		Timezone:       "UTC",
	})
	require.ErrorIs(t, err, ErrInvalidInput)

	_, err = normalizeInput(Input{
		StartTimestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		EndTimestamp:   time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC).Unix(),
		Granularity:    GranularityHour,
		Timezone:       "UTC",
	})
	require.ErrorIs(t, err, ErrRangeTooLarge)
}

func TestPreparedExportWritesBOMAndProtectsSpreadsheetFormulas(t *testing.T) {
	bucketStart := time.Date(2026, 7, 27, 8, 0, 0, 0, time.UTC)
	prepared := &PreparedExport{
		Granularity: GranularityHour,
		Timezone:    "UTC",
		rows: []ExportRow{
			{
				BucketStart:  bucketStart,
				BucketEnd:    bucketStart.Add(time.Hour),
				ModelName:    "=dangerous-model",
				RequestCount: 2,
				TokenUsed:    20,
				Quota:        10,
				RPM:          0.5,
				TPM:          5,
			},
		},
	}

	var output bytes.Buffer
	require.NoError(t, prepared.WriteTo(&output))
	require.True(t, bytes.HasPrefix(output.Bytes(), []byte{0xEF, 0xBB, 0xBF}))

	reader := csv.NewReader(
		strings.NewReader(string(bytes.TrimPrefix(
			output.Bytes(),
			[]byte{0xEF, 0xBB, 0xBF},
		))),
	)
	records, err := reader.ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 2)
	assert.Equal(t, "'=dangerous-model", records[1][4])
	assert.Equal(t, "2", records[1][5])
	assert.Equal(t, "20", records[1][6])
}
