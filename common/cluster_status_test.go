package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseClusterStatusRefreshIntervalSeconds(t *testing.T) {
	for _, value := range []string{"1", "5", "300", " 10 "} {
		t.Run(value, func(t *testing.T) {
			interval, err := ParseClusterStatusRefreshIntervalSeconds(value)

			require.NoError(t, err)
			assert.GreaterOrEqual(t, interval, MinClusterStatusRefreshIntervalSeconds)
			assert.LessOrEqual(t, interval, MaxClusterStatusRefreshIntervalSeconds)
		})
	}
}

func TestParseClusterStatusRefreshIntervalSecondsRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"", "five", "0", "301", "1.5"} {
		t.Run(value, func(t *testing.T) {
			_, err := ParseClusterStatusRefreshIntervalSeconds(value)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "between 1 and 300 seconds")
		})
	}
}

func TestDefaultClusterStatusRefreshIntervalUsesValidatedEnvironmentValue(t *testing.T) {
	t.Setenv("CLUSTER_TELEMETRY_POLL_INTERVAL_SECONDS", "12")
	assert.Equal(t, 12, DefaultClusterStatusRefreshInterval())

	t.Setenv("CLUSTER_TELEMETRY_POLL_INTERVAL_SECONDS", "301")
	assert.Equal(
		t,
		DefaultClusterStatusRefreshIntervalSeconds,
		DefaultClusterStatusRefreshInterval(),
	)
}
