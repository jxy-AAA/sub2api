package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	CSPNonceKey              = "csp_nonce"
	NonceTemplate            = "__CSP_NONCE__"
	CloudflareInsightsDomain = "https://static.cloudflareinsights.com"
	StripeDomain             = "https://*.stripe.com"
)

// GenerateNonce generates a cryptographically secure random nonce.
func GenerateNonce() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate CSP nonce: %w", err)
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}

func GetNonceFromContext(c *gin.Context) string {
	if nonce, exists := c.Get(CSPNonceKey); exists {
		if value, ok := nonce.(string); ok {
			return value
		}
	}
	return ""
}

// SecurityHeaders sets baseline security headers for all responses.
// getFrameSrcOrigins is an optional function that returns extra origins to inject into frame-src.
func SecurityHeaders(cfg config.CSPConfig, getFrameSrcOrigins func() []string) gin.HandlerFunc {
	policy := strings.TrimSpace(cfg.Policy)
	if policy == "" {
		policy = config.DefaultCSPPolicy
	}
	policy = enhanceCSPPolicy(policy)

	return func(c *gin.Context) {
		finalPolicy := policy
		if getFrameSrcOrigins != nil {
			for _, origin := range getFrameSrcOrigins() {
				if origin != "" {
					finalPolicy = addToDirective(finalPolicy, "frame-src", origin)
				}
			}
		}

		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		if isAPIRoutePath(c) {
			c.Next()
			return
		}

		if cfg.Enabled {
			nonce, err := GenerateNonce()
			if err != nil {
				logger.L().Warn("failed to generate CSP nonce; falling back to unsafe-inline CSP",
					zap.Error(err),
				)
				c.Header("Content-Security-Policy", strings.ReplaceAll(finalPolicy, NonceTemplate, "'unsafe-inline'"))
			} else {
				c.Set(CSPNonceKey, nonce)
				c.Header("Content-Security-Policy", strings.ReplaceAll(finalPolicy, NonceTemplate, "'nonce-"+nonce+"'"))
			}
		}
		c.Next()
	}
}

func isAPIRoutePath(c *gin.Context) bool {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return false
	}
	path := c.Request.URL.Path
	return strings.HasPrefix(path, "/v1/") ||
		strings.HasPrefix(path, "/v1beta/") ||
		strings.HasPrefix(path, "/antigravity/") ||
		strings.HasPrefix(path, "/responses") ||
		strings.HasPrefix(path, "/images")
}

func enhanceCSPPolicy(policy string) string {
	if !strings.Contains(policy, NonceTemplate) && !strings.Contains(policy, "'nonce-") {
		policy = addToDirective(policy, "script-src", NonceTemplate)
	}
	if !strings.Contains(policy, CloudflareInsightsDomain) {
		policy = addToDirective(policy, "script-src", CloudflareInsightsDomain)
	}
	if !strings.Contains(policy, "stripe.com") {
		policy = addToDirective(policy, "script-src", StripeDomain)
		policy = addToDirective(policy, "frame-src", StripeDomain)
	}
	return policy
}

// addToDirective adds a value to a specific CSP directive.
func addToDirective(policy, directive, value string) string {
	directivePrefix := directive + " "
	index := strings.Index(policy, directivePrefix)

	if index == -1 {
		defaultSrcIndex := strings.Index(policy, "default-src ")
		if defaultSrcIndex != -1 {
			endIndex := strings.Index(policy[defaultSrcIndex:], ";")
			if endIndex != -1 {
				insertPos := defaultSrcIndex + endIndex + 1
				return policy[:insertPos] + " " + directive + " 'self' " + value + ";" + policy[insertPos:]
			}
		}
		return directive + " 'self' " + value + "; " + policy
	}

	endIndex := strings.Index(policy[index:], ";")
	if endIndex == -1 {
		return policy + " " + value
	}

	insertPos := index + endIndex
	return policy[:insertPos] + " " + value + policy[insertPos:]
}
