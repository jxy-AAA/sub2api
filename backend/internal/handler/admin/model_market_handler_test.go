//go:build unit

package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	resp "github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type stubAdminModelCatalogRepository struct {
	items []*service.ModelCatalogItem
}

func (s *stubAdminModelCatalogRepository) List(ctx context.Context) ([]*service.ModelCatalogItem, error) {
	out := make([]*service.ModelCatalogItem, 0, len(s.items))
	for _, item := range s.items {
		cloned := *item
		out = append(out, &cloned)
	}
	return out, nil
}

func (s *stubAdminModelCatalogRepository) GetByID(ctx context.Context, id int64) (*service.ModelCatalogItem, error) {
	for _, item := range s.items {
		if item.ID == id {
			cloned := *item
			return &cloned, nil
		}
	}
	return nil, nil
}

func (s *stubAdminModelCatalogRepository) Create(ctx context.Context, item *service.ModelCatalogItem) (*service.ModelCatalogItem, error) {
	cloned := *item
	cloned.ID = int64(len(s.items) + 1)
	s.items = append(s.items, &cloned)
	return &cloned, nil
}

func (s *stubAdminModelCatalogRepository) Update(ctx context.Context, item *service.ModelCatalogItem) (*service.ModelCatalogItem, error) {
	cloned := *item
	return &cloned, nil
}

func (s *stubAdminModelCatalogRepository) Delete(ctx context.Context, id int64) error {
	return nil
}

type stubAdminChannelProvider struct {
	allChannels []service.Channel
}

func (s *stubAdminChannelProvider) ListAvailable(ctx context.Context) ([]service.AvailableChannel, error) {
	return []service.AvailableChannel{}, nil
}

func (s *stubAdminChannelProvider) ListAll(ctx context.Context) ([]service.Channel, error) {
	return append([]service.Channel(nil), s.allChannels...), nil
}

type stubAdminGroupProvider struct{}

func (s *stubAdminGroupProvider) GetAvailableGroups(ctx context.Context, userID int64) ([]service.Group, error) {
	return []service.Group{}, nil
}

func TestAdminModelMarketHandler_CreateSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &stubAdminModelCatalogRepository{}
	channelProvider := &stubAdminChannelProvider{}
	groupProvider := &stubAdminGroupProvider{}
	svc := service.NewModelMarketService(repo, channelProvider, groupProvider)
	h := NewModelMarketHandler(svc)

	body := bytes.NewBufferString(`{
		"model_id":"gpt-4o",
		"display_name":"GPT-4o",
		"provider_key":"openai",
		"protocol":"openai",
		"capabilities":["chat"],
		"status":"active"
	}`)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/model-market/models", body)
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	require.Equal(t, http.StatusOK, w.Code)
	var envelope resp.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
	require.Equal(t, 0, envelope.Code)
}

func TestAdminModelMarketHandler_ListSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &stubAdminModelCatalogRepository{
		items: []*service.ModelCatalogItem{
			{
				ID:           1,
				ModelID:      "gpt-4o",
				DisplayName:  "GPT-4o",
				ProviderKey:  service.PlatformOpenAI,
				Protocol:     service.PlatformOpenAI,
				Capabilities: []string{"chat"},
				Status:       service.ModelCatalogStatusActive,
			},
		},
	}
	channelProvider := &stubAdminChannelProvider{
		allChannels: []service.Channel{
			{
				ID:       9,
				Name:     "openai",
				Status:   service.StatusActive,
				GroupIDs: []int64{3},
				ModelPricing: []service.ChannelModelPricing{
					{Platform: service.PlatformOpenAI, Models: []string{"gpt-4o"}},
				},
			},
		},
	}
	groupProvider := &stubAdminGroupProvider{}
	svc := service.NewModelMarketService(repo, channelProvider, groupProvider)
	h := NewModelMarketHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/model-market/models?keyword=gpt", nil)

	h.List(c)

	require.Equal(t, http.StatusOK, w.Code)
	var envelope resp.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
	require.Equal(t, 0, envelope.Code)
}
