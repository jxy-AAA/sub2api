package main

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestShutdownTimeoutFromConfig(t *testing.T) {
	require.Equal(t, 30*time.Second, shutdownTimeoutFromConfig(nil))
	require.Equal(t, 45*time.Second, shutdownTimeoutFromConfig(&config.Config{
		Server: config.ServerConfig{ShutdownTimeout: 45},
	}))
}

func TestRunApplicationExecutesCleanupOnStartupError(t *testing.T) {
	cleanupCalled := false
	app := &Application{
		Server: &http.Server{Addr: "127.0.0.1:-1"},
		Cleanup: func() {
			cleanupCalled = true
		},
	}

	err := runApplication(app, time.Second, make(chan os.Signal))
	require.Error(t, err)
	require.True(t, cleanupCalled)
}
