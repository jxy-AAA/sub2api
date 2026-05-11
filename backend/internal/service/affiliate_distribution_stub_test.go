package service

import (
	"context"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type affiliateDistributionOverviewRepoStub struct {
	AffiliateRepository
	AffiliateDistributionRepository
	overview      *AgentDistributionOverview
	overviewErr   error
	directMembers []AgentDirectMember
	directErr     error
}

func (s *affiliateDistributionOverviewRepoStub) GetDistributionOverview(context.Context, int64) (*AgentDistributionOverview, error) {
	if s.overviewErr != nil {
		return nil, s.overviewErr
	}
	if s.overview == nil {
		return nil, nil
	}

	result := *s.overview
	return &result, nil
}

func (s *affiliateDistributionOverviewRepoStub) ListDirectSubordinates(context.Context, int64) ([]AgentDirectMember, error) {
	if s.directErr != nil {
		return nil, s.directErr
	}
	return append([]AgentDirectMember(nil), s.directMembers...), nil
}

func TestAffiliateServiceGetDistributionOverviewIncludesDirectChildren(t *testing.T) {
	repo := &affiliateDistributionOverviewRepoStub{
		overview: &AgentDistributionOverview{
			UserID:            21,
			InviteCode:        "AGENT21",
			InviteGroupRates:  nil,
			CurrentGroupRates: nil,
			MyGroupRates:      nil,
		},
		directMembers: []AgentDirectMember{{
			UserID:             33,
			Email:              "child@example.com",
			CurrentGroupRates:  nil,
			ParentCanEditRates: true,
		}},
	}
	service := NewAffiliateService(repo, nil, nil, nil)

	overview, err := service.GetDistributionOverview(context.Background(), 21)

	require.NoError(t, err)
	require.NotNil(t, overview)
	require.Len(t, overview.DirectChildren, 1)
	require.Equal(t, 1, overview.DirectChildrenCount)
	require.Equal(t, 1, overview.DirectMemberCount)
	require.NotNil(t, overview.InviteGroupRates)
	require.NotNil(t, overview.CurrentGroupRates)
	require.NotNil(t, overview.MyGroupRates)
	require.NotNil(t, overview.DirectChildren[0].CurrentGroupRates)
}

func TestAffiliateServiceGetDistributionOverviewReturnsDirectMemberError(t *testing.T) {
	repo := &affiliateDistributionOverviewRepoStub{
		overview: &AgentDistributionOverview{UserID: 21},
		directErr: infraerrors.ServiceUnavailable(
			"DIRECT_MEMBERS_FAILED",
			"failed to load direct members",
		),
	}
	service := NewAffiliateService(repo, nil, nil, nil)

	overview, err := service.GetDistributionOverview(context.Background(), 21)

	require.Nil(t, overview)
	require.Error(t, err)
	require.Equal(t, "DIRECT_MEMBERS_FAILED", infraerrors.Reason(err))
}
