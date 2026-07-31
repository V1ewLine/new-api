package dashboardexport

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/QuantumNous/new-api/model"
)

const (
	maxExportRows         = 200000
	maxExportModelOptions = 10000
)

var (
	ErrInvalidInput  = errors.New("invalid model analytics export request")
	ErrRangeTooLarge = errors.New("model analytics export range is too large")
	ErrTooManyRows   = errors.New("model analytics export exceeds the allowed row count")
	ErrNoData        = errors.New("no model analytics data found")
)

type Granularity string

const (
	GranularityHour Granularity = "hour"
	GranularityDay  Granularity = "day"
	GranularityWeek Granularity = "week"
)

type Input struct {
	StartTimestamp int64
	EndTimestamp   int64
	Granularity    Granularity
	Timezone       string
	ModelName      string
	UserID         int
	Username       string
}

type ModelOptionInput struct {
	StartTimestamp int64
	EndTimestamp   int64
	Granularity    Granularity
	Timezone       string
	UserID         int
	Username       string
}

type ExportRow struct {
	BucketStart  time.Time
	BucketEnd    time.Time
	ModelName    string
	RequestCount int64
	TokenUsed    int64
	Quota        int64
	RPM          float64
	TPM          float64
}

type PreparedExport struct {
	Filename       string
	ContentType    string
	Granularity    Granularity
	Timezone       string
	ModelName      string
	EffectiveStart time.Time
	EffectiveEnd   time.Time
	RowCount       int
	ModelCount     int
	rows           []ExportRow
}

type normalizedInput struct {
	location       *time.Location
	granularity    Granularity
	timezone       string
	modelName      string
	username       string
	effectiveStart time.Time
	effectiveEnd   time.Time
	userID         int
}

type aggregateKey struct {
	bucketStart int64
	modelName   string
}

type aggregateValue struct {
	requestCount int64
	tokenUsed    int64
	quota        int64
}

func ListModelOptions(
	ctx context.Context,
	input ModelOptionInput,
) ([]string, error) {
	normalized, err := normalizeInput(Input{
		StartTimestamp: input.StartTimestamp,
		EndTimestamp:   input.EndTimestamp,
		Granularity:    input.Granularity,
		Timezone:       input.Timezone,
		UserID:         input.UserID,
		Username:       input.Username,
	})
	if err != nil {
		return nil, err
	}

	options, err := model.ListModelAnalyticsExportModels(
		ctx,
		model.ModelAnalyticsExportFilter{
			StartTime: normalized.effectiveStart.Unix(),
			EndTime:   normalized.effectiveEnd.Unix(),
			UserID:    normalized.userID,
			Username:  normalized.username,
		},
	)
	if err != nil {
		return nil, err
	}
	if len(options) > maxExportModelOptions {
		return nil, ErrTooManyRows
	}
	return options, nil
}

func Prepare(ctx context.Context, input Input) (*PreparedExport, error) {
	normalized, err := normalizeInput(input)
	if err != nil {
		return nil, err
	}

	aggregates := make(map[aggregateKey]aggregateValue)
	err = model.IterateModelAnalyticsHourlyData(
		ctx,
		model.ModelAnalyticsExportFilter{
			StartTime: normalized.effectiveStart.Unix(),
			EndTime:   normalized.effectiveEnd.Unix(),
			UserID:    normalized.userID,
			Username:  normalized.username,
			ModelName: normalized.modelName,
		},
		func(hourly model.ModelAnalyticsHourlyData) error {
			modelName := strings.TrimSpace(hourly.ModelName)
			if modelName == "" {
				modelName = "Unknown"
			}
			bucketStart := floorTime(
				time.Unix(hourly.CreatedAt, 0).In(normalized.location),
				normalized.granularity,
			)
			key := aggregateKey{
				bucketStart: bucketStart.Unix(),
				modelName:   modelName,
			}
			value := aggregates[key]
			value.requestCount += hourly.Count
			value.tokenUsed += hourly.TokenUsed
			value.quota += hourly.Quota
			aggregates[key] = value
			if len(aggregates) > maxExportRows {
				return ErrTooManyRows
			}
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	if len(aggregates) == 0 {
		return nil, ErrNoData
	}

	rows := make([]ExportRow, 0, len(aggregates))
	modelNames := make(map[string]struct{})
	for key, value := range aggregates {
		bucketStart := time.Unix(key.bucketStart, 0).In(normalized.location)
		bucketEnd := addGranularity(bucketStart, normalized.granularity)
		minutes := bucketEnd.Sub(bucketStart).Minutes()
		if minutes <= 0 {
			return nil, ErrInvalidInput
		}
		rows = append(rows, ExportRow{
			BucketStart:  bucketStart,
			BucketEnd:    bucketEnd,
			ModelName:    key.modelName,
			RequestCount: value.requestCount,
			TokenUsed:    value.tokenUsed,
			Quota:        value.quota,
			RPM:          float64(value.requestCount) / minutes,
			TPM:          float64(value.tokenUsed) / minutes,
		})
		modelNames[key.modelName] = struct{}{}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].BucketStart.Equal(rows[j].BucketStart) {
			return strings.ToLower(rows[i].ModelName) <
				strings.ToLower(rows[j].ModelName)
		}
		return rows[i].BucketStart.Before(rows[j].BucketStart)
	})

	filenameModel := normalized.modelName
	if filenameModel == "" {
		filenameModel = "all-models"
	}
	return &PreparedExport{
		Filename: fmt.Sprintf(
			"new-api-model-analytics_%s_%s_%s_%s.csv",
			safeFilenamePart(filenameModel),
			normalized.granularity,
			normalized.effectiveStart.Format("20060102T150405"),
			normalized.effectiveEnd.Format("20060102T150405"),
		),
		ContentType:    "text/csv; charset=utf-8",
		Granularity:    normalized.granularity,
		Timezone:       normalized.timezone,
		ModelName:      normalized.modelName,
		EffectiveStart: normalized.effectiveStart,
		EffectiveEnd:   normalized.effectiveEnd,
		RowCount:       len(rows),
		ModelCount:     len(modelNames),
		rows:           rows,
	}, nil
}

func (prepared *PreparedExport) WriteTo(writer io.Writer) error {
	if _, err := writer.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return err
	}
	csvWriter := csv.NewWriter(writer)
	if err := csvWriter.Write([]string{
		"bucket_start",
		"bucket_end",
		"timezone",
		"granularity",
		"model_name",
		"request_count",
		"token_used",
		"quota",
		"rpm",
		"tpm",
	}); err != nil {
		return err
	}
	for _, row := range prepared.rows {
		if err := csvWriter.Write([]string{
			row.BucketStart.Format(time.RFC3339),
			row.BucketEnd.Format(time.RFC3339),
			safeCSVText(prepared.Timezone),
			string(prepared.Granularity),
			safeCSVText(row.ModelName),
			strconv.FormatInt(row.RequestCount, 10),
			strconv.FormatInt(row.TokenUsed, 10),
			strconv.FormatInt(row.Quota, 10),
			strconv.FormatFloat(row.RPM, 'f', 6, 64),
			strconv.FormatFloat(row.TPM, 'f', 6, 64),
		}); err != nil {
			return err
		}
	}
	csvWriter.Flush()
	return csvWriter.Error()
}

func normalizeInput(input Input) (normalizedInput, error) {
	if input.StartTimestamp <= 0 ||
		input.EndTimestamp <= input.StartTimestamp ||
		input.UserID < 0 {
		return normalizedInput{}, ErrInvalidInput
	}

	granularity := input.Granularity
	if granularity == "" {
		granularity = GranularityHour
	}
	switch granularity {
	case GranularityHour, GranularityDay, GranularityWeek:
	default:
		return normalizedInput{}, ErrInvalidInput
	}

	timezone := strings.TrimSpace(input.Timezone)
	if timezone == "" {
		timezone = "UTC"
	}
	if len(timezone) > 100 {
		return normalizedInput{}, ErrInvalidInput
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return normalizedInput{}, ErrInvalidInput
	}

	start := time.Unix(input.StartTimestamp, 0).In(location)
	end := time.Unix(input.EndTimestamp, 0).In(location)
	effectiveStart := floorTime(start, granularity)
	effectiveEnd := floorTime(end, granularity)
	if effectiveEnd.Before(end) {
		effectiveEnd = addGranularity(effectiveEnd, granularity)
	}
	if !effectiveStart.Before(effectiveEnd) {
		return normalizedInput{}, ErrInvalidInput
	}

	maxRange := 90 * 24 * time.Hour
	switch granularity {
	case GranularityDay:
		maxRange = 2 * 366 * 24 * time.Hour
	case GranularityWeek:
		maxRange = 5 * 366 * 24 * time.Hour
	}
	if effectiveEnd.Sub(effectiveStart) > maxRange {
		return normalizedInput{}, ErrRangeTooLarge
	}

	modelName := strings.TrimSpace(input.ModelName)
	username := strings.TrimSpace(input.Username)
	if len(modelName) > 255 || len(username) > 64 {
		return normalizedInput{}, ErrInvalidInput
	}

	return normalizedInput{
		location:       location,
		granularity:    granularity,
		timezone:       timezone,
		modelName:      modelName,
		username:       username,
		effectiveStart: effectiveStart,
		effectiveEnd:   effectiveEnd,
		userID:         input.UserID,
	}, nil
}

func floorTime(value time.Time, granularity Granularity) time.Time {
	switch granularity {
	case GranularityDay:
		return time.Date(
			value.Year(),
			value.Month(),
			value.Day(),
			0,
			0,
			0,
			0,
			value.Location(),
		)
	case GranularityWeek:
		dayStart := time.Date(
			value.Year(),
			value.Month(),
			value.Day(),
			0,
			0,
			0,
			0,
			value.Location(),
		)
		daysSinceMonday := (int(dayStart.Weekday()) + 6) % 7
		return dayStart.AddDate(0, 0, -daysSinceMonday)
	default:
		return time.Date(
			value.Year(),
			value.Month(),
			value.Day(),
			value.Hour(),
			0,
			0,
			0,
			value.Location(),
		)
	}
}

func addGranularity(value time.Time, granularity Granularity) time.Time {
	switch granularity {
	case GranularityDay:
		return value.AddDate(0, 0, 1)
	case GranularityWeek:
		return value.AddDate(0, 0, 7)
	default:
		return value.Add(time.Hour)
	}
}

func safeCSVText(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed == "" || !strings.ContainsRune("=+-@", rune(trimmed[0])) {
		return value
	}
	return "'" + value
}

func safeFilenamePart(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))
	lastWasDash := false
	for _, character := range strings.ToLower(value) {
		if character <= unicode.MaxASCII &&
			(unicode.IsLetter(character) || unicode.IsDigit(character)) {
			builder.WriteRune(character)
			lastWasDash = false
		} else if !lastWasDash {
			builder.WriteByte('-')
			lastWasDash = true
		}
		if builder.Len() >= 80 {
			break
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "model"
	}
	return result
}
