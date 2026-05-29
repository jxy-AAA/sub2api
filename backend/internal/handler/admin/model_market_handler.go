package admin

import (
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type ModelMarketHandler struct {
	modelMarketService *service.ModelMarketService
}

func NewModelMarketHandler(modelMarketService *service.ModelMarketService) *ModelMarketHandler {
	return &ModelMarketHandler{modelMarketService: modelMarketService}
}

type upsertModelCatalogItemRequest struct {
	ModelID       string         `json:"model_id" binding:"required,max=200"`
	DisplayName   string         `json:"display_name" binding:"omitempty,max=200"`
	ProviderKey   string         `json:"provider_key" binding:"required,max=50"`
	Protocol      string         `json:"protocol" binding:"omitempty,max=50"`
	Capabilities  []string       `json:"capabilities"`
	ContextWindow *int           `json:"context_window"`
	Description   string         `json:"description"`
	Tags          []string       `json:"tags"`
	Status        string         `json:"status" binding:"omitempty,oneof=active hidden disabled"`
	SortOrder     int            `json:"sort_order"`
	Metadata      map[string]any `json:"metadata"`
}

type modelCatalogChannelReferenceResponse struct {
	ChannelID     int64                        `json:"channel_id"`
	ChannelName   string                       `json:"channel_name"`
	ChannelStatus string                       `json:"channel_status"`
	Platform      string                       `json:"platform"`
	GroupIDs      []int64                      `json:"group_ids"`
	Pricing       *channelModelPricingResponse `json:"pricing"`
}

type adminModelCatalogItemResponse struct {
	ID            int64                                  `json:"id"`
	ModelID       string                                 `json:"model_id"`
	DisplayName   string                                 `json:"display_name"`
	ProviderKey   string                                 `json:"provider_key"`
	Protocol      string                                 `json:"protocol"`
	Capabilities  []string                               `json:"capabilities"`
	ContextWindow *int                                   `json:"context_window"`
	Description   string                                 `json:"description"`
	Tags          []string                               `json:"tags"`
	Status        string                                 `json:"status"`
	SortOrder     int                                    `json:"sort_order"`
	Metadata      map[string]any                         `json:"metadata"`
	CreatedAt     string                                 `json:"created_at"`
	UpdatedAt     string                                 `json:"updated_at"`
	ChannelRefs   []modelCatalogChannelReferenceResponse `json:"channel_refs"`
}

type importModelCatalogItemsResponse struct {
	ImportedCount int                             `json:"imported_count"`
	SkippedCount  int                             `json:"skipped_count"`
	Items         []adminModelCatalogItemResponse `json:"items"`
}

// List returns the admin model-catalog view with filters.
// GET /api/v1/admin/model-market/models
func (h *ModelMarketHandler) List(c *gin.Context) {
	items, err := h.modelMarketService.ListCatalog(c.Request.Context(), service.ModelCatalogListFilter{
		ProviderKey: strings.TrimSpace(c.Query("provider_key")),
		Protocol:    strings.TrimSpace(c.Query("protocol")),
		Capability:  strings.TrimSpace(c.Query("capability")),
		Status:      strings.TrimSpace(c.Query("status")),
		Keyword:     strings.TrimSpace(c.Query("keyword")),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]adminModelCatalogItemResponse, 0, len(items))
	for _, item := range items {
		out = append(out, adminModelCatalogItemToResponse(item))
	}
	response.Success(c, out)
}

// Create creates a model-catalog item.
// POST /api/v1/admin/model-market/models
func (h *ModelMarketHandler) Create(c *gin.Context) {
	var req upsertModelCatalogItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}

	item, err := h.modelMarketService.CreateCatalogItem(c.Request.Context(), toUpsertModelCatalogItemInput(req))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, adminModelCatalogItemToResponse(&service.AdminModelCatalogItem{
		ModelCatalogItem: item,
		ChannelRefs:      []service.ModelMarketChannelReference{},
	}))
}

// Update fully updates a model-catalog item.
// PUT /api/v1/admin/model-market/models/:id
func (h *ModelMarketHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_MODEL_CATALOG_ITEM_ID", "invalid model catalog item id"))
		return
	}

	var req upsertModelCatalogItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}

	item, err := h.modelMarketService.UpdateCatalogItem(c.Request.Context(), id, toUpsertModelCatalogItemInput(req))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, adminModelCatalogItemToResponse(&service.AdminModelCatalogItem{
		ModelCatalogItem: item,
		ChannelRefs:      []service.ModelMarketChannelReference{},
	}))
}

// Delete soft-deletes a model-catalog item.
// DELETE /api/v1/admin/model-market/models/:id
func (h *ModelMarketHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_MODEL_CATALOG_ITEM_ID", "invalid model catalog item id"))
		return
	}

	if err := h.modelMarketService.DeleteCatalogItem(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Model catalog item deleted successfully"})
}

// ImportFromChannels imports missing model-catalog items from current channels.
// POST /api/v1/admin/model-market/models/import-from-channels
func (h *ModelMarketHandler) ImportFromChannels(c *gin.Context) {
	result, err := h.modelMarketService.ImportFromChannels(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	items := make([]adminModelCatalogItemResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, adminModelCatalogItemToResponse(&service.AdminModelCatalogItem{
			ModelCatalogItem: item,
			ChannelRefs:      []service.ModelMarketChannelReference{},
		}))
	}

	response.Success(c, importModelCatalogItemsResponse{
		ImportedCount: result.ImportedCount,
		SkippedCount:  result.SkippedCount,
		Items:         items,
	})
}

func toUpsertModelCatalogItemInput(req upsertModelCatalogItemRequest) service.CreateModelCatalogItemInput {
	return service.CreateModelCatalogItemInput{
		ModelID:       req.ModelID,
		DisplayName:   req.DisplayName,
		ProviderKey:   req.ProviderKey,
		Protocol:      req.Protocol,
		Capabilities:  req.Capabilities,
		ContextWindow: req.ContextWindow,
		Description:   req.Description,
		Tags:          req.Tags,
		Status:        req.Status,
		SortOrder:     req.SortOrder,
		Metadata:      req.Metadata,
	}
}

func adminModelCatalogItemToResponse(item *service.AdminModelCatalogItem) adminModelCatalogItemResponse {
	if item == nil || item.ModelCatalogItem == nil {
		return adminModelCatalogItemResponse{
			Capabilities: []string{},
			Tags:         []string{},
			Metadata:     map[string]any{},
			ChannelRefs:  []modelCatalogChannelReferenceResponse{},
		}
	}

	channelRefs := make([]modelCatalogChannelReferenceResponse, 0, len(item.ChannelRefs))
	for _, ref := range item.ChannelRefs {
		var pricing *channelModelPricingResponse
		if ref.Pricing != nil {
			pricingValue := pricingToResponse(ref.Pricing)
			pricing = &pricingValue
		}
		channelRefs = append(channelRefs, modelCatalogChannelReferenceResponse{
			ChannelID:     ref.ChannelID,
			ChannelName:   ref.ChannelName,
			ChannelStatus: ref.ChannelStatus,
			Platform:      ref.Platform,
			GroupIDs:      append([]int64(nil), ref.GroupIDs...),
			Pricing:       pricing,
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

	return adminModelCatalogItemResponse{
		ID:            item.ID,
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
		CreatedAt:     item.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:     item.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		ChannelRefs:   channelRefs,
	}
}
