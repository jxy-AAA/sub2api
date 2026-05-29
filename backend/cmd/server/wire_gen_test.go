package main

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestProvideServiceBuildInfo(t *testing.T) {
	in := handler.BuildInfo{
		Version:   "v-test",
		BuildType: "release",
	}
	out := provideServiceBuildInfo(in)
	require.Equal(t, in.Version, out.Version)
	require.Equal(t, in.BuildType, out.BuildType)
}

func TestProvideCleanup_WithAggregatedDependencies_NoPanic(t *testing.T) {
	cfg := &config.Config{}

	oauthSvc := service.NewOAuthService(nil, nil)
	openAIOAuthSvc := service.NewOpenAIOAuthService(nil, nil)
	geminiOAuthSvc := service.NewGeminiOAuthService(nil, nil, nil, nil, cfg)
	antigravityOAuthSvc := service.NewAntigravityOAuthService(nil)

	tokenRefreshSvc := service.NewTokenRefreshService(
		nil,
		oauthSvc,
		openAIOAuthSvc,
		geminiOAuthSvc,
		antigravityOAuthSvc,
		nil,
		nil,
		cfg,
		nil,
	)
	accountExpirySvc := service.NewAccountExpiryService(nil, time.Second)
	subscriptionExpirySvc := service.NewSubscriptionExpiryService(nil, time.Second)
	pricingSvc := service.NewPricingService(cfg, nil)
	emailQueueSvc := service.NewEmailQueueService(nil, 1)
	billingCacheSvc := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg)
	idempotencyCleanupSvc := service.NewIdempotencyCleanupService(nil, cfg)
	schedulerSnapshotSvc := service.NewSchedulerSnapshotService(nil, nil, nil, nil, cfg)
	opsSystemLogSinkSvc := service.NewOpsSystemLogSink(nil)

	cleanupInfra := cleanupInfrastructure{}
	cleanupSvcSet := cleanupServices{
		OpsMetricsCollector:       &service.OpsMetricsCollector{},
		OpsAggregationService:     &service.OpsAggregationService{},
		OpsAlertEvaluatorService:  &service.OpsAlertEvaluatorService{},
		OpsCleanupService:         &service.OpsCleanupService{},
		OpsScheduledReportService: &service.OpsScheduledReportService{},
		OpsSystemLogSink:          opsSystemLogSinkSvc,
		SchedulerSnapshotService:  schedulerSnapshotSvc,
		TokenRefreshService:       tokenRefreshSvc,
		AccountExpiryService:      accountExpirySvc,
		SubscriptionExpiryService: subscriptionExpirySvc,
		UsageCleanupService:       &service.UsageCleanupService{},
		IdempotencyCleanupService: idempotencyCleanupSvc,
		PricingService:            pricingSvc,
		EmailQueueService:         emailQueueSvc,
		BillingCacheService:       billingCacheSvc,
		UsageRecordWorkerPool:     &service.UsageRecordWorkerPool{},
		SubscriptionService:       &service.SubscriptionService{},
		OAuthService:              oauthSvc,
		OpenAIOAuthService:        openAIOAuthSvc,
		GeminiOAuthService:        geminiOAuthSvc,
		AntigravityOAuthService:   antigravityOAuthSvc,
	}

	cleanup := provideCleanup(cleanupInfra, cleanupSvcSet)

	require.NotPanics(t, func() {
		cleanup(context.Background())
	})
}

func TestCleanupRegistry_RunExecutesSequentialAfterParallel(t *testing.T) {
	registry := &CleanupRegistry{}
	parallelDone := make(chan struct{})
	sequentialObservedDone := false

	registry.AddParallel("parallel", func() error {
		close(parallelDone)
		return nil
	})
	registry.AddSequential("sequential", func() error {
		select {
		case <-parallelDone:
			sequentialObservedDone = true
		default:
			sequentialObservedDone = false
		}
		return nil
	})

	registry.Run(context.Background())

	require.True(t, sequentialObservedDone)
}
