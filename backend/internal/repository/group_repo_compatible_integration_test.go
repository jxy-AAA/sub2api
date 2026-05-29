//go:build integration

package repository

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func groupIDsContain(groups []service.Group, targetID int64) bool {
	for i := range groups {
		if groups[i].ID == targetID {
			return true
		}
	}
	return false
}

func isOpenAIProtocolGroupPlatform(platform string) bool {
	return platform == service.PlatformOpenAI || platform == service.PlatformOpenAICompatible
}

func isAnthropicProtocolGroupPlatform(platform string) bool {
	return platform == service.PlatformAnthropic || platform == service.PlatformAnthropicCompatible
}

func (s *GroupRepoSuite) TestListWithFilters_OpenAICompatibleMatchesProtocolFamily() {
	openAIGroup := &service.Group{
		Name:             "openai-family-native",
		Platform:         service.PlatformOpenAI,
		RateMultiplier:   1.0,
		IsExclusive:      false,
		Status:           service.StatusActive,
		SubscriptionType: service.SubscriptionTypeStandard,
	}
	openAICompatibleGroup := &service.Group{
		Name:             "openai-family-compatible",
		Platform:         service.PlatformOpenAICompatible,
		RateMultiplier:   1.0,
		IsExclusive:      false,
		Status:           service.StatusActive,
		SubscriptionType: service.SubscriptionTypeStandard,
	}
	anthropicGroup := &service.Group{
		Name:             "openai-family-other",
		Platform:         service.PlatformAnthropic,
		RateMultiplier:   1.0,
		IsExclusive:      false,
		Status:           service.StatusActive,
		SubscriptionType: service.SubscriptionTypeStandard,
	}

	s.Require().NoError(s.repo.Create(s.ctx, openAIGroup))
	s.Require().NoError(s.repo.Create(s.ctx, openAICompatibleGroup))
	s.Require().NoError(s.repo.Create(s.ctx, anthropicGroup))

	groups, _, err := s.repo.ListWithFilters(
		s.ctx,
		pagination.PaginationParams{Page: 1, PageSize: 50},
		service.PlatformOpenAICompatible,
		"",
		"",
		nil,
	)
	s.Require().NoError(err)
	s.Require().True(groupIDsContain(groups, openAIGroup.ID))
	s.Require().True(groupIDsContain(groups, openAICompatibleGroup.ID))
	s.Require().False(groupIDsContain(groups, anthropicGroup.ID))
	for _, g := range groups {
		s.Require().True(isOpenAIProtocolGroupPlatform(g.Platform), "unexpected group platform: %s", g.Platform)
	}
}

func (s *GroupRepoSuite) TestListActiveByPlatform_AnthropicCompatibleMatchesProtocolFamily() {
	anthropicGroup := &service.Group{
		Name:             "anthropic-family-native",
		Platform:         service.PlatformAnthropic,
		RateMultiplier:   1.0,
		IsExclusive:      false,
		Status:           service.StatusActive,
		SubscriptionType: service.SubscriptionTypeStandard,
	}
	anthropicCompatibleGroup := &service.Group{
		Name:             "anthropic-family-compatible",
		Platform:         service.PlatformAnthropicCompatible,
		RateMultiplier:   1.0,
		IsExclusive:      false,
		Status:           service.StatusActive,
		SubscriptionType: service.SubscriptionTypeStandard,
	}
	disabledAnthropicCompatibleGroup := &service.Group{
		Name:             "anthropic-family-disabled",
		Platform:         service.PlatformAnthropicCompatible,
		RateMultiplier:   1.0,
		IsExclusive:      false,
		Status:           service.StatusDisabled,
		SubscriptionType: service.SubscriptionTypeStandard,
	}
	geminiGroup := &service.Group{
		Name:             "anthropic-family-other",
		Platform:         service.PlatformGemini,
		RateMultiplier:   1.0,
		IsExclusive:      false,
		Status:           service.StatusActive,
		SubscriptionType: service.SubscriptionTypeStandard,
	}

	s.Require().NoError(s.repo.Create(s.ctx, anthropicGroup))
	s.Require().NoError(s.repo.Create(s.ctx, anthropicCompatibleGroup))
	s.Require().NoError(s.repo.Create(s.ctx, disabledAnthropicCompatibleGroup))
	s.Require().NoError(s.repo.Create(s.ctx, geminiGroup))

	groups, err := s.repo.ListActiveByPlatform(s.ctx, service.PlatformAnthropicCompatible)
	s.Require().NoError(err)
	s.Require().True(groupIDsContain(groups, anthropicGroup.ID))
	s.Require().True(groupIDsContain(groups, anthropicCompatibleGroup.ID))
	s.Require().False(groupIDsContain(groups, disabledAnthropicCompatibleGroup.ID))
	s.Require().False(groupIDsContain(groups, geminiGroup.ID))
	for _, g := range groups {
		s.Require().True(isAnthropicProtocolGroupPlatform(g.Platform), "unexpected group platform: %s", g.Platform)
		s.Require().Equal(service.StatusActive, g.Status)
	}
}
