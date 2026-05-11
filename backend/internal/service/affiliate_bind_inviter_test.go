package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type affiliateBindRepoStub struct {
	AffiliateRepository
	selfSummary    *AffiliateSummary
	inviterSummary *AffiliateSummary
	ensureErr      error
	codeErr        error
	bindResult     bool
	bindErr        error
	bindCalled     int
	bindUserID     int64
	bindInviterID  int64
}

func (s *affiliateBindRepoStub) EnsureUserAffiliate(context.Context, int64) (*AffiliateSummary, error) {
	if s.ensureErr != nil {
		return nil, s.ensureErr
	}
	return s.selfSummary, nil
}

func (s *affiliateBindRepoStub) GetAffiliateByCode(context.Context, string) (*AffiliateSummary, error) {
	if s.codeErr != nil {
		return nil, s.codeErr
	}
	return s.inviterSummary, nil
}

func (s *affiliateBindRepoStub) BindInviter(_ context.Context, userID, inviterID int64) (bool, error) {
	s.bindCalled++
	s.bindUserID = userID
	s.bindInviterID = inviterID
	if s.bindErr != nil {
		return false, s.bindErr
	}
	return s.bindResult, nil
}

type affiliateEnabledSettingRepoStub struct{}

func (s *affiliateEnabledSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	return nil, errors.New("not implemented")
}

func (s *affiliateEnabledSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if key == SettingKeyAffiliateEnabled {
		return "true", nil
	}
	return "", errors.New("not implemented")
}

func (s *affiliateEnabledSettingRepoStub) Set(context.Context, string, string) error {
	return errors.New("not implemented")
}

func (s *affiliateEnabledSettingRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	return nil, errors.New("not implemented")
}

func (s *affiliateEnabledSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	return errors.New("not implemented")
}

func (s *affiliateEnabledSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return nil, errors.New("not implemented")
}

func (s *affiliateEnabledSettingRepoStub) Delete(context.Context, string) error {
	return errors.New("not implemented")
}

func TestAffiliateServiceBindInviterByCode_DoesNotShortCircuitDefaultRoot(t *testing.T) {
	repo := &affiliateBindRepoStub{
		selfSummary: &AffiliateSummary{
			UserID:        101,
			InviterID:     affiliateTestInt64Ptr(1),
			InviterSource: "default_root",
		},
		inviterSummary: &AffiliateSummary{
			UserID: 202,
		},
		bindResult: true,
	}
	svc := NewAffiliateService(repo, NewSettingService(&affiliateEnabledSettingRepoStub{}, nil), nil, nil)

	err := svc.BindInviterByCode(context.Background(), 101, "invite-202")

	require.NoError(t, err)
	require.Equal(t, 1, repo.bindCalled)
	require.Equal(t, int64(101), repo.bindUserID)
	require.Equal(t, int64(202), repo.bindInviterID)
}

func TestAffiliateServiceBindInviterByCode_ReturnsAlreadyBoundWhenRepositoryRejectsOverride(t *testing.T) {
	repo := &affiliateBindRepoStub{
		selfSummary: &AffiliateSummary{
			UserID:        101,
			InviterID:     affiliateTestInt64Ptr(9),
			InviterSource: "admin_override",
		},
		inviterSummary: &AffiliateSummary{
			UserID: 303,
		},
		bindResult: false,
	}
	svc := NewAffiliateService(repo, NewSettingService(&affiliateEnabledSettingRepoStub{}, nil), nil, nil)

	err := svc.BindInviterByCode(context.Background(), 101, "invite-303")

	require.ErrorIs(t, err, ErrAffiliateAlreadyBound)
	require.Equal(t, 1, repo.bindCalled)
	require.Equal(t, int64(303), repo.bindInviterID)
}

func affiliateTestInt64Ptr(value int64) *int64 {
	return &value
}
