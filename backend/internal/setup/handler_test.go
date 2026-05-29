package setup

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSetupGuardAllowsLocalRequest(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	resetSetupTokenTestState()

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Request.RemoteAddr = "127.0.0.1:12345"
	})
	router.Use(setupGuard())
	router.POST("/guarded", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/guarded", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status=%d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestSetupGuardRejectsRemoteRequestWithoutToken(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	resetSetupTokenTestState()

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Request.RemoteAddr = "203.0.113.10:12345"
	})
	router.Use(setupGuard())
	router.POST("/guarded", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/guarded", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.10")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status=%d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestSetupGuardAllowsLoopbackPeerDespiteForwardedHeader(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	resetSetupTokenTestState()

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Request.RemoteAddr = "127.0.0.1:12345"
	})
	router.Use(setupGuard())
	router.POST("/guarded", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/guarded", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.10")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status=%d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestSetupGuardRejectsForwardedLoopbackOnRemotePeer(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	resetSetupTokenTestState()

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Request.RemoteAddr = "203.0.113.10:12345"
	})
	router.Use(setupGuard())
	router.POST("/guarded", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/guarded", nil)
	req.Header.Set("X-Forwarded-For", "127.0.0.1")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status=%d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestSetupGuardAllowsRemoteRequestWithToken(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv(setupTokenEnvKey, "setup-token-123")
	resetSetupTokenTestState()

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Request.RemoteAddr = "203.0.113.10:12345"
	})
	router.Use(setupGuard())
	router.POST("/guarded", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/guarded", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.10")
	req.Header.Set(setupTokenHeaderName, "setup-token-123")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status=%d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestGetStatusReportsRemoteTokenRequirement(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv(setupTokenEnvKey, "setup-token-123")
	resetSetupTokenTestState()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodGet, "/setup/status", nil)
	req.RemoteAddr = "203.0.113.10:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.10")
	c.Request = req

	getStatus(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", recorder.Code, http.StatusOK)
	}

	var payload struct {
		Data SetupStatus `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !payload.Data.SetupTokenRequired {
		t.Fatalf("SetupTokenRequired=false, want true")
	}
	if !payload.Data.SetupTokenAvailable {
		t.Fatalf("SetupTokenAvailable=false, want true")
	}
}

func resetSetupTokenTestState() {
	setupTokenCached = ""
	setupTokenOnce = sync.Once{}
}
