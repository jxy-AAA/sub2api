//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOpenAIGatewayServiceRecordUsage_SettlesAffiliateDistributionAfterUsageLogPersist(t *testing.T) {
	usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
	billingRepo := &openAIRecordUsageBillingRepoStub{result: &UsageBillingApplyResult{Applied: true}}
	settlementSvc := &gatewayAffiliateDistributionSettlementStub{}
	svc := newOpenAIRecordUsageServiceWithBillingRepoForTest(usageRepo, billingRepo, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{}, nil)
	svc.SetAffiliateDistributionUsageSettlementService(settlementSvc)

	err := svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{
		Result: &OpenAIForwardResult{
			RequestID: "resp_affiliate_distribution",
			Usage: OpenAIUsage{
				InputTokens:  8,
				OutputTokens: 2,
			},
			Model:    "gpt-5.1",
			Duration: time.Second,
		},
		APIKey:  &APIKey{ID: 1000, Group: &Group{RateMultiplier: 1}},
		User:    &User{ID: 2000},
		Account: &Account{ID: 3000},
	})

	require.NoError(t, err)
	require.Equal(t, 1, settlementSvc.calls)
	require.NotNil(t, usageRepo.lastLog)
	require.Equal(t, usageRepo.lastLog.ID, settlementSvc.last.UsageLogID)
	require.Equal(t, usageRepo.lastLog.UserID, settlementSvc.last.UserID)
	require.Equal(t, usageRepo.lastLog.TotalCost, settlementSvc.last.TotalCost)
	require.Equal(t, usageRepo.lastLog.ActualCost, settlementSvc.last.ActualCost)
	require.Equal(t, usageRepo.lastLog.Model, settlementSvc.last.Model)
	require.Equal(t, usageRepo.lastLog.RequestedModel, settlementSvc.last.RequestedModel)
}
