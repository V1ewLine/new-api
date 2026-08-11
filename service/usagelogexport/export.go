package usagelogexport

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/QuantumNous/new-api/model"
)

const maxExportRows = 50000

var (
	ErrInvalidInput = errors.New("invalid usage log export request")
	ErrTooManyRows  = errors.New("usage log export exceeds the allowed row count")
	ErrNoData       = errors.New("no usage logs found")
)

type Input struct {
	StartTimestamp    int64
	EndTimestamp      int64
	Timezone          string
	LogType           int
	ModelName         string
	Username          string
	TokenName         string
	Channel           int
	Group             string
	RequestID         string
	UpstreamRequestID string
	UserID            int
}

type PreparedExport struct {
	Filename       string
	ContentType    string
	EffectiveStart time.Time
	EffectiveEnd   time.Time
	Timezone       string
	RowCount       int
	SelfOnly       bool
	location       *time.Location
	logs           []*model.Log
}

func Prepare(input Input) (*PreparedExport, error) {
	if input.StartTimestamp <= 0 ||
		input.EndTimestamp < input.StartTimestamp ||
		input.LogType < model.LogTypeUnknown ||
		input.LogType > model.LogTypeLogin ||
		input.Channel < 0 ||
		input.UserID < 0 {
		return nil, ErrInvalidInput
	}

	timezone := strings.TrimSpace(input.Timezone)
	if timezone == "" {
		timezone = "UTC"
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, ErrInvalidInput
	}

	var logs []*model.Log
	var total int64
	if input.UserID > 0 {
		logs, total, err = model.GetUserLogs(
			input.UserID,
			input.LogType,
			input.StartTimestamp,
			input.EndTimestamp,
			strings.TrimSpace(input.ModelName),
			strings.TrimSpace(input.TokenName),
			0,
			maxExportRows+1,
			strings.TrimSpace(input.Group),
			strings.TrimSpace(input.RequestID),
			strings.TrimSpace(input.UpstreamRequestID),
		)
	} else {
		logs, total, err = model.GetAllLogs(
			input.LogType,
			input.StartTimestamp,
			input.EndTimestamp,
			strings.TrimSpace(input.ModelName),
			strings.TrimSpace(input.Username),
			strings.TrimSpace(input.TokenName),
			0,
			maxExportRows+1,
			input.Channel,
			strings.TrimSpace(input.Group),
			strings.TrimSpace(input.RequestID),
			strings.TrimSpace(input.UpstreamRequestID),
		)
	}
	if err != nil {
		return nil, err
	}
	if total > maxExportRows || len(logs) > maxExportRows {
		return nil, ErrTooManyRows
	}
	if len(logs) == 0 {
		return nil, ErrNoData
	}

	start := time.Unix(input.StartTimestamp, 0).In(location)
	end := time.Unix(input.EndTimestamp, 0).In(location)
	scope := "all"
	if input.UserID > 0 {
		scope = "self"
	}
	return &PreparedExport{
		Filename: fmt.Sprintf(
			"new-api-usage-logs_%s_%s_%s.csv",
			scope,
			start.Format("20060102T150405"),
			end.Format("20060102T150405"),
		),
		ContentType:    "text/csv; charset=utf-8",
		EffectiveStart: start,
		EffectiveEnd:   end,
		Timezone:       timezone,
		RowCount:       len(logs),
		SelfOnly:       input.UserID > 0,
		location:       location,
		logs:           logs,
	}, nil
}

func (prepared *PreparedExport) WriteTo(writer io.Writer) error {
	if prepared == nil || prepared.location == nil {
		return ErrInvalidInput
	}
	if _, err := writer.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return err
	}

	csvWriter := csv.NewWriter(writer)
	headers := []string{
		"created_at",
		"created_at_unix",
		"timezone",
		"type",
		"type_name",
		"content",
		"token_name",
		"model_name",
		"quota",
		"prompt_tokens",
		"completion_tokens",
		"total_tokens",
		"use_time_seconds",
		"is_stream",
		"group",
		"request_id",
		"upstream_request_id",
		"other",
	}
	if !prepared.SelfOnly {
		headers = append(headers,
			"user_id",
			"username",
			"token_id",
			"channel_id",
			"channel_name",
			"ip",
		)
	}
	if err := csvWriter.Write(headers); err != nil {
		return err
	}

	for _, log := range prepared.logs {
		if log == nil {
			continue
		}
		row := []string{
			time.Unix(log.CreatedAt, 0).In(prepared.location).Format(time.RFC3339),
			strconv.FormatInt(log.CreatedAt, 10),
			safeCSVText(prepared.Timezone),
			strconv.Itoa(log.Type),
			logTypeName(log.Type),
			safeCSVText(log.Content),
			safeCSVText(log.TokenName),
			safeCSVText(log.ModelName),
			strconv.Itoa(log.Quota),
			strconv.Itoa(log.PromptTokens),
			strconv.Itoa(log.CompletionTokens),
			strconv.Itoa(log.PromptTokens + log.CompletionTokens),
			strconv.Itoa(log.UseTime),
			strconv.FormatBool(log.IsStream),
			safeCSVText(log.Group),
			safeCSVText(log.RequestId),
			safeCSVText(log.UpstreamRequestId),
			safeCSVText(log.Other),
		}
		if !prepared.SelfOnly {
			row = append(row,
				strconv.Itoa(log.UserId),
				safeCSVText(log.Username),
				strconv.Itoa(log.TokenId),
				strconv.Itoa(log.ChannelId),
				safeCSVText(log.ChannelName),
				safeCSVText(log.Ip),
			)
		}
		if err := csvWriter.Write(row); err != nil {
			return err
		}
	}
	csvWriter.Flush()
	return csvWriter.Error()
}

func logTypeName(logType int) string {
	switch logType {
	case model.LogTypeTopup:
		return "topup"
	case model.LogTypeConsume:
		return "consume"
	case model.LogTypeManage:
		return "manage"
	case model.LogTypeSystem:
		return "system"
	case model.LogTypeError:
		return "error"
	case model.LogTypeRefund:
		return "refund"
	case model.LogTypeLogin:
		return "login"
	default:
		return "unknown"
	}
}

func safeCSVText(value string) string {
	trimmed := strings.TrimLeftFunc(value, unicode.IsSpace)
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
