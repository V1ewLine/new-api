package clusterstatus

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLinkSecret(t *testing.T, baseURL string, bearerToken string) string {
	t.Helper()
	payload, err := common.Marshal(temporaryLinkPayload{
		BaseURL:     baseURL,
		BearerToken: bearerToken,
	})
	require.NoError(t, err)
	return temporaryLinkSecretPrefix + base64.RawURLEncoding.EncodeToString(payload)
}

func TestTemporaryLinkResolverResolvesOpaqueConnection(t *testing.T) {
	linkSecret := testLinkSecret(t, "https://agent.example/internal/", "agent-token")

	connection, err := (TemporaryLinkResolver{}).Resolve(context.Background(), linkSecret)

	require.NoError(t, err)
	assert.Equal(t, "https://agent.example/internal", connection.BaseURL)
	assert.Equal(t, "agent-token", connection.BearerToken)
}

func TestBuildTemporaryLinkSecretAcceptsIPAndPortWithoutScheme(t *testing.T) {
	linkSecret, err := BuildTemporaryLinkSecret("10.0.0.8:9100", "agent-token")
	require.NoError(t, err)

	connection, err := (TemporaryLinkResolver{}).Resolve(context.Background(), linkSecret)
	require.NoError(t, err)
	assert.Equal(t, "http://10.0.0.8:9100", connection.BaseURL)
	assert.Equal(t, "agent-token", connection.BearerToken)
}

func TestBuildTemporaryLinkSecretRejectsInvalidConnectionFields(t *testing.T) {
	testCases := map[string]struct {
		address string
		token   string
	}{
		"missing port":  {address: "agent.example", token: "token"},
		"path included": {address: "https://agent.example:9443/internal", token: "token"},
		"missing token": {address: "https://agent.example:9443", token: ""},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			_, err := BuildTemporaryLinkSecret(testCase.address, testCase.token)
			require.ErrorIs(t, err, ErrInvalidLinkSecret)
		})
	}
}

func TestTemporaryLinkResolverRejectsUnsafeOrMalformedValues(t *testing.T) {
	testCases := map[string]string{
		"unknown prefix":    "other.payload",
		"embedded userinfo": testLinkSecret(t, "https://user:pass@agent.example", "token"),
		"query string":      testLinkSecret(t, "https://agent.example?token=leak", "token"),
		"unsupported scheme": testLinkSecret(
			t,
			"file:///etc/passwd",
			"token",
		),
		"missing token": testLinkSecret(t, "https://agent.example", ""),
	}

	for name, linkSecret := range testCases {
		t.Run(name, func(t *testing.T) {
			_, err := (TemporaryLinkResolver{}).Resolve(context.Background(), linkSecret)
			require.ErrorIs(t, err, ErrInvalidLinkSecret)
		})
	}
}

func TestAESGCMSecretProtectorEncryptsAndAuthenticatesSecret(t *testing.T) {
	protector, err := NewAESGCMSecretProtector("test-crypto-secret")
	require.NoError(t, err)
	plaintext := testLinkSecret(t, "https://agent.example", "sensitive-token")

	ciphertext, err := protector.Protect(plaintext)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(ciphertext, encryptedSecretPrefix))
	assert.NotContains(t, ciphertext, plaintext)
	assert.NotContains(t, ciphertext, "sensitive-token")

	decrypted, err := protector.Unprotect(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)

	tampered := ciphertext[:len(ciphertext)-1] + "A"
	_, err = protector.Unprotect(tampered)
	require.Error(t, err)
}
