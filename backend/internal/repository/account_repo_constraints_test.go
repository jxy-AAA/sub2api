package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAccountRepositoryValidateAccountForPersistenceRejectsInvalidEnums(t *testing.T) {
	repo := &accountRepository{}

	err := repo.validateAccountForPersistence(context.Background(), &service.Account{
		Platform: "invalid-platform",
		Type:     service.AccountTypeOAuth,
		Status:   service.StatusActive,
	})
	require.ErrorIs(t, err, service.ErrAccountInvalidPlatform)

	err = repo.validateAccountForPersistence(context.Background(), &service.Account{
		Platform: service.PlatformOpenAI,
		Type:     "invalid-type",
		Status:   service.StatusActive,
	})
	require.ErrorIs(t, err, service.ErrAccountInvalidType)

	err = repo.validateAccountForPersistence(context.Background(), &service.Account{
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		Status:   "invalid-status",
	})
	require.ErrorIs(t, err, service.ErrAccountInvalidStatus)
}

func TestAccountRepositoryValidateAccountForPersistenceAcceptsCompatiblePlatforms(t *testing.T) {
	repo := &accountRepository{}

	for _, platform := range []string{
		service.PlatformOpenAICompatible,
		service.PlatformAnthropicCompatible,
	} {
		err := repo.validateAccountForPersistence(context.Background(), &service.Account{
			Platform: platform,
			Type:     service.AccountTypeUpstream,
			Status:   service.StatusActive,
		})
		require.NoError(t, err, "platform=%s", platform)
	}
}

func TestAccountRepositoryValidateAccountForPersistenceAcceptsExistingTLSProfile(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &accountRepository{sql: db}

	mock.ExpectQuery("SELECT EXISTS\\(SELECT 1 FROM tls_fingerprint_profiles WHERE id = \\$1\\)").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	err := repo.validateAccountForPersistence(context.Background(), &service.Account{
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		Status:   service.StatusActive,
		Extra: map[string]any{
			"tls_fingerprint_profile_id": "7",
		},
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAccountRepositoryValidateAccountForPersistenceRejectsMissingTLSProfile(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &accountRepository{sql: db}

	mock.ExpectQuery("SELECT EXISTS\\(SELECT 1 FROM tls_fingerprint_profiles WHERE id = \\$1\\)").
		WithArgs(int64(99)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	err := repo.validateAccountForPersistence(context.Background(), &service.Account{
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		Status:   service.StatusActive,
		Extra: map[string]any{
			"tls_fingerprint_profile_id": 99,
		},
	})
	require.ErrorIs(t, err, service.ErrAccountTLSFingerprintProfileNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAccountRepositoryValidateAccountForPersistenceRejectsMalformedTLSProfileReference(t *testing.T) {
	repo := &accountRepository{}

	err := repo.validateAccountForPersistence(context.Background(), &service.Account{
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		Status:   service.StatusActive,
		Extra: map[string]any{
			"tls_fingerprint_profile_id": "not-a-number",
		},
	})
	require.ErrorIs(t, err, service.ErrAccountInvalidTLSFingerprintProfileReference)
}
