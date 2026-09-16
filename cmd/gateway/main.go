package main

import (
	"context"
	"errors"
	"gateway/internal/config"
	"gateway/internal/httperr"
	"gateway/internal/logger"
	"gateway/internal/middleware"
	"gateway/internal/routing"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load .env file
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load env")
		os.Exit(1)
	}

	// Start logger
	lgr := logger.New(cfg.Server.LogLevel)
	slog.SetDefault(lgr)
	slog.Info("gateway starting", "addr", cfg.Server.Addr, "log_level", cfg.Server.LogLevel)

	// Load config file
	cfgyml, err := routing.LoadFile(cfg.ConfigFile)
	if err != nil {
		slog.Error("Failed to load config")
		os.Exit(1)
	}

	table, err := routing.NewTable(cfgyml)
	if err != nil {
		slog.Error("Failed to build routing table")
		os.Exit(1)
	}

	// build new gin router
	router := gin.New()

	// registers middleware
	router.Use(middleware.Recovery())
	router.Use(middleware.Logging())
	router.Use(middleware.Timeout(10 * time.Second))

	// Loop through Routes and check if proxy is present.
	// Change context so the timeout gets baked in directly.
	for _, route := range table.Routes {
		router.Handle(route.Method, route.Path, func(ctx *gin.Context) {
			entry, ok := table.ProxyFor(route.Backend)
			if !ok {
				httperr.Respond(ctx, http.StatusBadGateway, "bad_gateway", "backend not found")
				return
			}

			// We do not handle timeouts. Proxy does.
			// Returns new context with an auto-cancel baked in.
			// Also returns cancel func early timeout cancel if request was fulfilled
			reqCtx, cancel := context.WithTimeout(ctx.Request.Context(), entry.Timeout)
			// Cancels timer after request was fulfilled
			defer cancel()
			// Shallow copy of the original request with the timeout attached
			ctx.Request = ctx.Request.WithContext(reqCtx)

			// Serve HTTP-Server for each proxy entry in routes
			entry.Proxy.ServeHTTP(ctx.Writer, ctx.Request)
		})
	}
	router.NoRoute(func(ctx *gin.Context) {
		httperr.Respond(ctx, http.StatusNotFound, "not_found", "route not found")
	})

	router.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	readTimeout, err := time.ParseDuration(cfg.Server.ReadTimeout)
	if err != nil {
		slog.Error("Read Timeout", "error", err)
	}
	writeTimeout, err := time.ParseDuration(cfg.Server.WriteTimeout)
	if err != nil {
		slog.Error("Write Timeout", "error", err)
	}
	shutdownTimeout, err := time.ParseDuration(cfg.Server.ShutdownTimeout)
	if err != nil {
		slog.Error("Write Timeout", "error", err)
	}

	server := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      router,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	// Creates a signal channel
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Reads the signal channel until SIGINT/SIGTERM arrives
	<-sigCh
	slog.Info("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	shErr := server.Shutdown(ctx)
	if shErr != nil {
		slog.Error("shutdown error", "error", err)
		os.Exit(1)
	} else {
		slog.Info("shutdown complete")
		os.Exit(0)
	}
}
