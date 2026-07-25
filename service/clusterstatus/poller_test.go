package clusterstatus

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultPollConfigUsesTenSecondRequestTimeout(t *testing.T) {
	t.Setenv("CLUSTER_TELEMETRY_REQUEST_TIMEOUT_SECONDS", "")
	t.Setenv("CLUSTER_TELEMETRY_LEASE_TTL_SECONDS", "")

	config := DefaultPollConfig()

	assert.Equal(t, 10*time.Second, config.RequestTimeout)
	assert.Equal(t, 15*time.Second, config.LeaseTTL)
}

func TestDefaultPollConfigKeepsRequestTimeoutOverride(t *testing.T) {
	t.Setenv("CLUSTER_TELEMETRY_REQUEST_TIMEOUT_SECONDS", "7")
	t.Setenv("CLUSTER_TELEMETRY_LEASE_TTL_SECONDS", "")

	config := DefaultPollConfig()

	assert.Equal(t, 7*time.Second, config.RequestTimeout)
	assert.Equal(t, 12*time.Second, config.LeaseTTL)
}

func TestPollConfigAppliesIntervalChangesWithoutRestart(t *testing.T) {
	interval := 5 * time.Second
	config := PollConfig{
		Interval: time.Second,
		IntervalProvider: func() time.Duration {
			return interval
		},
	}

	assert.Equal(t, 5*time.Second, config.currentInterval())

	interval = 12 * time.Second
	assert.Equal(t, 12*time.Second, config.currentInterval())
}

func TestPollConfigFallsBackToStaticInterval(t *testing.T) {
	config := PollConfig{
		Interval:         3 * time.Second,
		IntervalProvider: func() time.Duration { return 0 },
	}

	assert.Equal(t, 3*time.Second, config.currentInterval())
}
