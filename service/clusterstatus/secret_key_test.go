package clusterstatus

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveClusterSecretProtectionKeyPrefersConfiguredEnvironment(t *testing.T) {
	t.Setenv("CRYPTO_SECRET", "configured-crypto-secret")
	t.Setenv("SESSION_SECRET", "configured-session-secret")
	t.Setenv(clusterSecretKeyFileEnv, filepath.Join(t.TempDir(), "must-not-be-created"))

	secret, source, err := resolveClusterSecretProtectionKey()

	require.NoError(t, err)
	assert.Equal(t, "configured-crypto-secret", secret)
	assert.Equal(t, "CRYPTO_SECRET", source)
}

func TestResolveClusterSecretProtectionKeyFallsBackToSessionSecret(t *testing.T) {
	t.Setenv("CRYPTO_SECRET", "")
	t.Setenv("SESSION_SECRET", "configured-session-secret")
	t.Setenv(clusterSecretKeyFileEnv, filepath.Join(t.TempDir(), "must-not-be-created"))

	secret, source, err := resolveClusterSecretProtectionKey()

	require.NoError(t, err)
	assert.Equal(t, "configured-session-secret", secret)
	assert.Equal(t, "SESSION_SECRET", source)
}

func TestResolveClusterSecretProtectionKeyPersistsGeneratedKey(t *testing.T) {
	t.Setenv("CRYPTO_SECRET", "")
	t.Setenv("SESSION_SECRET", "")
	keyPath := filepath.Join(t.TempDir(), "cluster-secret.key")
	t.Setenv(clusterSecretKeyFileEnv, keyPath)

	firstSecret, firstSource, err := resolveClusterSecretProtectionKey()
	require.NoError(t, err)
	firstProtector, err := NewAESGCMSecretProtector(firstSecret)
	require.NoError(t, err)
	ciphertext, err := firstProtector.Protect("agent-connection-secret")
	require.NoError(t, err)

	secondSecret, secondSource, err := resolveClusterSecretProtectionKey()
	require.NoError(t, err)
	secondProtector, err := NewAESGCMSecretProtector(secondSecret)
	require.NoError(t, err)
	plaintext, err := secondProtector.Unprotect(ciphertext)

	require.NoError(t, err)
	assert.Equal(t, firstSecret, secondSecret)
	assert.Equal(t, "agent-connection-secret", plaintext)
	assert.Equal(t, "generated_file:"+keyPath, firstSource)
	assert.Equal(t, "file:"+keyPath, secondSource)
	if runtime.GOOS != "windows" {
		info, statErr := os.Stat(keyPath)
		require.NoError(t, statErr)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
}

func TestClusterSecretKeyFilePathDefaultsBesideSQLiteDatabase(t *testing.T) {
	t.Setenv(clusterSecretKeyFileEnv, "")
	previousSQLitePath := common.SQLitePath
	previousDatabaseType := common.MainDatabaseType()
	t.Cleanup(func() {
		common.SQLitePath = previousSQLitePath
		common.SetMainDatabaseType(previousDatabaseType)
	})
	databaseDirectory := t.TempDir()
	common.SQLitePath = filepath.Join(databaseDirectory, "one-api.db") + "?_busy_timeout=30000"
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)

	keyPath, err := clusterSecretKeyFilePath()

	require.NoError(t, err)
	assert.Equal(t, filepath.Join(databaseDirectory, defaultClusterSecretKeyFile), keyPath)
}

func TestResolveClusterSecretProtectionKeyRejectsInvalidExistingFile(t *testing.T) {
	t.Setenv("CRYPTO_SECRET", "")
	t.Setenv("SESSION_SECRET", "")
	keyPath := filepath.Join(t.TempDir(), "cluster-secret.key")
	t.Setenv(clusterSecretKeyFileEnv, keyPath)
	require.NoError(t, os.WriteFile(keyPath, []byte("invalid-key\n"), 0o600))

	_, _, err := resolveClusterSecretProtectionKey()

	require.Error(t, err)
	assert.ErrorContains(t, err, "is invalid")
	content, readErr := os.ReadFile(keyPath)
	require.NoError(t, readErr)
	assert.Equal(t, "invalid-key\n", string(content))
}
