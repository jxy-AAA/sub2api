package middleware

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// RateLimitFailureMode Redis 故障策略
type RateLimitFailureMode int

const (
	RateLimitFailureModeUnset RateLimitFailureMode = iota
	RateLimitFailClose
	RateLimitFailOpen
)

// RateLimitOptions 限流可选配置
type RateLimitOptions struct {
	FailureMode RateLimitFailureMode
}

var rateLimitScript = redis.NewScript(`
local current = redis.call('INCR', KEYS[1])
local ttl = redis.call('PTTL', KEYS[1])
local repaired = 0
if current == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
elseif ttl == -1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
  repaired = 1
end
return {current, repaired}
`)

// rateLimitRun 允许测试覆盖脚本执行逻辑
var rateLimitRun = func(ctx context.Context, client *redis.Client, key string, windowMillis int64) (int64, bool, error) {
	values, err := rateLimitScript.Run(ctx, client, []string{key}, windowMillis).Slice()
	if err != nil {
		return 0, false, err
	}
	if len(values) < 2 {
		return 0, false, fmt.Errorf("rate limit script returned %d values", len(values))
	}
	count, err := parseInt64(values[0])
	if err != nil {
		return 0, false, err
	}
	repaired, err := parseInt64(values[1])
	if err != nil {
		return 0, false, err
	}
	return count, repaired == 1, nil
}

// RateLimiter Redis 速率限制器
type RateLimiter struct {
	redis  *redis.Client
	prefix string
}

// NewRateLimiter 创建速率限制器实例
func NewRateLimiter(redisClient *redis.Client) *RateLimiter {
	return &RateLimiter{
		redis:  redisClient,
		prefix: "rate_limit:",
	}
}

// Limit 返回速率限制中间件
func (r *RateLimiter) Limit(key string, limit int, window time.Duration) gin.HandlerFunc {
	return r.LimitWithOptions(key, limit, window, RateLimitOptions{})
}

// LimitWithOptions 返回速率限制中间件（带可选配置）
func (r *RateLimiter) LimitWithOptions(key string, limit int, window time.Duration, opts RateLimitOptions) gin.HandlerFunc {
	failureMode := resolveFailureMode(opts)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		redisKey := r.prefix + key + ":" + ip

		windowMillis := windowTTLMillis(window)
		count, repaired, err := rateLimitRun(c.Request.Context(), r.redis, redisKey, windowMillis)
		if err != nil {
			logger.L().Warn("rate limit redis backend error",
				zap.String("key", redisKey),
				zap.String("failure_mode", failureModeLabel(failureMode)),
				zap.Error(err),
			)
			if failureMode == RateLimitFailClose {
				abortRateLimitUnavailable(c)
				return
			}
			c.Next()
			return
		}
		if repaired {
			logger.L().Info("rate limit ttl repaired",
				zap.String("key", redisKey),
				zap.Int64("window_ms", windowMillis),
			)
		}

		if count > int64(limit) {
			abortRateLimitExceeded(c)
			return
		}

		c.Next()
	}
}

func windowTTLMillis(window time.Duration) int64 {
	ttl := window.Milliseconds()
	if ttl < 1 {
		return 1
	}
	return ttl
}

func abortRateLimitExceeded(c *gin.Context) {
	response.AbortWithDetails(c, 429, "Too many requests, please try again later", "RATE_LIMIT_EXCEEDED", nil)
}

func abortRateLimitUnavailable(c *gin.Context) {
	response.AbortWithDetails(c, 429, "Request throttling is temporarily unavailable. Please retry later", "RATE_LIMIT_BACKEND_UNAVAILABLE", nil)
}

func resolveFailureMode(opts RateLimitOptions) RateLimitFailureMode {
	switch opts.FailureMode {
	case RateLimitFailClose, RateLimitFailOpen:
		return opts.FailureMode
	}

	switch normalizeFailureModeName(viper.GetString("rate_limit.redis_failure_mode")) {
	case RateLimitFailOpen:
		return RateLimitFailOpen
	case RateLimitFailClose:
		return RateLimitFailClose
	}

	for _, envName := range []string{"RATE_LIMIT_REDIS_FAILURE_MODE", "SUB2API_RATE_LIMIT_REDIS_FAILURE_MODE"} {
		switch normalizeFailureModeName(os.Getenv(envName)) {
		case RateLimitFailOpen:
			return RateLimitFailOpen
		case RateLimitFailClose:
			return RateLimitFailClose
		}
	}

	return RateLimitFailClose
}

func normalizeFailureModeName(raw string) RateLimitFailureMode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "fail-open", "fail_open", "open":
		return RateLimitFailOpen
	case "fail-close", "fail_close", "close", "":
		return RateLimitFailClose
	default:
		return RateLimitFailureModeUnset
	}
}

func failureModeLabel(mode RateLimitFailureMode) string {
	if mode == RateLimitFailClose {
		return "fail-close"
	}
	return "fail-open"
}

func parseInt64(value any) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, err
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("unexpected value type %T", value)
	}
}
