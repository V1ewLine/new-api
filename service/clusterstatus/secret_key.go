package clusterstatus

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const (
	clusterSecretKeyFileEnv      = "CLUSTER_SECRET_KEY_FILE"
	defaultClusterSecretKeyFile  = ".new-api-cluster-secret.key"
	clusterSecretKeyBytes        = 32
	maxClusterSecretKeyFileBytes = 128
)

func resolveClusterSecretProtectionKey() (string, string, error) {
	if secret := os.Getenv("CRYPTO_SECRET"); secret != "" {
		return secret, "CRYPTO_SECRET", nil
	}
	if secret := os.Getenv("SESSION_SECRET"); secret != "" {
		return secret, "SESSION_SECRET", nil
	}

	keyPath, err := clusterSecretKeyFilePath()
	if err != nil {
		return "", "", err
	}
	secret, created, err := loadOrCreateClusterSecretKeyFile(keyPath)
	if err != nil {
		return "", "", err
	}
	source := "file"
	if created {
		source = "generated_file"
	}
	return secret, fmt.Sprintf("%s:%s", source, keyPath), nil
}

func clusterSecretKeyFilePath() (string, error) {
	if configuredPath := strings.TrimSpace(os.Getenv(clusterSecretKeyFileEnv)); configuredPath != "" {
		return filepath.Abs(configuredPath)
	}

	keyDirectory, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve cluster secret key directory: %w", err)
	}
	if common.UsingMainDatabase(common.DatabaseTypeSQLite) {
		sqlitePath := strings.TrimSpace(common.SQLitePath)
		sqlitePath = strings.TrimPrefix(sqlitePath, "file:")
		if queryIndex := strings.IndexByte(sqlitePath, '?'); queryIndex >= 0 {
			sqlitePath = sqlitePath[:queryIndex]
		}
		if sqlitePath != "" && sqlitePath != ":memory:" {
			absoluteSQLitePath, err := filepath.Abs(sqlitePath)
			if err != nil {
				return "", fmt.Errorf("resolve SQLite database path for cluster secret key: %w", err)
			}
			keyDirectory = filepath.Dir(absoluteSQLitePath)
		}
	}
	return filepath.Join(keyDirectory, defaultClusterSecretKeyFile), nil
}

func loadOrCreateClusterSecretKeyFile(path string) (string, bool, error) {
	secret, err := readClusterSecretKeyFile(path)
	if err == nil {
		return secret, false, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", false, err
	}

	randomBytes := make([]byte, clusterSecretKeyBytes)
	if _, err := io.ReadFull(rand.Reader, randomBytes); err != nil {
		return "", false, fmt.Errorf("generate cluster secret key: %w", err)
	}
	secret = base64.RawURLEncoding.EncodeToString(randomBytes)

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		secret, readErr := readClusterSecretKeyFile(path)
		return secret, false, readErr
	}
	if err != nil {
		return "", false, fmt.Errorf("create cluster secret key file %q: %w", path, err)
	}

	complete := false
	defer func() {
		if complete {
			return
		}
		_ = file.Close()
		_ = os.Remove(path)
	}()
	if _, err := file.WriteString(secret + "\n"); err != nil {
		return "", false, fmt.Errorf("write cluster secret key file %q: %w", path, err)
	}
	if err := file.Sync(); err != nil {
		return "", false, fmt.Errorf("sync cluster secret key file %q: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return "", false, fmt.Errorf("close cluster secret key file %q: %w", path, err)
	}
	complete = true
	return secret, true, nil
}

func readClusterSecretKeyFile(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("cluster secret key file %q is not a regular file", path)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return "", fmt.Errorf("cluster secret key file %q must have permissions 0600", path)
	}
	if info.Size() > maxClusterSecretKeyFileBytes {
		return "", fmt.Errorf("cluster secret key file %q is too large", path)
	}

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, maxClusterSecretKeyFileBytes+1))
	if err != nil {
		return "", fmt.Errorf("read cluster secret key file %q: %w", path, err)
	}
	if len(content) > maxClusterSecretKeyFileBytes {
		return "", fmt.Errorf("cluster secret key file %q is too large", path)
	}
	secret := strings.TrimSpace(string(content))
	decoded, err := base64.RawURLEncoding.DecodeString(secret)
	if err != nil || len(decoded) != clusterSecretKeyBytes {
		return "", fmt.Errorf("cluster secret key file %q is invalid", path)
	}
	return secret, nil
}
