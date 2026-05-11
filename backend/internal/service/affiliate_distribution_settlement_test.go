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

type affiliateDistributionSettlementServiceStub struct {
	calls int
	last  AffiliateDistributionUsageSettlementCommand
}

func (s *affiliateDistributionSettlementServiceStub) SettleUsage(ctx context.Context, cmd AffiliateDistributionUsageSettlementCommand) (bool, error) {
	s.calls++
	s.last = cmd
	return true, nil
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

func TestSettleAffiliateDistributionUsageBestEffortSkipsUngroupedUsage(t *testing.T) {
	settlement := &affiliateDistributionSettlementServiceStub{}

	settleAffiliateDistributionUsageBestEffort(context.Background(), settlement, &UsageLog{
		ID:        101,
		UserID:    202,
		GroupID:   nil,
		Model:     "gpt-5.1",
		TotalCost: 10,
		CreatedAt: time.Unix(123, 0),
	}, "service.test")

	require.Equal(t, 0, settlement.calls)
}

func TestSettleAffiliateDistributionUsageBestEffortPassesGroupedUsage(t *testing.T) {
	settlement := &affiliateDistributionSettlementServiceStub{}
	groupID := int64(303)

	settleAffiliateDistributionUsageBestEffort(context.Background(), settlement, &UsageLog{
		ID:             101,
		UserID:         202,
		GroupID:        &groupID,
		Model:          "mapped-model",
		RequestedModel: "requested-model",
		TotalCost:      10,
		ActualCost:     2,
		CreatedAt:      time.Unix(123, 0),
	}, "service.test")

	require.Equal(t, 1, settlement.calls)
	require.Equal(t, groupID, settlement.last.GroupID)
	require.Equal(t, int64(101), settlement.last.UsageLogID)
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

type retryableAffiliateDistributionSettlementStoreStub struct {
	statuses map[int64]string
	done     []int64
	failed   []int64
}

func (s *retryableAffiliateDistributionSettlementStoreStub) TryBeginUsageSettlement(_ context.Context, usageLogID int64) (bool, error) {
	if s.statuses == nil {
		s.statuses = make(map[int64]string)
	}
	switch s.statuses[usageLogID] {
	case "", "failed":
		s.statuses[usageLogID] = "processing"
		return true, nil
	case "processing", "done":
		return false, nil
	default:
		return false, nil
	}
}

func (s *retryableAffiliateDistributionSettlementStoreStub) MarkUsageSettlementDone(_ context.Context, usageLogID int64) error {
	if s.statuses == nil {
		s.statuses = make(map[int64]string)
	}
	s.statuses[usageLogID] = "done"
	s.done = append(s.done, usageLogID)
	return nil
}

func (s *retryableAffiliateDistributionSettlementStoreStub) MarkUsageSettlementFailed(_ context.Context, usageLogID int64) error {
	if s.statuses == nil {
		s.statuses = make(map[int64]string)
	}
	s.statuses[usageLogID] = "failed"
	s.failed = append(s.failed, usageLogID)
	return nil
}

func TestAffiliateDistributionSettlementService_RetriesAfterFailure(t *testing.T) {
	store := &retryableAffiliateDistributionSettlementStoreStub{}
	processor := &affiliateDistributionSettlementProcessorStub{
		err: assertiveTestError("boom"),
	}
	svc := NewAffiliateDistributionSettlementService(store, processor)

	cmd := AffiliateDistributionUsageSettlementCommand{UsageLogID: 77}

	applied, err := svc.SettleUsage(context.Background(), cmd)
	require.Error(t, err)
	require.False(t, applied)
	require.Equal(t, []int64{77}, store.failed)
	require.Equal(t, 1, processor.calls)

	processor.err = nil
	applied, err = svc.SettleUsage(context.Background(), cmd)
	require.NoError(t, err)
	require.True(t, applied)
	require.Equal(t, []int64{77}, store.done)
	require.Equal(t, 2, processor.calls)
}

type assertiveTestError string

func (e assertiveTestError) Error() string {
	return string(e)
}
