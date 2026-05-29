//go:build unit

package server_test

import (
	"reflect"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/server/routes"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newModelMarketContractHandlers() *handler.Handlers {
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

func collectModelMarketRouteSet(routesInfo gin.RoutesInfo) map[string]struct{} {
	routeSet := make(map[string]struct{}, len(routesInfo))
	for _, route := range routesInfo {
		routeSet[route.Method+" "+route.Path] = struct{}{}
	}
	return routeSet
}

func TestModelMarketRoutesAreRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	v1 := router.Group("/api/v1")
	handlers := newModelMarketContractHandlers()

	routes.RegisterAuthRoutes(v1, handlers, servermiddleware.JWTAuthMiddleware(func(c *gin.Context) {
		c.Next()
	}), nil, nil)
	routes.RegisterUserRoutes(v1, handlers, servermiddleware.JWTAuthMiddleware(func(c *gin.Context) {
		c.Next()
	}), nil)
	routes.RegisterAdminRoutes(v1, handlers, servermiddleware.AdminAuthMiddleware(func(c *gin.Context) {
		c.Next()
	}))

	routeSet := collectModelMarketRouteSet(router.Routes())
	expected := []string{
		"GET /api/v1/model-market/models",
		"GET /api/v1/admin/model-market/models",
		"POST /api/v1/admin/model-market/models",
		"PUT /api/v1/admin/model-market/models/:id",
		"DELETE /api/v1/admin/model-market/models/:id",
		"POST /api/v1/admin/model-market/models/import-from-channels",
	}

	for _, route := range expected {
		_, ok := routeSet[route]
		require.Truef(t, ok, "expected model market route to be registered: %s", route)
	}
}
