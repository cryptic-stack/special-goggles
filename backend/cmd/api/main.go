package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cryptic-stack/special-goggles/backend/internal/ap/delivery"
	"github.com/cryptic-stack/special-goggles/backend/internal/config"
	"github.com/cryptic-stack/special-goggles/backend/internal/domain/accounts"
	httpapi "github.com/cryptic-stack/special-goggles/backend/internal/http"
	postgresstore "github.com/cryptic-stack/special-goggles/backend/internal/storage/postgres"
	redisstore "github.com/cryptic-stack/special-goggles/backend/internal/storage/redis"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pg, err := connectPostgresWithRetry(ctx, cfg.DBDSN, logger)
	if err != nil {
		logger.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pg.Close()

	if err := postgresstore.RunMigrations(ctx, pg, cfg.MigrationsDir, logger); err != nil {
		logger.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	if cfg.AppEnv == "dev" {
		if err := accounts.SeedDevAlice(ctx, pg, cfg.AppBaseURL, cfg.AppDomain, cfg.DevSeedPassword, logger); err != nil {
			logger.Error("failed to seed dev actor", "error", err)
			os.Exit(1)
		}
	}

	redisClient, err := connectRedisWithRetry(ctx, cfg.RedisAddr, logger)
	if err != nil {
		logger.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	deliveryWorker := delivery.NewWorker(pg, logger)
	go deliveryWorker.Run(ctx)

	router := httpapi.NewRouter(httpapi.Dependencies{
		Config: cfg,
		Logger: logger,
		PG:     pg,
	})

	server := &http.Server{
		Addr:              cfg.AppListenAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("server shutdown failed", "error", err)
		}
	}()

	logger.Info("api listening", "addr", cfg.AppListenAddr, "env", cfg.AppEnv)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server exited unexpectedly", "error", err)
		os.Exit(1)
	}
}

func connectPostgresWithRetry(ctx context.Context, dsn string, logger *slog.Logger) (*pgxpool.Pool, error) {
	var lastErr error

	for attempt := 1; attempt <= 10; attempt++ {
		pool, err := postgresstore.Open(ctx, dsn)
		if err == nil {
			return pool, nil
		}
		lastErr = err

		logger.Warn("postgres unavailable, retrying", "attempt", attempt, "error", err)
		if err := waitOrCancel(ctx, 2*time.Second); err != nil {
			return nil, err
		}
	}

	return nil, errors.New("postgres unavailable after retries: " + lastErr.Error())
}

func connectRedisWithRetry(ctx context.Context, addr string, logger *slog.Logger) (*redis.Client, error) {
	var lastErr error

	for attempt := 1; attempt <= 10; attempt++ {
		client, err := redisstore.Open(ctx, addr)
		if err == nil {
			return client, nil
		}
		lastErr = err

		logger.Warn("redis unavailable, retrying", "attempt", attempt, "error", err)
		if err := waitOrCancel(ctx, 2*time.Second); err != nil {
			return nil, err
		}
	}

	return nil, errors.New("redis unavailable after retries: " + lastErr.Error())
}

func waitOrCancel(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
