package routes

import (
	"reflect"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newRouteRegistrationHandlers() *handler.Handlers {
	handlers := &handler.Handlers{}
	root := reflect.ValueOf(handlers).Elem()
	for index := 0; index < root.NumField(); index++ {
		field := root.Field(index)
		if field.Kind() != reflect.Pointer || !field.CanSet() || !field.IsNil() {
			continue
		}
		field.Set(reflect.New(field.Type().Elem()))
		if field.Elem().Kind() != reflect.Struct {
			continue
		}
		for innerIndex := 0; innerIndex < field.Elem().NumField(); innerIndex++ {
			inner := field.Elem().Field(innerIndex)
			if inner.Kind() == reflect.Pointer && inner.CanSet() && inner.IsNil() {
				inner.Set(reflect.New(inner.Type().Elem()))
			}
		}
	}
	return handlers
}

func collectRouteSet(routes gin.RoutesInfo) map[string]struct{} {
	set := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		set[route.Method+" "+route.Path] = struct{}{}
	}
	return set
}

func TestCriticalRoutesRemainRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	v1 := router.Group("/api/v1")
	handlers := newRouteRegistrationHandlers()

	RegisterAuthRoutes(v1, handlers, servermiddleware.JWTAuthMiddleware(func(c *gin.Context) {
		c.Next()
	}), nil, nil)
	RegisterUserRoutes(v1, handlers, servermiddleware.JWTAuthMiddleware(func(c *gin.Context) {
		c.Next()
	}), nil)
	RegisterAdminRoutes(v1, handlers, servermiddleware.AdminAuthMiddleware(func(c *gin.Context) {
		c.Next()
	}))

	routeSet := collectRouteSet(router.Routes())
	expected := []string{
		"POST /api/v1/auth/register",
		"POST /api/v1/auth/refresh",
		"POST /api/v1/auth/logout",
		"POST /api/v1/auth/validate-affiliate-code",
		"POST /api/v1/auth/validate-invitation-code",
		"POST /api/v1/auth/forgot-password",
		"POST /api/v1/auth/reset-password",
		"GET /api/v1/announcements",
		"GET /api/v1/admin/groups/:id/subscriptions",
		"GET /api/v1/admin/users/:id/subscriptions",
		"GET /api/v1/admin/settings/admin-api-key",
		"GET /api/v1/admin/settings/overload-cooldown",
		"GET /api/v1/admin/settings/rate-limit-429-cooldown",
		"GET /api/v1/admin/settings/stream-timeout",
		"GET /api/v1/admin/settings/rectifier",
		"GET /api/v1/admin/backups/schedule",
		"GET /api/v1/admin/usage",
		"GET /api/v1/admin/user-attributes",
		"PUT /api/v1/admin/api-keys/:id",
		"GET /api/v1/admin/tls-fingerprint-profiles",
		"GET /api/v1/admin/affiliates/tree",
	}

	for _, route := range expected {
		_, ok := routeSet[route]
		require.Truef(t, ok, "expected route to be registered: %s", route)
	}
}
