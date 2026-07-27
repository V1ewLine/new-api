package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/logger"
	"github.com/gin-gonic/gin"
)

const (
	redisRateLimitNamespace     = "rateLimit:v2"
	loginIPRateLimitMark        = "LGI"
	loginAccountRateLimitMark   = "LGA"
	passwordLoginAPIRequestPath = "/api/user/login"
)

// Redis rate limiting intentionally uses a fixed window. The single Lua script
// makes increment, expiry, and the limit decision atomic, while retaining the
// simple fixed-window behavior: traffic at a window boundary can burst up to
// twice the configured limit. Do not replace this with a sliding-window ZSET
// unless that externally visible behavior is intentionally changed.
const redisFixedWindowScript = `
local count = redis.call('INCR', KEYS[1])
if count == 1 then
  redis.call('EXPIRE', KEYS[1], ARGV[2])
end
local ttl = redis.call('TTL', KEYS[1])
if ttl < 0 then
  redis.call('EXPIRE', KEYS[1], ARGV[2])
  ttl = redis.call('TTL', KEYS[1])
end
if count > tonumber(ARGV[1]) then
  return {0, count, ttl}
end
return {1, count, ttl}
`

var inMemoryRateLimiter common.InMemoryRateLimiter

var defNext = func(c *gin.Context) {
	c.Next()
}

func redisIPRateLimitKey(mark string, clientIP string) string {
	return fmt.Sprintf("%s:ip:%s:%s", redisRateLimitNamespace, mark, clientIP)
}

func redisUserRateLimitKey(mark string, userID int) string {
	return fmt.Sprintf("%s:user:%s:%d", redisRateLimitNamespace, mark, userID)
}

func loginAccountRateLimitKey(clientIP string, username string) string {
	normalizedUsername := strings.ToLower(strings.TrimSpace(username))
	identityHash := sha256.Sum256([]byte(clientIP + "\x00" + normalizedUsername))
	return fmt.Sprintf("%s:login:%s:%x", redisRateLimitNamespace, loginAccountRateLimitMark, identityHash)
}

func isPasswordLoginAPIRequest(c *gin.Context) bool {
	return c.Request.Method == http.MethodPost && c.Request.URL.Path == passwordLoginAPIRequestPath
}

func redisReplyInteger(value interface{}) (int64, error) {
	switch typed := value.(type) {
	case int64:
		return typed, nil
	case string:
		return strconv.ParseInt(typed, 10, 64)
	case []byte:
		return strconv.ParseInt(string(typed), 10, 64)
	default:
		return 0, fmt.Errorf("unexpected Redis integer reply type %T", value)
	}
}

func redisFixedWindowTake(ctx context.Context, key string, maxRequestNum int, duration int64) (bool, int64, int64, error) {
	if common.RDB == nil {
		return false, 0, 0, errors.New("Redis client is not initialized")
	}
	if key == "" {
		return false, 0, 0, errors.New("rate limit key is empty")
	}
	if maxRequestNum <= 0 {
		return false, 0, 0, errors.New("rate limit maximum must be positive")
	}
	if duration <= 0 {
		return false, 0, 0, errors.New("rate limit duration must be positive")
	}

	values, err := common.RDB.Eval(
		ctx,
		redisFixedWindowScript,
		[]string{key},
		maxRequestNum,
		duration,
	).Slice()
	if err != nil {
		return false, 0, 0, err
	}
	if len(values) != 3 {
		return false, 0, 0, fmt.Errorf("unexpected Redis rate limit reply length %d", len(values))
	}

	allowedValue, err := redisReplyInteger(values[0])
	if err != nil {
		return false, 0, 0, err
	}
	count, err := redisReplyInteger(values[1])
	if err != nil {
		return false, 0, 0, err
	}
	ttlSeconds, err := redisReplyInteger(values[2])
	if err != nil {
		return false, 0, 0, err
	}

	return allowedValue == 1, count, ttlSeconds, nil
}

func redisRateLimiter(c *gin.Context, maxRequestNum int, duration int64, mark string) {
	allowed, _, ttlSeconds, err := redisFixedWindowTake(
		c.Request.Context(),
		redisIPRateLimitKey(mark, c.ClientIP()),
		maxRequestNum,
		duration,
	)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("rate limit check failed (mark=%s): %v", mark, err))
		c.Status(http.StatusInternalServerError)
		c.Abort()
		return
	}
	if !allowed {
		writeRateLimited(c, ttlSeconds)
	}
}

func memoryRateLimiter(c *gin.Context, maxRequestNum int, duration int64, mark string) {
	key := mark + c.ClientIP()
	if !inMemoryRateLimiter.Request(key, maxRequestNum, duration) {
		writeRateLimited(c, duration)
		return
	}
}

// writeRateLimited rejects the request with 429 and a Retry-After hint so
// clients can back off instead of treating the rejection as a fatal error.
// The in-memory limiter cannot report the remaining window, so callers
// without a TTL pass the full window duration as a conservative upper bound.
func writeRateLimited(c *gin.Context, retryAfterSeconds int64) {
	if retryAfterSeconds > 0 {
		c.Header("Retry-After", strconv.FormatInt(retryAfterSeconds, 10))
	}
	c.Status(http.StatusTooManyRequests)
	c.Abort()
}

func rateLimitFactory(maxRequestNum int, duration int64, mark string) func(c *gin.Context) {
	if common.RedisEnabled {
		return func(c *gin.Context) {
			redisRateLimiter(c, maxRequestNum, duration, mark)
		}
	}
	// It's safe to call multi times.
	inMemoryRateLimiter.Init(common.RateLimitKeyExpirationDuration)
	return func(c *gin.Context) {
		memoryRateLimiter(c, maxRequestNum, duration, mark)
	}
}

func GlobalWebRateLimit() func(c *gin.Context) {
	if common.GlobalWebRateLimitEnable {
		return rateLimitFactory(common.GlobalWebRateLimitNum, common.GlobalWebRateLimitDuration, "GW")
	}
	return defNext
}

func GlobalAPIRateLimit() func(c *gin.Context) {
	if common.GlobalApiRateLimitEnable {
		globalLimiter := rateLimitFactory(common.GlobalApiRateLimitNum, common.GlobalApiRateLimitDuration, "GA")
		return func(c *gin.Context) {
			if isPasswordLoginAPIRequest(c) {
				c.Next()
				return
			}
			globalLimiter(c)
		}
	}
	return defNext
}

// LoginRateLimit keeps password login independent from the shared critical
// operation bucket. A broader IP bucket still protects the endpoint from
// username spraying, while the account bucket prevents one user behind a
// reverse proxy from consuming every other user's login allowance.
func LoginRateLimit() gin.HandlerFunc {
	inMemoryRateLimiter.Init(common.RateLimitKeyExpirationDuration)

	return func(c *gin.Context) {
		if common.GlobalApiRateLimitEnable {
			allowed, retryAfterSeconds, err := takeRateLimit(
				c,
				redisIPRateLimitKey(loginIPRateLimitMark, c.ClientIP()),
				common.GlobalApiRateLimitNum,
				common.GlobalApiRateLimitDuration,
			)
			if err != nil {
				logger.LogError(c.Request.Context(), fmt.Sprintf("login IP rate limit check failed: %v", err))
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
			if !allowed {
				writeLoginRateLimited(c, retryAfterSeconds)
				return
			}
		}

		if !common.CriticalRateLimitEnable {
			c.Next()
			return
		}

		username := readLoginRateLimitUsername(c)
		if username == "" {
			c.Next()
			return
		}

		allowed, retryAfterSeconds, err := takeRateLimit(
			c,
			loginAccountRateLimitKey(c.ClientIP(), username),
			common.CriticalRateLimitNum,
			common.CriticalRateLimitDuration,
		)
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("login account rate limit check failed: %v", err))
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if !allowed {
			writeLoginRateLimited(c, retryAfterSeconds)
			return
		}

		c.Next()
	}
}

func readLoginRateLimitUsername(c *gin.Context) string {
	if c.Request.Body == nil {
		return ""
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return ""
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))

	var request struct {
		Username string `json:"username"`
	}
	if err := common.Unmarshal(body, &request); err != nil {
		return ""
	}
	return strings.TrimSpace(request.Username)
}

func takeRateLimit(c *gin.Context, key string, maxRequestNum int, duration int64) (bool, int64, error) {
	if common.RedisEnabled {
		allowed, _, retryAfterSeconds, err := redisFixedWindowTake(
			c.Request.Context(),
			key,
			maxRequestNum,
			duration,
		)
		return allowed, retryAfterSeconds, err
	}

	if inMemoryRateLimiter.Request(key, maxRequestNum, duration) {
		return true, duration, nil
	}
	return false, duration, nil
}

func writeLoginRateLimited(c *gin.Context, retryAfterSeconds int64) {
	if retryAfterSeconds <= 0 {
		retryAfterSeconds = 1
	}
	c.Header("Retry-After", strconv.FormatInt(retryAfterSeconds, 10))
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
		"success": false,
		"code":    "LOGIN_RATE_LIMITED",
		"message": common.TranslateMessage(c, i18n.MsgLoginRateLimitReached, map[string]any{
			"Seconds": retryAfterSeconds,
		}),
	})
}

func CriticalRateLimit() func(c *gin.Context) {
	if common.CriticalRateLimitEnable {
		return rateLimitFactory(common.CriticalRateLimitNum, common.CriticalRateLimitDuration, "CT")
	}
	return defNext
}

func DownloadRateLimit() func(c *gin.Context) {
	return rateLimitFactory(common.DownloadRateLimitNum, common.DownloadRateLimitDuration, "DW")
}

func UploadRateLimit() func(c *gin.Context) {
	return rateLimitFactory(common.UploadRateLimitNum, common.UploadRateLimitDuration, "UP")
}

// userRateLimitFactory creates a rate limiter keyed by authenticated user ID
// instead of client IP, making it resistant to proxy rotation attacks.
// Must be used AFTER authentication middleware (UserAuth).
func userRateLimitFactory(maxRequestNum int, duration int64, mark string) func(c *gin.Context) {
	if common.RedisEnabled {
		return func(c *gin.Context) {
			userID := c.GetInt("id")
			if userID == 0 {
				c.Status(http.StatusUnauthorized)
				c.Abort()
				return
			}
			userRedisRateLimiter(c, maxRequestNum, duration, redisUserRateLimitKey(mark, userID))
		}
	}
	// It's safe to call multi times.
	inMemoryRateLimiter.Init(common.RateLimitKeyExpirationDuration)
	return func(c *gin.Context) {
		userID := c.GetInt("id")
		if userID == 0 {
			c.Status(http.StatusUnauthorized)
			c.Abort()
			return
		}
		key := fmt.Sprintf("%s:user:%d", mark, userID)
		if !inMemoryRateLimiter.Request(key, maxRequestNum, duration) {
			writeRateLimited(c, duration)
			return
		}
	}
}

// userRedisRateLimiter is like redisRateLimiter but accepts a pre-built key
// (to support user-ID-based keys).
func userRedisRateLimiter(c *gin.Context, maxRequestNum int, duration int64, key string) {
	allowed, _, ttlSeconds, err := redisFixedWindowTake(c.Request.Context(), key, maxRequestNum, duration)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("rate limit check failed (key=%s): %v", key, err))
		c.Status(http.StatusInternalServerError)
		c.Abort()
		return
	}
	if !allowed {
		writeRateLimited(c, ttlSeconds)
	}
}

// SearchRateLimit returns a per-user rate limiter for search endpoints.
// Configurable via SEARCH_RATE_LIMIT_ENABLE / SEARCH_RATE_LIMIT / SEARCH_RATE_LIMIT_DURATION.
func SearchRateLimit() func(c *gin.Context) {
	if !common.SearchRateLimitEnable {
		return defNext
	}
	return userRateLimitFactory(common.SearchRateLimitNum, common.SearchRateLimitDuration, "SR")
}
