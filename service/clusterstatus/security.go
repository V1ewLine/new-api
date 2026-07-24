package clusterstatus

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const (
	temporaryLinkSecretPrefix = "sgta1."
	encryptedSecretPrefix     = "cluster-secret:v1:"
	maxLinkSecretLength       = 16 * 1024
	maxAgentBaseURLLength     = 2 * 1024
	maxAgentBearerTokenLength = 8 * 1024
)

type temporaryLinkPayload struct {
	BaseURL     string `json:"base_url"`
	BearerToken string `json:"bearer_token"`
}

// TemporaryLinkResolver is the phase-one compatibility boundary for the
// Agent's current URL + Bearer Token configuration. The service builds the
// complete sgta1 value internally. A future Agent enrollment-key resolver can
// replace this implementation without changing controllers or stored clusters.
type TemporaryLinkResolver struct{}

func BuildTemporaryLinkSecret(agentAddress string, bearerToken string) (string, error) {
	agentAddress = strings.TrimSpace(agentAddress)
	bearerToken = strings.TrimSpace(bearerToken)
	if agentAddress == "" || len(agentAddress) > maxAgentBaseURLLength ||
		bearerToken == "" || len(bearerToken) > maxAgentBearerTokenLength {
		return "", ErrInvalidLinkSecret
	}
	if !strings.Contains(agentAddress, "://") {
		agentAddress = "http://" + agentAddress
	}

	parsed, err := parseAgentBaseURL(agentAddress)
	if err != nil || parsed.Port() == "" || parsed.Path != "" {
		return "", ErrInvalidLinkSecret
	}
	payloadBytes, err := common.Marshal(temporaryLinkPayload{
		BaseURL:     parsed.String(),
		BearerToken: bearerToken,
	})
	if err != nil {
		return "", err
	}
	linkSecret := temporaryLinkSecretPrefix + base64.RawURLEncoding.EncodeToString(payloadBytes)
	if len(linkSecret) > maxLinkSecretLength {
		return "", ErrInvalidLinkSecret
	}
	return linkSecret, nil
}

func (TemporaryLinkResolver) Resolve(_ context.Context, linkSecret string) (ResolvedAgentConnection, error) {
	linkSecret = strings.TrimSpace(linkSecret)
	if len(linkSecret) <= len(temporaryLinkSecretPrefix) || len(linkSecret) > maxLinkSecretLength {
		return ResolvedAgentConnection{}, ErrInvalidLinkSecret
	}
	if !strings.HasPrefix(linkSecret, temporaryLinkSecretPrefix) {
		return ResolvedAgentConnection{}, ErrInvalidLinkSecret
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(linkSecret, temporaryLinkSecretPrefix))
	if err != nil {
		return ResolvedAgentConnection{}, ErrInvalidLinkSecret
	}
	var payload temporaryLinkPayload
	if err := common.Unmarshal(payloadBytes, &payload); err != nil {
		return ResolvedAgentConnection{}, ErrInvalidLinkSecret
	}

	payload.BaseURL = strings.TrimSpace(payload.BaseURL)
	payload.BearerToken = strings.TrimSpace(payload.BearerToken)
	if payload.BaseURL == "" || len(payload.BaseURL) > maxAgentBaseURLLength ||
		payload.BearerToken == "" || len(payload.BearerToken) > maxAgentBearerTokenLength {
		return ResolvedAgentConnection{}, ErrInvalidLinkSecret
	}

	parsed, err := parseAgentBaseURL(payload.BaseURL)
	if err != nil {
		return ResolvedAgentConnection{}, ErrInvalidLinkSecret
	}

	return ResolvedAgentConnection{
		BaseURL:     strings.TrimRight(parsed.String(), "/"),
		BearerToken: payload.BearerToken,
	}, nil
}

func parseAgentBaseURL(baseURL string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, ErrInvalidLinkSecret
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, ErrInvalidLinkSecret
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawPath = ""
	if parsed.Path == "." {
		parsed.Path = ""
	}
	return parsed, nil
}

type AESGCMSecretProtector struct {
	key [sha256.Size]byte
}

func NewAESGCMSecretProtector(secret string) (*AESGCMSecretProtector, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, errors.New("cluster secret protection key is empty")
	}
	return &AESGCMSecretProtector{key: sha256.Sum256([]byte(secret))}, nil
}

func (protector *AESGCMSecretProtector) Protect(plaintext string) (string, error) {
	if protector == nil || plaintext == "" {
		return "", errors.New("cluster secret is empty")
	}
	block, err := aes.NewCipher(protector.key[:])
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := aead.Seal(nil, nonce, []byte(plaintext), []byte(encryptedSecretPrefix))
	payload := append(nonce, sealed...)
	return encryptedSecretPrefix + base64.RawURLEncoding.EncodeToString(payload), nil
}

func (protector *AESGCMSecretProtector) Unprotect(ciphertext string) (string, error) {
	if protector == nil || !strings.HasPrefix(ciphertext, encryptedSecretPrefix) {
		return "", errors.New("invalid encrypted cluster secret")
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(ciphertext, encryptedSecretPrefix))
	if err != nil {
		return "", errors.New("invalid encrypted cluster secret")
	}
	block, err := aes.NewCipher(protector.key[:])
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(payload) <= aead.NonceSize() {
		return "", errors.New("invalid encrypted cluster secret")
	}
	plaintext, err := aead.Open(nil, payload[:aead.NonceSize()], payload[aead.NonceSize():], []byte(encryptedSecretPrefix))
	if err != nil {
		return "", errors.New("unable to decrypt cluster secret")
	}
	return string(plaintext), nil
}
