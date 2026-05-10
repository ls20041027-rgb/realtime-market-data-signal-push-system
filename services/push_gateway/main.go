package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"push_gateway/api"
	"push_gateway/config"
	kafkax "push_gateway/kafka"
	"push_gateway/storage"
	"push_gateway/ws"

	"github.com/gin-gonic/gin"
)

func main() {
	os.Exit(run())
}

func run() int {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config failed", "err", err)
		return 1
	}

	logger := newLogger(cfg.Runtime.LogLevel)
	slog.SetDefault(logger)

	logger.Info("push_gateway starting",
		"pid", os.Getpid(),
		"listen_addr", cfg.HTTP.ListenAddr,
		"kafka_group", cfg.Kafka.GroupID,
		"redis_host", cfg.Redis.Host,
		"postgres_host", cfg.Postgres.Host,
	)

	redisStore, err := storage.NewRedis(cfg.Redis)
	if err != nil {
		logger.Error("redis init failed", "err", err)
		return 2
	}
	defer closeOrLog(logger, "redis", redisStore.Close)

	mysqlStore, err := storage.NewPostgres(cfg.Postgres)
	if err != nil {
		logger.Error("postgres init failed", "err", err)
		return 2
	}
	defer closeOrLog(logger, "postgres", mysqlStore.Close)

	hub := ws.NewHub(cfg.WS)
	hubCtx, hubCancel := context.WithCancel(context.Background())
	defer hubCancel()
	hubDone := make(chan struct{})
	go func() {
		defer close(hubDone)
		hub.Run(hubCtx)
	}()

	consumer := kafkax.NewConsumer(cfg.Kafka, hub)
	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	defer consumerCancel()
	consumerDone := make(chan struct{})
	go func() {
		defer close(consumerDone)
		consumer.Run(consumerCtx)
	}()

	deps := api.Deps{
		Cfg:      cfg,
		Redis:    redisStore,
		MySQL:    mysqlStore,
		Hub:      hub,
		Consumer: consumer,
		StartAt:  time.Now(),
	}
	router := api.NewRouter(deps)

	metricsCtx, metricsCancel := context.WithCancel(context.Background())
	defer metricsCancel()
	api.StartMetricsLoop(metricsCtx, deps)

	router.GET("/ws", gin.HandlerFunc(ws.HandleWS(hub, cfg.WS)))
	router.GET("/ws/market", gin.HandlerFunc(ws.HandleWS(hub, cfg.WS)))
	router.GET("/ws/signals", gin.HandlerFunc(ws.HandleWS(hub, cfg.WS)))

	server := &http.Server{
		Addr:         cfg.HTTP.ListenAddr,
		Handler:      router,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}

	serverErrCh := make(chan error, 1)
	go func() {
		logger.Info("http server listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
			return
		}
		serverErrCh <- nil
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received, graceful stopping",
			"shutdown_timeout", cfg.HTTP.ShutdownTimeout.String())
	case err := <-serverErrCh:
		if err != nil {
			logger.Error("http server exited abnormally", "err", err)
			return 3
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("http server shutdown failed", "err", err)
	} else {
		logger.Info("http server stopped")
	}

	hub.Close()
	hubCancel()
	waitWithTimeout(hubDone, cfg.HTTP.ShutdownTimeout, logger, "hub")

	consumerCancel()
	if err := consumer.Close(); err != nil {
		logger.Error("consumer close failed", "err", err)
	}
	waitWithTimeout(consumerDone, cfg.HTTP.ShutdownTimeout, logger, "consumer")

	logger.Info("push_gateway stopped", "goroutines", runtime.NumGoroutine())
	return 0
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(strings.ToUpper(strings.TrimSpace(level)))); err != nil {
		lvl = slog.LevelInfo
	}
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     lvl,
		AddSource: false,
	})
	return slog.New(handler).With("service", "push_gateway")
}

func closeOrLog(logger *slog.Logger, name string, closeFn func() error) {
	start := time.Now()
	if err := closeFn(); err != nil {
		logger.Error("component close failed", "component", name, "err", err)
		return
	}
	logger.Info("component closed", "component", name, "cost_ms", time.Since(start).Milliseconds())
}

func waitWithTimeout(done <-chan struct{}, timeout time.Duration, logger *slog.Logger, name string) {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case <-done:
		logger.Info("component stopped", "component", name)
	case <-t.C:
		logger.Warn("component stop timeout, continuing", "component", name, "timeout", timeout.String())
	}
}
