package service

import "strings"

func IsStaticKeyAccountType(accountType string) bool {
	switch strings.TrimSpace(accountType) {
	case AccountTypeAPIKey, AccountTypeUpstream:
		return true
	default:
		return false
	}
}

func IsOpenAIProtocolPlatform(platform string) bool {
	switch strings.TrimSpace(platform) {
	case PlatformOpenAI, PlatformOpenAICompatible:
		return true
	default:
		return false
	}
}

func OpenAIProtocolPlatforms() []string {
	return []string{PlatformOpenAI, PlatformOpenAICompatible}
}

func AnthropicProtocolPlatforms() []string {
	return []string{PlatformAnthropic, PlatformAnthropicCompatible}
}

func ProtocolCompatiblePlatforms(platform string) []string {
	switch strings.TrimSpace(platform) {
	case PlatformOpenAI, PlatformOpenAICompatible:
		return OpenAIProtocolPlatforms()
	case PlatformAnthropic, PlatformAnthropicCompatible:
		return AnthropicProtocolPlatforms()
	default:
		if strings.TrimSpace(platform) == "" {
			return nil
		}
		return []string{strings.TrimSpace(platform)}
	}
}

func DefaultGroupNameCandidates(platform string) []string {
	platform = strings.TrimSpace(platform)
	if platform == "" {
		return nil
	}
	names := []string{platform + "-default"}
	switch platform {
	case PlatformOpenAICompatible:
		names = append(names, PlatformOpenAI+"-default")
	case PlatformAnthropicCompatible:
		names = append(names, PlatformAnthropic+"-default")
	}
	return names
}

func IsOAuthOnlyGroupPlatform(platform string) bool {
	switch {
	case IsOpenAIProtocolPlatform(platform), IsAnthropicProtocolPlatform(platform):
		return true
	case strings.TrimSpace(platform) == PlatformAntigravity, strings.TrimSpace(platform) == PlatformGemini:
		return true
	default:
		return false
	}
}

func SameProtocolPlatform(left, right string) bool {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if left == "" || right == "" {
		return false
	}
	if left == right {
		return true
	}
	if IsOpenAIProtocolPlatform(left) && IsOpenAIProtocolPlatform(right) {
		return true
	}
	if IsAnthropicProtocolPlatform(left) && IsAnthropicProtocolPlatform(right) {
		return true
	}
	return false
}

func IsAnthropicProtocolPlatform(platform string) bool {
	switch strings.TrimSpace(platform) {
	case PlatformAnthropic, PlatformAnthropicCompatible:
		return true
	default:
		return false
	}
}
