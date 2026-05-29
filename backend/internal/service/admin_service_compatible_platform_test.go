package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type compatibleAdminGroupRepo struct {
	GroupRepository
	groups       map[int64]*Group
	activeGroups []Group
}

func (r *compatibleAdminGroupRepo) GetByID(_ context.Context, id int64) (*Group, error) {
	if group, ok := r.groups[id]; ok {
		return group, nil
	}
	return nil, ErrGroupNotFound
}

func (r *compatibleAdminGroupRepo) ListActiveByPlatform(_ context.Context, _ string) ([]Group, error) {
	return append([]Group(nil), r.activeGroups...), nil
}

type compatibleAdminAccountRepo struct {
	AccountRepository
	accounts   map[int64]*Account
	created    *Account
	bound      map[int64][]int64
	bulkCalled bool
}

func (r *compatibleAdminAccountRepo) Create(_ context.Context, account *Account) error {
	if account.ID == 0 {
		account.ID = 1
	}
	copyAccount := *account
	r.created = &copyAccount
	if r.accounts == nil {
		r.accounts = map[int64]*Account{}
	}
	r.accounts[copyAccount.ID] = &copyAccount
	return nil
}

func (r *compatibleAdminAccountRepo) GetByID(_ context.Context, id int64) (*Account, error) {
	if account, ok := r.accounts[id]; ok {
		copyAccount := *account
		return &copyAccount, nil
	}
	return nil, ErrAccountNotFound
}

func (r *compatibleAdminAccountRepo) GetByIDs(_ context.Context, ids []int64) ([]*Account, error) {
	accounts := make([]*Account, 0, len(ids))
	for _, id := range ids {
		if account, ok := r.accounts[id]; ok {
			copyAccount := *account
			accounts = append(accounts, &copyAccount)
		}
	}
	return accounts, nil
}

func (r *compatibleAdminAccountRepo) Update(_ context.Context, account *Account) error {
	if r.accounts == nil {
		r.accounts = map[int64]*Account{}
	}
	copyAccount := *account
	r.accounts[copyAccount.ID] = &copyAccount
	return nil
}

func (r *compatibleAdminAccountRepo) BindGroups(_ context.Context, accountID int64, groupIDs []int64) error {
	if r.bound == nil {
		r.bound = map[int64][]int64{}
	}
	r.bound[accountID] = append([]int64(nil), groupIDs...)
	return nil
}

func (r *compatibleAdminAccountRepo) BulkUpdate(_ context.Context, _ []int64, _ AccountBulkUpdate) (int64, error) {
	r.bulkCalled = true
	return 0, nil
}

func TestAdminServiceCreateCompatibleAccountBindsProtocolDefaultGroup(t *testing.T) {
	groupRepo := &compatibleAdminGroupRepo{
		groups: map[int64]*Group{
			10: {ID: 10, Name: "openai-default", Platform: PlatformOpenAI, Status: StatusActive},
		},
		activeGroups: []Group{
			{ID: 10, Name: "openai-default", Platform: PlatformOpenAI, Status: StatusActive},
		},
	}
	accountRepo := &compatibleAdminAccountRepo{}
	svc := &adminServiceImpl{accountRepo: accountRepo, groupRepo: groupRepo}

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:        "compat",
		Platform:    PlatformOpenAICompatible,
		Type:        AccountTypeUpstream,
		Credentials: map[string]any{"base_url": "https://compat.example", "api_key": "sk-test"},
	})
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, []int64{10}, accountRepo.bound[account.ID])
}

func TestAdminServiceCreateCompatibleAccountRejectsOAuthOnlyGroup(t *testing.T) {
	groupRepo := compatibleOAuthOnlyGroupRepo()
	accountRepo := &compatibleAdminAccountRepo{}
	svc := &adminServiceImpl{accountRepo: accountRepo, groupRepo: groupRepo}

	_, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:                  "compat",
		Platform:              PlatformOpenAICompatible,
		Type:                  AccountTypeUpstream,
		Credentials:           map[string]any{"base_url": "https://compat.example", "api_key": "sk-test"},
		GroupIDs:              []int64{20},
		SkipMixedChannelCheck: true,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "OAuth")
	require.Nil(t, accountRepo.created)
}

func TestAdminServiceUpdateCompatibleAccountRejectsOAuthOnlyGroup(t *testing.T) {
	groupRepo := compatibleOAuthOnlyGroupRepo()
	groupIDs := []int64{20}
	accountRepo := &compatibleAdminAccountRepo{
		accounts: map[int64]*Account{
			1: {
				ID:          1,
				Name:        "compat",
				Platform:    PlatformOpenAICompatible,
				Type:        AccountTypeUpstream,
				Credentials: map[string]any{"base_url": "https://compat.example", "api_key": "sk-test"},
			},
		},
	}
	svc := &adminServiceImpl{accountRepo: accountRepo, groupRepo: groupRepo}

	_, err := svc.UpdateAccount(context.Background(), 1, &UpdateAccountInput{
		GroupIDs:              &groupIDs,
		SkipMixedChannelCheck: true,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "OAuth")
	require.Empty(t, accountRepo.bound)
}

func TestAdminServiceBulkUpdateCompatibleAccountRejectsOAuthOnlyGroup(t *testing.T) {
	groupRepo := compatibleOAuthOnlyGroupRepo()
	groupIDs := []int64{20}
	accountRepo := &compatibleAdminAccountRepo{
		accounts: map[int64]*Account{
			1: {
				ID:       1,
				Name:     "compat",
				Platform: PlatformOpenAICompatible,
				Type:     AccountTypeUpstream,
			},
		},
	}
	svc := &adminServiceImpl{accountRepo: accountRepo, groupRepo: groupRepo}

	_, err := svc.BulkUpdateAccounts(context.Background(), &BulkUpdateAccountsInput{
		AccountIDs:            []int64{1},
		GroupIDs:              &groupIDs,
		SkipMixedChannelCheck: true,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "OAuth")
	require.False(t, accountRepo.bulkCalled)
	require.Empty(t, accountRepo.bound)
}

func compatibleOAuthOnlyGroupRepo() *compatibleAdminGroupRepo {
	return &compatibleAdminGroupRepo{
		groups: map[int64]*Group{
			20: {
				ID:               20,
				Name:             "openai-compatible-oauth",
				Platform:         PlatformOpenAICompatible,
				Status:           StatusActive,
				RequireOAuthOnly: true,
			},
		},
	}
}
