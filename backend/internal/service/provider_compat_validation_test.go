package service

import "testing"

func TestValidateCompatibleProviderAccount(t *testing.T) {
	t.Run("accepts openai compatible static key credentials", func(t *testing.T) {
		err := validateCompatibleProviderAccount(PlatformOpenAICompatible, AccountTypeUpstream, map[string]any{
			"base_url":        "https://openai-compatible.example",
			"api_key":         "sk-test",
			"models_endpoint": "https://openai-compatible.example/v1/models",
			"headers": map[string]any{
				"x-provider": "test",
			},
		})
		if err != nil {
			t.Fatalf("validateCompatibleProviderAccount() error = %v, want nil", err)
		}
	})

	t.Run("accepts anthropic compatible static key credentials", func(t *testing.T) {
		err := validateCompatibleProviderAccount(PlatformAnthropicCompatible, AccountTypeAPIKey, map[string]any{
			"base_url": "https://anthropic-compatible.example",
			"api_key":  "sk-ant-test",
			"headers": map[string]string{
				"anthropic-beta": "tools-2024-04-04",
			},
		})
		if err != nil {
			t.Fatalf("validateCompatibleProviderAccount() error = %v, want nil", err)
		}
	})

	t.Run("rejects oauth on compatible providers", func(t *testing.T) {
		err := validateCompatibleProviderAccount(PlatformOpenAICompatible, AccountTypeOAuth, map[string]any{
			"base_url": "https://example.com",
			"api_key":  "sk-test",
		})
		if err == nil {
			t.Fatal("expected error for oauth compatible provider")
		}
	})

	t.Run("rejects malformed custom headers", func(t *testing.T) {
		err := validateCompatibleProviderAccount(PlatformAnthropicCompatible, AccountTypeUpstream, map[string]any{
			"base_url": "https://example.com",
			"api_key":  "sk-test",
			"headers": map[string]any{
				"x-provider": 1,
			},
		})
		if err == nil {
			t.Fatal("expected error for non-string header value")
		}
	})
}
