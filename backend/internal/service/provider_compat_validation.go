package service

import (
	"fmt"
	"strings"
)

func validateCompatibleProviderAccount(platform, accountType string, credentials map[string]any) error {
	switch strings.TrimSpace(platform) {
	case PlatformOpenAICompatible:
		return validateStaticCompatibleProviderCredentials(platform, accountType, credentials, true)
	case PlatformAnthropicCompatible:
		return validateStaticCompatibleProviderCredentials(platform, accountType, credentials, false)
	default:
		return nil
	}
}

func validateStaticCompatibleProviderCredentials(platform, accountType string, credentials map[string]any, allowModelsEndpoint bool) error {
	if !IsStaticKeyAccountType(accountType) {
		return fmt.Errorf("%s accounts only support %s or %s type", platform, AccountTypeAPIKey, AccountTypeUpstream)
	}
	if strings.TrimSpace(readCredentialString(credentials, "base_url")) == "" {
		return fmt.Errorf("%s credentials.base_url is required", platform)
	}
	if strings.TrimSpace(readCredentialString(credentials, "api_key")) == "" {
		return fmt.Errorf("%s credentials.api_key is required", platform)
	}
	if headersRaw, ok := credentials["headers"]; ok && headersRaw != nil {
		if _, ok := normalizeHeaderMap(headersRaw); !ok {
			return fmt.Errorf("%s credentials.headers must be an object of string values", platform)
		}
	}
	if !allowModelsEndpoint {
		return nil
	}
	if modelsEndpointRaw, ok := credentials["models_endpoint"]; ok && modelsEndpointRaw != nil {
		if strings.TrimSpace(readCredentialString(credentials, "models_endpoint")) == "" {
			return fmt.Errorf("%s credentials.models_endpoint must be a non-empty string when provided", platform)
		}
	}
	return nil
}

func readCredentialString(credentials map[string]any, key string) string {
	if credentials == nil {
		return ""
	}
	value, ok := credentials[key]
	if !ok || value == nil {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}

func normalizeHeaderMap(raw any) (map[string]string, bool) {
	switch headers := raw.(type) {
	case map[string]string:
		return headers, true
	case map[string]any:
		out := make(map[string]string, len(headers))
		for key, value := range headers {
			text, ok := value.(string)
			if !ok {
				return nil, false
			}
			out[key] = text
		}
		return out, true
	default:
		return nil, false
	}
}
