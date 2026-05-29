//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/server"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
)

type Application struct {
	Server  *http.Server
	Cleanup func(context.Context)
}

type cleanupStep struct {
	name string
	fn   func() error
}

type CleanupRegistry struct {
	parallel   []cleanupStep
	sequential []cleanupStep
}

func (r *CleanupRegistry) AddParallel(name string, fn func() error) {
	if fn == nil {
		return
	}
	r.parallel = append(r.parallel, cleanupStep{name: name, fn: fn})
}

func (r *CleanupRegistry) AddSequential(name string, fn func() error) {
	if fn == nil {
		return
	}
	r.sequential = append(r.sequential, cleanupStep{name: name, fn: fn})
}

func (r *CleanupRegistry) Run(ctx context.Context) {
	if !runCleanupParallel(ctx, r.parallel) {
		return
	}
	if !runCleanupSequential(ctx, r.sequential) {
		return
	}
	slog.Info("cleanup steps completed")
}

type cleanupResult struct {
	name string
	err  error
}

func runCleanupParallel(ctx context.Context, steps []cleanupStep) bool {
	if len(steps) == 0 {
		return true
	}
	if err := ctx.Err(); err != nil {
		slog.Warn("cleanup skipped before parallel steps", "error", err)
		return false
	}

	results := make(chan cleanupResult, len(steps))
	pending := make(map[string]struct{}, len(steps))

	for i := range steps {
		step := steps[i]
		pending[step.name] = struct{}{}
		go func(step cleanupStep) {
			results <- cleanupResult{name: step.name, err: step.fn()}
		}(step)
	}

	for completed := 0; completed < len(steps); completed++ {
		select {
		case result := <-results:
			delete(pending, result.name)
			logCleanupStepResult(result)
		case <-ctx.Done():
			slog.Warn(
				"cleanup timed out while waiting for parallel steps",
				"remaining_steps", joinCleanupStepNames(pending),
				"error", ctx.Err(),
			)
			return false
		}
	}

	return true
}

func runCleanupSequential(ctx context.Context, steps []cleanupStep) bool {
	for i := range steps {
		step := steps[i]
		if err := ctx.Err(); err != nil {
			slog.Warn(
				"cleanup timed out before sequential step",
				"step", step.name,
				"error", err,
			)
			return false
		}

		resultCh := make(chan cleanupResult, 1)
		go func(step cleanupStep) {
			resultCh <- cleanupResult{name: step.name, err: step.fn()}
		}(step)

		select {
		case result := <-resultCh:
			logCleanupStepResult(result)
		case <-ctx.Done():
			slog.Warn(
				"cleanup timed out during sequential step",
				"step", step.name,
				"error", ctx.Err(),
			)
			return false
		}
	}

	return true
}

func logCleanupStepResult(result cleanupResult) {
	if result.err != nil {
		slog.Warn("cleanup step failed", "step", result.name, "error", result.err)
		return
	}
	slog.Info("cleanup step succeeded", "step", result.name)
}

func joinCleanupStepNames(steps map[string]struct{}) string {
	if len(steps) == 0 {
		return ""
	}

	names := make([]string, 0, len(steps))
	for name := range steps {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

type cleanupInfrastructure struct {
	EntClient *ent.Client
	Redis     *redis.Client
}

func (c cleanupInfrastructure) Register(registry *CleanupRegistry) {
	registry.AddSequential("SharedReqClients", func() error {
		repository.CloseSharedReqClients()
		return nil
	})
	registry.AddSequential("SharedHTTPClients", func() error {
		httpclient.CloseSharedClients()
		return nil
	})
	registry.AddSequential("Redis", func() error {
		if c.Redis == nil {
			return nil
		}
		return c.Redis.Close()
	})
	registry.AddSequential("Ent", func() error {
		if c.EntClient == nil {
			return nil
		}
		return c.EntClient.Close()
	})
}

type cleanupServices struct {
	OpsMetricsCollector               *service.OpsMetricsCollector
	OpsAggregationService             *service.OpsAggregationService
	OpsAlertEvaluatorService          *service.OpsAlertEvaluatorService
	OpsCleanupService                 *service.OpsCleanupService
	OpsScheduledReportService         *service.OpsScheduledReportService
	OpsSystemLogSink                  *service.OpsSystemLogSink
	SchedulerSnapshotService          *service.SchedulerSnapshotService
	TokenRefreshService               *service.TokenRefreshService
	AccountExpiryService              *service.AccountExpiryService
	SubscriptionExpiryService         *service.SubscriptionExpiryService
	UsageCleanupService               *service.UsageCleanupService
	IdempotencyCleanupService         *service.IdempotencyCleanupService
	PricingService                    *service.PricingService
	EmailQueueService                 *service.EmailQueueService
	BillingCacheService               *service.BillingCacheService
	UsageRecordWorkerPool             *service.UsageRecordWorkerPool
	SubscriptionService               *service.SubscriptionService
	OAuthService                      *service.OAuthService
	OpenAIOAuthService                *service.OpenAIOAuthService
	GeminiOAuthService                *service.GeminiOAuthService
	AntigravityOAuthService           *service.AntigravityOAuthService
	OpenAIGatewayService              *service.OpenAIGatewayService
	ScheduledTestRunnerService        *service.ScheduledTestRunnerService
	BackupService                     *service.BackupService
	PaymentOrderExpiryService         *service.PaymentOrderExpiryService
	ChannelMonitorRunner              *service.ChannelMonitorRunner
	AffiliateDistributionMonthlyReset *service.AffiliateDistributionMonthlyResetService
}

func (s cleanupServices) Register(registry *CleanupRegistry, ctx context.Context) {
	registry.AddParallel("AffiliateDistributionMonthlyResetService", func() error {
		if s.AffiliateDistributionMonthlyReset != nil {
			s.AffiliateDistributionMonthlyReset.Stop()
		}
		return nil
	})
	registry.AddParallel("OpsScheduledReportService", func() error {
		if s.OpsScheduledReportService != nil {
			s.OpsScheduledReportService.Stop()
		}
		return nil
	})
	registry.AddParallel("OpsCleanupService", func() error {
		if s.OpsCleanupService != nil {
			s.OpsCleanupService.Stop()
		}
		return nil
	})
	registry.AddParallel("OpsSystemLogSink", func() error {
		if s.OpsSystemLogSink != nil {
			s.OpsSystemLogSink.Stop()
		}
		return nil
	})
	registry.AddParallel("OpsAlertEvaluatorService", func() error {
		if s.OpsAlertEvaluatorService != nil {
			s.OpsAlertEvaluatorService.Stop()
		}
		return nil
	})
	registry.AddParallel("OpsAggregationService", func() error {
		if s.OpsAggregationService != nil {
			s.OpsAggregationService.Stop()
		}
		return nil
	})
	registry.AddParallel("OpsMetricsCollector", func() error {
		if s.OpsMetricsCollector != nil {
			s.OpsMetricsCollector.Stop()
		}
		return nil
	})
	registry.AddParallel("SchedulerSnapshotService", func() error {
		if s.SchedulerSnapshotService != nil {
			s.SchedulerSnapshotService.Stop()
		}
		return nil
	})
	registry.AddParallel("UsageCleanupService", func() error {
		if s.UsageCleanupService != nil {
			s.UsageCleanupService.Stop()
		}
		return nil
	})
	registry.AddParallel("IdempotencyCleanupService", func() error {
		if s.IdempotencyCleanupService != nil {
			s.IdempotencyCleanupService.Stop()
		}
		return nil
	})
	registry.AddParallel("TokenRefreshService", func() error {
		if s.TokenRefreshService != nil {
			s.TokenRefreshService.Stop()
		}
		return nil
	})
	registry.AddParallel("AccountExpiryService", func() error {
		if s.AccountExpiryService != nil {
			s.AccountExpiryService.Stop()
		}
		return nil
	})
	registry.AddParallel("SubscriptionExpiryService", func() error {
		if s.SubscriptionExpiryService != nil {
			s.SubscriptionExpiryService.Stop()
		}
		return nil
	})
	registry.AddParallel("SubscriptionService", func() error {
		if s.SubscriptionService != nil {
			s.SubscriptionService.Stop()
		}
		return nil
	})
	registry.AddParallel("PricingService", func() error {
		if s.PricingService != nil {
			s.PricingService.Stop()
		}
		return nil
	})
	registry.AddParallel("EmailQueueService", func() error {
		if s.EmailQueueService != nil {
			s.EmailQueueService.Stop()
		}
		return nil
	})
	registry.AddParallel("BillingCacheService", func() error {
		if s.BillingCacheService != nil {
			s.BillingCacheService.Stop()
		}
		return nil
	})
	registry.AddParallel("UsageRecordWorkerPool", func() error {
		if s.UsageRecordWorkerPool != nil {
			s.UsageRecordWorkerPool.Stop()
		}
		return nil
	})
	registry.AddParallel("OAuthService", func() error {
		if s.OAuthService != nil {
			s.OAuthService.Stop()
		}
		return nil
	})
	registry.AddParallel("OpenAIOAuthService", func() error {
		if s.OpenAIOAuthService != nil {
			s.OpenAIOAuthService.Stop()
		}
		return nil
	})
	registry.AddParallel("GeminiOAuthService", func() error {
		if s.GeminiOAuthService != nil {
			s.GeminiOAuthService.Stop()
		}
		return nil
	})
	registry.AddParallel("AntigravityOAuthService", func() error {
		if s.AntigravityOAuthService != nil {
			s.AntigravityOAuthService.Stop()
		}
		return nil
	})
	registry.AddParallel("OpenAIWSPool", func() error {
		if s.OpenAIGatewayService != nil {
			s.OpenAIGatewayService.CloseOpenAIWSPool()
		}
		return nil
	})
	registry.AddParallel("ScheduledTestRunnerService", func() error {
		if s.ScheduledTestRunnerService != nil {
			s.ScheduledTestRunnerService.Stop()
		}
		return nil
	})
	registry.AddParallel("BackupService", func() error {
		if s.BackupService != nil {
			return s.BackupService.StopWithContext(ctx)
		}
		return nil
	})
	registry.AddParallel("PaymentOrderExpiryService", func() error {
		if s.PaymentOrderExpiryService != nil {
			s.PaymentOrderExpiryService.Stop()
		}
		return nil
	})
	registry.AddParallel("ChannelMonitorRunner", func() error {
		if s.ChannelMonitorRunner != nil {
			s.ChannelMonitorRunner.Stop()
		}
		return nil
	})
}

func initializeApplication(buildInfo handler.BuildInfo) (*Application, error) {
	wire.Build(
		config.ProviderSet,
		repository.ProviderSet,
		service.ProviderSet,
		payment.ProviderSet,
		middleware.ProviderSet,
		handler.ProviderSet,
		server.ProviderSet,
		providePrivacyClientFactory,
		provideServiceBuildInfo,
		wire.Struct(new(cleanupInfrastructure), "*"),
		wire.Struct(new(cleanupServices), "*"),
		provideCleanup,
		wire.Struct(new(Application), "Server", "Cleanup"),
	)
	return nil, nil
}

func providePrivacyClientFactory() service.PrivacyClientFactory {
	return repository.CreatePrivacyReqClient
}

func provideServiceBuildInfo(buildInfo handler.BuildInfo) service.BuildInfo {
	return service.BuildInfo{
		Version:   buildInfo.Version,
		BuildType: buildInfo.BuildType,
	}
}

func provideCleanup(infra cleanupInfrastructure, services cleanupServices) func(context.Context) {
	return func(ctx context.Context) {
		if ctx == nil {
			ctx = context.Background()
		}
		registry := &CleanupRegistry{}
		services.Register(registry, ctx)
		infra.Register(registry)
		registry.Run(ctx)
	}
}
