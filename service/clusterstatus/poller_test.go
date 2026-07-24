package clusterstatus

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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
