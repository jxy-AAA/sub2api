package service

import (
	"context"
	"fmt"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

var (
	ErrAccountNotFound                              = infraerrors.NotFound("ACCOUNT_NOT_FOUND", "account not found")
	ErrAccountNilInput                              = infraerrors.BadRequest("ACCOUNT_NIL_INPUT", "account input cannot be nil")
	ErrAccountInvalidPlatform                       = infraerrors.BadRequest("ACCOUNT_INVALID_PLATFORM", "account platform is invalid")
	ErrAccountInvalidType                           = infraerrors.BadRequest("ACCOUNT_INVALID_TYPE", "account type is invalid")
	ErrAccountInvalidStatus                         = infraerrors.BadRequest("ACCOUNT_INVALID_STATUS", "account status is invalid")
	ErrAccountInvalidTLSFingerprintProfileReference = infraerrors.BadRequest("ACCOUNT_INVALID_TLS_FINGERPRINT_PROFILE_REFERENCE", "tls fingerprint profile reference must be a positive integer, 0, or -1")
	ErrAccountTLSFingerprintProfileNotFound         = infraerrors.BadRequest("ACCOUNT_TLS_FINGERPRINT_PROFILE_NOT_FOUND", "tls fingerprint profile does not exist")
)

const AccountListGroupUngrouped int64 = -1
const AccountPrivacyModeUnsetFilter = "__unset__"

type AccountRepository interface {
	Create(ctx context.Context, account *Account) error
	GetByID(ctx context.Context, id int64) (*Account, error)
	GetByIDs(ctx context.Context, ids []int64) ([]*Account, error)
	ExistsByID(ctx context.Context, id int64) (bool, error)
	GetByCRSAccountID(ctx context.Context, crsAccountID string) (*Account, error)
	FindByExtraField(ctx context.Context, key string, value any) ([]Account, error)
	ListCRSAccountIDs(ctx context.Context) (map[string]int64, error)
	Update(ctx context.Context, account *Account) error
	Delete(ctx context.Context, id int64) error

	List(ctx context.Context, params pagination.PaginationParams) ([]Account, *pagination.PaginationResult, error)
	ListWithFilters(ctx context.Context, params pagination.PaginationParams, platform, accountType, status, search string, groupID int64, privacyMode string) ([]Account, *pagination.PaginationResult, error)
	ListByGroup(ctx context.Context, groupID int64) ([]Account, error)
	ListActive(ctx context.Context) ([]Account, error)
	ListByPlatform(ctx context.Context, platform string) ([]Account, error)

	UpdateLastUsed(ctx context.Context, id int64) error
	BatchUpdateLastUsed(ctx context.Context, updates map[int64]time.Time) error
	SetError(ctx context.Context, id int64, errorMsg string) error
	ClearError(ctx context.Context, id int64) error
	SetSchedulable(ctx context.Context, id int64, schedulable bool) error
	AutoPauseExpiredAccounts(ctx context.Context, now time.Time) (int64, error)
	BindGroups(ctx context.Context, accountID int64, groupIDs []int64) error

	ListSchedulable(ctx context.Context) ([]Account, error)
	ListSchedulableByGroupID(ctx context.Context, groupID int64) ([]Account, error)
	ListSchedulableByPlatform(ctx context.Context, platform string) ([]Account, error)
	ListSchedulableByGroupIDAndPlatform(ctx context.Context, groupID int64, platform string) ([]Account, error)
	ListSchedulableByPlatforms(ctx context.Context, platforms []string) ([]Account, error)
	ListSchedulableByGroupIDAndPlatforms(ctx context.Context, groupID int64, platforms []string) ([]Account, error)
	ListSchedulableUngroupedByPlatform(ctx context.Context, platform string) ([]Account, error)
	ListSchedulableUngroupedByPlatforms(ctx context.Context, platforms []string) ([]Account, error)

	SetRateLimited(ctx context.Context, id int64, resetAt time.Time) error
	SetModelRateLimit(ctx context.Context, id int64, scope string, resetAt time.Time) error
	SetOverloaded(ctx context.Context, id int64, until time.Time) error
	SetTempUnschedulable(ctx context.Context, id int64, until time.Time, reason string) error
	ClearTempUnschedulable(ctx context.Context, id int64) error
	ClearRateLimit(ctx context.Context, id int64) error
	ClearAntigravityQuotaScopes(ctx context.Context, id int64) error
	ClearModelRateLimits(ctx context.Context, id int64) error
	UpdateSessionWindow(ctx context.Context, id int64, start, end *time.Time, status string) error
	UpdateExtra(ctx context.Context, id int64, updates map[string]any) error
	BulkUpdate(ctx context.Context, ids []int64, updates AccountBulkUpdate) (int64, error)
	IncrementQuotaUsed(ctx context.Context, id int64, amount float64) error
	ResetQuotaUsed(ctx context.Context, id int64) error
}

type AccountBulkUpdate struct {
	Name           *string
	ProxyID        *int64
	Concurrency    *int
	Priority       *int
	RateMultiplier *float64
	LoadFactor     *int
	Status         *string
	Schedulable    *bool
	Credentials    map[string]any
	Extra          map[string]any
}

type CreateAccountRequest struct {
	Name               string         `json:"name"`
	Notes              *string        `json:"notes"`
	Platform           string         `json:"platform"`
	Type               string         `json:"type"`
	Credentials        map[string]any `json:"credentials"`
	Extra              map[string]any `json:"extra"`
	ProxyID            *int64         `json:"proxy_id"`
	Concurrency        int            `json:"concurrency"`
	Priority           int            `json:"priority"`
	GroupIDs           []int64        `json:"group_ids"`
	ExpiresAt          *time.Time     `json:"expires_at"`
	AutoPauseOnExpired *bool          `json:"auto_pause_on_expired"`
}

type UpdateAccountRequest struct {
	Name               *string         `json:"name"`
	Notes              *string         `json:"notes"`
	Credentials        *map[string]any `json:"credentials"`
	Extra              *map[string]any `json:"extra"`
	ProxyID            *int64          `json:"proxy_id"`
	Concurrency        *int            `json:"concurrency"`
	Priority           *int            `json:"priority"`
	Status             *string         `json:"status"`
	GroupIDs           *[]int64        `json:"group_ids"`
	ExpiresAt          *time.Time      `json:"expires_at"`
	AutoPauseOnExpired *bool           `json:"auto_pause_on_expired"`
}

type AccountService struct {
	accountRepo AccountRepository
	groupRepo   GroupRepository
}

type groupExistenceBatchChecker interface {
	ExistsByIDs(ctx context.Context, ids []int64) (map[int64]bool, error)
}

type accountTransactionRunner interface {
	WithTx(ctx context.Context, fn func(txCtx context.Context) error) error
}

func NewAccountService(accountRepo AccountRepository, groupRepo GroupRepository) *AccountService {
	return &AccountService{
		accountRepo: accountRepo,
		groupRepo:   groupRepo,
	}
}

func (s *AccountService) Create(ctx context.Context, req CreateAccountRequest) (*Account, error) {
	if len(req.GroupIDs) > 0 {
		if err := s.validateGroupIDsExist(ctx, req.GroupIDs); err != nil {
			return nil, err
		}
	}

	account := &Account{
		Name:        req.Name,
		Notes:       normalizeAccountNotes(req.Notes),
		Platform:    req.Platform,
		Type:        req.Type,
		Credentials: req.Credentials,
		Extra:       req.Extra,
		ProxyID:     req.ProxyID,
		Concurrency: req.Concurrency,
		Priority:    req.Priority,
		Status:      StatusActive,
		ExpiresAt:   req.ExpiresAt,
	}
	if req.AutoPauseOnExpired != nil {
		account.AutoPauseOnExpired = *req.AutoPauseOnExpired
	} else {
		account.AutoPauseOnExpired = true
	}

	if err := s.validateRequireOAuthOnlyGroups(ctx, account.Type, req.GroupIDs); err != nil {
		return nil, err
	}

	if txRunner, ok := s.accountRepo.(accountTransactionRunner); ok {
		if err := txRunner.WithTx(ctx, func(txCtx context.Context) error {
			if err := s.accountRepo.Create(txCtx, account); err != nil {
				return fmt.Errorf("create account: %w", err)
			}
			if len(req.GroupIDs) > 0 {
				if err := s.accountRepo.BindGroups(txCtx, account.ID, req.GroupIDs); err != nil {
					return fmt.Errorf("bind groups: %w", err)
				}
			}
			return nil
		}); err != nil {
			return nil, err
		}
		account.GroupIDs = copyAccountGroupIDs(req.GroupIDs)
		return account, nil
	}

	if err := s.accountRepo.Create(ctx, account); err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}
	if len(req.GroupIDs) > 0 {
		if err := s.accountRepo.BindGroups(ctx, account.ID, req.GroupIDs); err != nil {
			rollbackErr := s.accountRepo.Delete(ctx, account.ID)
			if rollbackErr != nil {
				return nil, fmt.Errorf("bind groups: %w (rollback create failed: %v)", err, rollbackErr)
			}
			return nil, fmt.Errorf("bind groups: %w", err)
		}
	}

	account.GroupIDs = copyAccountGroupIDs(req.GroupIDs)
	return account, nil
}

func (s *AccountService) GetByID(ctx context.Context, id int64) (*Account, error) {
	account, err := s.accountRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}
	return account, nil
}

func (s *AccountService) List(ctx context.Context, params pagination.PaginationParams) ([]Account, *pagination.PaginationResult, error) {
	accounts, paginationResult, err := s.accountRepo.List(ctx, params)
	if err != nil {
		return nil, nil, fmt.Errorf("list accounts: %w", err)
	}
	return accounts, paginationResult, nil
}

func (s *AccountService) ListByPlatform(ctx context.Context, platform string) ([]Account, error) {
	accounts, err := s.accountRepo.ListByPlatform(ctx, platform)
	if err != nil {
		return nil, fmt.Errorf("list accounts by platform: %w", err)
	}
	return accounts, nil
}

func (s *AccountService) ListByGroup(ctx context.Context, groupID int64) ([]Account, error) {
	accounts, err := s.accountRepo.ListByGroup(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("list accounts by group: %w", err)
	}
	return accounts, nil
}

func (s *AccountService) Update(ctx context.Context, id int64, req UpdateAccountRequest) (*Account, error) {
	account, err := s.accountRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}

	original := cloneAccount(account)
	updated := cloneAccount(account)

	if req.Name != nil {
		updated.Name = *req.Name
	}
	if req.Notes != nil {
		updated.Notes = normalizeAccountNotes(req.Notes)
	}
	if req.Credentials != nil {
		updated.Credentials = *req.Credentials
	}
	if req.Extra != nil {
		updated.Extra = *req.Extra
	}
	if req.ProxyID != nil {
		updated.ProxyID = req.ProxyID
	}
	if req.Concurrency != nil {
		updated.Concurrency = *req.Concurrency
	}
	if req.Priority != nil {
		updated.Priority = *req.Priority
	}
	if req.Status != nil {
		updated.Status = *req.Status
	}
	if req.ExpiresAt != nil {
		updated.ExpiresAt = req.ExpiresAt
	}
	if req.AutoPauseOnExpired != nil {
		updated.AutoPauseOnExpired = *req.AutoPauseOnExpired
	}

	if req.GroupIDs != nil {
		if err := s.validateGroupIDsExist(ctx, *req.GroupIDs); err != nil {
			return nil, err
		}
	}

	targetGroupIDs := copyAccountGroupIDs(updated.GroupIDs)
	if req.GroupIDs != nil {
		targetGroupIDs = copyAccountGroupIDs(*req.GroupIDs)
	}
	if err := s.validateRequireOAuthOnlyGroups(ctx, updated.Type, targetGroupIDs); err != nil {
		return nil, err
	}

	if txRunner, ok := s.accountRepo.(accountTransactionRunner); ok {
		if err := txRunner.WithTx(ctx, func(txCtx context.Context) error {
			if err := s.accountRepo.Update(txCtx, updated); err != nil {
				return fmt.Errorf("update account: %w", err)
			}
			if req.GroupIDs != nil {
				if err := s.accountRepo.BindGroups(txCtx, updated.ID, targetGroupIDs); err != nil {
					return fmt.Errorf("bind groups: %w", err)
				}
			}
			return nil
		}); err != nil {
			return nil, err
		}
		updated.GroupIDs = targetGroupIDs
		return updated, nil
	}

	if err := s.accountRepo.Update(ctx, updated); err != nil {
		return nil, fmt.Errorf("update account: %w", err)
	}
	if req.GroupIDs != nil {
		if err := s.accountRepo.BindGroups(ctx, updated.ID, targetGroupIDs); err != nil {
			rollbackErr := s.restoreAccountState(ctx, original)
			if rollbackErr != nil {
				return nil, fmt.Errorf("bind groups: %w (rollback update failed: %v)", err, rollbackErr)
			}
			return nil, fmt.Errorf("bind groups: %w", err)
		}
	}

	updated.GroupIDs = targetGroupIDs
	return updated, nil
}

func (s *AccountService) Delete(ctx context.Context, id int64) error {
	exists, err := s.accountRepo.ExistsByID(ctx, id)
	if err != nil {
		return fmt.Errorf("check account: %w", err)
	}
	if !exists {
		return ErrAccountNotFound
	}

	if err := s.accountRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete account: %w", err)
	}
	return nil
}

func (s *AccountService) validateGroupIDsExist(ctx context.Context, groupIDs []int64) error {
	if len(groupIDs) == 0 {
		return nil
	}
	if s.groupRepo == nil {
		return fmt.Errorf("group repository not configured")
	}

	if batchChecker, ok := s.groupRepo.(groupExistenceBatchChecker); ok {
		existsByID, err := batchChecker.ExistsByIDs(ctx, groupIDs)
		if err != nil {
			return fmt.Errorf("check groups exists: %w", err)
		}
		for _, groupID := range groupIDs {
			if groupID <= 0 {
				return fmt.Errorf("get group: %w", ErrGroupNotFound)
			}
			if !existsByID[groupID] {
				return fmt.Errorf("get group: %w", ErrGroupNotFound)
			}
		}
		return nil
	}

	for _, groupID := range groupIDs {
		_, err := s.groupRepo.GetByID(ctx, groupID)
		if err != nil {
			return fmt.Errorf("get group: %w", err)
		}
	}
	return nil
}

func (s *AccountService) validateRequireOAuthOnlyGroups(ctx context.Context, accountType string, groupIDs []int64) error {
	if accountType != AccountTypeAPIKey || len(groupIDs) == 0 {
		return nil
	}
	if s.groupRepo == nil {
		return fmt.Errorf("group repository not configured")
	}

	for _, groupID := range groupIDs {
		group, err := s.groupRepo.GetByID(ctx, groupID)
		if err != nil {
			return err
		}
		if !group.RequireOAuthOnly {
			continue
		}
		switch group.Platform {
		case PlatformOpenAI, PlatformAntigravity, PlatformAnthropic, PlatformGemini:
			return fmt.Errorf("分组 [%s] 仅允许 OAuth 账号，apikey 类型账号无法加入", group.Name)
		}
	}
	return nil
}

func (s *AccountService) restoreAccountState(ctx context.Context, account *Account) error {
	if account == nil {
		return nil
	}
	if err := s.accountRepo.Update(ctx, cloneAccount(account)); err != nil {
		return fmt.Errorf("restore account: %w", err)
	}
	if err := s.accountRepo.BindGroups(ctx, account.ID, copyAccountGroupIDs(account.GroupIDs)); err != nil {
		return fmt.Errorf("restore groups: %w", err)
	}
	return nil
}

func cloneAccount(account *Account) *Account {
	if account == nil {
		return nil
	}

	cloned := *account
	cloned.Notes = cloneStringPointer(account.Notes)
	cloned.ProxyID = cloneInt64Pointer(account.ProxyID)
	cloned.RateMultiplier = cloneFloat64Pointer(account.RateMultiplier)
	cloned.LoadFactor = cloneIntPointer(account.LoadFactor)
	cloned.LastUsedAt = cloneTimePointer(account.LastUsedAt)
	cloned.ExpiresAt = cloneTimePointer(account.ExpiresAt)
	cloned.RateLimitedAt = cloneTimePointer(account.RateLimitedAt)
	cloned.RateLimitResetAt = cloneTimePointer(account.RateLimitResetAt)
	cloned.OverloadUntil = cloneTimePointer(account.OverloadUntil)
	cloned.TempUnschedulableUntil = cloneTimePointer(account.TempUnschedulableUntil)
	cloned.SessionWindowStart = cloneTimePointer(account.SessionWindowStart)
	cloned.SessionWindowEnd = cloneTimePointer(account.SessionWindowEnd)
	cloned.Credentials = cloneAnyMap(account.Credentials)
	cloned.Extra = cloneAnyMap(account.Extra)
	cloned.GroupIDs = copyAccountGroupIDs(account.GroupIDs)
	if len(account.Groups) > 0 {
		cloned.Groups = append([]*Group(nil), account.Groups...)
	}
	if len(account.AccountGroups) > 0 {
		cloned.AccountGroups = append([]AccountGroup(nil), account.AccountGroups...)
	}
	return &cloned
}

func cloneAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func copyAccountGroupIDs(groupIDs []int64) []int64 {
	if len(groupIDs) == 0 {
		return nil
	}
	return append([]int64(nil), groupIDs...)
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneInt64Pointer(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneFloat64Pointer(value *float64) *float64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func (s *AccountService) UpdateStatus(ctx context.Context, id int64, status string, errorMessage string) error {
	account, err := s.accountRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get account: %w", err)
	}

	account.Status = status
	account.ErrorMessage = errorMessage

	if err := s.accountRepo.Update(ctx, account); err != nil {
		return fmt.Errorf("update account: %w", err)
	}
	return nil
}

func (s *AccountService) UpdateLastUsed(ctx context.Context, id int64) error {
	if err := s.accountRepo.UpdateLastUsed(ctx, id); err != nil {
		return fmt.Errorf("update last used: %w", err)
	}
	return nil
}

func (s *AccountService) GetCredential(ctx context.Context, id int64, key string) (string, error) {
	account, err := s.accountRepo.GetByID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("get account: %w", err)
	}
	return account.GetCredential(key), nil
}

func (s *AccountService) TestCredentials(ctx context.Context, id int64) error {
	account, err := s.accountRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get account: %w", err)
	}

	switch account.Platform {
	case PlatformAnthropic:
		return nil
	case PlatformOpenAI:
		return nil
	case PlatformGemini:
		return nil
	default:
		return fmt.Errorf("unsupported platform: %s", account.Platform)
	}
}
