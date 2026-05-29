//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type adminAPIKeySettingRepoStub struct {
	values map[string]string
}

func (s *adminAPIKeySettingRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	value, ok := s.values[key]
	if !ok {
		return nil, ErrSettingNotFound
	}
	return &Setting{Key: key, Value: value}, nil
}

func (s *adminAPIKeySettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	setting, err := s.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return setting.Value, nil
}

func (s *adminAPIKeySettingRepoStub) Set(ctx context.Context, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}

func (s *adminAPIKeySettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *adminAPIKeySettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *adminAPIKeySettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *adminAPIKeySettingRepoStub) Delete(ctx context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func TestSettingService_GenerateAdminAPIKey_StoresHashRecord(t *testing.T) {
	repo := &adminAPIKeySettingRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	rawKey, err := svc.GenerateAdminAPIKey(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, rawKey)

	stored := repo.values[SettingKeyAdminAPIKey]
	require.NotEmpty(t, stored)
	require.NotEqual(t, rawKey, stored)
	require.NotContains(t, stored, rawKey)

	var record adminAPIKeyStoredRecord
	require.NoError(t, json.Unmarshal([]byte(stored), &record))
	require.Equal(t, adminAPIKeyRecordVersion, record.Version)
	require.Equal(t, "admin:*", record.Scope)
	require.Equal(t, hashAdminAPIKey(rawKey), record.Hash)
	require.False(t, record.CreatedAt.IsZero())
}

func TestSettingService_GetAdminAPIKeyStatusDetail_DoesNotReturnRawKeyForHashedRecord(t *testing.T) {
	rawKey := "admin-test-key-secret"
	createdAt := time.Date(2026, 5, 28, 1, 2, 3, 0, time.UTC)
	stored, err := marshalAdminAPIKeyRecord(rawKey, createdAt)
	require.NoError(t, err)

	repo := &adminAPIKeySettingRepoStub{
		values: map[string]string{
			SettingKeyAdminAPIKey: stored,
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	detail, err := svc.GetAdminAPIKeyStatusDetail(context.Background())
	require.NoError(t, err)
	require.True(t, detail.Exists)
	require.Equal(t, "configured", detail.MaskedKey)
	require.Equal(t, createdAt.Format(time.RFC3339), detail.CreatedAt)
	require.NotEmpty(t, detail.PrincipalID)
	require.NotContains(t, detail.PrincipalID, rawKey)
}

func TestSettingService_ValidateAdminAPIKey_SupportsLegacyPlaintextRecord(t *testing.T) {
	rawKey := "admin-legacy-secret"
	repo := &adminAPIKeySettingRepoStub{
		values: map[string]string{
			SettingKeyAdminAPIKey: rawKey,
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	record, ok, err := svc.ValidateAdminAPIKey(context.Background(), rawKey)
	require.NoError(t, err)
	require.True(t, ok)
	require.False(t, record.LegacyPlain)
	require.Equal(t, "configured", record.MaskedKey)
	require.Equal(t, hashAdminAPIKey(rawKey), record.Hash)

	detail, err := svc.GetAdminAPIKeyStatusDetail(context.Background())
	require.NoError(t, err)
	require.True(t, detail.Exists)
	require.Equal(t, "configured", detail.MaskedKey)
	require.NotEqual(t, rawKey, repo.values[SettingKeyAdminAPIKey])
	require.NotContains(t, repo.values[SettingKeyAdminAPIKey], rawKey)
}
