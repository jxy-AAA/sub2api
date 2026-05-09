package service

import (
	"context"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

type AffiliateDistributionUsageSettlementCommand struct {
	UsageLogID     int64
	UserID         int64
	Model          string
	RequestedModel string
	TotalCost      float64
	ActualCost     float64
	UsageCreatedAt time.Time
}

type AffiliateDistributionUsageSettlementService interface {
	SettleUsage(ctx context.Context, cmd AffiliateDistributionUsageSettlementCommand) (applied bool, err error)
}

type AffiliateDistributionUsageSettlementStore interface {
	TryBeginUsageSettlement(ctx context.Context, usageLogID int64) (bool, error)
	MarkUsageSettlementDone(ctx context.Context, usageLogID int64) error
	MarkUsageSettlementFailed(ctx context.Context, usageLogID int64) error
}

type AffiliateDistributionUsageSettlementProcessor interface {
	SettleUsageDistribution(ctx context.Context, cmd AffiliateDistributionUsageSettlementCommand) error
}

type AffiliateDistributionSettlementService struct {
	store     AffiliateDistributionUsageSettlementStore
	processor AffiliateDistributionUsageSettlementProcessor
}

func NewAffiliateDistributionSettlementService(
	store AffiliateDistributionUsageSettlementStore,
	processor AffiliateDistributionUsageSettlementProcessor,
) *AffiliateDistributionSettlementService {
	return &AffiliateDistributionSettlementService{
		store:     store,
		processor: processor,
	}
}

func (s *AffiliateDistributionSettlementService) SettleUsage(ctx context.Context, cmd AffiliateDistributionUsageSettlementCommand) (bool, error) {
	if s == nil || s.store == nil || s.processor == nil || cmd.UsageLogID <= 0 {
		return false, nil
	}

	claimed, err := s.store.TryBeginUsageSettlement(ctx, cmd.UsageLogID)
	if err != nil || !claimed {
		return false, err
	}

	if err := s.processor.SettleUsageDistribution(ctx, cmd); err != nil {
		_ = s.store.MarkUsageSettlementFailed(ctx, cmd.UsageLogID)
		return false, err
	}
	if err := s.store.MarkUsageSettlementDone(ctx, cmd.UsageLogID); err != nil {
		_ = s.store.MarkUsageSettlementFailed(ctx, cmd.UsageLogID)
		return false, err
	}
	return true, nil
}

func (s *GatewayService) SetAffiliateDistributionUsageSettlementService(settlementService AffiliateDistributionUsageSettlementService) {
	if s != nil {
		s.affiliateDistributionSettlementService = settlementService
	}
}

func (s *OpenAIGatewayService) SetAffiliateDistributionUsageSettlementService(settlementService AffiliateDistributionUsageSettlementService) {
	if s != nil {
		s.affiliateDistributionSettlementService = settlementService
	}
}

func settleAffiliateDistributionUsageBestEffort(
	ctx context.Context,
	settlementService AffiliateDistributionUsageSettlementService,
	usageLog *UsageLog,
	logKey string,
) {
	if settlementService == nil || usageLog == nil || usageLog.ID <= 0 {
		return
	}

	settlementCtx, cancel := detachedBillingContext(ctx)
	defer cancel()

	cmd := AffiliateDistributionUsageSettlementCommand{
		UsageLogID:     usageLog.ID,
		UserID:         usageLog.UserID,
		Model:          strings.TrimSpace(usageLog.Model),
		RequestedModel: strings.TrimSpace(usageLog.RequestedModel),
		TotalCost:      usageLog.TotalCost,
		ActualCost:     usageLog.ActualCost,
		UsageCreatedAt: usageLog.CreatedAt,
	}

	if _, err := settlementService.SettleUsage(settlementCtx, cmd); err != nil {
		logger.LegacyPrintf(logKey, "Affiliate distribution usage settlement failed: usage_log_id=%d err=%v", usageLog.ID, err)
	}
}
