package main

//go:generate go run github.com/google/wire/cmd/wire

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/setup"
	"github.com/Wei-Shaw/sub2api/internal/web"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

//go:embed VERSION
var embeddedVersion string

// Build-time variables (can be set by ldflags)
var (
	Version   = ""
	Commit    = "unknown"
	Date      = "unknown"
	BuildType = "source"
)

func init() {
	if strings.TrimSpace(Version) != "" {
		return
	}

	Version = strings.TrimSpace(embeddedVersion)
	if Version == "" {
		Version = "0.0.0-dev"
	}
}

func main() {
	os.Exit(realMain())
}

func realMain() int {
	logger.InitBootstrap()
	defer logger.Sync()

	if err := run(); err != nil {
		log.Printf("Sub2API exited with error: %v", err)
		return 1
	}
	return 0
}

func run() error {
	setupMode := flag.Bool("setup", false, "Run setup wizard in CLI mode")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		log.Printf("Sub2API %s (commit: %s, built: %s)\n", Version, Commit, Date)
		return nil
	}

	if *setupMode {
		if err := setup.RunCLI(); err != nil {
			return fmt.Errorf("setup failed: %w", err)
		}
		return nil
	}

	if setup.NeedsSetup() {
		if setup.AutoSetupEnabled() {
			log.Println("Auto setup mode enabled...")
			if err := setup.AutoSetupFromEnv(); err != nil {
				return fmt.Errorf("auto setup failed: %w", err)
			}
		} else {
			log.Println("First run detected, starting setup wizard...")
			return runSetupServer()
		}
	}

	return runMainServer()
}

func runSetupServer() error {
	r := gin.New()
	if err := r.SetTrustedProxies(nil); err != nil {
		return fmt.Errorf("disable setup trusted proxies: %w", err)
	}
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS(config.CORSConfig{}))
	r.Use(middleware.SecurityHeaders(config.CSPConfig{Enabled: true, Policy: config.DefaultCSPPolicy}, nil))

	setup.RegisterRoutes(r)

	if web.HasEmbeddedFrontend() {
		r.Use(web.ServeEmbeddedFrontend())
	}

	addr := config.GetServerAddress()
	log.Printf("Setup wizard available at http://%s", addr)
	log.Println("Complete the setup wizard to configure Sub2API")

	server := &http.Server{
		Addr:              addr,
		Handler:           h2c.NewHandler(r, &http2.Server{}),
		ReadHeaderTimeout: 30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("start setup server: %w", err)
	}
	return nil
}

func runMainServer() error {
	cfg, err := config.LoadForBootstrap()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := logger.Init(logger.OptionsFromConfig(cfg.Log)); err != nil {
		return fmt.Errorf("initialize logger: %w", err)
	}
	if cfg.RunMode == config.RunModeSimple {
		log.Println("⚠️  WARNING: Running in SIMPLE mode - billing and quota checks are DISABLED")
	}

	buildInfo := handler.BuildInfo{
		Version:   Version,
		BuildType: BuildType,
	}

	app, err := initializeApplication(buildInfo)
	if err != nil {
		return fmt.Errorf("initialize application: %w", err)
	}

	return runApplication(app, shutdownTimeoutFromConfig(cfg), nil)
}

func runApplication(app *Application, shutdownTimeout time.Duration, signals <-chan os.Signal) error {
	if app == nil || app.Server == nil {
		return errors.New("application server not initialized")
	}
	return serveApplication(app, shutdownTimeout, signals)
}

func serveApplication(app *Application, shutdownTimeout time.Duration, signals <-chan os.Signal) (retErr error) {
	if app == nil || app.Server == nil {
		return errors.New("server is nil")
	}
	server := app.Server
	if shutdownTimeout <= 0 {
		shutdownTimeout = 30 * time.Second
	}
	defer func() {
		if app.Cleanup == nil {
			return
		}
		cleanupCtx := context.Background()
		if shutdownTimeout > 0 {
			var cancel context.CancelFunc
			cleanupCtx, cancel = context.WithTimeout(context.Background(), shutdownTimeout)
			defer cancel()
		}
		app.Cleanup(cleanupCtx)
	}()

	serverErrCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
		}
		close(serverErrCh)
	}()

	log.Printf("Server started on %s", server.Addr)

	if signals == nil {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(quit)
		signals = quit
	}

	select {
	case err, ok := <-serverErrCh:
		if ok && err != nil {
			return fmt.Errorf("start server: %w", err)
		}
		log.Println("Server exited")
		return nil
	case sig, ok := <-signals:
		if !ok {
			return errors.New("signal channel closed unexpectedly")
		}
		log.Printf("Shutting down server on signal %s...", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		_ = server.Close()
		return fmt.Errorf("shutdown server after %s: %w", shutdownTimeout, err)
	}

	log.Println("Server exited")
	return nil
}

func shutdownTimeoutFromConfig(cfg *config.Config) time.Duration {
	if cfg != nil && cfg.Server.ShutdownTimeout > 0 {
		return time.Duration(cfg.Server.ShutdownTimeout) * time.Second
	}
	return 30 * time.Second
}
