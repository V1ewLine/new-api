package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ClusterHealthStatus string

const (
	ClusterHealthUnknown  ClusterHealthStatus = "unknown"
	ClusterHealthOnline   ClusterHealthStatus = "online"
	ClusterHealthPartial  ClusterHealthStatus = "partial"
	ClusterHealthAbnormal ClusterHealthStatus = "abnormal"
	ClusterHealthOffline  ClusterHealthStatus = "offline"
)

type ClusterCredentialStatus string

const (
	ClusterCredentialPending ClusterCredentialStatus = "pending"
	ClusterCredentialActive  ClusterCredentialStatus = "active"
)

type Cluster struct {
	ID                   int64                   `json:"id" gorm:"primaryKey"`
	ModelID              int                     `json:"model_id" gorm:"index;not null"`
	ModelNameSnapshot    string                  `json:"model_name_snapshot" gorm:"type:varchar(128);not null"`
	Name                 string                  `json:"name" gorm:"type:varchar(128);not null;index"`
	LinkSecretCiphertext string                  `json:"-" gorm:"type:text;not null"`
	CredentialStatus     ClusterCredentialStatus `json:"credential_status" gorm:"type:varchar(32);index"`
	CredentialVersion    int                     `json:"credential_version"`
	CredentialIssuedAt   int64                   `json:"credential_issued_at" gorm:"bigint"`
	CredentialVerifiedAt int64                   `json:"credential_verified_at" gorm:"bigint"`
	Enabled              bool                    `json:"enabled" gorm:"not null"`
	HealthStatus         ClusterHealthStatus     `json:"health_status" gorm:"type:varchar(32);not null;index"`
	LastPolledAt         int64                   `json:"last_polled_at" gorm:"bigint"`
	LastSuccessAt        int64                   `json:"last_success_at" gorm:"bigint"`
	ConsecutiveFailures  int                     `json:"consecutive_failures"`
	LastErrorCode        string                  `json:"last_error_code,omitempty" gorm:"type:varchar(64)"`
	LastFailurePayload   string                  `json:"-" gorm:"type:text"`
	NextPollAt           int64                   `json:"-" gorm:"bigint;index"`
	PollLockedBy         string                  `json:"-" gorm:"type:varchar(128);index"`
	PollLockedUntil      int64                   `json:"-" gorm:"bigint;index"`
	CreatedAt            int64                   `json:"created_at" gorm:"bigint;index"`
	UpdatedAt            int64                   `json:"updated_at" gorm:"bigint;index"`
}

type ClusterTelemetryLatest struct {
	ClusterID         int64  `json:"cluster_id" gorm:"primaryKey"`
	SchemaVersion     string `json:"schema_version" gorm:"type:varchar(32);not null"`
	CollectionID      string `json:"collection_id" gorm:"type:varchar(64);index"`
	RawPayload        string `json:"-" gorm:"type:text;not null"`
	NormalizedPayload string `json:"-" gorm:"type:text;not null"`
	CollectedAt       int64  `json:"collected_at" gorm:"bigint;index"`
	UpdatedAt         int64  `json:"updated_at" gorm:"bigint;index"`
}

func (ClusterTelemetryLatest) TableName() string {
	return "cluster_telemetry_latest"
}

type ClusterListFilter struct {
	Search      string
	ModelID     int
	Health      ClusterHealthStatus
	EnabledOnly bool
}

func (cluster *Cluster) BeforeCreate(_ *gorm.DB) error {
	now := common.GetTimestamp()
	if cluster.CredentialStatus == "" {
		cluster.CredentialStatus = ClusterCredentialPending
	}
	if cluster.CredentialVersion <= 0 {
		cluster.CredentialVersion = 1
	}
	if cluster.CredentialIssuedAt == 0 {
		cluster.CredentialIssuedAt = now
	}
	if cluster.HealthStatus == "" {
		cluster.HealthStatus = ClusterHealthUnknown
	}
	if cluster.NextPollAt == 0 {
		cluster.NextPollAt = now
	}
	if cluster.CreatedAt == 0 {
		cluster.CreatedAt = now
	}
	cluster.UpdatedAt = now
	return nil
}

func CreateCluster(cluster *Cluster) error {
	if cluster == nil {
		return errors.New("cluster is required")
	}
	return DB.Create(cluster).Error
}

func InitializeClusterCredentialStatuses() error {
	if err := DB.Model(&Cluster{}).
		Where("(credential_status = ? OR credential_status IS NULL) AND last_success_at > ?", "", 0).
		Updates(map[string]any{
			"credential_status":      ClusterCredentialActive,
			"credential_verified_at": gorm.Expr("last_success_at"),
		}).Error; err != nil {
		return err
	}
	if err := DB.Model(&Cluster{}).
		Where("credential_status = ? OR credential_status IS NULL", "").
		Updates(map[string]any{
			"credential_status": ClusterCredentialPending,
			"health_status":     ClusterHealthUnknown,
		}).Error; err != nil {
		return err
	}
	if err := DB.Model(&Cluster{}).
		Where("credential_version <= ? OR credential_version IS NULL", 0).
		Update("credential_version", 1).Error; err != nil {
		return err
	}
	return DB.Model(&Cluster{}).
		Where("credential_issued_at = ? OR credential_issued_at IS NULL", 0).
		Update("credential_issued_at", gorm.Expr("created_at")).Error
}

func RotateClusterCredential(id int64, ciphertext string) (bool, error) {
	if id <= 0 || ciphertext == "" {
		return false, nil
	}
	now := common.GetTimestamp()
	result := DB.Model(&Cluster{}).
		Where("id = ? AND poll_locked_until < ?", id, now).
		Updates(map[string]any{
			"link_secret_ciphertext": ciphertext,
			"credential_status":      ClusterCredentialPending,
			"credential_version":     gorm.Expr("credential_version + ?", 1),
			"credential_issued_at":   now,
			"credential_verified_at": int64(0),
			"health_status":          ClusterHealthUnknown,
			"consecutive_failures":   0,
			"last_error_code":        "",
			"last_failure_payload":   "",
			"next_poll_at":           now,
			"updated_at":             now,
		})
	return result.RowsAffected == 1, result.Error
}

func DeleteClusterByID(id int64) (bool, error) {
	if id <= 0 {
		return false, nil
	}

	deleted := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("id = ?", id).Delete(&Cluster{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		if err := tx.Where("cluster_id = ?", id).Delete(&ClusterTelemetryLatest{}).Error; err != nil {
			return err
		}
		deleted = true
		return nil
	})
	return deleted, err
}

func GetClusterByID(id int64) (*Cluster, error) {
	var cluster Cluster
	err := DB.Where("id = ?", id).First(&cluster).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &cluster, nil
}

func ListClusters(filter ClusterListFilter) ([]*Cluster, error) {
	clusters := make([]*Cluster, 0)
	query := DB.Model(&Cluster{})
	if filter.EnabledOnly {
		query = query.Where("enabled = ?", true)
	}
	if search := strings.TrimSpace(filter.Search); search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR model_name_snapshot LIKE ?", like, like)
	}
	if filter.ModelID > 0 {
		query = query.Where("model_id = ?", filter.ModelID)
	}
	if filter.Health != "" {
		query = query.Where("health_status = ?", filter.Health)
	}
	err := query.Order("model_id ASC, id ASC").Find(&clusters).Error
	return clusters, err
}

func ListClustersByModelID(modelID int) ([]*Cluster, error) {
	clusters := make([]*Cluster, 0)
	err := DB.Where("model_id = ?", modelID).Order("id ASC").Find(&clusters).Error
	return clusters, err
}

func ListDueClusterIDs(now int64, limit int) ([]int64, error) {
	if limit <= 0 {
		limit = 100
	}
	ids := make([]int64, 0)
	err := DB.Model(&Cluster{}).
		Where("enabled = ? AND next_poll_at <= ? AND poll_locked_until < ?", true, now, now).
		Order("next_poll_at ASC, id ASC").
		Limit(limit).
		Pluck("id", &ids).Error
	return ids, err
}

func ScheduleEnabledClustersForPoll(now int64) error {
	return DB.Model(&Cluster{}).
		Where("enabled = ?", true).
		UpdateColumn("next_poll_at", now).Error
}

func ClaimClusterPoll(id int64, runnerID string, now int64, lockUntil int64, force bool) (bool, error) {
	query := DB.Model(&Cluster{}).
		Where("id = ? AND enabled = ? AND poll_locked_until < ?", id, true, now)
	if !force {
		query = query.Where("next_poll_at <= ?", now)
	}
	result := query.Updates(map[string]any{
		"poll_locked_by":    runnerID,
		"poll_locked_until": lockUntil,
		"updated_at":        now,
	})
	return result.RowsAffected == 1, result.Error
}

func ReleaseClusterPoll(id int64, runnerID string) error {
	result := DB.Model(&Cluster{}).
		Where("id = ? AND poll_locked_by = ?", id, runnerID).
		Updates(map[string]any{
			"poll_locked_by":    "",
			"poll_locked_until": int64(0),
			"updated_at":        common.GetTimestamp(),
		})
	return result.Error
}

func SaveClusterPollSuccess(clusterID int64, runnerID string, health ClusterHealthStatus, nextPollAt int64, telemetry *ClusterTelemetryLatest) error {
	if telemetry == nil {
		return errors.New("cluster telemetry is required")
	}
	now := common.GetTimestamp()
	return DB.Transaction(func(tx *gorm.DB) error {
		telemetry.ClusterID = clusterID
		telemetry.UpdatedAt = now
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "cluster_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"schema_version":     telemetry.SchemaVersion,
				"collection_id":      telemetry.CollectionID,
				"raw_payload":        telemetry.RawPayload,
				"normalized_payload": telemetry.NormalizedPayload,
				"collected_at":       telemetry.CollectedAt,
				"updated_at":         now,
			}),
		}).Create(telemetry).Error; err != nil {
			return err
		}

		result := tx.Model(&Cluster{}).
			Where("id = ? AND poll_locked_by = ?", clusterID, runnerID).
			Updates(map[string]any{
				"health_status":          health,
				"credential_status":      ClusterCredentialActive,
				"credential_verified_at": now,
				"last_polled_at":         now,
				"last_success_at":        now,
				"consecutive_failures":   0,
				"last_error_code":        "",
				"last_failure_payload":   "",
				"next_poll_at":           nextPollAt,
				"poll_locked_by":         "",
				"poll_locked_until":      int64(0),
				"updated_at":             now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("cluster poll lease lost")
		}
		return nil
	})
}

func SaveClusterPollFailure(clusterID int64, runnerID string, health ClusterHealthStatus, errorCode string, diagnosticPayload string, nextPollAt int64) error {
	now := common.GetTimestamp()
	updates := map[string]any{
		"health_status":        health,
		"last_polled_at":       now,
		"consecutive_failures": gorm.Expr("consecutive_failures + ?", 1),
		"last_error_code":      errorCode,
		"next_poll_at":         nextPollAt,
		"poll_locked_by":       "",
		"poll_locked_until":    int64(0),
		"updated_at":           now,
	}
	if diagnosticPayload != "" {
		updates["last_failure_payload"] = diagnosticPayload
	}
	result := DB.Model(&Cluster{}).
		Where("id = ? AND poll_locked_by = ?", clusterID, runnerID).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("cluster poll lease lost")
	}
	return nil
}

func GetLatestClusterTelemetry(clusterID int64) (*ClusterTelemetryLatest, error) {
	var telemetry ClusterTelemetryLatest
	err := DB.Where("cluster_id = ?", clusterID).First(&telemetry).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &telemetry, nil
}

func GetLatestClusterTelemetryMap(clusterIDs []int64) (map[int64]*ClusterTelemetryLatest, error) {
	result := make(map[int64]*ClusterTelemetryLatest, len(clusterIDs))
	if len(clusterIDs) == 0 {
		return result, nil
	}
	rows := make([]*ClusterTelemetryLatest, 0, len(clusterIDs))
	if err := DB.Where("cluster_id IN ?", clusterIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.ClusterID] = row
	}
	return result, nil
}
