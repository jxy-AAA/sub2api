// Package middleware provides HTTP middleware for authentication, authorization, and request processing.
package middleware

import (
	"errors"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// NewAdminAuthMiddleware creates admin authentication middleware.
func NewAdminAuthMiddleware(
	authService *service.AuthService,
	userService *service.UserService,
	settingService *service.SettingService,
) AdminAuthMiddleware {
	return AdminAuthMiddleware(adminAuth(authService, userService, settingService))
}

func adminAuth(
	authService *service.AuthService,
	userService *service.UserService,
	settingService *service.SettingService,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isWebSocketUpgradeRequest(c) {
			if token := extractJWTFromWebSocketSubprotocol(c); token != "" {
				if !validateJWTForAdmin(c, token, authService, userService) {
					return
				}
				c.Next()
				return
			}
		}

		apiKey := c.GetHeader("x-api-key")
		if apiKey != "" {
			if !validateAdminAPIKey(c, apiKey, settingService) {
				return
			}
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				token := strings.TrimSpace(parts[1])
				if token == "" {
					AbortWithError(c, 401, "UNAUTHORIZED", "Authorization required")
					return
				}
				if !validateJWTForAdmin(c, token, authService, userService) {
					return
				}
				c.Next()
				return
			}
		}

		AbortWithError(c, 401, "UNAUTHORIZED", "Authorization required")
	}
}

func isWebSocketUpgradeRequest(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}
	upgrade := strings.ToLower(strings.TrimSpace(c.GetHeader("Upgrade")))
	if upgrade != "websocket" {
		return false
	}
	connection := strings.ToLower(c.GetHeader("Connection"))
	return strings.Contains(connection, "upgrade")
}

func extractJWTFromWebSocketSubprotocol(c *gin.Context) string {
	if c == nil {
		return ""
	}
	raw := strings.TrimSpace(c.GetHeader("Sec-WebSocket-Protocol"))
	if raw == "" {
		return ""
	}

	for _, part := range strings.Split(raw, ",") {
		p := strings.TrimSpace(part)
		if strings.HasPrefix(p, "jwt.") {
			token := strings.TrimSpace(strings.TrimPrefix(p, "jwt."))
			if token != "" {
				return token
			}
		}
	}
	return ""
}

func validateAdminAPIKey(
	c *gin.Context,
	key string,
	settingService *service.SettingService,
) bool {
	record, ok, err := settingService.ValidateAdminAPIKey(c.Request.Context(), key)
	if err != nil {
		AbortWithError(c, 500, "INTERNAL_ERROR", "Internal server error")
		return false
	}

	if !ok {
		AbortWithError(c, 401, "INVALID_ADMIN_KEY", "Invalid admin API key")
		return false
	}

	principal := service.DeriveAdminAPIKeyPrincipalFromHash(record.Hash)

	c.Set(string(ContextKeyUser), AuthSubject{
		UserID:        0,
		Concurrency:   0,
		PrincipalID:   principal.PrincipalID,
		PrincipalType: principal.PrincipalType,
		IsSystem:      true,
	})
	c.Set(string(ContextKeyUserRole), service.RoleAdmin)
	c.Set("auth_method", "admin_api_key")
	return true
}

func validateJWTForAdmin(
	c *gin.Context,
	token string,
	authService *service.AuthService,
	userService *service.UserService,
) bool {
	claims, err := authService.ValidateToken(token)
	if err != nil {
		if errors.Is(err, service.ErrTokenExpired) {
			AbortWithError(c, 401, "TOKEN_EXPIRED", "Token has expired")
			return false
		}
		AbortWithError(c, 401, "INVALID_TOKEN", "Invalid token")
		return false
	}

	user, err := userService.GetByID(c.Request.Context(), claims.UserID)
	if err != nil {
		AbortWithError(c, 401, "USER_NOT_FOUND", "User not found")
		return false
	}

	if !user.IsActive() {
		AbortWithError(c, 401, "USER_INACTIVE", "User account is not active")
		return false
	}

	if claims.TokenVersion != user.TokenVersion {
		AbortWithError(c, 401, "TOKEN_REVOKED", "Token has been revoked (password changed)")
		return false
	}

	if !user.IsAdmin() {
		AbortWithError(c, 403, "FORBIDDEN", "Admin access required")
		return false
	}

	c.Set(string(ContextKeyUser), AuthSubject{
		UserID:        user.ID,
		Concurrency:   user.Concurrency,
		PrincipalID:   "user:" + strconv.FormatInt(user.ID, 10),
		PrincipalType: "user",
	})
	c.Set(string(ContextKeyUserRole), user.Role)
	c.Set("auth_method", "jwt")

	return true
}
