//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubModelCatalogRepository struct {
	items      []*ModelCatalogItem
	created    []*ModelCatalogItem
	createErr  error
	listErr    error
	getByIDErr error
	deleteErr  error
}

func (s *stubModelCatalogRepository) List(ctx context.Context) ([]*ModelCatalogItem, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	out := make([]*ModelCatalogItem, 0, len(s.items))
	for _, item := range s.items {
		cloned := *item
		out = append(out, &cloned)
	}
	return out, nil
}

func (s *stubModelCatalogRepository) GetByID(ctx context.Context, id int64) (*ModelCatalogItem, error) {
	if s.getByIDErr != nil {
		return nil, s.getByIDErr
	}
	for _, item := range s.items {
		if item.ID == id {
			cloned := *item
			return &cloned, nil
		}
	}
	return nil, nil
}

func (s *stubModelCatalogRepository) Create(ctx context.Context, item *ModelCatalogItem) (*ModelCatalogItem, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}
	cloned := *item
	cloned.ID = int64(len(s.items) + len(s.created) + 1)
	s.created = append(s.created, &cloned)
	s.items = append(s.items, &cloned)
	return &cloned, nil
}

func (s *stubModelCatalogRepository) Update(ctx context.Context, item *ModelCatalogItem) (*ModelCatalogItem, error) {
	for idx, existing := range s.items {
		if existing.ID == item.ID {
			cloned := *item
			s.items[idx] = &cloned
			return &cloned, nil
		}
	}
	return nil, ErrModelCatalogItemNotFound
}

func (s *stubModelCatalogRepository) Delete(ctx context.Context, id int64) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	for idx, item := range s.items {
		if item.ID == id {
			s.items = append(s.items[:idx], s.items[idx+1:]...)
			return nil
		}
	}
	return ErrModelCatalogItemNotFound
}

type stubModelMarketChannelProvider struct {
	availableChannels []AvailableChannel
	allChannels       []Channel
	listAvailableErr  error
	listAllErr        error
}

func (s *stubModelMarketChannelProvider) ListAvailable(ctx context.Context) ([]AvailableChannel, error) {
	if s.listAvailableErr != nil {
		return nil, s.listAvailableErr
	}
	return append([]AvailableChannel(nil), s.availableChannels...), nil
}

func (s *stubModelMarketChannelProvider) ListAll(ctx context.Context) ([]Channel, error) {
	if s.listAllErr != nil {
		return nil, s.listAllErr
	}
	return append([]Channel(nil), s.allChannels...), nil
}

type stubModelMarketGroupProvider struct {
	groups []Group
	err    error
}

func (s *stubModelMarketGroupProvider) GetAvailableGroups(ctx context.Context, userID int64) ([]Group, error) {
	if s.err != nil {
		return nil, s.err
	}
	return append([]Group(nil), s.groups...), nil
}

func TestModelMarketServiceListUserModelsFiltersAndEnriches(t *testing.T) {
	repo := &stubModelCatalogRepository{
		items: []*ModelCatalogItem{
			{
				ID:           1,
				ModelID:      "claude-visible",
				DisplayName:  "Claude Visible",
				ProviderKey:  PlatformAnthropic,
				Protocol:     PlatformAnthropic,
				Capabilities: []string{"chat"},
				Tags:         []string{"recommended"},
				Status:       ModelCatalogStatusActive,
			},
			{
				ID:          2,
				ModelID:     "claude-hidden",
				DisplayName: "Claude Hidden",
				ProviderKey: PlatformAnthropic,
				Protocol:    PlatformAnthropic,
				Status:      ModelCatalogStatusHidden,
			},
		},
	}
	channelProvider := &stubModelMarketChannelProvider{
		availableChannels: []AvailableChannel{
			{
				ID:          10,
				Name:        "anthropic-main",
				Description: "Anthropic channel",
				Status:      StatusActive,
				Groups: []AvailableGroupRef{
					{ID: 100, Name: "anthropic-group", Platform: PlatformAnthropic, SubscriptionType: SubscriptionTypeStandard},
				},
				SupportedModels: []SupportedModel{
					{Name: "claude-visible", Platform: PlatformAnthropic},
					{Name: "claude-hidden", Platform: PlatformAnthropic},
				},
			},
			{
				ID:     11,
				Name:   "openai-main",
				Status: StatusActive,
				Groups: []AvailableGroupRef{
					{ID: 101, Name: "openai-group", Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard},
				},
				SupportedModels: []SupportedModel{
					{Name: "gpt-4o", Platform: PlatformOpenAI},
				},
			},
		},
	}
	groupProvider := &stubModelMarketGroupProvider{
		groups: []Group{
			{ID: 100, Name: "anthropic-group", Platform: PlatformAnthropic, SubscriptionType: SubscriptionTypeStandard},
		},
	}

	svc := NewModelMarketService(repo, channelProvider, groupProvider)
	items, err := svc.ListUserModels(context.Background(), 7)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "claude-visible", items[0].ModelID)
	require.Equal(t, "Claude Visible", items[0].DisplayName)
	require.Equal(t, []string{"recommended"}, items[0].Tags)
	require.Len(t, items[0].Channels, 1)
	require.Equal(t, "anthropic-main", items[0].Channels[0].Name)
	require.Len(t, items[0].Channels[0].Groups, 1)
	require.Equal(t, int64(100), items[0].Channels[0].Groups[0].ID)
}

func TestModelMarketServiceImportFromChannelsSkipsExisting(t *testing.T) {
	repo := &stubModelCatalogRepository{
		items: []*ModelCatalogItem{
			{
				ID:          1,
				ModelID:     "claude-visible",
				DisplayName: "Claude Visible",
				ProviderKey: PlatformAnthropic,
				Protocol:    PlatformAnthropic,
				Status:      ModelCatalogStatusActive,
			},
		},
	}
	channelProvider := &stubModelMarketChannelProvider{
		allChannels: []Channel{
			{
				ID:     1,
				Status: StatusActive,
				ModelPricing: []ChannelModelPricing{
					{Platform: PlatformAnthropic, Models: []string{"claude-visible", "claude-new"}},
					{Platform: PlatformOpenAI, Models: []string{"gpt-4o"}},
				},
			},
		},
	}

	svc := NewModelMarketService(repo, channelProvider, &stubModelMarketGroupProvider{})
	result, err := svc.ImportFromChannels(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, result.ImportedCount)
	require.Equal(t, 1, result.SkippedCount)
	require.Len(t, result.Items, 2)
	require.Equal(t, "claude-new", result.Items[0].ModelID)
	require.Equal(t, "gpt-4o", result.Items[1].ModelID)
}

func TestModelMarketServiceListCatalogAppliesFiltersAndChannelRefs(t *testing.T) {
	repo := &stubModelCatalogRepository{
		items: []*ModelCatalogItem{
			{
				ID:           1,
				ModelID:      "gpt-4o",
				DisplayName:  "GPT-4o",
				ProviderKey:  PlatformOpenAI,
				Protocol:     PlatformOpenAI,
				Capabilities: []string{"chat"},
				Status:       ModelCatalogStatusActive,
			},
			{
				ID:           2,
				ModelID:      "imagen-3",
				DisplayName:  "Imagen 3",
				ProviderKey:  PlatformGemini,
				Protocol:     PlatformGemini,
				Capabilities: []string{"image"},
				Status:       ModelCatalogStatusDisabled,
			},
		},
	}
	channelProvider := &stubModelMarketChannelProvider{
		allChannels: []Channel{
			{
				ID:       55,
				Name:     "openai",
				Status:   StatusActive,
				GroupIDs: []int64{9},
				ModelPricing: []ChannelModelPricing{
					{Platform: PlatformOpenAI, Models: []string{"gpt-4o"}},
				},
			},
		},
	}

	svc := NewModelMarketService(repo, channelProvider, &stubModelMarketGroupProvider{})
	items, err := svc.ListCatalog(context.Background(), ModelCatalogListFilter{
		ProviderKey: PlatformOpenAI,
		Capability:  "chat",
		Status:      ModelCatalogStatusActive,
		Keyword:     "gpt",
	})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "gpt-4o", items[0].ModelID)
	require.Len(t, items[0].ChannelRefs, 1)
	require.Equal(t, int64(55), items[0].ChannelRefs[0].ChannelID)
}

func TestModelMarketServiceListCatalogFiltersChannelRefGroupsByPlatform(t *testing.T) {
	repo := &stubModelCatalogRepository{
		items: []*ModelCatalogItem{
			{
				ID:          1,
				ModelID:     "gpt-4o",
				DisplayName: "GPT-4o",
				ProviderKey: PlatformOpenAI,
				Protocol:    PlatformOpenAI,
				Status:      ModelCatalogStatusActive,
			},
		},
	}
	channelProvider := &stubModelMarketChannelProvider{
		allChannels: []Channel{
			{
				ID:       55,
				Name:     "mixed-channel",
				Status:   StatusActive,
				GroupIDs: []int64{9, 10},
				ModelPricing: []ChannelModelPricing{
					{Platform: PlatformOpenAI, Models: []string{"gpt-4o"}},
				},
			},
		},
		availableChannels: []AvailableChannel{
			{
				ID:     55,
				Name:   "mixed-channel",
				Status: StatusActive,
				Groups: []AvailableGroupRef{
					{ID: 9, Name: "OpenAI Group", Platform: PlatformOpenAI},
					{ID: 10, Name: "Gemini Group", Platform: PlatformGemini},
				},
				SupportedModels: []SupportedModel{
					{Name: "gpt-4o", Platform: PlatformOpenAI},
				},
			},
		},
	}

	svc := NewModelMarketService(repo, channelProvider, &stubModelMarketGroupProvider{})
	items, err := svc.ListCatalog(context.Background(), ModelCatalogListFilter{})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Len(t, items[0].ChannelRefs, 1)
	require.Equal(t, []int64{9}, items[0].ChannelRefs[0].GroupIDs)
}
