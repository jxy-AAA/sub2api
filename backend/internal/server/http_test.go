package server

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
)

func TestProvideHTTPServer_EnforcesGlobalBodyLimitWithAndWithoutH2C(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/test", func(c *gin.Context) {
		_, err := io.ReadAll(c.Request.Body)
		if err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				c.Status(http.StatusRequestEntityTooLarge)
				return
			}
			c.Status(http.StatusBadRequest)
			return
		}
		c.Status(http.StatusOK)
	})

	tests := []struct {
		name       string
		h2cEnabled bool
	}{
		{name: "http1", h2cEnabled: false},
		{name: "h2c", h2cEnabled: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Gateway.MaxBodySize = 4
			cfg.Server.H2C.Enabled = tt.h2cEnabled

			srv := ProvideHTTPServer(cfg, router)
			req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader([]byte("12345")))
			rec := httptest.NewRecorder()

			srv.Handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
			}
		})
	}
}

func TestProvideHTTPServer_SetsReadBodyTimeoutAndConnContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	cfg := &config.Config{}
	cfg.Server.ReadHeaderTimeout = 30
	cfg.Server.ReadBodyTimeout = 11
	cfg.Server.IdleTimeout = 60

	srv := ProvideHTTPServer(cfg, router)

	if srv.ConnContext == nil {
		t.Fatal("ConnContext should be configured for body read timeouts")
	}

	if srv.ReadHeaderTimeout.Seconds() != 30 {
		t.Fatalf("ReadHeaderTimeout = %s, want 30s", srv.ReadHeaderTimeout)
	}
}
