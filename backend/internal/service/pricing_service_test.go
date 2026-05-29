package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

func TestParsePricingData_ParsesPriorityAndServiceTierFields(t *testing.T) {
	svc := &PricingService{}
	body := []byte(`{
		"gpt-5.4": {
			"input_cost_per_token": 0.0000025,
			"input_cost_per_token_priority": 0.000005,
			"output_cost_per_token": 0.000015,
			"output_cost_per_token_priority": 0.00003,
			"cache_creation_input_token_cost": 0.0000025,
			"cache_read_input_token_cost": 0.00000025,
			"cache_read_input_token_cost_priority": 0.0000005,
			"supports_service_tier": true,
			"supports_prompt_caching": true,
			"litellm_provider": "openai",
			"mode": "chat"
		}
	}`)

	data, err := svc.parsePricingData(body)
	require.NoError(t, err)
	pricing := data["gpt-5.4"]
	require.NotNil(t, pricing)
	require.InDelta(t, 5e-6, pricing.InputCostPerTokenPriority, 1e-12)
	require.InDelta(t, 3e-5, pricing.OutputCostPerTokenPriority, 1e-12)
	require.InDelta(t, 5e-7, pricing.CacheReadInputTokenCostPriority, 1e-12)
	require.True(t, pricing.SupportsServiceTier)
}

func TestGetModelPricing_Gpt53CodexSparkUsesGpt51CodexPricing(t *testing.T) {
	sparkPricing := &LiteLLMModelPricing{InputCostPerToken: 1}
	gpt53Pricing := &LiteLLMModelPricing{InputCostPerToken: 9}

	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"gpt-5.1-codex": sparkPricing,
			"gpt-5.3":       gpt53Pricing,
		},
	}

	got := svc.GetModelPricing("gpt-5.3-codex-spark")
	require.Same(t, sparkPricing, got)
}

func TestGetModelPricing_Gpt53CodexFallbackStillUsesGpt52Codex(t *testing.T) {
	gpt52CodexPricing := &LiteLLMModelPricing{InputCostPerToken: 2}

	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"gpt-5.2-codex": gpt52CodexPricing,
		},
	}

	got := svc.GetModelPricing("gpt-5.3-codex")
	require.Same(t, gpt52CodexPricing, got)
}

func TestGetModelPricing_OpenAIFallbackMatchedLoggedAsInfo(t *testing.T) {
	logSink, restore := captureStructuredLog(t)
	defer restore()

	gpt52CodexPricing := &LiteLLMModelPricing{InputCostPerToken: 2}
	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"gpt-5.2-codex": gpt52CodexPricing,
		},
	}

	got := svc.GetModelPricing("gpt-5.3-codex")
	require.Same(t, gpt52CodexPricing, got)

	require.True(t, logSink.ContainsMessageAtLevel("[Pricing] OpenAI fallback matched gpt-5.3-codex -> gpt-5.2-codex", "info"))
	require.False(t, logSink.ContainsMessageAtLevel("[Pricing] OpenAI fallback matched gpt-5.3-codex -> gpt-5.2-codex", "warn"))
}

func TestGetModelPricing_Gpt54UsesStaticFallbackWhenRemoteMissing(t *testing.T) {
	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"gpt-5.1-codex": &LiteLLMModelPricing{InputCostPerToken: 1.25e-6},
		},
	}

	got := svc.GetModelPricing("gpt-5.4")
	require.NotNil(t, got)
	require.InDelta(t, 2.5e-6, got.InputCostPerToken, 1e-12)
	require.InDelta(t, 1.5e-5, got.OutputCostPerToken, 1e-12)
	require.InDelta(t, 2.5e-7, got.CacheReadInputTokenCost, 1e-12)
	require.Equal(t, 272000, got.LongContextInputTokenThreshold)
	require.InDelta(t, 2.0, got.LongContextInputCostMultiplier, 1e-12)
	require.InDelta(t, 1.5, got.LongContextOutputCostMultiplier, 1e-12)
}

func TestGetModelPricing_OpenAICompactAliasUsesStaticFallback(t *testing.T) {
	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"gpt-5.1-codex": {InputCostPerToken: 1.25e-6},
		},
	}

	got := svc.GetModelPricing("openai/gpt5.5")
	require.NotNil(t, got)
	require.InDelta(t, 2.5e-6, got.InputCostPerToken, 1e-12)
	require.InDelta(t, 1.5e-5, got.OutputCostPerToken, 1e-12)
}

func TestGetModelPricing_Gpt54MiniUsesDedicatedStaticFallbackWhenRemoteMissing(t *testing.T) {
	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"gpt-5.1-codex": {InputCostPerToken: 1.25e-6},
		},
	}

	got := svc.GetModelPricing("gpt-5.4-mini")
	require.NotNil(t, got)
	require.InDelta(t, 7.5e-7, got.InputCostPerToken, 1e-12)
	require.InDelta(t, 4.5e-6, got.OutputCostPerToken, 1e-12)
	require.InDelta(t, 7.5e-8, got.CacheReadInputTokenCost, 1e-12)
	require.Zero(t, got.LongContextInputTokenThreshold)
}

func TestGetModelPricing_Gpt54NanoUsesDedicatedStaticFallbackWhenRemoteMissing(t *testing.T) {
	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"gpt-5.1-codex": {InputCostPerToken: 1.25e-6},
		},
	}

	got := svc.GetModelPricing("gpt-5.4-nano")
	require.NotNil(t, got)
	require.InDelta(t, 2e-7, got.InputCostPerToken, 1e-12)
	require.InDelta(t, 1.25e-6, got.OutputCostPerToken, 1e-12)
	require.InDelta(t, 2e-8, got.CacheReadInputTokenCost, 1e-12)
	require.Zero(t, got.LongContextInputTokenThreshold)
}

func TestGetModelPricing_ImageModelDoesNotFallbackToTextModel(t *testing.T) {
	imagePricing := &LiteLLMModelPricing{InputCostPerToken: 3}
	textPricing := &LiteLLMModelPricing{InputCostPerToken: 9}

	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"gpt-image-2": imagePricing,
			"gpt-5.4":     textPricing,
		},
	}

	got := svc.GetModelPricing("gpt-image-3")
	require.Same(t, imagePricing, got)
}

func TestParsePricingData_PreservesPriorityAndServiceTierFields(t *testing.T) {
	raw := map[string]any{
		"gpt-5.4": map[string]any{
			"input_cost_per_token":                 2.5e-6,
			"input_cost_per_token_priority":        5e-6,
			"output_cost_per_token":                15e-6,
			"output_cost_per_token_priority":       30e-6,
			"cache_read_input_token_cost":          0.25e-6,
			"cache_read_input_token_cost_priority": 0.5e-6,
			"supports_service_tier":                true,
			"supports_prompt_caching":              true,
			"litellm_provider":                     "openai",
			"mode":                                 "chat",
		},
	}
	body, err := json.Marshal(raw)
	require.NoError(t, err)

	svc := &PricingService{}
	pricingMap, err := svc.parsePricingData(body)
	require.NoError(t, err)

	pricing := pricingMap["gpt-5.4"]
	require.NotNil(t, pricing)
	require.InDelta(t, 2.5e-6, pricing.InputCostPerToken, 1e-12)
	require.InDelta(t, 5e-6, pricing.InputCostPerTokenPriority, 1e-12)
	require.InDelta(t, 15e-6, pricing.OutputCostPerToken, 1e-12)
	require.InDelta(t, 30e-6, pricing.OutputCostPerTokenPriority, 1e-12)
	require.InDelta(t, 0.25e-6, pricing.CacheReadInputTokenCost, 1e-12)
	require.InDelta(t, 0.5e-6, pricing.CacheReadInputTokenCostPriority, 1e-12)
	require.True(t, pricing.SupportsServiceTier)
}

func TestParsePricingData_PreservesServiceTierPriorityFields(t *testing.T) {
	svc := &PricingService{}
	pricingData, err := svc.parsePricingData([]byte(`{
		"gpt-5.4": {
			"input_cost_per_token": 0.0000025,
			"input_cost_per_token_priority": 0.000005,
			"output_cost_per_token": 0.000015,
			"output_cost_per_token_priority": 0.00003,
			"cache_read_input_token_cost": 0.00000025,
			"cache_read_input_token_cost_priority": 0.0000005,
			"supports_service_tier": true,
			"litellm_provider": "openai",
			"mode": "chat"
		}
	}`))
	require.NoError(t, err)

	pricing := pricingData["gpt-5.4"]
	require.NotNil(t, pricing)
	require.InDelta(t, 0.0000025, pricing.InputCostPerToken, 1e-12)
	require.InDelta(t, 0.000005, pricing.InputCostPerTokenPriority, 1e-12)
	require.InDelta(t, 0.000015, pricing.OutputCostPerToken, 1e-12)
	require.InDelta(t, 0.00003, pricing.OutputCostPerTokenPriority, 1e-12)
	require.InDelta(t, 0.00000025, pricing.CacheReadInputTokenCost, 1e-12)
	require.InDelta(t, 0.0000005, pricing.CacheReadInputTokenCostPriority, 1e-12)
	require.True(t, pricing.SupportsServiceTier)
}

type stubPricingRemoteClient struct {
	fetchPricingJSON func(ctx context.Context, url string) ([]byte, error)
	fetchHashText    func(ctx context.Context, url string) (string, error)
}

func (s *stubPricingRemoteClient) FetchPricingJSON(ctx context.Context, url string) ([]byte, error) {
	if s.fetchPricingJSON != nil {
		return s.fetchPricingJSON(ctx, url)
	}
	return nil, nil
}

func (s *stubPricingRemoteClient) FetchHashText(ctx context.Context, url string) (string, error) {
	if s.fetchHashText != nil {
		return s.fetchHashText(ctx, url)
	}
	return "", nil
}

func pricingPersistenceTestConfig(dataDir string) *config.Config {
	return &config.Config{
		Pricing: config.PricingConfig{
			DataDir:                  dataDir,
			RemoteURL:                "https://pricing.example.com/model_pricing.json",
			HashURL:                  "https://pricing.example.com/model_pricing.sha256",
			HashCheckIntervalMinutes: 10,
			UpdateIntervalHours:      24,
			FallbackFile:             filepath.Join(dataDir, "fallback_model_pricing.json"),
		},
		Security: config.SecurityConfig{
			URLAllowlist: config.URLAllowlistConfig{
				Enabled:           true,
				PricingHosts:      []string{"pricing.example.com"},
				AllowPrivateHosts: false,
			},
		},
	}
}

func pricingPersistenceTestBody(inputCost, outputCost float64) []byte {
	return []byte(`{"gpt-5.4":{"input_cost_per_token":` +
		strconv.FormatFloat(inputCost, 'f', -1, 64) +
		`,"output_cost_per_token":` +
		strconv.FormatFloat(outputCost, 'f', -1, 64) +
		`,"litellm_provider":"openai","mode":"chat"}}`)
}

func pricingPersistenceTestHash(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func TestDownloadPricingData_WriteFailureDoesNotAdvanceMemoryState(t *testing.T) {
	dataDir := t.TempDir()
	cfg := pricingPersistenceTestConfig(dataDir)

	existingBody := pricingPersistenceTestBody(0.1, 0.2)
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "model_pricing.json"), existingBody, 0o644))

	newBody := pricingPersistenceTestBody(0.9, 1.1)
	svc := NewPricingService(cfg, &stubPricingRemoteClient{
		fetchPricingJSON: func(context.Context, string) ([]byte, error) { return newBody, nil },
		fetchHashText:    func(context.Context, string) (string, error) { return pricingPersistenceTestHash(newBody), nil },
	})
	require.NoError(t, svc.loadPricingData(filepath.Join(dataDir, "model_pricing.json")))

	beforeHash := svc.localHash
	before := svc.GetModelPricing("gpt-5.4")
	require.NotNil(t, before)
	require.InDelta(t, 0.1, before.InputCostPerToken, 1e-12)

	writeFile := svc.writeFile
	svc.writeFile = func(path string, data []byte, perm os.FileMode) error {
		if path == svc.getHashFilePath() {
			return assert.AnError
		}
		return writeFile(path, data, perm)
	}

	err := svc.downloadPricingData()
	require.ErrorIs(t, err, assert.AnError)

	after := svc.GetModelPricing("gpt-5.4")
	require.NotNil(t, after)
	require.InDelta(t, 0.1, after.InputCostPerToken, 1e-12)
	require.Equal(t, beforeHash, svc.localHash)
}

func TestDownloadPricingData_SuccessUpdatesDiskAndMemoryState(t *testing.T) {
	dataDir := t.TempDir()
	cfg := pricingPersistenceTestConfig(dataDir)

	newBody := pricingPersistenceTestBody(0.9, 1.1)
	remoteHash := pricingPersistenceTestHash(newBody)
	svc := NewPricingService(cfg, &stubPricingRemoteClient{
		fetchPricingJSON: func(context.Context, string) ([]byte, error) { return newBody, nil },
		fetchHashText:    func(context.Context, string) (string, error) { return remoteHash, nil },
	})

	require.NoError(t, svc.downloadPricingData())

	got := svc.GetModelPricing("gpt-5.4")
	require.NotNil(t, got)
	require.InDelta(t, 0.9, got.InputCostPerToken, 1e-12)
	require.InDelta(t, 1.1, got.OutputCostPerToken, 1e-12)
	require.Equal(t, remoteHash, svc.localHash)

	savedBody, err := os.ReadFile(svc.getPricingFilePath())
	require.NoError(t, err)
	assert.JSONEq(t, string(newBody), string(savedBody))

	savedHash, err := os.ReadFile(svc.getHashFilePath())
	require.NoError(t, err)
	require.Equal(t, remoteHash+"\n", string(savedHash))
}

func TestUseFallbackPricing_WriteFailureDoesNotAdvanceMemoryState(t *testing.T) {
	dataDir := t.TempDir()
	cfg := pricingPersistenceTestConfig(dataDir)

	existingBody := pricingPersistenceTestBody(0.1, 0.2)
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "model_pricing.json"), existingBody, 0o644))
	require.NoError(t, os.WriteFile(cfg.Pricing.FallbackFile, pricingPersistenceTestBody(0.7, 0.8), 0o644))

	svc := NewPricingService(cfg, &stubPricingRemoteClient{})
	require.NoError(t, svc.loadPricingData(filepath.Join(dataDir, "model_pricing.json")))

	beforeHash := svc.localHash
	before := svc.GetModelPricing("gpt-5.4")
	require.NotNil(t, before)
	require.InDelta(t, 0.1, before.InputCostPerToken, 1e-12)

	writeFile := svc.writeFile
	svc.writeFile = func(path string, data []byte, perm os.FileMode) error {
		if path == svc.getHashFilePath() {
			return assert.AnError
		}
		return writeFile(path, data, perm)
	}

	err := svc.useFallbackPricing()
	require.ErrorIs(t, err, assert.AnError)

	after := svc.GetModelPricing("gpt-5.4")
	require.NotNil(t, after)
	require.InDelta(t, 0.1, after.InputCostPerToken, 1e-12)
	require.Equal(t, beforeHash, svc.localHash)
}

func TestDownloadPricingData_HashMismatchFailsClosedAndKeepsExistingCache(t *testing.T) {
	logSink, restore := captureStructuredLog(t)
	defer restore()

	dataDir := t.TempDir()
	pricingFile := filepath.Join(dataDir, "model_pricing.json")
	existingBody := []byte(`{"gpt-5.4":{"input_cost_per_token":0.1,"output_cost_per_token":0.2,"litellm_provider":"openai","mode":"chat"}}`)
	require.NoError(t, os.WriteFile(pricingFile, existingBody, 0o644))

	cfg := &config.Config{
		Pricing: config.PricingConfig{
			DataDir:                  dataDir,
			RemoteURL:                "https://pricing.example.com/model_pricing.json",
			HashURL:                  "https://pricing.example.com/model_pricing.sha256",
			HashCheckIntervalMinutes: 10,
			UpdateIntervalHours:      24,
		},
		Security: config.SecurityConfig{
			URLAllowlist: config.URLAllowlistConfig{
				Enabled:           true,
				PricingHosts:      []string{"pricing.example.com"},
				AllowPrivateHosts: false,
			},
		},
	}

	verifiedHash := "8d4f0f5a6c5d57b9a2dc3f9c41d7d2585a8a4f6f7542024d31d13d4d5f0a4ef2"
	mismatchedBody := []byte(`{"gpt-5.4":{"input_cost_per_token":9.9,"output_cost_per_token":8.8,"litellm_provider":"openai","mode":"chat"}}`)
	svc := NewPricingService(cfg, &stubPricingRemoteClient{
		fetchPricingJSON: func(context.Context, string) ([]byte, error) { return mismatchedBody, nil },
		fetchHashText:    func(context.Context, string) (string, error) { return verifiedHash, nil },
	})
	require.NoError(t, svc.loadPricingData(pricingFile))

	before := svc.GetModelPricing("gpt-5.4")
	require.NotNil(t, before)
	require.InDelta(t, 0.1, before.InputCostPerToken, 1e-12)

	err := svc.downloadPricingData()
	require.ErrorIs(t, err, ErrPricingHashMismatch)

	after := svc.GetModelPricing("gpt-5.4")
	require.NotNil(t, after)
	require.InDelta(t, 0.1, after.InputCostPerToken, 1e-12)

	savedBody, readErr := os.ReadFile(pricingFile)
	require.NoError(t, readErr)
	assert.JSONEq(t, string(existingBody), string(savedBody))
	require.True(t, logSink.ContainsMessageAtLevel("[Pricing] Hash mismatch blocked: remote="+verifiedHash[:8]+" data=", "warn"))
}

func TestDownloadPricingData_HashFetchFailureFailsClosedAndKeepsExistingCache(t *testing.T) {
	logSink, restore := captureStructuredLog(t)
	defer restore()

	dataDir := t.TempDir()
	pricingFile := filepath.Join(dataDir, "model_pricing.json")
	existingBody := []byte(`{"gpt-5.4":{"input_cost_per_token":0.3,"output_cost_per_token":0.4,"litellm_provider":"openai","mode":"chat"}}`)
	require.NoError(t, os.WriteFile(pricingFile, existingBody, 0o644))

	cfg := &config.Config{
		Pricing: config.PricingConfig{
			DataDir:                  dataDir,
			RemoteURL:                "https://pricing.example.com/model_pricing.json",
			HashURL:                  "https://pricing.example.com/model_pricing.sha256",
			HashCheckIntervalMinutes: 10,
			UpdateIntervalHours:      24,
		},
		Security: config.SecurityConfig{
			URLAllowlist: config.URLAllowlistConfig{
				Enabled:           true,
				PricingHosts:      []string{"pricing.example.com"},
				AllowPrivateHosts: false,
			},
		},
	}

	remoteBody := []byte(`{"gpt-5.4":{"input_cost_per_token":7.7,"output_cost_per_token":8.8,"litellm_provider":"openai","mode":"chat"}}`)
	svc := NewPricingService(cfg, &stubPricingRemoteClient{
		fetchPricingJSON: func(context.Context, string) ([]byte, error) { return remoteBody, nil },
		fetchHashText:    func(context.Context, string) (string, error) { return "", assert.AnError },
	})
	require.NoError(t, svc.loadPricingData(pricingFile))

	err := svc.downloadPricingData()
	require.ErrorIs(t, err, ErrPricingVerificationFailed)

	after := svc.GetModelPricing("gpt-5.4")
	require.NotNil(t, after)
	require.InDelta(t, 0.3, after.InputCostPerToken, 1e-12)

	savedBody, readErr := os.ReadFile(pricingFile)
	require.NoError(t, readErr)
	assert.JSONEq(t, string(existingBody), string(savedBody))
	require.True(t, logSink.ContainsMessageAtLevel("[Pricing] Remote hash verification failed:", "warn"))
}

func TestFetchRemoteHash_RejectsEmptyAndMalformedHash(t *testing.T) {
	cfg := &config.Config{
		Pricing: config.PricingConfig{
			HashURL: "https://pricing.example.com/model_pricing.sha256",
		},
		Security: config.SecurityConfig{
			URLAllowlist: config.URLAllowlistConfig{
				Enabled:           true,
				PricingHosts:      []string{"pricing.example.com"},
				AllowPrivateHosts: false,
			},
		},
	}

	testCases := []struct {
		name        string
		hashText    string
		errContains string
	}{
		{name: "empty", hashText: "   ", errContains: "remote hash is empty"},
		{name: "wrong length", hashText: "abc123", errContains: "remote hash has invalid length"},
		{name: "invalid hex", hashText: strings.Repeat("z", sha256.Size*2), errContains: "remote hash is not valid hex"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewPricingService(cfg, &stubPricingRemoteClient{
				fetchHashText: func(context.Context, string) (string, error) { return tc.hashText, nil },
			})
			_, err := svc.fetchRemoteHash()
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.errContains)
		})
	}
}
