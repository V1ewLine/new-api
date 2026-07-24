package common

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	MinClusterStatusRefreshIntervalSeconds     = 1
	MaxClusterStatusRefreshIntervalSeconds     = 300
	DefaultClusterStatusRefreshIntervalSeconds = 5
)

func DefaultClusterStatusRefreshInterval() int {
	interval := GetEnvOrDefault(
		"CLUSTER_TELEMETRY_POLL_INTERVAL_SECONDS",
		DefaultClusterStatusRefreshIntervalSeconds,
	)
	if interval < MinClusterStatusRefreshIntervalSeconds ||
		interval > MaxClusterStatusRefreshIntervalSeconds {
		return DefaultClusterStatusRefreshIntervalSeconds
	}
	return interval
}

func ParseClusterStatusRefreshIntervalSeconds(value string) (int, error) {
	interval, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil ||
		interval < MinClusterStatusRefreshIntervalSeconds ||
		interval > MaxClusterStatusRefreshIntervalSeconds {
		return 0, fmt.Errorf(
			"cluster status refresh interval must be an integer between %d and %d seconds",
			MinClusterStatusRefreshIntervalSeconds,
			MaxClusterStatusRefreshIntervalSeconds,
		)
	}
	return interval, nil
}

func GetClusterStatusRefreshIntervalSeconds() int {
	OptionMapRWMutex.RLock()
	defer OptionMapRWMutex.RUnlock()
	return ClusterStatusRefreshIntervalSeconds
}
