package clusterstatus

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	rootservice "github.com/QuantumNous/new-api/service"
)

var (
	defaultServiceMu sync.RWMutex
	defaultService   *Service
	defaultPoller    *Poller
)

type Poller struct {
	service   *Service
	config    PollConfig
	runnerID  string
	ctx       context.Context
	cancel    context.CancelFunc
	wakeup    chan struct{}
	semaphore chan struct{}
	running   sync.Map
	wg        sync.WaitGroup
}

func DefaultPollConfig() PollConfig {
	intervalSeconds := common.GetEnvOrDefault("CLUSTER_TELEMETRY_POLL_INTERVAL_SECONDS", 5)
	if intervalSeconds < 1 {
		intervalSeconds = 1
	}
	timeoutSeconds := common.GetEnvOrDefault("CLUSTER_TELEMETRY_REQUEST_TIMEOUT_SECONDS", 3)
	if timeoutSeconds < 1 {
		timeoutSeconds = 1
	}
	maxConcurrency := common.GetEnvOrDefault("CLUSTER_TELEMETRY_MAX_CONCURRENCY", 8)
	if maxConcurrency < 1 {
		maxConcurrency = 1
	}
	failureThreshold := common.GetEnvOrDefault("CLUSTER_TELEMETRY_FAILURE_THRESHOLD", 3)
	if failureThreshold < 1 {
		failureThreshold = 1
	}
	maxBodyBytes := common.GetEnvOrDefault("CLUSTER_TELEMETRY_MAX_BODY_BYTES", 2*1024*1024)
	if maxBodyBytes < 1024 {
		maxBodyBytes = 1024
	}
	leaseSeconds := common.GetEnvOrDefault("CLUSTER_TELEMETRY_LEASE_TTL_SECONDS", timeoutSeconds+5)
	if leaseSeconds <= timeoutSeconds {
		leaseSeconds = timeoutSeconds + 5
	}
	maxBackoffSeconds := common.GetEnvOrDefault("CLUSTER_TELEMETRY_MAX_BACKOFF_SECONDS", 300)
	if maxBackoffSeconds < intervalSeconds {
		maxBackoffSeconds = intervalSeconds
	}
	return PollConfig{
		Interval:         time.Duration(intervalSeconds) * time.Second,
		RequestTimeout:   time.Duration(timeoutSeconds) * time.Second,
		MaxConcurrency:   maxConcurrency,
		FailureThreshold: failureThreshold,
		MaxBodyBytes:     int64(maxBodyBytes),
		LeaseTTL:         time.Duration(leaseSeconds) * time.Second,
		MaxBackoff:       time.Duration(maxBackoffSeconds) * time.Second,
	}
}

func Initialize() error {
	config := DefaultPollConfig()
	protector, err := NewAESGCMSecretProtector(common.CryptoSecret)
	if err != nil {
		return err
	}
	clusterService := NewService(
		TemporaryLinkResolver{},
		protector,
		NewHTTPAgentClient(rootservice.GetSSRFProtectedHTTPClient(), config.MaxBodyBytes),
		SchemaV1Adapter{},
		NewDefaultHealthEvaluator(config.FailureThreshold),
		EmptyHistoryRepository{},
		rootservice.ValidateSSRFProtectedFetchURL,
		config,
	)

	defaultServiceMu.Lock()
	defaultService = clusterService
	if common.IsMasterNode {
		defaultPoller = NewPoller(clusterService, config)
		defaultPoller.Start()
	}
	defaultServiceMu.Unlock()
	return nil
}

func DefaultService() (*Service, error) {
	defaultServiceMu.RLock()
	defer defaultServiceMu.RUnlock()
	if defaultService == nil {
		return nil, errors.New("cluster status service is not initialized")
	}
	return defaultService, nil
}

func Shutdown(ctx context.Context) error {
	defaultServiceMu.RLock()
	poller := defaultPoller
	defaultServiceMu.RUnlock()
	if poller == nil {
		return nil
	}
	return poller.Stop(ctx)
}

func NewPoller(service *Service, config PollConfig) *Poller {
	ctx, cancel := context.WithCancel(context.Background())
	return &Poller{
		service:   service,
		config:    config,
		runnerID:  fmt.Sprintf("%s-cluster-%s", common.NodeName, common.GetRandomString(8)),
		ctx:       ctx,
		cancel:    cancel,
		wakeup:    make(chan struct{}, 1),
		semaphore: make(chan struct{}, config.MaxConcurrency),
	}
}

func (poller *Poller) Start() {
	if poller == nil || poller.service == nil {
		return
	}
	poller.wg.Add(1)
	go func() {
		defer poller.wg.Done()
		tickInterval := poller.config.Interval / 2
		if tickInterval > time.Second {
			tickInterval = time.Second
		}
		if tickInterval < 250*time.Millisecond {
			tickInterval = 250 * time.Millisecond
		}
		ticker := time.NewTicker(tickInterval)
		defer ticker.Stop()
		poller.dispatchDue()
		for {
			select {
			case <-poller.ctx.Done():
				return
			case <-ticker.C:
				poller.dispatchDue()
			case <-poller.wakeup:
				poller.dispatchDue()
			}
		}
	}()
	logger.LogInfo(context.Background(), fmt.Sprintf(
		"cluster telemetry poller started: interval=%s timeout=%s concurrency=%d",
		poller.config.Interval,
		poller.config.RequestTimeout,
		poller.config.MaxConcurrency,
	))
}

func (poller *Poller) Stop(ctx context.Context) error {
	if poller == nil {
		return nil
	}
	poller.cancel()
	done := make(chan struct{})
	go func() {
		poller.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (poller *Poller) dispatchDue() {
	if poller.ctx.Err() != nil {
		return
	}
	ids, err := model.ListDueClusterIDs(common.GetTimestamp(), poller.config.MaxConcurrency*4)
	if err != nil {
		logger.LogWarn(context.Background(), fmt.Sprintf("cluster telemetry due query failed: %v", err))
		return
	}
	for _, clusterID := range ids {
		if _, loaded := poller.running.LoadOrStore(clusterID, struct{}{}); loaded {
			continue
		}
		select {
		case poller.semaphore <- struct{}{}:
			poller.wg.Add(1)
			go poller.poll(clusterID)
		default:
			poller.running.Delete(clusterID)
			return
		}
	}
}

func (poller *Poller) poll(clusterID int64) {
	defer poller.wg.Done()
	defer func() {
		<-poller.semaphore
		poller.running.Delete(clusterID)
	}()
	err := poller.service.PollCluster(poller.ctx, clusterID, poller.runnerID, false)
	if err == nil || errors.Is(err, ErrClusterPollInProgress) || errors.Is(err, context.Canceled) {
		return
	}
	logger.LogWarn(context.Background(), fmt.Sprintf(
		"cluster telemetry poll failed: cluster_id=%d error=%v",
		clusterID,
		err,
	))
}

func RefreshCluster(ctx context.Context, clusterID int64) (*ClusterResponse, error) {
	service, err := DefaultService()
	if err != nil {
		return nil, err
	}
	runnerID := fmt.Sprintf("%s-manual-%s", common.NodeName, common.GetRandomString(8))
	if err := service.PollCluster(ctx, clusterID, runnerID, true); err != nil {
		return nil, err
	}
	return service.GetCluster(clusterID)
}
