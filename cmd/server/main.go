package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/ecommerce/feature-management/internal/api"
	"github.com/ecommerce/feature-management/internal/cache"
	"github.com/ecommerce/feature-management/internal/config"
	"github.com/ecommerce/feature-management/internal/db"
	"github.com/ecommerce/feature-management/internal/evaluator"
	"github.com/ecommerce/feature-management/internal/observability"
	"github.com/ecommerce/feature-management/internal/repository"
	"github.com/ecommerce/feature-management/internal/service"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// ── Config ──────────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// ── Logger ───────────────────────────────────────────────────────────────
	logger, err := observability.NewLogger(cfg.Log)
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer logger.Sync() //nolint:errcheck

	// ── Tracer ───────────────────────────────────────────────────────────────
	ctx := context.Background()
	tp, err := observability.NewTracerProvider(ctx, cfg.Observability)
	if err != nil {
		logger.Warn("failed to init tracer, continuing without tracing", zap.Error(err))
	} else {
		defer tp.Shutdown(ctx) //nolint:errcheck
	}

	// ── PostgreSQL ───────────────────────────────────────────────────────────
	pool, err := db.NewPool(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer pool.Close()
	logger.Info("connected to postgres")

	// ── Redis ────────────────────────────────────────────────────────────────
	redisOpts, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		return fmt.Errorf("parse redis url: %w", err)
	}
	if cfg.Redis.Password != "" {
		redisOpts.Password = cfg.Redis.Password
	}
	redisOpts.DB = cfg.Redis.DB
	redisClient := redis.NewClient(redisOpts)
	defer redisClient.Close()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("connect to redis: %w", err)
	}
	logger.Info("connected to redis")

	// ── Prometheus metrics ───────────────────────────────────────────────────
	metrics := observability.NewMetrics()

	// ── L1 ristretto cache ───────────────────────────────────────────────────
	ristrettoCache, err := ristretto.NewCache(&ristretto.Config[string, []byte]{
		NumCounters: 1e7,     // track 10M keys for admission
		MaxCost:     1 << 28, // 256 MB
		BufferItems: 64,
	})
	if err != nil {
		return fmt.Errorf("init ristretto: %w", err)
	}

	// ── Cache layers ─────────────────────────────────────────────────────────
	l1 := cache.NewL1Cache(ristrettoCache)
	l2 := cache.NewL2Cache(redisClient)
	tiered := cache.NewTieredCache(l1, l2, metrics)

	// ── Repositories ─────────────────────────────────────────────────────────
	flagRepo := repository.NewFlagRepo(pool)
	ruleRepo := repository.NewRuleRepo(pool)
	segmentRepo := repository.NewSegmentRepo(pool)
	auditRepo := repository.NewAuditRepo(pool)
	appRepo := repository.NewApplicationRepo(pool)
	envRepo := repository.NewEnvironmentRepo(pool)

	// ── Cache invalidator (Redis pub/sub) ────────────────────────────────────
	invalidator := cache.NewInvalidator(redisClient, tiered, logger)
	invCtx, invCancel := context.WithCancel(ctx)
	defer invCancel()
	go invalidator.Run(invCtx)

	// ── Evaluator engine ─────────────────────────────────────────────────────
	engine := evaluator.NewEngine(tiered, flagRepo, segmentRepo, envRepo)

	// ── Services ─────────────────────────────────────────────────────────────
	flagSvc := service.NewFlagService(flagRepo, ruleRepo, auditRepo, tiered, invalidator)
	evalSvc := service.NewEvaluationService(engine)
	segmentSvc := service.NewSegmentService(segmentRepo, auditRepo, invalidator)
	auditSvc := service.NewAuditService(auditRepo)
	appSvc := service.NewApplicationService(appRepo, envRepo)

	// ── HTTP server ───────────────────────────────────────────────────────────
	router := api.NewRouter(flagSvc, evalSvc, segmentSvc, auditSvc, appSvc, metrics, logger, cfg.Auth.APIKey)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// ── Graceful shutdown ────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("starting HTTP server", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", zap.Error(err))
		}
	}()

	<-quit
	logger.Info("shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", zap.Error(err))
		return err
	}

	logger.Info("server stopped")
	return nil
}
