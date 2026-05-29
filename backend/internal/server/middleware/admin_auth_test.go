//go:build unit

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAdminAuthJWTValidatesTokenVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{JWT: config.JWTConfig{Secret: "test-secret", ExpireHour: 1}}
	authService := service.NewAuthService(nil, nil, nil, nil, cfg, nil, nil, nil, nil, nil, nil, nil)

	admin := &service.User{
		ID:           1,
		Email:        "admin@example.com",
		Role:         service.RoleAdmin,
		Status:       service.StatusActive,
		TokenVersion: 2,
		Concurrency:  1,
	}

	userRepo := &stubUserRepo{
		getByID: func(ctx context.Context, id int64) (*service.User, error) {
			if id != admin.ID {
				return nil, service.ErrUserNotFound
			}
			clone := *admin
			return &clone, nil
		},
	}
	userService := service.NewUserService(userRepo, nil, nil, nil)

	router := gin.New()
	router.Use(gin.HandlerFunc(NewAdminAuthMiddleware(authService, userService, nil)))
	router.GET("/t", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	t.Run("token_version_mismatch_rejected", func(t *testing.T) {
		token, err := authService.GenerateToken(&service.User{
			ID:           admin.ID,
			Email:        admin.Email,
			Role:         admin.Role,
			TokenVersion: admin.TokenVersion - 1,
		})
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/t", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.Contains(t, w.Body.String(), "TOKEN_REVOKED")
	})

	t.Run("token_version_match_allows", func(t *testing.T) {
		token, err := authService.GenerateToken(&service.User{
			ID:           admin.ID,
			Email:        admin.Email,
			Role:         admin.Role,
			TokenVersion: admin.TokenVersion,
		})
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/t", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("websocket_token_version_mismatch_rejected", func(t *testing.T) {
		token, err := authService.GenerateToken(&service.User{
			ID:           admin.ID,
			Email:        admin.Email,
			Role:         admin.Role,
			TokenVersion: admin.TokenVersion - 1,
		})
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/t", nil)
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Connection", "Upgrade")
		req.Header.Set("Sec-WebSocket-Protocol", "sub2api-admin, jwt."+token)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.Contains(t, w.Body.String(), "TOKEN_REVOKED")
	})

	t.Run("websocket_token_version_match_allows", func(t *testing.T) {
		token, err := authService.GenerateToken(&service.User{
			ID:           admin.ID,
			Email:        admin.Email,
			Role:         admin.Role,
			TokenVersion: admin.TokenVersion,
		})
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/t", nil)
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Connection", "Upgrade")
		req.Header.Set("Sec-WebSocket-Protocol", "sub2api-admin, jwt."+token)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAdminAuthAPIKeyUsesExplicitSystemPrincipalWithoutHumanAttribution(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var getFirstAdminCalled atomic.Bool
	repo := &stubUserRepo{
		getByID: func(ctx context.Context, id int64) (*service.User, error) {
			return nil, service.ErrUserNotFound
		},
	}
	repo.getFirstAdmin = func(ctx context.Context) (*service.User, error) {
		getFirstAdminCalled.Store(true)
		return &service.User{
			ID:          42,
			Role:        service.RoleAdmin,
			Status:      service.StatusActive,
			Concurrency: 7,
		}, nil
	}

	settingSvc := service.NewSettingService(&adminAuthSettingRepoStub{
		values: map[string]string{},
	}, &config.Config{})
	rawKey, err := settingSvc.GenerateAdminAPIKey(context.Background())
	require.NoError(t, err)
	userService := service.NewUserService(repo, nil, nil, nil)

	router := gin.New()
	router.Use(gin.HandlerFunc(NewAdminAuthMiddleware(nil, userService, settingSvc)))
	router.GET("/t", func(c *gin.Context) {
		subject, ok := GetAuthSubjectFromContext(c)
		require.True(t, ok)
		role, ok := GetUserRoleFromContext(c)
		require.True(t, ok)
		c.JSON(http.StatusOK, gin.H{
			"user_id":        subject.UserID,
			"concurrency":    subject.Concurrency,
			"principal_id":   subject.PrincipalID,
			"principal_type": subject.PrincipalType,
			"is_system":      subject.IsSystem,
			"role":           role,
			"auth_method":    c.GetString("auth_method"),
		})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/t", nil)
	req.Header.Set("x-api-key", rawKey)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.False(t, getFirstAdminCalled.Load())
	require.Contains(t, w.Body.String(), `"user_id":0`)
	require.Contains(t, w.Body.String(), `"concurrency":0`)
	require.Contains(t, w.Body.String(), `"principal_type":"admin_api_key"`)
	require.Contains(t, w.Body.String(), `"principal_id":"admin-key:`)
	require.Contains(t, w.Body.String(), `"is_system":true`)
	require.Contains(t, w.Body.String(), `"auth_method":"admin_api_key"`)
	require.Contains(t, w.Body.String(), `"role":"admin"`)
}

type stubUserRepo struct {
	getByID       func(ctx context.Context, id int64) (*service.User, error)
	getFirstAdmin func(ctx context.Context) (*service.User, error)
}

func (s *stubUserRepo) Create(ctx context.Context, user *service.User) error {
	panic("unexpected Create call")
}

func (s *stubUserRepo) GetByID(ctx context.Context, id int64) (*service.User, error) {
	if s.getByID == nil {
		panic("GetByID not stubbed")
	}
	return s.getByID(ctx, id)
}

func (s *stubUserRepo) GetByEmail(ctx context.Context, email string) (*service.User, error) {
	panic("unexpected GetByEmail call")
}

func (s *stubUserRepo) GetFirstAdmin(ctx context.Context) (*service.User, error) {
	if s.getFirstAdmin == nil {
		panic("unexpected GetFirstAdmin call")
	}
	return s.getFirstAdmin(ctx)
}

func (s *stubUserRepo) Update(ctx context.Context, user *service.User) error {
	panic("unexpected Update call")
}

func (s *stubUserRepo) Delete(ctx context.Context, id int64) error {
	panic("unexpected Delete call")
}

func (s *stubUserRepo) GetUserAvatar(ctx context.Context, userID int64) (*service.UserAvatar, error) {
	return nil, nil
}

func (s *stubUserRepo) UpsertUserAvatar(ctx context.Context, userID int64, input service.UpsertUserAvatarInput) (*service.UserAvatar, error) {
	panic("unexpected UpsertUserAvatar call")
}

func (s *stubUserRepo) DeleteUserAvatar(ctx context.Context, userID int64) error {
	panic("unexpected DeleteUserAvatar call")
}

func (s *stubUserRepo) List(ctx context.Context, params pagination.PaginationParams) ([]service.User, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}

func (s *stubUserRepo) ListWithFilters(ctx context.Context, params pagination.PaginationParams, filters service.UserListFilters) ([]service.User, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}

func (s *stubUserRepo) GetLatestUsedAtByUserIDs(ctx context.Context, userIDs []int64) (map[int64]*time.Time, error) {
	panic("unexpected GetLatestUsedAtByUserIDs call")
}

func (s *stubUserRepo) GetLatestUsedAtByUserID(ctx context.Context, userID int64) (*time.Time, error) {
	panic("unexpected GetLatestUsedAtByUserID call")
}

func (s *stubUserRepo) UpdateUserLastActiveAt(ctx context.Context, userID int64, activeAt time.Time) error {
	panic("unexpected UpdateUserLastActiveAt call")
}

func (s *stubUserRepo) UpdateBalance(ctx context.Context, id int64, amount float64) error {
	panic("unexpected UpdateBalance call")
}

func (s *stubUserRepo) DeductBalance(ctx context.Context, id int64, amount float64) error {
	panic("unexpected DeductBalance call")
}

func (s *stubUserRepo) UpdateConcurrency(ctx context.Context, id int64, amount int) error {
	panic("unexpected UpdateConcurrency call")
}

func (s *stubUserRepo) BatchSetConcurrency(context.Context, []int64, int) (int, error) { return 0, nil }
func (s *stubUserRepo) BatchAddConcurrency(context.Context, []int64, int) (int, error) { return 0, nil }

func (s *stubUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	panic("unexpected ExistsByEmail call")
}

func (s *stubUserRepo) RemoveGroupFromAllowedGroups(ctx context.Context, groupID int64) (int64, error) {
	panic("unexpected RemoveGroupFromAllowedGroups call")
}

func (s *stubUserRepo) RemoveGroupFromUserAllowedGroups(ctx context.Context, userID int64, groupID int64) error {
	panic("unexpected RemoveGroupFromUserAllowedGroups call")
}

func (s *stubUserRepo) AddGroupToAllowedGroups(ctx context.Context, userID int64, groupID int64) error {
	panic("unexpected AddGroupToAllowedGroups call")
}

func (s *stubUserRepo) ListUserAuthIdentities(ctx context.Context, userID int64) ([]service.UserAuthIdentityRecord, error) {
	panic("unexpected ListUserAuthIdentities call")
}

func (s *stubUserRepo) UnbindUserAuthProvider(context.Context, int64, string) error {
	panic("unexpected UnbindUserAuthProvider call")
}

func (s *stubUserRepo) UpdateTotpSecret(ctx context.Context, userID int64, encryptedSecret *string) error {
	panic("unexpected UpdateTotpSecret call")
}

func (s *stubUserRepo) EnableTotp(ctx context.Context, userID int64) error {
	panic("unexpected EnableTotp call")
}

func (s *stubUserRepo) DisableTotp(ctx context.Context, userID int64) error {
	panic("unexpected DisableTotp call")
}

type adminAuthSettingRepoStub struct {
	values map[string]string
}

func (s *adminAuthSettingRepoStub) Get(ctx context.Context, key string) (*service.Setting, error) {
	value, ok := s.values[key]
	if !ok {
		return nil, service.ErrSettingNotFound
	}
	return &service.Setting{Key: key, Value: value}, nil
}

func (s *adminAuthSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	setting, err := s.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return setting.Value, nil
}

func (s *adminAuthSettingRepoStub) Set(ctx context.Context, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}

func (s *adminAuthSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func (s *adminAuthSettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *adminAuthSettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	result := make(map[string]string, len(s.values))
	for key, value := range s.values {
		result[key] = value
	}
	return result, nil
}

func (s *adminAuthSettingRepoStub) Delete(ctx context.Context, key string) error {
	delete(s.values, key)
	return nil
}
