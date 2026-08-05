package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelResponsesCapabilityStoresStreamingModesIndependently(t *testing.T) {
	const channelId = 88001
	const modelName = "model-with-separate-stream-support"
	require.NoError(t, DeleteChannelResponsesCapabilities([]int{channelId}))
	t.Cleanup(func() {
		require.NoError(t, DeleteChannelResponsesCapabilities([]int{channelId}))
	})

	require.NoError(t, SaveChannelResponsesCapabilityMode(
		channelId,
		modelName,
		false,
		ResponsesCapabilityModeNative,
		"",
	))
	require.NoError(t, SaveChannelResponsesCapabilityMode(
		channelId,
		modelName,
		true,
		ResponsesCapabilityModeChatCompletions,
		"",
	))

	capability, err := GetChannelResponsesCapability(channelId, modelName)
	require.NoError(t, err)
	assert.Equal(t, ResponsesCapabilityModeNative, capability.NonStreamMode)
	assert.Equal(t, ResponsesCapabilityModeChatCompletions, capability.StreamMode)
	assert.Positive(t, capability.NonStreamDetectedAt)
	assert.Positive(t, capability.StreamDetectedAt)

	require.NoError(t, SaveChannelResponsesCapabilityMode(
		channelId,
		modelName,
		false,
		ResponsesCapabilityModeUnknown,
		"temporary timeout",
	))
	capability, err = GetChannelResponsesCapability(channelId, modelName)
	require.NoError(t, err)
	assert.Equal(t, ResponsesCapabilityModeUnknown, capability.NonStreamMode)
	assert.Equal(t, "temporary timeout", capability.NonStreamLastError)
	assert.Equal(t, ResponsesCapabilityModeChatCompletions, capability.StreamMode)
}

func TestChannelResponsesCapabilityStoresNativeTextCompatibilityMode(t *testing.T) {
	const channelId = 88002
	const modelName = "sglang-native-text-compat"
	require.NoError(t, DeleteChannelResponsesCapabilities([]int{channelId}))
	t.Cleanup(func() {
		require.NoError(t, DeleteChannelResponsesCapabilities([]int{channelId}))
	})

	require.NoError(t, SaveChannelResponsesCapabilityMode(
		channelId,
		modelName,
		false,
		ResponsesCapabilityModeNativeTextCompat,
		"",
	))

	capability, err := GetChannelResponsesCapability(channelId, modelName)
	require.NoError(t, err)
	assert.Equal(t, ResponsesCapabilityModeNativeTextCompat, capability.NonStreamMode)
}
