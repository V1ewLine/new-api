package model

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ResponsesCapabilityMode string

const (
	ResponsesCapabilityModeUnknown          ResponsesCapabilityMode = "unknown"
	ResponsesCapabilityModeNative           ResponsesCapabilityMode = "native"
	ResponsesCapabilityModeNativeTextCompat ResponsesCapabilityMode = "native_text_compat"
	ResponsesCapabilityModeChatCompletions  ResponsesCapabilityMode = "chat_completions"
)

type ChannelResponsesCapability struct {
	Id                  int                     `json:"id"`
	ChannelId           int                     `json:"channel_id" gorm:"uniqueIndex:idx_channel_responses_capability"`
	Model               string                  `json:"model" gorm:"type:text"`
	ModelHash           string                  `json:"-" gorm:"type:varchar(64);uniqueIndex:idx_channel_responses_capability"`
	NonStreamMode       ResponsesCapabilityMode `json:"non_stream_mode" gorm:"type:varchar(32)"`
	StreamMode          ResponsesCapabilityMode `json:"stream_mode" gorm:"type:varchar(32)"`
	NonStreamDetectedAt int64                   `json:"non_stream_detected_at" gorm:"bigint"`
	StreamDetectedAt    int64                   `json:"stream_detected_at" gorm:"bigint"`
	NonStreamLastError  string                  `json:"non_stream_last_error" gorm:"type:text"`
	StreamLastError     string                  `json:"stream_last_error" gorm:"type:text"`
	UpdatedAt           int64                   `json:"updated_at" gorm:"bigint"`
}

func GetChannelResponsesCapability(channelId int, modelName string) (*ChannelResponsesCapability, error) {
	if DB == nil {
		return nil, gorm.ErrInvalidDB
	}
	capability := &ChannelResponsesCapability{}
	modelName = strings.TrimSpace(modelName)
	err := DB.Where("channel_id = ? AND model_hash = ?", channelId, responsesCapabilityModelHash(modelName)).
		First(capability).Error
	if err != nil {
		return nil, err
	}
	return capability, nil
}

func ListChannelResponsesCapabilities(channelId int) ([]ChannelResponsesCapability, error) {
	if DB == nil {
		return nil, gorm.ErrInvalidDB
	}
	var capabilities []ChannelResponsesCapability
	err := DB.Where("channel_id = ?", channelId).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "model"}}).
		Find(&capabilities).Error
	return capabilities, err
}

func SaveChannelResponsesCapabilityMode(
	channelId int,
	modelName string,
	isStream bool,
	mode ResponsesCapabilityMode,
	lastError string,
) error {
	if DB == nil {
		return gorm.ErrInvalidDB
	}
	modelName = strings.TrimSpace(modelName)
	if channelId <= 0 || modelName == "" {
		return errors.New("channel id and model are required")
	}
	if mode != ResponsesCapabilityModeNative &&
		mode != ResponsesCapabilityModeNativeTextCompat &&
		mode != ResponsesCapabilityModeChatCompletions {
		mode = ResponsesCapabilityModeUnknown
	}
	if len(lastError) > 1024 {
		lastError = lastError[:1024]
	}

	now := common.GetTimestamp()
	capability := ChannelResponsesCapability{
		ChannelId:     channelId,
		Model:         modelName,
		ModelHash:     responsesCapabilityModelHash(modelName),
		NonStreamMode: ResponsesCapabilityModeUnknown,
		StreamMode:    ResponsesCapabilityModeUnknown,
		UpdatedAt:     now,
	}
	updateColumns := []string{"updated_at"}
	if isStream {
		capability.StreamMode = mode
		capability.StreamDetectedAt = now
		capability.StreamLastError = lastError
		updateColumns = append(updateColumns, "stream_mode", "stream_detected_at", "stream_last_error")
	} else {
		capability.NonStreamMode = mode
		capability.NonStreamDetectedAt = now
		capability.NonStreamLastError = lastError
		updateColumns = append(updateColumns, "non_stream_mode", "non_stream_detected_at", "non_stream_last_error")
	}

	return DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "channel_id"},
			{Name: "model_hash"},
		},
		DoUpdates: clause.AssignmentColumns(append(updateColumns, "model")),
	}).Create(&capability).Error
}

func DeleteChannelResponsesCapabilities(channelIds []int) error {
	if len(channelIds) == 0 || DB == nil {
		return nil
	}
	return DB.Where("channel_id IN ?", channelIds).Delete(&ChannelResponsesCapability{}).Error
}

func responsesCapabilityModelHash(modelName string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(modelName)))
	return hex.EncodeToString(sum[:])
}
