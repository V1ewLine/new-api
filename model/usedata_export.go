package model

import (
	"context"
	"sort"
	"strings"

	"gorm.io/gorm"
)

const maxModelAnalyticsExportOptions = 10000

type ModelAnalyticsExportFilter struct {
	StartTime int64
	EndTime   int64
	UserID    int
	Username  string
	ModelName string
}

type ModelAnalyticsHourlyData struct {
	ModelName string
	CreatedAt int64
	Count     int64
	TokenUsed int64
	Quota     int64
}

func ListModelAnalyticsExportModels(
	ctx context.Context,
	filter ModelAnalyticsExportFilter,
) ([]string, error) {
	modelNames := make([]string, 0)
	query := applyModelAnalyticsExportFilter(
		DB.WithContext(ctx).Table("quota_data").Distinct("model_name"),
		filter,
	)
	if err := query.
		Order("model_name ASC").
		Limit(maxModelAnalyticsExportOptions+1).
		Pluck("model_name", &modelNames).Error; err != nil {
		return nil, err
	}

	uniqueNames := make(map[string]struct{}, len(modelNames))
	normalizedNames := make([]string, 0, len(modelNames))
	for _, modelName := range modelNames {
		modelName = strings.TrimSpace(modelName)
		if modelName == "" {
			continue
		}
		if _, exists := uniqueNames[modelName]; exists {
			continue
		}
		uniqueNames[modelName] = struct{}{}
		normalizedNames = append(normalizedNames, modelName)
	}
	sort.Slice(normalizedNames, func(i, j int) bool {
		return strings.ToLower(normalizedNames[i]) < strings.ToLower(normalizedNames[j])
	})
	return normalizedNames, nil
}

func IterateModelAnalyticsHourlyData(
	ctx context.Context,
	filter ModelAnalyticsExportFilter,
	visit func(ModelAnalyticsHourlyData) error,
) error {
	query := applyModelAnalyticsExportFilter(
		DB.WithContext(ctx).Table("quota_data").
			Select(
				"model_name, created_at, sum(count) as count, "+
					"sum(token_used) as token_used, sum(quota) as quota",
			),
		filter,
	).
		Group("model_name, created_at").
		Order("created_at ASC, model_name ASC")

	rows, err := query.Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var row ModelAnalyticsHourlyData
		if err := rows.Scan(
			&row.ModelName,
			&row.CreatedAt,
			&row.Count,
			&row.TokenUsed,
			&row.Quota,
		); err != nil {
			return err
		}
		if err := visit(row); err != nil {
			return err
		}
	}
	return rows.Err()
}

func applyModelAnalyticsExportFilter(
	query *gorm.DB,
	filter ModelAnalyticsExportFilter,
) *gorm.DB {
	filtered := query.Where(
		"created_at >= ? and created_at < ?",
		filter.StartTime,
		filter.EndTime,
	)
	if filter.UserID > 0 {
		filtered = filtered.Where("user_id = ?", filter.UserID)
	}
	if username := strings.TrimSpace(filter.Username); username != "" {
		filtered = filtered.Where("username = ?", username)
	}
	if modelName := strings.TrimSpace(filter.ModelName); modelName != "" {
		filtered = filtered.Where("model_name = ?", modelName)
	}
	return filtered
}
