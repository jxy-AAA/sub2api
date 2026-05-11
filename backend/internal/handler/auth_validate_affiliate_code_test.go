//go:build unit

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type authValidateSettingRepoStub struct {
	values map[string]string
}

func (s *authValidateSettingRepoStub) Get(context.Context, string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (s *authValidateSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", service.ErrSettingNotFound
}

func (s *authValidateSettingRepoStub) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (s *authValidateSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	values := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			values[key] = value
		}
	}
	return values, nil
}

func (s *authValidateSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *authValidateSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *authValidateSettingRepoStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

func newValidateAffiliateSettingService(enabled bool) *service.SettingService {
	value := "false"
	if enabled {
		value = "true"
	}
	return service.NewSettingService(&authValidateSettingRepoStub{
		values: map[string]string{
			service.SettingKeyAffiliateEnabled: value,
		},
	}, &config.Config{})
}

func TestAuthHandlerValidateAffiliateCodeDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/validate-affiliate-code", bytes.NewReader([]byte(`{"code":"AFF123"}`)))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := NewAuthHandler(&config.Config{}, nil, nil, newValidateAffiliateSettingService(false), nil, nil, nil)
	handler.ValidateAffiliateCode(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Valid     bool   `json:"valid"`
			ErrorCode string `json:"error_code"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.False(t, resp.Data.Valid)
	require.Equal(t, "AFFILIATE_DISABLED", resp.Data.ErrorCode)
}

func TestAuthHandlerValidateAffiliateCodeOmitsUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	settingSvc := newValidateAffiliateSettingService(true)
	affiliateSvc := service.NewAffiliateService(&affiliateDistributionRepoStub{
		affiliateByCode: &service.AffiliateSummary{UserID: 88, AffCode: "AFF123"},
	}, settingSvc, nil, nil)
	authSvc := service.NewAuthService(nil, nil, nil, nil, &config.Config{}, settingSvc, nil, nil, nil, nil, nil, affiliateSvc)
	handler := NewAuthHandler(&config.Config{}, authSvc, nil, settingSvc, nil, nil, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/validate-affiliate-code", bytes.NewReader([]byte(`{"code":"AFF123"}`)))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ValidateAffiliateCode(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Valid   bool   `json:"valid"`
			AffCode string `json:"aff_code"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.True(t, resp.Data.Valid)
	require.Equal(t, "AFF123", resp.Data.AffCode)
	require.NotContains(t, recorder.Body.String(), "user_id")
}
