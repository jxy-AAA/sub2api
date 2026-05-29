package routes

import (
	"context"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type HealthDependencies struct {
	SettingService *service.SettingService
	RedisClient    *redis.Client
	Postgres       routeHealthChecker
	Migrations     healthMigrationChecker
	Redis          routeHealthChecker
	Timeout        time.Duration
}

type redisReadinessChecker struct {
	client      *redis.Client
	pingTimeout time.Duration
}

func (c redisReadinessChecker) Check(ctx context.Context) error {
	if c.client == nil {
		return context.DeadlineExceeded
	}
	timeout := c.pingTimeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return c.client.Ping(pingCtx).Err()
}

func RegisterCommonRoutes(r *gin.Engine, deps HealthDependencies) {
	timeout := deps.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}

	postgres := deps.Postgres
	if postgres == nil {
		postgres = postgresReadinessChecker{
			source:           deps.SettingService,
			pingTimeout:      timeout,
			migrationChecker: deps.Migrations,
		}
	}
	redisChecker := deps.Redis
	if redisChecker == nil {
		redisChecker = redisReadinessChecker{
			client:      deps.RedisClient,
			pingTimeout: timeout,
		}
	}

	r.GET("/livez", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		checks := gin.H{
			"postgres":   "ok",
			"redis":      "ok",
			"migrations": "ok",
		}
		status := "ok"
		statusCode := http.StatusOK

		if err := postgres.Check(ctx); err != nil {
			checks["postgres"] = err.Error()
			checks["migrations"] = err.Error()
			status = "degraded"
			statusCode = http.StatusServiceUnavailable
		}
		if err := redisChecker.Check(ctx); err != nil {
			checks["redis"] = err.Error()
			status = "degraded"
			statusCode = http.StatusServiceUnavailable
		}

		c.JSON(statusCode, gin.H{
			"status": status,
			"checks": checks,
		})
	})

	r.POST("/api/event_logging/batch", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	r.GET("/setup/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"needs_setup": false,
				"step":        "completed",
			},
		})
	})
}
