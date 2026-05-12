//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type gatewayAffiliateDistributionSettlementStub struct {
	calls int
	last  AffiliateDistributionUsageSettlementCommand
	err   error
}

func (s *gatewayAffiliateDistributionSettlementStub) SettleUsage(ctx context.Context, cmd AffiliateDistributionUsageSettlementCommand) (bool, error) {
	s.calls++
	s.last = cmd
	if s.err != nil {
		return false, s.err
	}
	return true, nil
}

func TestGatewayServiceRecordUsage_SettlesAffiliateDistributionAfterUsageLogPersist(t *testing.T) {
	usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
	billingRepo := &openAIRecordUsageBillingRepoStub{result: &UsageBillingApplyResult{Applied: true}}
	settlementSvc := &gatewayAffiliateDistributionSettlementStub{}
	groupID := int64(801)
	svc := newGatewayRecordUsageServiceWithBillingRepoForTest(usageRepo, billingRepo, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{})
	svc.SetAffiliateDistributionUsageSettlementService(settlementSvc)

	err := svc.RecordUsage(context.Background(), &RecordUsageInput{
		Result: &ForwardResult{
			RequestID: "gateway_affiliate_distribution",
			Usage: ClaudeUsage{
				InputTokens:  10,
				OutputTokens: 6,
			},
			Model:    "claude-sonnet-4",
			Duration: time.Second,
		},
		APIKey:  &APIKey{ID: 501, GroupID: &groupID, Quota: 100},
		User:    &User{ID: 601},
		Account: &Account{ID: 701},
	})

	require.NoError(t, err)
	require.Equal(t, 1, settlementSvc.calls)
	require.NotNil(t, usageRepo.lastLog)
	require.Equal(t, usageRepo.lastLog.ID, settlementSvc.last.UsageLogID)
	require.Equal(t, usageRepo.lastLog.UserID, settlementSvc.last.UserID)
	require.Equal(t, groupID, settlementSvc.last.GroupID)
	require.Equal(t, usageRepo.lastLog.TotalCost, settlementSvc.last.TotalCost)
	require.Equal(t, usageRepo.lastLog.ActualCost, settlementSvc.last.ActualCost)
	require.Equal(t, usageRepo.lastLog.Model, settlementSvc.last.Model)
	require.Equal(t, usageRepo.lastLog.RequestedModel, settlementSvc.last.RequestedModel)
}
