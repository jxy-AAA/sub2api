package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type affiliateDistributionSettlementStoreStub struct {
	claims map[int64]bool
	done   []int64
}

func (s *affiliateDistributionSettlementStoreStub) TryBeginUsageSettlement(ctx context.Context, usageLogID int64) (bool, error) {
	if s.claims == nil {
		s.claims = make(map[int64]bool)
	}
	if s.claims[usageLogID] {
		return false, nil
	}
	s.claims[usageLogID] = true
	return true, nil
}

func (s *affiliateDistributionSettlementStoreStub) MarkUsageSettlementDone(ctx context.Context, usageLogID int64) error {
	s.done = append(s.done, usageLogID)
	return nil
}

func (s *affiliateDistributionSettlementStoreStub) MarkUsageSettlementFailed(ctx context.Context, usageLogID int64) error {
	return nil
}

type affiliateDistributionSettlementProcessorStub struct {
	calls int
	last  AffiliateDistributionUsageSettlementCommand
	err   error
}

func (s *affiliateDistributionSettlementProcessorStub) SettleUsageDistribution(ctx context.Context, cmd AffiliateDistributionUsageSettlementCommand) error {
	s.calls++
	s.last = cmd
	return s.err
}

func TestAffiliateDistributionSettlementService_Idempotent(t *testing.T) {
	store := &affiliateDistributionSettlementStoreStub{}
	processor := &affiliateDistributionSettlementProcessorStub{}
	svc := NewAffiliateDistributionSettlementService(store, processor)

	cmd := AffiliateDistributionUsageSettlementCommand{
		UsageLogID:     101,
		UserID:         202,
		Model:          "gpt-5.1",
		RequestedModel: "gpt-5.1",
		TotalCost:      200,
		ActualCost:     40,
		UsageCreatedAt: time.Unix(123, 0),
	}

	applied, err := svc.SettleUsage(context.Background(), cmd)
	require.NoError(t, err)
	require.True(t, applied)

	applied, err = svc.SettleUsage(context.Background(), cmd)
	require.NoError(t, err)
	require.False(t, applied)
	require.Equal(t, 1, processor.calls)
	require.Equal(t, int64(101), processor.last.UsageLogID)
	require.Equal(t, []int64{101}, store.done)
}

func TestAffiliateDistributionSettlementService_SkipsDuplicatesWithoutProcessing(t *testing.T) {
	store := &affiliateDistributionSettlementStoreStub{}
	processor := &affiliateDistributionSettlementProcessorStub{}
	svc := NewAffiliateDistributionSettlementService(store, processor)

	_, err := svc.SettleUsage(context.Background(), AffiliateDistributionUsageSettlementCommand{UsageLogID: 1})
	require.NoError(t, err)
	_, err = svc.SettleUsage(context.Background(), AffiliateDistributionUsageSettlementCommand{UsageLogID: 1})
	require.NoError(t, err)
	require.Equal(t, 1, processor.calls)
}
