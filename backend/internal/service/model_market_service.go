package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	ModelCatalogStatusActive   = "active"
	ModelCatalogStatusHidden   = "hidden"
	ModelCatalogStatusDisabled = "disabled"
)

var (
	ErrModelCatalogItemNotFound = infraerrors.NotFound("MODEL_CATALOG_ITEM_NOT_FOUND", "model catalog item not found")
	ErrModelCatalogItemExists   = infraerrors.Conflict("MODEL_CATALOG_ITEM_EXISTS", "model catalog item already exists")
)

type ModelCatalogItem struct {
	ID            int64
	ModelID       string
	DisplayName   string
	ProviderKey   string
	Protocol      string
	Capabilities  []string
	ContextWindow *int
	Description   string
	Tags          []string
	Status        string
	SortOrder     int
	Metadata      map[string]any
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time
}

type ModelCatalogListFilter struct {
	ProviderKey string
	Protocol    string
	Capability  string
	Status      string
	Keyword     string
}

type CreateModelCatalogItemInput struct {
	ModelID       string
	DisplayName   string
	ProviderKey   string
	Protocol      string
	Capabilities  []string
	ContextWindow *int
	Description   string
	Tags          []string
	Status        string
	SortOrder     int
	Metadata      map[string]any
}

type UpdateModelCatalogItemInput = CreateModelCatalogItemInput

type ModelCatalogRepository interface {
	List(ctx context.Context) ([]*ModelCatalogItem, error)
	GetByID(ctx context.Context, id int64) (*ModelCatalogItem, error)
	Create(ctx context.Context, item *ModelCatalogItem) (*ModelCatalogItem, error)
	Update(ctx context.Context, item *ModelCatalogItem) (*ModelCatalogItem, error)
	Delete(ctx context.Context, id int64) error
}

type modelMarketChannelProvider interface {
	ListAvailable(ctx context.Context) ([]AvailableChannel, error)
	ListAll(ctx context.Context) ([]Channel, error)
}

type modelMarketGroupProvider interface {
	GetAvailableGroups(ctx context.Context, userID int64) ([]Group, error)
}

type ModelMarketChannelReference struct {
	ChannelID     int64
	ChannelName   string
	ChannelStatus string
	Platform      string
	GroupIDs      []int64
	Pricing       *ChannelModelPricing
}

type AdminModelCatalogItem struct {
	*ModelCatalogItem
	ChannelRefs []ModelMarketChannelReference
}

type UserModelMarketChannel struct {
	ID          int64
	Name        string
	Description string
	Platform    string
	Groups      []AvailableGroupRef
	Pricing     *ChannelModelPricing
}

type UserModelMarketItem struct {
	ModelID       string
	DisplayName   string
	ProviderKey   string
	Protocol      string
	Capabilities  []string
	ContextWindow *int
	Description   string
	Tags          []string
	Status        string
	SortOrder     int
	Metadata      map[string]any
	Channels      []UserModelMarketChannel
}

type ImportModelCatalogItemsResult struct {
	ImportedCount int
	SkippedCount  int
	Items         []*ModelCatalogItem
}

type ModelMarketService struct {
	repo            ModelCatalogRepository
	channelProvider modelMarketChannelProvider
	groupProvider   modelMarketGroupProvider
}

func NewModelMarketService(
	repo ModelCatalogRepository,
	channelProvider modelMarketChannelProvider,
	groupProvider modelMarketGroupProvider,
) *ModelMarketService {
	return &ModelMarketService{
		repo:            repo,
		channelProvider: channelProvider,
		groupProvider:   groupProvider,
	}
}

func (s *ModelMarketService) ListCatalog(ctx context.Context, filter ModelCatalogListFilter) ([]*AdminModelCatalogItem, error) {
	items, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list model catalog items: %w", err)
	}

	channelRefs, err := s.buildAdminChannelReferences(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]*AdminModelCatalogItem, 0, len(items))
	for _, item := range items {
		normalized := normalizeModelCatalogItem(item)
		if !matchesModelCatalogFilter(normalized, filter) {
			continue
		}
		key := modelCatalogKey(normalized.ProviderKey, normalized.ModelID)
		filtered = append(filtered, &AdminModelCatalogItem{
			ModelCatalogItem: normalized,
			ChannelRefs:      channelRefs[key],
		})
	}
	sortAdminModelCatalogItems(filtered)
	return filtered, nil
}

func (s *ModelMarketService) CreateCatalogItem(ctx context.Context, input CreateModelCatalogItemInput) (*ModelCatalogItem, error) {
	item, err := newModelCatalogItemFromInput(input)
	if err != nil {
		return nil, err
	}
	created, err := s.repo.Create(ctx, item)
	if err != nil {
		return nil, fmt.Errorf("create model catalog item: %w", err)
	}
	return normalizeModelCatalogItem(created), nil
}

func (s *ModelMarketService) UpdateCatalogItem(ctx context.Context, id int64, input UpdateModelCatalogItemInput) (*ModelCatalogItem, error) {
	if id <= 0 {
		return nil, infraerrors.BadRequest("INVALID_MODEL_CATALOG_ITEM_ID", "invalid model catalog item id")
	}

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get model catalog item: %w", err)
	}
	if existing == nil {
		return nil, ErrModelCatalogItemNotFound
	}

	item, err := newModelCatalogItemFromInput(input)
	if err != nil {
		return nil, err
	}
	item.ID = id
	item.CreatedAt = existing.CreatedAt

	updated, err := s.repo.Update(ctx, item)
	if err != nil {
		return nil, fmt.Errorf("update model catalog item: %w", err)
	}
	return normalizeModelCatalogItem(updated), nil
}

func (s *ModelMarketService) DeleteCatalogItem(ctx context.Context, id int64) error {
	if id <= 0 {
		return infraerrors.BadRequest("INVALID_MODEL_CATALOG_ITEM_ID", "invalid model catalog item id")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete model catalog item: %w", err)
	}
	return nil
}

func (s *ModelMarketService) ImportFromChannels(ctx context.Context) (*ImportModelCatalogItemsResult, error) {
	channels, err := s.channelProvider.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list channels: %w", err)
	}

	existingItems, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list model catalog items: %w", err)
	}

	existingByKey := make(map[string]*ModelCatalogItem, len(existingItems))
	for _, item := range existingItems {
		normalized := normalizeModelCatalogItem(item)
		existingByKey[modelCatalogKey(normalized.ProviderKey, normalized.ModelID)] = normalized
	}

	candidates := buildImportCandidates(channels)
	result := &ImportModelCatalogItemsResult{
		Items: make([]*ModelCatalogItem, 0, len(candidates)),
	}

	for _, candidate := range candidates {
		key := modelCatalogKey(candidate.ProviderKey, candidate.ModelID)
		if _, exists := existingByKey[key]; exists {
			result.SkippedCount++
			continue
		}

		created, err := s.repo.Create(ctx, candidate)
		if err != nil {
			return nil, fmt.Errorf("import model catalog item %s/%s: %w", candidate.ProviderKey, candidate.ModelID, err)
		}
		normalized := normalizeModelCatalogItem(created)
		existingByKey[key] = normalized
		result.Items = append(result.Items, normalized)
	}

	result.ImportedCount = len(result.Items)
	sort.SliceStable(result.Items, func(i, j int) bool {
		return compareModelCatalogItems(result.Items[i], result.Items[j]) < 0
	})
	return result, nil
}

func (s *ModelMarketService) ListUserModels(ctx context.Context, userID int64) ([]*UserModelMarketItem, error) {
	if userID <= 0 {
		return nil, infraerrors.BadRequest("INVALID_USER_ID", "invalid user id")
	}

	availableGroups, err := s.groupProvider.GetAvailableGroups(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get available groups: %w", err)
	}
	allowedGroupIDs := make(map[int64]struct{}, len(availableGroups))
	for i := range availableGroups {
		allowedGroupIDs[availableGroups[i].ID] = struct{}{}
	}

	channels, err := s.channelProvider.ListAvailable(ctx)
	if err != nil {
		return nil, fmt.Errorf("list available channels: %w", err)
	}

	catalogItems, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list model catalog items: %w", err)
	}
	activeCatalog, suppressedCatalog := buildCatalogVisibilityIndexes(catalogItems)

	type aggregate struct {
		item       *UserModelMarketItem
		channelSet map[string]struct{}
	}

	aggregates := make(map[string]*aggregate)
	for _, channel := range channels {
		if channel.Status != StatusActive {
			continue
		}

		visibleGroups := filterAllowedAvailableGroups(channel.Groups, allowedGroupIDs)
		if len(visibleGroups) == 0 {
			continue
		}
		visiblePlatforms := make(map[string]struct{}, len(visibleGroups))
		for _, group := range visibleGroups {
			if group.Platform == "" {
				continue
			}
			visiblePlatforms[group.Platform] = struct{}{}
		}

		for _, supportedModel := range channel.SupportedModels {
			if _, ok := visiblePlatforms[supportedModel.Platform]; !ok {
				continue
			}

			key := modelCatalogKey(supportedModel.Platform, supportedModel.Name)
			if _, suppressed := suppressedCatalog[key]; suppressed {
				continue
			}

			catalogItem := activeCatalog[key]
			agg, exists := aggregates[key]
			if !exists {
				agg = &aggregate{
					item: &UserModelMarketItem{
						ModelID:       strings.TrimSpace(supportedModel.Name),
						DisplayName:   strings.TrimSpace(supportedModel.Name),
						ProviderKey:   normalizeLowerString(supportedModel.Platform),
						Protocol:      defaultProtocolForPlatform(supportedModel.Platform),
						Capabilities:  []string{},
						Description:   "",
						Tags:          []string{},
						Status:        ModelCatalogStatusActive,
						SortOrder:     0,
						Metadata:      map[string]any{},
						Channels:      []UserModelMarketChannel{},
						ContextWindow: nil,
					},
					channelSet: make(map[string]struct{}),
				}
				if catalogItem != nil {
					applyCatalogMetadataToUserItem(agg.item, catalogItem)
				} else {
					agg.item.Capabilities = inferCapabilities(supportedModel.Name, supportedModel.Pricing)
				}
				aggregates[key] = agg
			}

			channelKey := fmt.Sprintf("%d:%s:%s", channel.ID, channel.Name, supportedModel.Platform)
			if _, seen := agg.channelSet[channelKey]; seen {
				continue
			}
			agg.channelSet[channelKey] = struct{}{}
			agg.item.Channels = append(agg.item.Channels, UserModelMarketChannel{
				ID:          channel.ID,
				Name:        channel.Name,
				Description: channel.Description,
				Platform:    supportedModel.Platform,
				Groups:      cloneAvailableGroupRefs(visibleGroups, supportedModel.Platform),
				Pricing:     cloneChannelPricing(supportedModel.Pricing),
			})
		}
	}

	out := make([]*UserModelMarketItem, 0, len(aggregates))
	for _, aggregate := range aggregates {
		sort.SliceStable(aggregate.item.Channels, func(i, j int) bool {
			if aggregate.item.Channels[i].Name != aggregate.item.Channels[j].Name {
				return strings.ToLower(aggregate.item.Channels[i].Name) < strings.ToLower(aggregate.item.Channels[j].Name)
			}
			return aggregate.item.Channels[i].Platform < aggregate.item.Channels[j].Platform
		})
		out = append(out, aggregate.item)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].SortOrder != out[j].SortOrder {
			return out[i].SortOrder < out[j].SortOrder
		}
		if strings.ToLower(out[i].DisplayName) != strings.ToLower(out[j].DisplayName) {
			return strings.ToLower(out[i].DisplayName) < strings.ToLower(out[j].DisplayName)
		}
		return strings.ToLower(out[i].ModelID) < strings.ToLower(out[j].ModelID)
	})

	return out, nil
}

func newModelCatalogItemFromInput(input CreateModelCatalogItemInput) (*ModelCatalogItem, error) {
	item := &ModelCatalogItem{
		ModelID:       strings.TrimSpace(input.ModelID),
		DisplayName:   strings.TrimSpace(input.DisplayName),
		ProviderKey:   normalizeLowerString(input.ProviderKey),
		Protocol:      normalizeModelCatalogProtocol(input.Protocol, input.ProviderKey),
		Capabilities:  normalizeStringList(input.Capabilities),
		ContextWindow: input.ContextWindow,
		Description:   strings.TrimSpace(input.Description),
		Tags:          normalizeStringList(input.Tags),
		Status:        normalizeModelCatalogStatus(input.Status),
		SortOrder:     input.SortOrder,
		Metadata:      cloneMetadataMap(input.Metadata),
	}

	if item.ModelID == "" {
		return nil, infraerrors.BadRequest("MODEL_CATALOG_MODEL_ID_REQUIRED", "model_id is required")
	}
	if item.DisplayName == "" {
		item.DisplayName = item.ModelID
	}
	if item.ProviderKey == "" {
		return nil, infraerrors.BadRequest("MODEL_CATALOG_PROVIDER_KEY_REQUIRED", "provider_key is required")
	}
	if item.Protocol == "" {
		return nil, infraerrors.BadRequest("MODEL_CATALOG_PROTOCOL_REQUIRED", "protocol is required")
	}
	if item.Status == "" {
		return nil, infraerrors.BadRequest("MODEL_CATALOG_STATUS_INVALID", "status must be active, hidden, or disabled")
	}
	if item.ContextWindow != nil && *item.ContextWindow <= 0 {
		return nil, infraerrors.BadRequest("MODEL_CATALOG_CONTEXT_WINDOW_INVALID", "context_window must be greater than 0")
	}
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}

	return item, nil
}

func normalizeModelCatalogItem(item *ModelCatalogItem) *ModelCatalogItem {
	if item == nil {
		return nil
	}
	normalized := *item
	normalized.ModelID = strings.TrimSpace(normalized.ModelID)
	normalized.DisplayName = strings.TrimSpace(normalized.DisplayName)
	if normalized.DisplayName == "" {
		normalized.DisplayName = normalized.ModelID
	}
	normalized.ProviderKey = normalizeLowerString(normalized.ProviderKey)
	normalized.Protocol = normalizeModelCatalogProtocol(normalized.Protocol, normalized.ProviderKey)
	if normalized.Protocol == "" {
		normalized.Protocol = defaultProtocolForPlatform(normalized.ProviderKey)
	}
	normalized.Capabilities = normalizeStringList(normalized.Capabilities)
	normalized.Description = strings.TrimSpace(normalized.Description)
	normalized.Tags = normalizeStringList(normalized.Tags)
	normalized.Status = normalizeModelCatalogStatus(normalized.Status)
	if normalized.Metadata == nil {
		normalized.Metadata = map[string]any{}
	} else {
		normalized.Metadata = cloneMetadataMap(normalized.Metadata)
	}
	return &normalized
}

func normalizeModelCatalogStatus(status string) string {
	switch normalizeLowerString(status) {
	case "", ModelCatalogStatusActive:
		return ModelCatalogStatusActive
	case ModelCatalogStatusHidden:
		return ModelCatalogStatusHidden
	case ModelCatalogStatusDisabled:
		return ModelCatalogStatusDisabled
	default:
		return ""
	}
}

func sortAdminModelCatalogItems(items []*AdminModelCatalogItem) {
	sort.SliceStable(items, func(i, j int) bool {
		return compareModelCatalogItems(items[i].ModelCatalogItem, items[j].ModelCatalogItem) < 0
	})
}

func compareModelCatalogItems(left, right *ModelCatalogItem) int {
	if left == nil && right == nil {
		return 0
	}
	if left == nil {
		return 1
	}
	if right == nil {
		return -1
	}
	if left.SortOrder != right.SortOrder {
		if left.SortOrder < right.SortOrder {
			return -1
		}
		return 1
	}
	if strings.ToLower(left.DisplayName) != strings.ToLower(right.DisplayName) {
		if strings.ToLower(left.DisplayName) < strings.ToLower(right.DisplayName) {
			return -1
		}
		return 1
	}
	if strings.ToLower(left.ModelID) != strings.ToLower(right.ModelID) {
		if strings.ToLower(left.ModelID) < strings.ToLower(right.ModelID) {
			return -1
		}
		return 1
	}
	if left.ID < right.ID {
		return -1
	}
	if left.ID > right.ID {
		return 1
	}
	return 0
}

func matchesModelCatalogFilter(item *ModelCatalogItem, filter ModelCatalogListFilter) bool {
	if item == nil {
		return false
	}
	if providerKey := normalizeLowerString(filter.ProviderKey); providerKey != "" && item.ProviderKey != providerKey {
		return false
	}
	if protocol := normalizeLowerString(filter.Protocol); protocol != "" && item.Protocol != protocol {
		return false
	}
	if status := normalizeLowerString(filter.Status); status != "" && item.Status != status {
		return false
	}
	if capability := normalizeLowerString(filter.Capability); capability != "" {
		matched := false
		for _, itemCapability := range item.Capabilities {
			if normalizeLowerString(itemCapability) == capability {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if keyword := normalizeLowerString(strings.TrimSpace(filter.Keyword)); keyword != "" {
		if !strings.Contains(normalizeLowerString(item.ModelID), keyword) &&
			!strings.Contains(normalizeLowerString(item.DisplayName), keyword) &&
			!strings.Contains(normalizeLowerString(item.ProviderKey), keyword) &&
			!strings.Contains(normalizeLowerString(item.Protocol), keyword) &&
			!strings.Contains(normalizeLowerString(item.Description), keyword) &&
			!containsStringKeyword(item.Tags, keyword) {
			return false
		}
	}
	return true
}

func buildCatalogVisibilityIndexes(items []*ModelCatalogItem) (map[string]*ModelCatalogItem, map[string]struct{}) {
	active := make(map[string]*ModelCatalogItem)
	suppressed := make(map[string]struct{})
	for _, item := range items {
		normalized := normalizeModelCatalogItem(item)
		key := modelCatalogKey(normalized.ProviderKey, normalized.ModelID)
		switch normalized.Status {
		case ModelCatalogStatusActive:
			active[key] = normalized
		case ModelCatalogStatusHidden, ModelCatalogStatusDisabled:
			suppressed[key] = struct{}{}
		}
	}
	return active, suppressed
}

func buildImportCandidates(channels []Channel) []*ModelCatalogItem {
	candidatesByKey := make(map[string]*ModelCatalogItem)
	for i := range channels {
		supportedModels := channels[i].SupportedModels()
		for _, supportedModel := range supportedModels {
			modelID := strings.TrimSpace(supportedModel.Name)
			providerKey := normalizeLowerString(supportedModel.Platform)
			if modelID == "" || providerKey == "" {
				continue
			}
			key := modelCatalogKey(providerKey, modelID)
			if _, exists := candidatesByKey[key]; exists {
				continue
			}

			candidatesByKey[key] = normalizeModelCatalogItem(&ModelCatalogItem{
				ModelID:       modelID,
				DisplayName:   modelID,
				ProviderKey:   providerKey,
				Protocol:      defaultProtocolForPlatform(providerKey),
				Capabilities:  inferCapabilities(modelID, supportedModel.Pricing),
				ContextWindow: nil,
				Description:   "",
				Tags:          []string{},
				Status:        ModelCatalogStatusActive,
				SortOrder:     0,
				Metadata:      map[string]any{},
			})
		}
	}

	candidates := make([]*ModelCatalogItem, 0, len(candidatesByKey))
	for _, candidate := range candidatesByKey {
		candidates = append(candidates, candidate)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return compareModelCatalogItems(candidates[i], candidates[j]) < 0
	})
	return candidates
}

func (s *ModelMarketService) buildAdminChannelReferences(ctx context.Context) (map[string][]ModelMarketChannelReference, error) {
	channels, err := s.channelProvider.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list channels: %w", err)
	}
	availableChannels, err := s.channelProvider.ListAvailable(ctx)
	if err != nil {
		return nil, fmt.Errorf("list available channels: %w", err)
	}
	availableGroupIndex := buildAvailableChannelGroupIndex(availableChannels)

	refs := make(map[string][]ModelMarketChannelReference)
	for i := range channels {
		channel := channels[i]
		supportedModels := channel.SupportedModels()
		for _, supportedModel := range supportedModels {
			modelID := strings.TrimSpace(supportedModel.Name)
			providerKey := normalizeLowerString(supportedModel.Platform)
			if modelID == "" || providerKey == "" {
				continue
			}
			key := modelCatalogKey(providerKey, modelID)
			groupIDs := append([]int64(nil), channel.GroupIDs...)
			if groupsByPlatform, ok := availableGroupIndex[channel.ID]; ok {
				groupIDs = append([]int64(nil), groupsByPlatform[providerKey]...)
			}
			refs[key] = append(refs[key], ModelMarketChannelReference{
				ChannelID:     channel.ID,
				ChannelName:   channel.Name,
				ChannelStatus: channel.Status,
				Platform:      supportedModel.Platform,
				GroupIDs:      groupIDs,
				Pricing:       cloneChannelPricing(supportedModel.Pricing),
			})
		}
	}

	for key := range refs {
		sort.SliceStable(refs[key], func(i, j int) bool {
			if refs[key][i].ChannelName != refs[key][j].ChannelName {
				return strings.ToLower(refs[key][i].ChannelName) < strings.ToLower(refs[key][j].ChannelName)
			}
			return refs[key][i].ChannelID < refs[key][j].ChannelID
		})
	}

	return refs, nil
}

func buildAvailableChannelGroupIndex(channels []AvailableChannel) map[int64]map[string][]int64 {
	index := make(map[int64]map[string][]int64, len(channels))
	for i := range channels {
		platformGroups := make(map[string][]int64)
		for _, group := range channels[i].Groups {
			platformKey := normalizeLowerString(group.Platform)
			platformGroups[platformKey] = append(platformGroups[platformKey], group.ID)
		}
		index[channels[i].ID] = platformGroups
	}
	return index
}

func applyCatalogMetadataToUserItem(item *UserModelMarketItem, catalogItem *ModelCatalogItem) {
	if item == nil || catalogItem == nil {
		return
	}
	if catalogItem.DisplayName != "" {
		item.DisplayName = catalogItem.DisplayName
	}
	if catalogItem.ProviderKey != "" {
		item.ProviderKey = catalogItem.ProviderKey
	}
	if catalogItem.Protocol != "" {
		item.Protocol = catalogItem.Protocol
	}
	item.Capabilities = append([]string(nil), catalogItem.Capabilities...)
	item.ContextWindow = cloneOptionalIntPointer(catalogItem.ContextWindow)
	item.Description = catalogItem.Description
	item.Tags = append([]string(nil), catalogItem.Tags...)
	item.Status = catalogItem.Status
	item.SortOrder = catalogItem.SortOrder
	item.Metadata = cloneMetadataMap(catalogItem.Metadata)
}

func filterAllowedAvailableGroups(groups []AvailableGroupRef, allowed map[int64]struct{}) []AvailableGroupRef {
	filtered := make([]AvailableGroupRef, 0, len(groups))
	for _, group := range groups {
		if _, ok := allowed[group.ID]; !ok {
			continue
		}
		filtered = append(filtered, group)
	}
	return filtered
}

func cloneAvailableGroupRefs(groups []AvailableGroupRef, platform string) []AvailableGroupRef {
	out := make([]AvailableGroupRef, 0, len(groups))
	for _, group := range groups {
		if platform != "" && group.Platform != platform {
			continue
		}
		out = append(out, group)
	}
	return out
}

func cloneChannelPricing(pricing *ChannelModelPricing) *ChannelModelPricing {
	if pricing == nil {
		return nil
	}
	cloned := pricing.Clone()
	return &cloned
}

func cloneMetadataMap(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}
	return cloned
}

func cloneOptionalIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func normalizeLowerString(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeStringList(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return []string{}
	}
	return normalized
}

func containsStringKeyword(values []string, keyword string) bool {
	for _, value := range values {
		if strings.Contains(normalizeLowerString(value), keyword) {
			return true
		}
	}
	return false
}

func modelCatalogKey(providerKey, modelID string) string {
	return normalizeLowerString(providerKey) + "\x00" + strings.ToLower(strings.TrimSpace(modelID))
}

func defaultProtocolForPlatform(platform string) string {
	normalized := normalizeLowerString(platform)
	if IsOpenAIProtocolPlatform(normalized) || normalized == PlatformGemini {
		return PlatformOpenAI
	}
	if IsAnthropicProtocolPlatform(normalized) || normalized == PlatformAntigravity {
		return PlatformAnthropic
	}
	return PlatformOpenAI
}

func normalizeModelCatalogProtocol(protocol, providerKey string) string {
	switch normalizeLowerString(protocol) {
	case "":
		return defaultProtocolForPlatform(providerKey)
	case PlatformOpenAI, PlatformOpenAICompatible, PlatformGemini:
		return PlatformOpenAI
	case PlatformAnthropic, PlatformAnthropicCompatible, PlatformAntigravity:
		return PlatformAnthropic
	default:
		return ""
	}
}

func inferCapabilities(modelID string, pricing *ChannelModelPricing) []string {
	lowerModelID := strings.ToLower(strings.TrimSpace(modelID))
	capabilities := []string{}

	if pricing != nil && (pricing.BillingMode == BillingModeImage || pricing.ImageOutputPrice != nil) {
		capabilities = append(capabilities, "image")
	}

	if len(capabilities) == 0 {
		if strings.Contains(lowerModelID, "image") {
			capabilities = append(capabilities, "image")
		} else {
			capabilities = append(capabilities, "chat")
		}
	}

	return normalizeStringList(capabilities)
}
