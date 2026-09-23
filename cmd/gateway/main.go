package main

import (
	"context"
	"errors"
	"gateway/internal/auth"
	"gateway/internal/config"
	"gateway/internal/httperr"
	"gateway/internal/logger"
	"gateway/internal/middleware"
	"gateway/internal/routing"
	"gateway/internal/store"
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

	pool, err := store.Open(context.Background(), cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to open postgres pool", "error", err)
		// manual close because os.Exit stops without respecting defer
		os.Exit(1)
	}
	defer pool.Close()

	err = store.Migrate("migrations", cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to apply migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("database connected")

	userStore := store.NewUserStore(pool)

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

	// Requests get checked against the set global Request Timeout
	requestTimeout, err := time.ParseDuration(cfg.Server.RequestTimeout)
	if err != nil {
		slog.Error("Request Timeout not parsable", "error", err)
		os.Exit(1)
	}
	table.CheckTimeouts(requestTimeout)

	// build new gin router
	router := gin.New()

	// registers middleware
	router.Use(middleware.Recovery())
	router.Use(middleware.Logging())
	router.Use(middleware.Timeout(10 * time.Second))

	api := router.Group(routing.RoutePrefix)

	// Loop through Routes and check if proxy is present.
	// Change context so the timeout gets baked in directly.
	for _, route := range table.Routes {
		api.Handle(route.Method, route.Path, func(ctx *gin.Context) {
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
	router.POST("/register", func(ctx *gin.Context) {
		type Req struct {
			Email    string
			Password string
		}
		var req Req

		if err := ctx.ShouldBindJSON(&req); err != nil {
			httperr.Respond(ctx, http.StatusBadRequest, "invalid_input", "can't parse request")
			slog.Error("Error in Request Body", "error", err)
			return
		}
		if req.Email == "" {
			httperr.Respond(ctx, http.StatusBadRequest, "invalid_input", "email field cannot be empty")
			return
		}
		// WARN: only basic pw rules
		if req.Password == "" || len(req.Password) < 8 {
			httperr.Respond(ctx, http.StatusBadRequest, "invalid_input", "password not in the correct format")
			return
		}

		pwHash, err := auth.Hash(req.Password)
		if err != nil {
			httperr.Respond(ctx, http.StatusInternalServerError, "internal_error", "internal server error")
			slog.Error("PW hashing failed", "error", err)
			return
		}
		newUser, err := userStore.Create(ctx.Request.Context(), req.Email, pwHash)
		if err != nil {
			if errors.Is(err, store.ErrEmailExists) {
				httperr.Respond(ctx, http.StatusConflict, "email_exists", "email already registered")
				return
			}
			httperr.Respond(ctx, http.StatusInternalServerError, "internal_error", "registration failed")
			slog.Error("User creation failed", "error", err)
			return
		}
		ctx.JSON(http.StatusCreated, gin.H{"id": newUser.ID, "email": newUser.Email})
	})

	readTimeout, err := time.ParseDuration(cfg.Server.ReadTimeout)
	if err != nil {
		slog.Error("Read Timeout", "error", err)
		os.Exit(1)
	}
	writeTimeout, err := time.ParseDuration(cfg.Server.WriteTimeout)
	if err != nil {
		slog.Error("Write Timeout", "error", err)
		os.Exit(1)
	}
	shutdownTimeout, err := time.ParseDuration(cfg.Server.ShutdownTimeout)
	if err != nil {
		slog.Error("Write Timeout", "error", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:           cfg.Server.Addr,
		Handler:        router,
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		MaxHeaderBytes: 1 << 20, // 1MB (1 * 20**20)
	}

	// Buffered signal channel — capacity 1 so a signal is not dropped
	// if the receiver is not ready the instant it arrives.
	sigCh := make(chan os.Signal, 1)

	// Route SIGINT/SIGTERM to sigCh
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Run the server in a goroutine so the main goroutine is free to
	// wait for a signal. ListenAndServe blocks until the server stops.
	go func() {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Block here until a signal arrives on sigCh.
	// Main goroutine waits.
	<-sigCh
	slog.Info("shutdown signal received")

	// Give in-flight handlers a deadline to finish. shutdownTimeout
	// comes from config. defer cancel() releases the timer resources
	// even if Shutdown returns early.
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// Shutdown stops accepting new requests and waits for active
	// ones to finish (only to deadline)
	shErr := server.Shutdown(ctx)
	if shErr != nil {
		// Shutdown timed out or failed — some connections were dropped.
		slog.Error("shutdown error", "error", shErr)
		os.Exit(1)
	} else {
		// All in-flight requests completed within the deadline.
		slog.Info("shutdown complete")
		pool.Close()
		os.Exit(0)
	}
}
