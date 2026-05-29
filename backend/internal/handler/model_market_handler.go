package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type ModelMarketHandler struct {
	modelMarketService *service.ModelMarketService
}

func NewModelMarketHandler(modelMarketService *service.ModelMarketService) *ModelMarketHandler {
	return &ModelMarketHandler{modelMarketService: modelMarketService}
}

type userModelMarketChannelResponse struct {
	ID          int64                      `json:"id"`
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	Platform    string                     `json:"platform"`
	Groups      []userAvailableGroup       `json:"groups"`
	Pricing     *userSupportedModelPricing `json:"pricing"`
}

type userModelMarketItemResponse struct {
	ModelID       string                           `json:"model_id"`
	DisplayName   string                           `json:"display_name"`
	ProviderKey   string                           `json:"provider_key"`
	Protocol      string                           `json:"protocol"`
	Capabilities  []string                         `json:"capabilities"`
	ContextWindow *int                             `json:"context_window"`
	Description   string                           `json:"description"`
	Tags          []string                         `json:"tags"`
	Status        string                           `json:"status"`
	SortOrder     int                              `json:"sort_order"`
	Metadata      map[string]any                   `json:"metadata"`
	Channels      []userModelMarketChannelResponse `json:"channels"`
}

// List returns the current user's visible model-market entries.
// GET /api/v1/model-market/models
func (h *ModelMarketHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	items, err := h.modelMarketService.ListUserModels(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]userModelMarketItemResponse, 0, len(items))
	for _, item := range items {
		out = append(out, toUserModelMarketItemResponse(item))
	}
	response.Success(c, out)
}

func toUserModelMarketItemResponse(item *service.UserModelMarketItem) userModelMarketItemResponse {
	if item == nil {
		return userModelMarketItemResponse{
			Capabilities: []string{},
			Tags:         []string{},
			Metadata:     map[string]any{},
			Channels:     []userModelMarketChannelResponse{},
		}
	}

	channels := make([]userModelMarketChannelResponse, 0, len(item.Channels))
	for _, channel := range item.Channels {
		channels = append(channels, userModelMarketChannelResponse{
			ID:          channel.ID,
			Name:        channel.Name,
			Description: channel.Description,
			Platform:    channel.Platform,
			Groups:      toUserAvailableGroups(channel.Groups),
			Pricing:     toUserPricing(channel.Pricing),
		})
	}

	capabilities := append([]string(nil), item.Capabilities...)
	if capabilities == nil {
		capabilities = []string{}
	}
	tags := append([]string(nil), item.Tags...)
	if tags == nil {
		tags = []string{}
	}
	metadata := item.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}

	return userModelMarketItemResponse{
		ModelID:       item.ModelID,
		DisplayName:   item.DisplayName,
		ProviderKey:   item.ProviderKey,
		Protocol:      item.Protocol,
		Capabilities:  capabilities,
		ContextWindow: item.ContextWindow,
		Description:   item.Description,
		Tags:          tags,
		Status:        item.Status,
		SortOrder:     item.SortOrder,
		Metadata:      metadata,
		Channels:      channels,
	}
}

func toUserAvailableGroups(groups []service.AvailableGroupRef) []userAvailableGroup {
	out := make([]userAvailableGroup, 0, len(groups))
	for _, group := range groups {
		out = append(out, userAvailableGroup{
			ID:               group.ID,
			Name:             group.Name,
			Platform:         group.Platform,
			SubscriptionType: group.SubscriptionType,
			RateMultiplier:   group.RateMultiplier,
			IsExclusive:      group.IsExclusive,
		})
	}
	return out
}
