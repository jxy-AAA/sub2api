package schema

import (
	"testing"

	"entgo.io/ent/entc/load"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestAccountPlatformEnumIncludesCompatibleProviders(t *testing.T) {
	spec, err := (&load.Config{Path: "."}).Load()
	require.NoError(t, err)

	schemas := map[string]*load.Schema{}
	for _, schema := range spec.Schemas {
		schemas[schema.Name] = schema
	}

	accountSchema := requireSchema(t, schemas, "Account")
	requireFieldEnumValues(t, accountSchema, "platform",
		domain.PlatformAnthropic,
		domain.PlatformOpenAI,
		domain.PlatformGemini,
		domain.PlatformAntigravity,
		domain.PlatformOpenAICompatible,
		domain.PlatformAnthropicCompatible,
	)
}
