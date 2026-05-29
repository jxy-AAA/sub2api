//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	resp "github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type stubHandlerModelCatalogRepository struct {
	items []*service.ModelCatalogItem
}

func (s *stubHandlerModelCatalogRepository) List(ctx context.Context) ([]*service.ModelCatalogItem, error) {
	out := make([]*service.ModelCatalogItem, 0, len(s.items))
	for _, item := range s.items {
		cloned := *item
		out = append(out, &cloned)
	}
	return out, nil
}

func (s *stubHandlerModelCatalogRepository) GetByID(ctx context.Context, id int64) (*service.ModelCatalogItem, error) {
	for _, item := range s.items {
		if item.ID == id {
			cloned := *item
			return &cloned, nil
		}
	}
	return nil, nil
}

func (s *stubHandlerModelCatalogRepository) Create(ctx context.Context, item *service.ModelCatalogItem) (*service.ModelCatalogItem, error) {
	cloned := *item
	cloned.ID = int64(len(s.items) + 1)
	s.items = append(s.items, &cloned)
	return &cloned, nil
}

func (s *stubHandlerModelCatalogRepository) Update(ctx context.Context, item *service.ModelCatalogItem) (*service.ModelCatalogItem, error) {
	cloned := *item
	return &cloned, nil
}

func (s *stubHandlerModelCatalogRepository) Delete(ctx context.Context, id int64) error {
	return nil
}

type stubHandlerModelMarketChannelProvider struct {
	availableChannels []service.AvailableChannel
	allChannels       []service.Channel
}

func (s *stubHandlerModelMarketChannelProvider) ListAvailable(ctx context.Context) ([]service.AvailableChannel, error) {
	return append([]service.AvailableChannel(nil), s.availableChannels...), nil
}

func (s *stubHandlerModelMarketChannelProvider) ListAll(ctx context.Context) ([]service.Channel, error) {
	return append([]service.Channel(nil), s.allChannels...), nil
}

type stubHandlerModelMarketGroupProvider struct {
	groups []service.Group
}

func (s *stubHandlerModelMarketGroupProvider) GetAvailableGroups(ctx context.Context, userID int64) ([]service.Group, error) {
	return append([]service.Group(nil), s.groups...), nil
}

func TestModelMarketHandler_Unauthenticated401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &ModelMarketHandler{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/model-market/models", nil)

	h.List(c)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestModelMarketHandler_ListSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &stubHandlerModelCatalogRepository{
		items: []*service.ModelCatalogItem{
			{
				ID:           1,
				ModelID:      "claude-visible",
				DisplayName:  "Claude Visible",
				ProviderKey:  service.PlatformAnthropic,
				Protocol:     service.PlatformAnthropic,
				Capabilities: []string{"chat"},
				Status:       service.ModelCatalogStatusActive,
			},
		},
	}
	channelProvider := &stubHandlerModelMarketChannelProvider{
		availableChannels: []service.AvailableChannel{
			{
				ID:     9,
				Name:   "anthropic",
				Status: service.StatusActive,
				Groups: []service.AvailableGroupRef{
					{ID: 7, Name: "default", Platform: service.PlatformAnthropic, SubscriptionType: service.SubscriptionTypeStandard},
				},
				SupportedModels: []service.SupportedModel{
					{Name: "claude-visible", Platform: service.PlatformAnthropic},
				},
			},
		},
	}
	groupProvider := &stubHandlerModelMarketGroupProvider{
		groups: []service.Group{{ID: 7, Name: "default", Platform: service.PlatformAnthropic, SubscriptionType: service.SubscriptionTypeStandard}},
	}
	svc := service.NewModelMarketService(repo, channelProvider, groupProvider)
	h := NewModelMarketHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/model-market/models", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 77})

	h.List(c)

	require.Equal(t, http.StatusOK, w.Code)
	var envelope resp.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
	require.Equal(t, 0, envelope.Code)

	raw, err := json.Marshal(envelope.Data)
	require.NoError(t, err)
	var items []map[string]any
	require.NoError(t, json.Unmarshal(raw, &items))
	require.Len(t, items, 1)
	require.Equal(t, "claude-visible", items[0]["model_id"])
	require.Equal(t, "Claude Visible", items[0]["display_name"])
}
