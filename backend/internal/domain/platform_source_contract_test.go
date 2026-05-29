package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlatformConstantsExposeCompatibleProviders(t *testing.T) {
	require.Equal(t, []string{
		"anthropic",
		"openai",
		"gemini",
		"antigravity",
		"openai_compatible",
		"anthropic_compatible",
	}, []string{
		PlatformAnthropic,
		PlatformOpenAI,
		PlatformGemini,
		PlatformAntigravity,
		PlatformOpenAICompatible,
		PlatformAnthropicCompatible,
	})
}
