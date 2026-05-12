//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type accountRepoAtomicityState struct {
	accountsByID map[int64]*Account
}

type accountRepoAtomicityTxKey struct{}

type mockAccountRepoForAtomicity struct {
	mockAccountRepoForPlatform
	nextID      int64
	bindErr     error
	createCalls int
	updateCalls int
	bindCalls   int
	deleteCalls int
	withTxCalls int
}

func newMockAccountRepoForAtomicity(accounts ...*Account) *mockAccountRepoForAtomicity {
	repo := &mockAccountRepoForAtomicity{
		nextID: 100,
		mockAccountRepoForPlatform: mockAccountRepoForPlatform{
			accountsByID: make(map[int64]*Account),
		},
	}
	for _, account := range accounts {
		if account == nil {
			continue
		}
		cloned := cloneAccount(account)
		repo.accountsByID[cloned.ID] = cloned
		if cloned.ID >= repo.nextID {
			repo.nextID = cloned.ID + 1
		}
	}
	return repo
}

func (m *mockAccountRepoForAtomicity) WithTx(ctx context.Context, fn func(txCtx context.Context) error) error {
	m.withTxCalls++
	txState := &accountRepoAtomicityState{
		accountsByID: cloneAtomicAccountMap(m.accountsByID),
	}
	txCtx := context.WithValue(ctx, accountRepoAtomicityTxKey{}, txState)
	if err := fn(txCtx); err != nil {
		return err
	}
	m.accountsByID = txState.accountsByID
	return nil
}

func (m *mockAccountRepoForAtomicity) Create(ctx context.Context, account *Account) error {
	m.createCalls++
	if account == nil {
		return ErrAccountNilInput
	}
	state := m.stateFromContext(ctx)
	cloned := cloneAccount(account)
	if cloned.ID == 0 {
		cloned.ID = m.nextID
		m.nextID++
	}
	account.ID = cloned.ID
	state.accountsByID[cloned.ID] = cloned
	return nil
}

func (m *mockAccountRepoForAtomicity) GetByID(ctx context.Context, id int64) (*Account, error) {
	state := m.stateFromContext(ctx)
	account, ok := state.accountsByID[id]
	if !ok {
		return nil, ErrAccountNotFound
	}
	return cloneAccount(account), nil
}

func (m *mockAccountRepoForAtomicity) Update(ctx context.Context, account *Account) error {
	m.updateCalls++
	if account == nil {
		return ErrAccountNilInput
	}
	state := m.stateFromContext(ctx)
	if _, ok := state.accountsByID[account.ID]; !ok {
		return ErrAccountNotFound
	}
	state.accountsByID[account.ID] = cloneAccount(account)
	return nil
}

func (m *mockAccountRepoForAtomicity) Delete(ctx context.Context, id int64) error {
	m.deleteCalls++
	state := m.stateFromContext(ctx)
	delete(state.accountsByID, id)
	return nil
}

func (m *mockAccountRepoForAtomicity) BindGroups(ctx context.Context, accountID int64, groupIDs []int64) error {
	m.bindCalls++
	if m.bindErr != nil {
		return m.bindErr
	}
	state := m.stateFromContext(ctx)
	account, ok := state.accountsByID[accountID]
	if !ok {
		return ErrAccountNotFound
	}
	account.GroupIDs = copyAccountGroupIDs(groupIDs)
	return nil
}

func (m *mockAccountRepoForAtomicity) ExistsByID(ctx context.Context, id int64) (bool, error) {
	state := m.stateFromContext(ctx)
	_, ok := state.accountsByID[id]
	return ok, nil
}

func (m *mockAccountRepoForAtomicity) stateFromContext(ctx context.Context) *accountRepoAtomicityState {
	if state, ok := ctx.Value(accountRepoAtomicityTxKey{}).(*accountRepoAtomicityState); ok {
		return state
	}
	return &accountRepoAtomicityState{accountsByID: m.accountsByID}
}

func cloneAtomicAccountMap(in map[int64]*Account) map[int64]*Account {
	out := make(map[int64]*Account, len(in))
	for id, account := range in {
		out[id] = cloneAccount(account)
	}
	return out
}

type mockAccountRepoForAtomicityNoTx struct {
	mockAccountRepoForPlatform
	nextID      int64
	bindErr     error
	bindErrOnce error
	createCalls int
	updateCalls int
	bindCalls   int
	deleteCalls int
}

func newMockAccountRepoForAtomicityNoTx(accounts ...*Account) *mockAccountRepoForAtomicityNoTx {
	repo := &mockAccountRepoForAtomicityNoTx{
		nextID: 100,
		mockAccountRepoForPlatform: mockAccountRepoForPlatform{
			accountsByID: make(map[int64]*Account),
		},
	}
	for _, account := range accounts {
		if account == nil {
			continue
		}
		cloned := cloneAccount(account)
		repo.accountsByID[cloned.ID] = cloned
		if cloned.ID >= repo.nextID {
			repo.nextID = cloned.ID + 1
		}
	}
	return repo
}

func (m *mockAccountRepoForAtomicityNoTx) Create(ctx context.Context, account *Account) error {
	m.createCalls++
	if account == nil {
		return ErrAccountNilInput
	}
	cloned := cloneAccount(account)
	if cloned.ID == 0 {
		cloned.ID = m.nextID
		m.nextID++
	}
	account.ID = cloned.ID
	m.accountsByID[cloned.ID] = cloned
	return nil
}

func (m *mockAccountRepoForAtomicityNoTx) GetByID(ctx context.Context, id int64) (*Account, error) {
	account, ok := m.accountsByID[id]
	if !ok {
		return nil, ErrAccountNotFound
	}
	return cloneAccount(account), nil
}

func (m *mockAccountRepoForAtomicityNoTx) Update(ctx context.Context, account *Account) error {
	m.updateCalls++
	if account == nil {
		return ErrAccountNilInput
	}
	if _, ok := m.accountsByID[account.ID]; !ok {
		return ErrAccountNotFound
	}
	m.accountsByID[account.ID] = cloneAccount(account)
	return nil
}

func (m *mockAccountRepoForAtomicityNoTx) Delete(ctx context.Context, id int64) error {
	m.deleteCalls++
	delete(m.accountsByID, id)
	return nil
}

func (m *mockAccountRepoForAtomicityNoTx) BindGroups(ctx context.Context, accountID int64, groupIDs []int64) error {
	m.bindCalls++
	if m.bindErrOnce != nil {
		err := m.bindErrOnce
		m.bindErrOnce = nil
		return err
	}
	if m.bindErr != nil {
		return m.bindErr
	}
	account, ok := m.accountsByID[accountID]
	if !ok {
		return ErrAccountNotFound
	}
	account.GroupIDs = copyAccountGroupIDs(groupIDs)
	return nil
}

func TestAccountServiceCreateRollsBackWhenBindGroupsFails(t *testing.T) {
	repo := newMockAccountRepoForAtomicity()
	repo.bindErr = errors.New("bind failed")
	groupRepo := &mockGroupRepoForGateway{
		groups: map[int64]*Group{
			1: {ID: 1, Name: "g1"},
		},
	}
	service := NewAccountService(repo, groupRepo)

	_, err := service.Create(context.Background(), CreateAccountRequest{
		Name:        "new-account",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Credentials: map[string]any{},
		Extra:       map[string]any{},
		Concurrency: 1,
		Priority:    1,
		GroupIDs:    []int64{1},
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "bind groups")
	require.Equal(t, 1, repo.withTxCalls)
	require.Len(t, repo.accountsByID, 0)
}

func TestAccountServiceCreateWithTxCommitsAccountAndGroups(t *testing.T) {
	repo := newMockAccountRepoForAtomicity()
	groupRepo := &mockGroupRepoForGateway{
		groups: map[int64]*Group{
			1: {ID: 1, Name: "g1"},
			2: {ID: 2, Name: "g2"},
		},
	}
	service := NewAccountService(repo, groupRepo)

	account, err := service.Create(context.Background(), CreateAccountRequest{
		Name:        "new-account",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Credentials: map[string]any{"token": "new"},
		Extra:       map[string]any{"mode": "fresh"},
		Concurrency: 1,
		Priority:    1,
		GroupIDs:    []int64{1, 2},
	})

	require.NoError(t, err)
	require.Equal(t, 1, repo.withTxCalls)
	require.Equal(t, 1, repo.createCalls)
	require.Equal(t, 1, repo.bindCalls)
	require.Equal(t, 0, repo.deleteCalls)
	require.Equal(t, []int64{1, 2}, account.GroupIDs)

	stored, getErr := repo.GetByID(context.Background(), account.ID)
	require.NoError(t, getErr)
	require.Equal(t, "new-account", stored.Name)
	require.Equal(t, []int64{1, 2}, stored.GroupIDs)
	require.Equal(t, map[string]any{"token": "new"}, stored.Credentials)
	require.Equal(t, map[string]any{"mode": "fresh"}, stored.Extra)
}

func TestAccountServiceCreateWithoutTxRollsBackWhenBindGroupsFails(t *testing.T) {
	repo := newMockAccountRepoForAtomicityNoTx()
	repo.bindErr = errors.New("bind failed")
	groupRepo := &mockGroupRepoForGateway{
		groups: map[int64]*Group{
			1: {ID: 1, Name: "g1"},
		},
	}
	service := NewAccountService(repo, groupRepo)

	_, err := service.Create(context.Background(), CreateAccountRequest{
		Name:        "new-account",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Credentials: map[string]any{},
		Extra:       map[string]any{},
		Concurrency: 1,
		Priority:    1,
		GroupIDs:    []int64{1},
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "bind groups")
	require.Equal(t, 1, repo.createCalls)
	require.Equal(t, 1, repo.bindCalls)
	require.Equal(t, 1, repo.deleteCalls)
	require.Len(t, repo.accountsByID, 0)
}

func TestAccountServiceUpdateRollsBackWhenBindGroupsFails(t *testing.T) {
	original := &Account{
		ID:          7,
		Name:        "original",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Credentials: map[string]any{"token": "old"},
		Extra:       map[string]any{"mode": "old"},
		Concurrency: 1,
		Priority:    1,
		GroupIDs:    []int64{1},
	}
	repo := newMockAccountRepoForAtomicity(original)
	repo.bindErr = errors.New("bind failed")
	groupRepo := &mockGroupRepoForGateway{
		groups: map[int64]*Group{
			1: {ID: 1, Name: "g1"},
			2: {ID: 2, Name: "g2"},
		},
	}
	service := NewAccountService(repo, groupRepo)
	newName := "updated"

	_, err := service.Update(context.Background(), original.ID, UpdateAccountRequest{
		Name:     &newName,
		GroupIDs: &[]int64{2},
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "bind groups")
	require.Equal(t, 1, repo.withTxCalls)

	stored, getErr := repo.GetByID(context.Background(), original.ID)
	require.NoError(t, getErr)
	require.Equal(t, "original", stored.Name)
	require.Equal(t, []int64{1}, stored.GroupIDs)
}

func TestAccountServiceUpdateWithTxCommitsAccountAndGroups(t *testing.T) {
	original := &Account{
		ID:          7,
		Name:        "original",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Credentials: map[string]any{"token": "old"},
		Extra:       map[string]any{"mode": "old"},
		Concurrency: 1,
		Priority:    1,
		GroupIDs:    []int64{1},
	}
	repo := newMockAccountRepoForAtomicity(original)
	groupRepo := &mockGroupRepoForGateway{
		groups: map[int64]*Group{
			1: {ID: 1, Name: "g1"},
			2: {ID: 2, Name: "g2"},
		},
	}
	service := NewAccountService(repo, groupRepo)
	newName := "updated"
	newCredentials := map[string]any{"token": "new"}

	account, err := service.Update(context.Background(), original.ID, UpdateAccountRequest{
		Name:        &newName,
		Credentials: &newCredentials,
		GroupIDs:    &[]int64{2},
	})

	require.NoError(t, err)
	require.Equal(t, 1, repo.withTxCalls)
	require.Equal(t, 1, repo.updateCalls)
	require.Equal(t, 1, repo.bindCalls)
	require.Equal(t, "updated", account.Name)
	require.Equal(t, []int64{2}, account.GroupIDs)
	require.Equal(t, map[string]any{"token": "new"}, account.Credentials)

	stored, getErr := repo.GetByID(context.Background(), original.ID)
	require.NoError(t, getErr)
	require.Equal(t, "updated", stored.Name)
	require.Equal(t, []int64{2}, stored.GroupIDs)
	require.Equal(t, map[string]any{"token": "new"}, stored.Credentials)
	require.Equal(t, map[string]any{"mode": "old"}, stored.Extra)
}

func TestAccountServiceUpdateWithoutTxRollsBackWhenBindGroupsFails(t *testing.T) {
	original := &Account{
		ID:          7,
		Name:        "original",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Credentials: map[string]any{"token": "old"},
		Extra:       map[string]any{"mode": "old"},
		Concurrency: 1,
		Priority:    1,
		GroupIDs:    []int64{1},
	}
	repo := newMockAccountRepoForAtomicityNoTx(original)
	repo.bindErrOnce = errors.New("bind failed")
	groupRepo := &mockGroupRepoForGateway{
		groups: map[int64]*Group{
			1: {ID: 1, Name: "g1"},
			2: {ID: 2, Name: "g2"},
		},
	}
	service := NewAccountService(repo, groupRepo)
	newName := "updated"

	_, err := service.Update(context.Background(), original.ID, UpdateAccountRequest{
		Name:     &newName,
		GroupIDs: &[]int64{2},
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "bind groups")
	require.Equal(t, 2, repo.updateCalls)
	require.Equal(t, 2, repo.bindCalls)

	stored, getErr := repo.GetByID(context.Background(), original.ID)
	require.NoError(t, getErr)
	require.Equal(t, "original", stored.Name)
	require.Equal(t, []int64{1}, stored.GroupIDs)
	require.Equal(t, map[string]any{"token": "old"}, stored.Credentials)
	require.Equal(t, map[string]any{"mode": "old"}, stored.Extra)
}

func TestAccountServiceCreateValidatesRequireOAuthOnlyBeforeWrite(t *testing.T) {
	repo := newMockAccountRepoForAtomicity()
	groupRepo := &mockGroupRepoForGateway{
		groups: map[int64]*Group{
			11: {ID: 11, Name: "oauth-only", Platform: PlatformOpenAI, RequireOAuthOnly: true},
		},
	}
	service := NewAccountService(repo, groupRepo)

	_, err := service.Create(context.Background(), CreateAccountRequest{
		Name:        "api-key-account",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{},
		Extra:       map[string]any{},
		Concurrency: 1,
		Priority:    1,
		GroupIDs:    []int64{11},
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "OAuth")
	require.Zero(t, repo.createCalls)
	require.Zero(t, repo.bindCalls)
}

func TestAccountServiceUpdateValidatesRequireOAuthOnlyBeforeWrite(t *testing.T) {
	existing := &Account{
		ID:          9,
		Name:        "api-key-account",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Credentials: map[string]any{},
		Extra:       map[string]any{},
		Concurrency: 1,
		Priority:    1,
	}
	repo := newMockAccountRepoForAtomicity(existing)
	groupRepo := &mockGroupRepoForGateway{
		groups: map[int64]*Group{
			12: {ID: 12, Name: "oauth-only", Platform: PlatformOpenAI, RequireOAuthOnly: true},
		},
	}
	service := NewAccountService(repo, groupRepo)

	_, err := service.Update(context.Background(), existing.ID, UpdateAccountRequest{
		GroupIDs: &[]int64{12},
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "OAuth")
	require.Zero(t, repo.updateCalls)
	require.Zero(t, repo.bindCalls)
}
