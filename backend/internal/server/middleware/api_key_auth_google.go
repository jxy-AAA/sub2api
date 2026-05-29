package middleware

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/googleapi"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// APIKeyAuthGoogle is a Google-style error wrapper for API key auth.
func APIKeyAuthGoogle(apiKeyService *service.APIKeyService, cfg *config.Config) gin.HandlerFunc {
	return APIKeyAuthWithSubscriptionGoogle(apiKeyService, nil, cfg)
}

// APIKeyAuthWithSubscriptionGoogle behaves like ApiKeyAuthWithSubscription but returns Google-style errors:
// {"error":{"code":401,"message":"...","status":"UNAUTHENTICATED"}}
//
// It is intended for Gemini native endpoints (/v1beta) to match Gemini SDK expectations.
func APIKeyAuthWithSubscriptionGoogle(apiKeyService *service.APIKeyService, subscriptionService *service.SubscriptionService, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		queryAPIKey, queryKey := stripGoogleAPIKeyQuery(c.Request)
		if queryAPIKey != "" {
			abortWithGoogleError(c, 400, "Query parameter api_key is deprecated. Use Authorization header or key instead.")
			return
		}
		apiKeyString, source := extractAPIKeyForGoogle(c, cfg, queryKey)
		if c.IsAborted() {
			return
		}
		if apiKeyString == "" {
			abortWithGoogleError(c, 401, "API key is required")
			return
		}
		if source == googleAPIKeySourceQueryKey {
			c.Header("Warning", `299 - "Query parameter key is compatibility-only and disabled by default; use x-goog-api-key, Authorization, or x-api-key header"`)
		}

		apiKey, err := apiKeyService.GetByKey(c.Request.Context(), apiKeyString)
		if err != nil {
			if errors.Is(err, service.ErrAPIKeyNotFound) {
				abortWithGoogleError(c, 401, "Invalid API key")
				return
			}
			abortWithGoogleError(c, 500, "Failed to validate API key")
			return
		}

		if !apiKey.IsActive() &&
			apiKey.Status != service.StatusAPIKeyExpired &&
			apiKey.Status != service.StatusAPIKeyQuotaExhausted {
			abortWithGoogleError(c, 401, "API key is disabled")
			return
		}
		if len(apiKey.IPWhitelist) > 0 || len(apiKey.IPBlacklist) > 0 {
			clientIP := ip.GetTrustedClientIP(c)
			allowed, _ := ip.CheckIPRestrictionWithCompiledRules(clientIP, apiKey.CompiledIPWhitelist, apiKey.CompiledIPBlacklist)
			if !allowed {
				abortWithGoogleError(c, 403, "Access denied")
				return
			}
		}
		if apiKey.User == nil {
			abortWithGoogleError(c, 401, "User associated with API key not found")
			return
		}
		if !apiKey.User.IsActive() {
			abortWithGoogleError(c, 401, "User account is not active")
			return
		}

		// 简易模式：跳过余额和订阅检查
		if cfg.RunMode == config.RunModeSimple {
			c.Set(string(ContextKeyAPIKey), apiKey)
			c.Set(string(ContextKeyUser), AuthSubject{
				UserID:      apiKey.User.ID,
				Concurrency: apiKey.User.Concurrency,
			})
			c.Set(string(ContextKeyUserRole), apiKey.User.Role)
			setGroupContext(c, apiKey.Group)
			_ = apiKeyService.TouchLastUsed(c.Request.Context(), apiKey.ID)
			c.Next()
			return
		}

		switch apiKey.Status {
		case service.StatusAPIKeyQuotaExhausted:
			abortWithGoogleError(c, http.StatusTooManyRequests, "API key quota exhausted")
			return
		case service.StatusAPIKeyExpired:
			abortWithGoogleError(c, http.StatusForbidden, "API key expired")
			return
		}
		if apiKey.IsExpired() {
			abortWithGoogleError(c, http.StatusForbidden, "API key expired")
			return
		}
		if apiKey.IsQuotaExhausted() {
			abortWithGoogleError(c, http.StatusTooManyRequests, "API key quota exhausted")
			return
		}

		isSubscriptionType := apiKey.Group != nil && apiKey.Group.IsSubscriptionType()
		if isSubscriptionType && subscriptionService != nil {
			subscription, err := subscriptionService.GetActiveSubscription(
				c.Request.Context(),
				apiKey.User.ID,
				apiKey.Group.ID,
			)
			if err != nil {
				abortWithGoogleError(c, 403, "No active subscription found for this group")
				return
			}

			needsMaintenance, err := subscriptionService.ValidateAndCheckLimits(subscription, apiKey.Group)
			if err != nil {
				status := 403
				if errors.Is(err, service.ErrDailyLimitExceeded) ||
					errors.Is(err, service.ErrWeeklyLimitExceeded) ||
					errors.Is(err, service.ErrMonthlyLimitExceeded) {
					status = 429
				}
				abortWithGoogleError(c, status, err.Error())
				return
			}

			c.Set(string(ContextKeySubscription), subscription)

			if needsMaintenance {
				maintenanceCopy := *subscription
				subscriptionService.DoWindowMaintenance(&maintenanceCopy)
			}
		} else {
			if apiKey.User.Balance <= 0 {
				abortWithGoogleError(c, 403, "Insufficient account balance")
				return
			}
		}

		c.Set(string(ContextKeyAPIKey), apiKey)
		c.Set(string(ContextKeyUser), AuthSubject{
			UserID:      apiKey.User.ID,
			Concurrency: apiKey.User.Concurrency,
		})
		c.Set(string(ContextKeyUserRole), apiKey.User.Role)
		setGroupContext(c, apiKey.Group)
		_ = apiKeyService.TouchLastUsed(c.Request.Context(), apiKey.ID)
		c.Next()
	}
}

type googleAPIKeySource int

const (
	googleAPIKeySourceNone googleAPIKeySource = iota
	googleAPIKeySourceXGoogAPIKey
	googleAPIKeySourceAuthorizationBearer
	googleAPIKeySourceXAPIKey
	googleAPIKeySourceQueryKey
)

// extractAPIKeyForGoogle extracts API key for Google/Gemini endpoints.
// Priority: x-goog-api-key > Authorization: Bearer > x-api-key > query key
// This allows OpenClaw and other clients using Bearer auth to work with Gemini endpoints.
func extractAPIKeyForGoogle(c *gin.Context, cfg *config.Config, queryKey string) (string, googleAPIKeySource) {
	// 1) preferred: Gemini native header
	if k := strings.TrimSpace(c.GetHeader("x-goog-api-key")); k != "" {
		return k, googleAPIKeySourceXGoogAPIKey
	}

	// 2) fallback: Authorization: Bearer <key>
	auth := strings.TrimSpace(c.GetHeader("Authorization"))
	if auth != "" {
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			if k := strings.TrimSpace(parts[1]); k != "" {
				return k, googleAPIKeySourceAuthorizationBearer
			}
		}
	}

	// 3) x-api-key header (backward compatibility)
	if k := strings.TrimSpace(c.GetHeader("x-api-key")); k != "" {
		return k, googleAPIKeySourceXAPIKey
	}

	// 4) query parameter key (for specific paths)
	if allowGoogleQueryKey(c.Request.URL.Path, cfg) {
		if v := strings.TrimSpace(queryKey); v != "" {
			return v, googleAPIKeySourceQueryKey
		}
	} else if strings.TrimSpace(queryKey) != "" {
		abortWithGoogleError(c, http.StatusBadRequest, "Query parameter key is not allowed on Gemini native endpoints unless explicit compatibility mode is enabled.")
		return "", googleAPIKeySourceNone
	}

	return "", googleAPIKeySourceNone
}

func allowGoogleQueryKey(path string, cfg *config.Config) bool {
	if cfg == nil || !cfg.Gemini.AllowNativeQueryAPIKey {
		return false
	}
	return strings.HasPrefix(path, "/v1beta") || strings.HasPrefix(path, "/antigravity/v1beta")
}

func stripGoogleAPIKeyQuery(req *http.Request) (string, string) {
	if req == nil || req.URL == nil {
		return "", ""
	}

	queryValues, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		return "", ""
	}

	apiKey := strings.TrimSpace(queryValues.Get("api_key"))
	queryKey := strings.TrimSpace(queryValues.Get("key"))
	if apiKey == "" && queryKey == "" {
		return "", ""
	}

	queryValues.Del("api_key")
	queryValues.Del("key")
	req.URL.RawQuery = queryValues.Encode()
	if req.RequestURI != "" {
		req.RequestURI = req.URL.RequestURI()
	}
	return apiKey, queryKey
}

func abortWithGoogleError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"code":    status,
			"message": message,
			"status":  googleapi.HTTPStatusToGoogleStatus(status),
		},
	})
	c.Abort()
}
