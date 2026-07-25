package clusterstatus

import (
	"context"
	"errors"
	"time"

	"github.com/QuantumNous/new-api/model"
)

var (
	ErrClusterNotFound              = errors.New("cluster not found")
	ErrClusterModelNotFound         = errors.New("cluster model not found")
	ErrClusterModelDisabled         = errors.New("cluster model is disabled")
	ErrClusterPollInProgress        = errors.New("cluster poll is already in progress")
	ErrInvalidLinkSecret            = errors.New("invalid cluster link secret")
	ErrClusterCredentialUnavailable = errors.New("cluster credential cannot be rotated")
	ErrClusterTrendInvalid          = errors.New("invalid cluster trend request")
)

type PollFailureError struct {
	Code string
}

func (err *PollFailureError) Error() string {
	if err == nil || err.Code == "" {
		return "cluster telemetry poll failed"
	}
	return "cluster telemetry poll failed: " + err.Code
}

type CreateClusterInput struct {
	ModelID      int
	Name         string
	AgentAddress string
}

type ResolvedAgentConnection struct {
	BaseURL     string
	BearerToken string
}

type ClusterLinkResolver interface {
	Resolve(ctx context.Context, linkSecret string) (ResolvedAgentConnection, error)
}

type SecretProtector interface {
	Protect(plaintext string) (string, error)
	Unprotect(ciphertext string) (string, error)
}

type TelemetryAgentClient interface {
	Fetch(ctx context.Context, connection ResolvedAgentConnection) ([]byte, error)
}

type TelemetrySchemaAdapter interface {
	Adapt(raw []byte, expectedModelName string) (*NormalizedTelemetry, error)
}

type TelemetryHistoryRepository interface {
	List(_ context.Context, _ int64, _ time.Time, _ time.Time) ([]NormalizedTelemetry, error)
}

type ClusterHealthEvaluator interface {
	Evaluate(telemetry *NormalizedTelemetry) model.ClusterHealthStatus
	FailureStatus(consecutiveFailures int) model.ClusterHealthStatus
}

type PollConfig struct {
	Interval         time.Duration
	IntervalProvider func() time.Duration
	RequestTimeout   time.Duration
	MaxConcurrency   int
	FailureThreshold int
	MaxBodyBytes     int64
	LeaseTTL         time.Duration
	MaxBackoff       time.Duration
}

func (config PollConfig) currentInterval() time.Duration {
	if config.IntervalProvider != nil {
		if interval := config.IntervalProvider(); interval > 0 {
			return interval
		}
	}
	return config.Interval
}

type NormalizedTelemetry struct {
	SchemaVersion string                    `json:"schema_version"`
	CollectionID  string                    `json:"collection_id"`
	Status        string                    `json:"status"`
	CollectedAt   string                    `json:"collected_at"`
	Identity      TelemetryIdentity         `json:"identity"`
	Engine        TelemetryEngine           `json:"engine"`
	Machine       TelemetryMachine          `json:"machine"`
	Alignment     TelemetryAlignment        `json:"alignment"`
	ModelMismatch bool                      `json:"model_mismatch"`
	Metrics       TelemetryAggregateMetrics `json:"metrics"`
}

type TelemetryIdentity struct {
	NodeID   string `json:"node_id"`
	EngineID string `json:"engine_id"`
	Model    string `json:"model"`
}

type TelemetryEngine struct {
	Up                bool             `json:"up"`
	Version           string           `json:"version"`
	ObservedAt        string           `json:"observed_at"`
	RequestDurationMS float64          `json:"request_duration_ms"`
	Loads             []map[string]any `json:"loads"`
	RunningRequests   *float64         `json:"running_requests,omitempty"`
	WaitingRequests   *float64         `json:"waiting_requests,omitempty"`
	TokenUsage        *float64         `json:"token_usage,omitempty"`
	Throughput        *float64         `json:"throughput,omitempty"`
	CacheUsage        *float64         `json:"cache_usage,omitempty"`
	ErrorCode         string           `json:"error_code,omitempty"`
}

type TelemetryMachine struct {
	Up             bool            `json:"up"`
	SampledAt      string          `json:"sampled_at"`
	WindowComplete bool            `json:"window_complete"`
	GPU            TelemetryGPU    `json:"gpu"`
	System         TelemetrySystem `json:"system"`
}

type TelemetryGPU struct {
	Available       bool                 `json:"available"`
	Count           int                  `json:"count"`
	PowerTotalWatts *float64             `json:"power_total_watts,omitempty"`
	DriverVersion   string               `json:"driver_version"`
	Devices         []TelemetryGPUDevice `json:"devices"`
}

type TelemetryGPUDevice struct {
	Index                    int      `json:"index"`
	UUID                     string   `json:"uuid"`
	Name                     string   `json:"name"`
	PowerWatts               *float64 `json:"power_watts,omitempty"`
	TemperatureCelsius       *float64 `json:"temperature_celsius,omitempty"`
	UtilizationPercent       *float64 `json:"utilization_percent,omitempty"`
	MemoryUtilizationPercent *float64 `json:"memory_utilization_percent,omitempty"`
	MemoryUsedBytes          *float64 `json:"memory_used_bytes,omitempty"`
	MemoryTotalBytes         *float64 `json:"memory_total_bytes,omitempty"`
	SMClockMHz               *float64 `json:"sm_clock_mhz,omitempty"`
}

type TelemetrySystem struct {
	Available                bool                `json:"available"`
	CPUUtilizationPercent    *float64            `json:"cpu_utilization_percent,omitempty"`
	CPUCount                 *float64            `json:"cpu_count,omitempty"`
	MemoryUsedBytes          *float64            `json:"memory_used_bytes,omitempty"`
	MemoryAvailableBytes     *float64            `json:"memory_available_bytes,omitempty"`
	MemoryTotalBytes         *float64            `json:"memory_total_bytes,omitempty"`
	MemoryUtilizationPercent *float64            `json:"memory_utilization_percent,omitempty"`
	LoadAverage              map[string]*float64 `json:"load_average"`
}

type TelemetryAlignment struct {
	Method  string   `json:"method"`
	Quality string   `json:"quality"`
	SkewMS  *float64 `json:"skew_ms,omitempty"`
}

type TelemetryAggregateMetrics struct {
	Requests          *float64 `json:"requests,omitempty"`
	RequestsSemantics string   `json:"requests_semantics,omitempty"`
	Tokens            *float64 `json:"tokens,omitempty"`
	TokensSemantics   string   `json:"tokens_semantics,omitempty"`
}

type ClusterResponse struct {
	ID                   int64                         `json:"id"`
	ModelID              int                           `json:"model_id"`
	ModelName            string                        `json:"model_name"`
	ModelAvailable       bool                          `json:"model_available"`
	Name                 string                        `json:"name"`
	Enabled              bool                          `json:"enabled"`
	HealthStatus         model.ClusterHealthStatus     `json:"health_status"`
	CredentialStatus     model.ClusterCredentialStatus `json:"credential_status"`
	CredentialVersion    int                           `json:"credential_version"`
	CredentialIssuedAt   int64                         `json:"credential_issued_at"`
	CredentialVerifiedAt int64                         `json:"credential_verified_at"`
	HasLinkSecret        bool                          `json:"has_link_secret"`
	LastPolledAt         int64                         `json:"last_polled_at"`
	LastSuccessAt        int64                         `json:"last_success_at"`
	ConsecutiveFailures  int                           `json:"consecutive_failures"`
	LastErrorCode        string                        `json:"last_error_code,omitempty"`
	CreatedAt            int64                         `json:"created_at"`
	UpdatedAt            int64                         `json:"updated_at"`
	Telemetry            *NormalizedTelemetry          `json:"telemetry,omitempty"`
}

type CredentialIssueResponse struct {
	Cluster        ClusterResponse `json:"cluster"`
	BootstrapToken string          `json:"bootstrap_token"`
}

type CredentialVerificationResponse struct {
	Verified  bool            `json:"verified"`
	ErrorCode string          `json:"error_code,omitempty"`
	Cluster   ClusterResponse `json:"cluster"`
}

type ModelOption struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Icon    string `json:"icon,omitempty"`
	Type    string `json:"type"`
	Enabled bool   `json:"enabled"`
}

type OverviewSummary struct {
	TotalClusters              int     `json:"total_clusters"`
	OnlineClusters             int     `json:"online_clusters"`
	AbnormalClusters           int     `json:"abnormal_clusters"`
	CurrentRequests            float64 `json:"current_requests"`
	CurrentTokenUsage          float64 `json:"current_token_usage"`
	CurrentRequestsAvailable   bool    `json:"current_requests_available"`
	CurrentTokenUsageAvailable bool    `json:"current_token_usage_available"`
	RequestsReportingClusters  int     `json:"requests_reporting_clusters"`
	TokensReportingClusters    int     `json:"tokens_reporting_clusters"`
	MonitoredClusters          int     `json:"monitored_clusters"`
	TotalRequests              float64 `json:"total_requests"`
	TotalTokens                float64 `json:"total_tokens"`
	RequestsAvailable          bool    `json:"requests_available"`
	TokensAvailable            bool    `json:"tokens_available"`
}

type ClusterAlert struct {
	ClusterID           int64                     `json:"cluster_id"`
	ClusterName         string                    `json:"cluster_name"`
	ModelName           string                    `json:"model_name"`
	HealthStatus        model.ClusterHealthStatus `json:"health_status"`
	ErrorCode           string                    `json:"error_code,omitempty"`
	LastPolledAt        int64                     `json:"last_polled_at"`
	LastSuccessAt       int64                     `json:"last_success_at"`
	ConsecutiveFailures int                       `json:"consecutive_failures"`
}

type ModelClusterSummary struct {
	ModelID                    int                       `json:"model_id"`
	ModelName                  string                    `json:"model_name"`
	Icon                       string                    `json:"icon,omitempty"`
	Type                       string                    `json:"type"`
	ModelAvailable             bool                      `json:"model_available"`
	HealthStatus               model.ClusterHealthStatus `json:"health_status"`
	ClusterCount               int                       `json:"cluster_count"`
	OnlineCount                int                       `json:"online_count"`
	AbnormalCount              int                       `json:"abnormal_count"`
	CurrentRequests            float64                   `json:"current_requests"`
	CurrentTokenUsage          float64                   `json:"current_token_usage"`
	CurrentRequestsAvailable   bool                      `json:"current_requests_available"`
	CurrentTokenUsageAvailable bool                      `json:"current_token_usage_available"`
	RequestsReportingClusters  int                       `json:"requests_reporting_clusters"`
	TokensReportingClusters    int                       `json:"tokens_reporting_clusters"`
	MonitoredClusters          int                       `json:"monitored_clusters"`
	TotalRequests              float64                   `json:"total_requests"`
	TotalTokens                float64                   `json:"total_tokens"`
	RequestsAvailable          bool                      `json:"requests_available"`
	TokensAvailable            bool                      `json:"tokens_available"`
	GPUUtilization             *float64                  `json:"gpu_utilization,omitempty"`
}

type ModelClusterGroup struct {
	Type   string                `json:"type"`
	Models []ModelClusterSummary `json:"models"`
}

type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

type OverviewResponse struct {
	Overview    OverviewSummary     `json:"overview"`
	ModelGroups []ModelClusterGroup `json:"model_groups"`
	Alerts      []ClusterAlert      `json:"alerts"`
	Pagination  Pagination          `json:"pagination"`
}

type ModelDetailResponse struct {
	Model    ModelOption         `json:"model"`
	Summary  ModelClusterSummary `json:"summary"`
	Clusters []ClusterResponse   `json:"clusters"`
}

type EmptyHistoryRepository struct{}

func (EmptyHistoryRepository) List(_ context.Context, _ int64, _ time.Time, _ time.Time) ([]NormalizedTelemetry, error) {
	return []NormalizedTelemetry{}, nil
}
