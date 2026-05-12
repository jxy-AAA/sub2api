package service

import (
	"context"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

var ErrAffiliateDistributionGroupIDRequired = infraerrors.ServiceUnavailable(
	"AFFILIATE_DISTRIBUTION_GROUP_ID_REQUIRED",
	"usage group id unavailable for settlement",
)

var ErrAffiliateDistributionSettlementInvalidInput = infraerrors.BadRequest(
	"AFFILIATE_DISTRIBUTION_SETTLEMENT_INVALID",
	"affiliate distribution settlement input is invalid",
)

var ErrAffiliateDistributionPaidCreditInvalidInput = infraerrors.BadRequest(
	"AFFILIATE_DISTRIBUTION_PAID_CREDIT_INVALID",
	"affiliate distribution paid credit input is invalid",
)

var ErrAffiliateDistributionPaidCreditConflict = infraerrors.Conflict(
	"AFFILIATE_DISTRIBUTION_PAID_CREDIT_CONFLICT",
	"affiliate distribution paid credit conflicts with existing order credit",
)

type AffiliateDistributionUsageSettlementCommand struct {
	UsageLogID     int64
	UserID         int64
	GroupID        int64
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

type AffiliateDistributionPaidCreditRecorder interface {
	RecordPaidCredit(ctx context.Context, userID, sourceOrderID int64, amountUSD float64, creditedAt time.Time) (bool, error)
	ReversePaidCredit(ctx context.Context, userID, sourceOrderID int64, amountUSD float64, reversedAt time.Time) (bool, error)
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
	if s == nil || s.store == nil || s.processor == nil {
		return false, ErrAffiliateDistributionUnavailable
	}
	if cmd.UsageLogID <= 0 || cmd.UserID <= 0 {
		return false, ErrAffiliateDistributionSettlementInvalidInput
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
	if usageLog.GroupID == nil || *usageLog.GroupID <= 0 {
		return
	}

	settlementCtx, cancel := detachedBillingContext(ctx)
	defer cancel()

	cmd := AffiliateDistributionUsageSettlementCommand{
		UsageLogID:     usageLog.ID,
		UserID:         usageLog.UserID,
		GroupID:        *usageLog.GroupID,
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
