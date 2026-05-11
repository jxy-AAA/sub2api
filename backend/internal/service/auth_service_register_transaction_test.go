//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type authRegisterAffiliateRepoStub struct {
	summary    *AffiliateSummary
	ensureErr  error
	getCodeErr error
	bindErr    error
}

func (s *authRegisterAffiliateRepoStub) EnsureUserAffiliate(_ context.Context, userID int64) (*AffiliateSummary, error) {
	if s.ensureErr != nil {
		return nil, s.ensureErr
	}
	if s.summary != nil {
		copyValue := *s.summary
		return &copyValue, nil
	}
	return &AffiliateSummary{UserID: userID, AffCode: "SELF001"}, nil
}

func (s *authRegisterAffiliateRepoStub) GetAffiliateByCode(_ context.Context, _ string) (*AffiliateSummary, error) {
	if s.getCodeErr != nil {
		return nil, s.getCodeErr
	}
	if s.summary == nil {
		return nil, ErrAffiliateProfileNotFound
	}
	copyValue := *s.summary
	return &copyValue, nil
}

func (s *authRegisterAffiliateRepoStub) BindInviter(context.Context, int64, int64) (bool, error) {
	if s.bindErr != nil {
		return false, s.bindErr
	}
	return true, nil
}

func (s *authRegisterAffiliateRepoStub) AccrueQuota(context.Context, int64, int64, float64, int, *int64) (bool, error) {
	return false, nil
}

func (s *authRegisterAffiliateRepoStub) GetAccruedRebateFromInvitee(context.Context, int64, int64) (float64, error) {
	return 0, nil
}

func (s *authRegisterAffiliateRepoStub) ThawFrozenQuota(context.Context, int64) (float64, error) {
	return 0, nil
}

func (s *authRegisterAffiliateRepoStub) TransferQuotaToBalance(context.Context, int64) (float64, float64, error) {
	return 0, 0, nil
}

func (s *authRegisterAffiliateRepoStub) ListInvitees(context.Context, int64, int) ([]AffiliateInvitee, error) {
	return nil, nil
}

func (s *authRegisterAffiliateRepoStub) UpdateUserAffCode(context.Context, int64, string) error {
	return nil
}

func (s *authRegisterAffiliateRepoStub) ResetUserAffCode(context.Context, int64) (string, error) {
	return "", nil
}

func (s *authRegisterAffiliateRepoStub) SetUserRebateRate(context.Context, int64, *float64) error {
	return nil
}

func (s *authRegisterAffiliateRepoStub) BatchSetUserRebateRate(context.Context, []int64, *float64) error {
	return nil
}

func (s *authRegisterAffiliateRepoStub) ListUsersWithCustomSettings(context.Context, AffiliateAdminFilter) ([]AffiliateAdminEntry, int64, error) {
	return nil, 0, nil
}

func (s *authRegisterAffiliateRepoStub) ListAffiliateInviteRecords(context.Context, AffiliateRecordFilter) ([]AffiliateInviteRecord, int64, error) {
	return nil, 0, nil
}

func (s *authRegisterAffiliateRepoStub) ListAffiliateRebateRecords(context.Context, AffiliateRecordFilter) ([]AffiliateRebateRecord, int64, error) {
	return nil, 0, nil
}

func (s *authRegisterAffiliateRepoStub) ListAffiliateTransferRecords(context.Context, AffiliateRecordFilter) ([]AffiliateTransferRecord, int64, error) {
	return nil, 0, nil
}

func (s *authRegisterAffiliateRepoStub) GetAffiliateUserOverview(context.Context, int64) (*AffiliateUserOverview, error) {
	return nil, nil
}

func TestAuthServiceRegisterRollsBackOnDefaultSubscriptionFailure(t *testing.T) {
	repo := &userRepoStub{nextID: 101}
	assigner := &defaultSubscriptionAssignerStub{err: errors.New("assign failed")}
	authService := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:                 "true",
		SettingKeyDefaultSubscriptions:                `[{"group_id":11,"validity_days":30}]`,
		SettingKeyAuthSourceDefaultEmailGrantOnSignup: "false",
	}, nil)
	authService.defaultSubAssigner = assigner

	_, _, err := authService.Register(context.Background(), "rollback-subscriptions@test.com", "password")

	require.ErrorIs(t, err, ErrServiceUnavailable)
	require.Equal(t, []int64{101}, repo.deletedIDs)
}

func TestAuthServiceRegisterRollsBackOnAffiliateBindFailure(t *testing.T) {
	repo := &userRepoStub{nextID: 102}
	bindErr := errors.New("bind failed")
	authService := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled: "true",
		SettingKeyAffiliateEnabled:    "true",
	}, nil)
	authService.affiliateService = NewAffiliateService(&authRegisterAffiliateRepoStub{
		summary: &AffiliateSummary{UserID: 900, AffCode: "AFF123"},
		bindErr: bindErr,
	}, authService.settingService, nil, nil)

	_, _, err := authService.RegisterWithVerification(context.Background(), "rollback-affiliate@test.com", "password", "", "", "AFF123")

	require.ErrorIs(t, err, bindErr)
	require.Equal(t, []int64{102}, repo.deletedIDs)
}

func TestAuthServiceRegisterRollsBackOnInvitationUseFailure(t *testing.T) {
	repo := &userRepoStub{nextID: 103}
	redeemRepo := &redeemCodeRepoStub{
		useErr: errors.New("use failed"),
		codesByCode: map[string]*RedeemCode{
			"INVITE123": {
				ID:     7,
				Code:   "INVITE123",
				Type:   RedeemTypeInvitation,
				Status: StatusUnused,
			},
		},
	}
	authService := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:   "true",
		SettingKeyInvitationCodeEnabled: "true",
	}, nil)
	authService.redeemRepo = redeemRepo

	_, _, err := authService.RegisterWithVerification(context.Background(), "rollback-invite@test.com", "password", "", "INVITE123", "")

	require.ErrorIs(t, err, ErrInvitationCodeInvalid)
	require.Equal(t, []int64{103}, repo.deletedIDs)
}
