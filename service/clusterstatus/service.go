package clusterstatus

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"gorm.io/gorm"
)

type Service struct {
	resolver         ClusterLinkResolver
	protector        SecretProtector
	client           TelemetryAgentClient
	adapter          TelemetrySchemaAdapter
	healthEvaluator  ClusterHealthEvaluator
	history          TelemetryHistoryRepository
	validateAgentURL func(string) error
	config           PollConfig
}

func NewService(
	resolver ClusterLinkResolver,
	protector SecretProtector,
	client TelemetryAgentClient,
	adapter TelemetrySchemaAdapter,
	healthEvaluator ClusterHealthEvaluator,
	history TelemetryHistoryRepository,
	validateAgentURL func(string) error,
	config PollConfig,
) *Service {
	return &Service{
		resolver:         resolver,
		protector:        protector,
		client:           client,
		adapter:          adapter,
		healthEvaluator:  healthEvaluator,
		history:          history,
		validateAgentURL: validateAgentURL,
		config:           config,
	}
}

func (service *Service) CreateCluster(ctx context.Context, input CreateClusterInput) (*CredentialIssueResponse, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.AgentAddress = strings.TrimSpace(input.AgentAddress)
	if input.ModelID <= 0 {
		return nil, ErrClusterModelNotFound
	}
	if input.Name == "" || len(input.Name) > 128 {
		return nil, errors.New("cluster name must contain 1 to 128 characters")
	}

	var linkedModel model.Model
	if err := model.DB.Where("id = ? AND status = ?", input.ModelID, 1).First(&linkedModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			var unavailableModel model.Model
			if findErr := model.DB.Unscoped().Where("id = ?", input.ModelID).First(&unavailableModel).Error; findErr == nil {
				return nil, ErrClusterModelDisabled
			}
			return nil, ErrClusterModelNotFound
		}
		return nil, err
	}

	bearerToken, err := GenerateAgentBearerToken()
	if err != nil {
		return nil, err
	}
	linkSecret, err := BuildTemporaryLinkSecret(input.AgentAddress, bearerToken)
	if err != nil {
		return nil, ErrInvalidLinkSecret
	}
	connection, err := service.resolver.Resolve(ctx, linkSecret)
	if err != nil {
		return nil, ErrInvalidLinkSecret
	}
	if service.validateAgentURL != nil {
		if err := service.validateAgentURL(connection.BaseURL); err != nil {
			return nil, errors.New("cluster Agent address is blocked by the outbound request policy")
		}
	}
	ciphertext, err := service.protector.Protect(linkSecret)
	if err != nil {
		return nil, err
	}

	cluster := &model.Cluster{
		ModelID:              linkedModel.Id,
		ModelNameSnapshot:    linkedModel.ModelName,
		Name:                 input.Name,
		LinkSecretCiphertext: ciphertext,
		Enabled:              true,
		HealthStatus:         model.ClusterHealthUnknown,
	}
	if err := model.CreateCluster(cluster); err != nil {
		return nil, err
	}
	return &CredentialIssueResponse{
		Cluster:        *service.clusterResponse(cluster, &linkedModel, nil),
		BootstrapToken: bearerToken,
	}, nil
}

func (service *Service) RotateCredential(ctx context.Context, clusterID int64) (*CredentialIssueResponse, error) {
	cluster, err := model.GetClusterByID(clusterID)
	if err != nil {
		return nil, err
	}
	if cluster == nil {
		return nil, ErrClusterNotFound
	}

	linkSecret, err := service.protector.Unprotect(cluster.LinkSecretCiphertext)
	if err != nil {
		return nil, ErrClusterCredentialUnavailable
	}
	connection, err := service.resolver.Resolve(ctx, linkSecret)
	if err != nil {
		return nil, ErrClusterCredentialUnavailable
	}
	if service.validateAgentURL != nil {
		if err := service.validateAgentURL(connection.BaseURL); err != nil {
			return nil, errors.New("cluster Agent address is blocked by the outbound request policy")
		}
	}

	bearerToken, err := GenerateAgentBearerToken()
	if err != nil {
		return nil, err
	}
	rotatedLinkSecret, err := BuildTemporaryLinkSecret(connection.BaseURL, bearerToken)
	if err != nil {
		return nil, ErrClusterCredentialUnavailable
	}
	ciphertext, err := service.protector.Protect(rotatedLinkSecret)
	if err != nil {
		return nil, err
	}
	rotated, err := model.RotateClusterCredential(clusterID, ciphertext)
	if err != nil {
		return nil, err
	}
	if !rotated {
		if existing, getErr := model.GetClusterByID(clusterID); getErr != nil {
			return nil, getErr
		} else if existing == nil {
			return nil, ErrClusterNotFound
		}
		return nil, ErrClusterPollInProgress
	}

	response, err := service.GetCluster(clusterID)
	if err != nil {
		return nil, err
	}
	return &CredentialIssueResponse{
		Cluster:        *response,
		BootstrapToken: bearerToken,
	}, nil
}

func (service *Service) VerifyCredential(ctx context.Context, clusterID int64) (*CredentialVerificationResponse, error) {
	runnerID := common.NodeName + "-credential-" + common.GetRandomString(8)
	err := service.PollCluster(ctx, clusterID, runnerID, true)
	if err == nil {
		cluster, getErr := service.GetCluster(clusterID)
		if getErr != nil {
			return nil, getErr
		}
		return &CredentialVerificationResponse{
			Verified: true,
			Cluster:  *cluster,
		}, nil
	}

	var pollFailure *PollFailureError
	if !errors.As(err, &pollFailure) {
		return nil, err
	}
	cluster, getErr := service.GetCluster(clusterID)
	if getErr != nil {
		return nil, getErr
	}
	return &CredentialVerificationResponse{
		Verified:  false,
		ErrorCode: pollFailure.Code,
		Cluster:   *cluster,
	}, nil
}

func (service *Service) ListModelOptions() ([]ModelOption, error) {
	models := make([]*model.Model, 0)
	if err := model.DB.Where("status = ?", 1).Order("model_name ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	options := make([]ModelOption, 0, len(models))
	for _, linkedModel := range models {
		options = append(options, modelOption(linkedModel, true))
	}
	return options, nil
}

func (service *Service) GetOverview(search string, modelID int, health model.ClusterHealthStatus, page int, pageSize int) (*OverviewResponse, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	filteredClusters, err := model.ListClusters(model.ClusterListFilter{
		Search:  search,
		ModelID: modelID,
		Health:  health,
	})
	if err != nil {
		return nil, err
	}
	allClusters, err := model.ListClusters(model.ClusterListFilter{})
	if err != nil {
		return nil, err
	}
	clusterIDs := make([]int64, 0, len(allClusters))
	modelIDs := make([]int, 0, len(allClusters))
	for _, cluster := range allClusters {
		clusterIDs = append(clusterIDs, cluster.ID)
		modelIDs = append(modelIDs, cluster.ModelID)
	}
	telemetryMap, err := model.GetLatestClusterTelemetryMap(clusterIDs)
	if err != nil {
		return nil, err
	}
	modelMap, err := loadClusterModels(modelIDs)
	if err != nil {
		return nil, err
	}

	summaryByModel := make(map[int]*ModelClusterSummary)
	response := &OverviewResponse{
		ModelGroups: []ModelClusterGroup{},
		Alerts:      []ClusterAlert{},
	}
	now := common.GetTimestamp()
	for _, cluster := range allClusters {
		response.Overview.TotalClusters++
		if cluster.CredentialStatus == model.ClusterCredentialActive {
			switch cluster.HealthStatus {
			case model.ClusterHealthOnline:
				response.Overview.OnlineClusters++
			case model.ClusterHealthPartial, model.ClusterHealthAbnormal, model.ClusterHealthOffline:
				response.Overview.AbnormalClusters++
			}
		}

		normalized := decodeLatestTelemetry(telemetryMap[cluster.ID])
		addCurrentClusterLoadToOverview(&response.Overview, cluster, normalized, now)

		if cluster.CredentialStatus != model.ClusterCredentialActive ||
			(cluster.HealthStatus != model.ClusterHealthAbnormal &&
				cluster.HealthStatus != model.ClusterHealthOffline &&
				cluster.HealthStatus != model.ClusterHealthPartial) {
			continue
		}
		modelName := cluster.ModelNameSnapshot
		if linkedModel := modelMap[cluster.ModelID]; linkedModel != nil {
			modelName = linkedModel.ModelName
		}
		response.Alerts = append(response.Alerts, ClusterAlert{
			ClusterID:           cluster.ID,
			ClusterName:         cluster.Name,
			ModelName:           modelName,
			HealthStatus:        cluster.HealthStatus,
			ErrorCode:           cluster.LastErrorCode,
			LastPolledAt:        cluster.LastPolledAt,
			LastSuccessAt:       cluster.LastSuccessAt,
			ConsecutiveFailures: cluster.ConsecutiveFailures,
		})
	}
	syncLegacyCurrentLoadOverview(&response.Overview)

	var gpuUtilizationSumByModel = make(map[int]float64)
	var gpuUtilizationCountByModel = make(map[int]int)
	for _, cluster := range filteredClusters {
		linkedModel := modelMap[cluster.ModelID]
		modelName := cluster.ModelNameSnapshot
		available := false
		icon := ""
		category := "other"
		if linkedModel != nil {
			modelName = linkedModel.ModelName
			available = linkedModel.Status == 1 && !linkedModel.DeletedAt.Valid
			icon = linkedModel.Icon
			category = modelCategory(linkedModel)
		}
		modelSummary := summaryByModel[cluster.ModelID]
		if modelSummary == nil {
			modelSummary = &ModelClusterSummary{
				ModelID:        cluster.ModelID,
				ModelName:      modelName,
				Icon:           icon,
				Type:           category,
				ModelAvailable: available,
				HealthStatus:   model.ClusterHealthUnknown,
			}
			summaryByModel[cluster.ModelID] = modelSummary
		}
		modelSummary.ClusterCount++
		if cluster.CredentialStatus == model.ClusterCredentialActive {
			if cluster.HealthStatus == model.ClusterHealthOnline {
				modelSummary.OnlineCount++
			}
			if cluster.HealthStatus == model.ClusterHealthPartial ||
				cluster.HealthStatus == model.ClusterHealthAbnormal ||
				cluster.HealthStatus == model.ClusterHealthOffline {
				modelSummary.AbnormalCount++
			}
			modelSummary.HealthStatus = mergeHealthStatus(modelSummary.HealthStatus, cluster.HealthStatus)
		}

		normalized := decodeLatestTelemetry(telemetryMap[cluster.ID])
		if normalized != nil {
			for _, device := range normalized.Machine.GPU.Devices {
				if device.UtilizationPercent == nil {
					continue
				}
				gpuUtilizationSumByModel[cluster.ModelID] += *device.UtilizationPercent
				gpuUtilizationCountByModel[cluster.ModelID]++
			}
		}
		addCurrentClusterLoadToModel(modelSummary, cluster, normalized, now)
	}

	modelSummaries := make([]ModelClusterSummary, 0, len(summaryByModel))
	for modelID, modelSummary := range summaryByModel {
		syncLegacyCurrentLoadModel(modelSummary)
		if count := gpuUtilizationCountByModel[modelID]; count > 0 {
			average := gpuUtilizationSumByModel[modelID] / float64(count)
			modelSummary.GPUUtilization = &average
		}
		modelSummaries = append(modelSummaries, *modelSummary)
	}
	sort.Slice(modelSummaries, func(i, j int) bool {
		if modelSummaries[i].Type == modelSummaries[j].Type {
			return strings.ToLower(modelSummaries[i].ModelName) < strings.ToLower(modelSummaries[j].ModelName)
		}
		return modelTypeOrder(modelSummaries[i].Type) < modelTypeOrder(modelSummaries[j].Type)
	})
	sort.Slice(response.Alerts, func(i, j int) bool {
		return response.Alerts[i].LastPolledAt > response.Alerts[j].LastPolledAt
	})
	if len(response.Alerts) > 20 {
		response.Alerts = response.Alerts[:20]
	}

	totalModels := len(modelSummaries)
	start := (page - 1) * pageSize
	if start > totalModels {
		start = totalModels
	}
	end := start + pageSize
	if end > totalModels {
		end = totalModels
	}
	pagedModels := modelSummaries[start:end]
	for _, category := range []string{"language", "embedding", "multimodal", "other"} {
		group := ModelClusterGroup{Type: category, Models: []ModelClusterSummary{}}
		for _, modelSummary := range pagedModels {
			if modelSummary.Type == category {
				group.Models = append(group.Models, modelSummary)
			}
		}
		if len(group.Models) > 0 {
			response.ModelGroups = append(response.ModelGroups, group)
		}
	}
	response.Pagination = Pagination{Page: page, PageSize: pageSize, Total: totalModels}
	return response, nil
}

func (service *Service) GetModelDetail(modelID int) (*ModelDetailResponse, error) {
	if modelID <= 0 {
		return nil, ErrClusterModelNotFound
	}
	var linkedModel model.Model
	if err := model.DB.Unscoped().Where("id = ?", modelID).First(&linkedModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrClusterModelNotFound
		}
		return nil, err
	}
	clusters, err := model.ListClustersByModelID(modelID)
	if err != nil {
		return nil, err
	}
	clusterIDs := make([]int64, 0, len(clusters))
	for _, cluster := range clusters {
		clusterIDs = append(clusterIDs, cluster.ID)
	}
	telemetryMap, err := model.GetLatestClusterTelemetryMap(clusterIDs)
	if err != nil {
		return nil, err
	}

	available := linkedModel.Status == 1 && !linkedModel.DeletedAt.Valid
	summary := ModelClusterSummary{
		ModelID:        linkedModel.Id,
		ModelName:      linkedModel.ModelName,
		Icon:           linkedModel.Icon,
		Type:           modelCategory(&linkedModel),
		ModelAvailable: available,
		HealthStatus:   model.ClusterHealthUnknown,
	}
	clusterResponses := make([]ClusterResponse, 0, len(clusters))
	var utilizationTotal float64
	var utilizationCount int
	now := common.GetTimestamp()
	for _, cluster := range clusters {
		normalized := decodeLatestTelemetry(telemetryMap[cluster.ID])
		clusterResponses = append(clusterResponses, *service.clusterResponse(cluster, &linkedModel, normalized))
		summary.ClusterCount++
		if cluster.CredentialStatus == model.ClusterCredentialActive {
			if cluster.HealthStatus == model.ClusterHealthOnline {
				summary.OnlineCount++
			}
			if cluster.HealthStatus == model.ClusterHealthPartial ||
				cluster.HealthStatus == model.ClusterHealthAbnormal ||
				cluster.HealthStatus == model.ClusterHealthOffline {
				summary.AbnormalCount++
			}
			summary.HealthStatus = mergeHealthStatus(summary.HealthStatus, cluster.HealthStatus)
		}
		addCurrentClusterLoadToModel(&summary, cluster, normalized, now)
		if normalized == nil {
			continue
		}
		for _, device := range normalized.Machine.GPU.Devices {
			if device.UtilizationPercent != nil {
				utilizationTotal += *device.UtilizationPercent
				utilizationCount++
			}
		}
	}
	if utilizationCount > 0 {
		average := utilizationTotal / float64(utilizationCount)
		summary.GPUUtilization = &average
	}
	syncLegacyCurrentLoadModel(&summary)
	return &ModelDetailResponse{
		Model:    modelOption(&linkedModel, available),
		Summary:  summary,
		Clusters: clusterResponses,
	}, nil
}

func (service *Service) GetCluster(clusterID int64) (*ClusterResponse, error) {
	cluster, err := model.GetClusterByID(clusterID)
	if err != nil {
		return nil, err
	}
	if cluster == nil {
		return nil, ErrClusterNotFound
	}
	modelMap, err := loadClusterModels([]int{cluster.ModelID})
	if err != nil {
		return nil, err
	}
	latest, err := model.GetLatestClusterTelemetry(clusterID)
	if err != nil {
		return nil, err
	}
	return service.clusterResponse(cluster, modelMap[cluster.ModelID], decodeLatestTelemetry(latest)), nil
}

func (service *Service) DeleteCluster(clusterID int64) error {
	deleted, err := model.DeleteClusterByID(clusterID)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrClusterNotFound
	}
	return nil
}

func (service *Service) GetLatestTelemetry(clusterID int64) (*NormalizedTelemetry, error) {
	if cluster, err := model.GetClusterByID(clusterID); err != nil {
		return nil, err
	} else if cluster == nil {
		return nil, ErrClusterNotFound
	}
	latest, err := model.GetLatestClusterTelemetry(clusterID)
	if err != nil {
		return nil, err
	}
	return decodeLatestTelemetry(latest), nil
}

func (service *Service) GetTelemetryHistory(ctx context.Context, clusterID int64) ([]NormalizedTelemetry, error) {
	if cluster, err := model.GetClusterByID(clusterID); err != nil {
		return nil, err
	} else if cluster == nil {
		return nil, ErrClusterNotFound
	}
	return service.history.List(ctx, clusterID, time.Time{}, time.Time{})
}

func (service *Service) PollCluster(ctx context.Context, clusterID int64, runnerID string, force bool) error {
	now := common.GetTimestamp()
	claimed, err := model.ClaimClusterPoll(clusterID, runnerID, now, now+int64(service.config.LeaseTTL.Seconds()), force)
	if err != nil {
		return err
	}
	if !claimed {
		return ErrClusterPollInProgress
	}

	cluster, err := model.GetClusterByID(clusterID)
	if err != nil {
		_ = model.ReleaseClusterPoll(clusterID, runnerID)
		return err
	}
	if cluster == nil {
		_ = model.ReleaseClusterPoll(clusterID, runnerID)
		return ErrClusterNotFound
	}
	linkSecret, err := service.protector.Unprotect(cluster.LinkSecretCiphertext)
	if err != nil {
		return service.finishPollFailure(cluster, runnerID, "CLUSTER_SECRET_INVALID")
	}
	connection, err := service.resolver.Resolve(ctx, linkSecret)
	if err != nil {
		return service.finishPollFailure(cluster, runnerID, "CLUSTER_LINK_INVALID")
	}
	if service.validateAgentURL != nil {
		if err := service.validateAgentURL(connection.BaseURL); err != nil {
			return service.finishPollFailure(cluster, runnerID, "AGENT_ADDRESS_BLOCKED")
		}
	}

	requestCtx, cancel := context.WithTimeout(ctx, service.config.RequestTimeout)
	raw, err := service.client.Fetch(requestCtx, connection)
	cancel()
	if err != nil {
		if errors.Is(err, context.Canceled) && errors.Is(ctx.Err(), context.Canceled) {
			_ = model.ReleaseClusterPoll(clusterID, runnerID)
			return context.Canceled
		}
		return service.finishPollFailure(cluster, runnerID, agentErrorCode(err))
	}
	normalized, err := service.adapter.Adapt(raw, cluster.ModelNameSnapshot)
	if err != nil {
		return service.finishPollFailureWithDiagnostic(cluster, runnerID, schemaErrorCode(err), string(raw))
	}
	normalizedPayload, err := common.Marshal(normalized)
	if err != nil {
		return service.finishPollFailure(cluster, runnerID, "TELEMETRY_NORMALIZE_FAILED")
	}
	collectedAt := common.GetTimestamp()
	if parsed, parseErr := time.Parse(time.RFC3339Nano, normalized.CollectedAt); parseErr == nil {
		collectedAt = parsed.Unix()
	}
	nextPollAt := common.GetTimestamp() + pollDelaySeconds(
		service.jittered(service.config.currentInterval(), cluster.ID),
	)
	err = model.SaveClusterPollSuccess(
		cluster.ID,
		runnerID,
		service.healthEvaluator.Evaluate(normalized),
		nextPollAt,
		&model.ClusterTelemetryLatest{
			SchemaVersion:     normalized.SchemaVersion,
			CollectionID:      normalized.CollectionID,
			RawPayload:        string(raw),
			NormalizedPayload: string(normalizedPayload),
			CollectedAt:       collectedAt,
		},
	)
	return err
}

func (service *Service) finishPollFailure(cluster *model.Cluster, runnerID string, errorCode string) error {
	return service.finishPollFailureWithDiagnostic(cluster, runnerID, errorCode, "")
}

func (service *Service) finishPollFailureWithDiagnostic(cluster *model.Cluster, runnerID string, errorCode string, diagnosticPayload string) error {
	failures := cluster.ConsecutiveFailures + 1
	backoff := service.config.currentInterval()
	for retry := 1; retry < failures && backoff < service.config.MaxBackoff; retry++ {
		backoff *= 2
	}
	if backoff > service.config.MaxBackoff {
		backoff = service.config.MaxBackoff
	}
	nextPollAt := common.GetTimestamp() + pollDelaySeconds(
		service.jittered(backoff, cluster.ID),
	)
	health := service.healthEvaluator.FailureStatus(failures)
	if cluster.CredentialStatus == model.ClusterCredentialPending {
		health = model.ClusterHealthUnknown
	}
	if err := model.SaveClusterPollFailure(
		cluster.ID,
		runnerID,
		health,
		errorCode,
		diagnosticPayload,
		nextPollAt,
	); err != nil {
		return err
	}
	return &PollFailureError{Code: errorCode}
}

func (service *Service) jittered(duration time.Duration, clusterID int64) time.Duration {
	if duration <= 0 {
		return time.Second
	}
	bucket := (clusterID*31 + time.Now().Unix()) % 21
	percent := float64(bucket-10) / 100
	return duration + time.Duration(float64(duration)*percent)
}

func pollDelaySeconds(delay time.Duration) int64 {
	if delay <= 0 {
		return 1
	}
	seconds := int64((delay + time.Second - 1) / time.Second)
	if seconds < 1 {
		return 1
	}
	return seconds
}

func (service *Service) clusterResponse(cluster *model.Cluster, linkedModel *model.Model, telemetry *NormalizedTelemetry) *ClusterResponse {
	modelName := cluster.ModelNameSnapshot
	modelAvailable := false
	if linkedModel != nil {
		modelName = linkedModel.ModelName
		modelAvailable = linkedModel.Status == 1 && !linkedModel.DeletedAt.Valid
	}
	return &ClusterResponse{
		ID:                   cluster.ID,
		ModelID:              cluster.ModelID,
		ModelName:            modelName,
		ModelAvailable:       modelAvailable,
		Name:                 cluster.Name,
		Enabled:              cluster.Enabled,
		HealthStatus:         cluster.HealthStatus,
		CredentialStatus:     cluster.CredentialStatus,
		CredentialVersion:    cluster.CredentialVersion,
		CredentialIssuedAt:   cluster.CredentialIssuedAt,
		CredentialVerifiedAt: cluster.CredentialVerifiedAt,
		HasLinkSecret:        cluster.LinkSecretCiphertext != "",
		LastPolledAt:         cluster.LastPolledAt,
		LastSuccessAt:        cluster.LastSuccessAt,
		ConsecutiveFailures:  cluster.ConsecutiveFailures,
		LastErrorCode:        cluster.LastErrorCode,
		CreatedAt:            cluster.CreatedAt,
		UpdatedAt:            cluster.UpdatedAt,
		Telemetry:            telemetry,
	}
}

func decodeLatestTelemetry(latest *model.ClusterTelemetryLatest) *NormalizedTelemetry {
	if latest == nil || latest.NormalizedPayload == "" {
		return nil
	}
	var telemetry NormalizedTelemetry
	if err := common.UnmarshalJsonStr(latest.NormalizedPayload, &telemetry); err != nil {
		return nil
	}
	if telemetry.Metrics.Requests != nil && telemetry.Metrics.RequestsSemantics == "" {
		telemetry.Metrics.RequestsSemantics = "unknown"
	}
	if telemetry.Metrics.Tokens != nil && telemetry.Metrics.TokensSemantics == "" {
		telemetry.Metrics.TokensSemantics = "unknown"
	}
	return &telemetry
}

func currentClusterLoad(
	cluster *model.Cluster,
	telemetry *NormalizedTelemetry,
	now int64,
) (requests float64, requestsAvailable bool, tokens float64, tokensAvailable bool, monitored bool) {
	if cluster == nil || !cluster.Enabled || cluster.CredentialStatus != model.ClusterCredentialActive {
		return 0, false, 0, false, false
	}
	monitored = true
	staleAfter := int64(common.GetClusterStatusRefreshIntervalSeconds() * 3)
	if staleAfter < 30 {
		staleAfter = 30
	}
	if telemetry == nil || cluster.LastSuccessAt <= 0 || now-cluster.LastSuccessAt > staleAfter {
		return 0, false, 0, false, true
	}
	running := nonNegativeFiniteMetric(telemetry.Engine.RunningRequests)
	waiting := nonNegativeFiniteMetric(telemetry.Engine.WaitingRequests)
	if running != nil && waiting != nil {
		requests = *running + *waiting
		requestsAvailable = true
	}
	tokenUsage := nonNegativeFiniteMetric(telemetry.Engine.TokenUsage)
	if tokenUsage != nil {
		tokens = *tokenUsage
		tokensAvailable = true
	}
	return requests, requestsAvailable, tokens, tokensAvailable, true
}

func addCurrentClusterLoadToOverview(
	summary *OverviewSummary,
	cluster *model.Cluster,
	telemetry *NormalizedTelemetry,
	now int64,
) {
	requests, requestsAvailable, tokens, tokensAvailable, monitored := currentClusterLoad(cluster, telemetry, now)
	if monitored {
		summary.MonitoredClusters++
	}
	if requestsAvailable {
		summary.CurrentRequests += requests
		summary.CurrentRequestsAvailable = true
		summary.RequestsReportingClusters++
	}
	if tokensAvailable {
		summary.CurrentTokenUsage += tokens
		summary.CurrentTokenUsageAvailable = true
		summary.TokensReportingClusters++
	}
}

func addCurrentClusterLoadToModel(
	summary *ModelClusterSummary,
	cluster *model.Cluster,
	telemetry *NormalizedTelemetry,
	now int64,
) {
	requests, requestsAvailable, tokens, tokensAvailable, monitored := currentClusterLoad(cluster, telemetry, now)
	if monitored {
		summary.MonitoredClusters++
	}
	if requestsAvailable {
		summary.CurrentRequests += requests
		summary.CurrentRequestsAvailable = true
		summary.RequestsReportingClusters++
	}
	if tokensAvailable {
		summary.CurrentTokenUsage += tokens
		summary.CurrentTokenUsageAvailable = true
		summary.TokensReportingClusters++
	}
}

func syncLegacyCurrentLoadOverview(summary *OverviewSummary) {
	summary.TotalRequests = summary.CurrentRequests
	summary.TotalTokens = summary.CurrentTokenUsage
	summary.RequestsAvailable = summary.CurrentRequestsAvailable
	summary.TokensAvailable = summary.CurrentTokenUsageAvailable
}

func syncLegacyCurrentLoadModel(summary *ModelClusterSummary) {
	summary.TotalRequests = summary.CurrentRequests
	summary.TotalTokens = summary.CurrentTokenUsage
	summary.RequestsAvailable = summary.CurrentRequestsAvailable
	summary.TokensAvailable = summary.CurrentTokenUsageAvailable
}

func loadClusterModels(modelIDs []int) (map[int]*model.Model, error) {
	result := make(map[int]*model.Model)
	if len(modelIDs) == 0 {
		return result, nil
	}
	unique := make(map[int]struct{}, len(modelIDs))
	ids := make([]int, 0, len(modelIDs))
	for _, modelID := range modelIDs {
		if modelID <= 0 {
			continue
		}
		if _, exists := unique[modelID]; exists {
			continue
		}
		unique[modelID] = struct{}{}
		ids = append(ids, modelID)
	}
	models := make([]*model.Model, 0, len(ids))
	if err := model.DB.Unscoped().Where("id IN ?", ids).Find(&models).Error; err != nil {
		return nil, err
	}
	for _, linkedModel := range models {
		result[linkedModel.Id] = linkedModel
	}
	return result, nil
}

func modelOption(linkedModel *model.Model, enabled bool) ModelOption {
	return ModelOption{
		ID:      linkedModel.Id,
		Name:    linkedModel.ModelName,
		Icon:    linkedModel.Icon,
		Type:    modelCategory(linkedModel),
		Enabled: enabled,
	}
}

func modelCategory(linkedModel *model.Model) string {
	if linkedModel == nil {
		return "other"
	}
	metadata := strings.ToLower(strings.Join([]string{
		linkedModel.Tags,
		linkedModel.Endpoints,
		linkedModel.Description,
	}, " "))
	switch {
	case strings.Contains(metadata, "embedding"):
		return "embedding"
	case strings.Contains(metadata, "image"),
		strings.Contains(metadata, "audio"),
		strings.Contains(metadata, "speech"),
		strings.Contains(metadata, "video"),
		strings.Contains(metadata, "multimodal"):
		return "multimodal"
	default:
		return "language"
	}
}

func modelTypeOrder(category string) int {
	switch category {
	case "language":
		return 0
	case "embedding":
		return 1
	case "multimodal":
		return 2
	default:
		return 3
	}
}

func mergeHealthStatus(current model.ClusterHealthStatus, next model.ClusterHealthStatus) model.ClusterHealthStatus {
	rank := map[model.ClusterHealthStatus]int{
		model.ClusterHealthUnknown:  0,
		model.ClusterHealthOnline:   1,
		model.ClusterHealthPartial:  2,
		model.ClusterHealthAbnormal: 3,
		model.ClusterHealthOffline:  4,
	}
	if rank[next] > rank[current] {
		return next
	}
	return current
}
