package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"push_gateway/api"
	"push_gateway/config"
	"push_gateway/internal/latency"
	log "push_gateway/internal/log"
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
		log.Errorf("load config failed: %v", err)
		return 1
	}

	if err := log.Init(cfg.Runtime.LogLevel, "logs", "push_gateway.log"); err != nil {
		log.Errorf("init logger failed: %v", err)
		return 1
	}
	defer log.Close()

	log.Infof("push_gateway starting pid=%d listen_addr=%s kafka_group=%s redis_host=%s postgres_host=%s",
		os.Getpid(), cfg.HTTP.ListenAddr, cfg.Kafka.GroupID, cfg.Redis.Host, cfg.Postgres.Host)

	redisStore, err := storage.NewRedis(cfg.Redis)
	if err != nil {
		log.Errorf("redis init failed: %v", err)
		return 2
	}
	defer closeOrLog("redis", redisStore.Close)

	latency.Init(redisStore.Client())
	defer latency.Close()

	mysqlStore, err := storage.NewPostgres(cfg.Postgres)
	if err != nil {
		log.Errorf("postgres init failed: %v", err)
		return 2
	}
	defer closeOrLog("postgres", mysqlStore.Close)

	// Auto-migrate: ensure users and watchlist tables exist
	if err := mysqlStore.EnsureUserTable(context.Background()); err != nil {
		log.Errorf("ensure users table failed: %v", err)
		return 2
	}
	if err := mysqlStore.EnsureWatchlistTable(context.Background()); err != nil {
		log.Errorf("ensure watchlist table failed: %v", err)
		return 2
	}

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
		Cfg:     cfg,
		Redis:   redisStore,
		MySQL:   mysqlStore,
		Hub:     hub,
		StartAt: time.Now(),
	}
	router := api.NewRouter(deps)

	metricsCtx, metricsCancel := context.WithCancel(context.Background())
	defer metricsCancel()
	api.StartMetricsLoop(metricsCtx, deps)
	api.StartResourceMonitor(deps)
	defer api.StopResourceMonitor()

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
		log.Infof("http server listening addr=%s", server.Addr)
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
		log.Infof("shutdown signal received, graceful stopping shutdown_timeout=%s",
			cfg.HTTP.ShutdownTimeout.String())
	case err := <-serverErrCh:
		if err != nil {
			log.Errorf("http server exited abnormally: %v", err)
			return 3
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Errorf("http server shutdown failed: %v", err)
	} else {
		log.Infof("http server stopped")
	}

	hub.Close()
	hubCancel()
	waitWithTimeout(hubDone, cfg.HTTP.ShutdownTimeout, "hub")

	consumerCancel()
	if err := consumer.Close(); err != nil {
		log.Errorf("consumer close failed: %v", err)
	}
	waitWithTimeout(consumerDone, cfg.HTTP.ShutdownTimeout, "consumer")

	log.Infof("push_gateway stopped goroutines=%d", runtime.NumGoroutine())
	return 0
}

func closeOrLog(name string, closeFn func() error) {
	start := time.Now()
	if err := closeFn(); err != nil {
		log.Errorf("component close failed component=%s err=%v", name, err)
		return
	}
	log.Infof("component closed component=%s cost_ms=%d", name, time.Since(start).Milliseconds())
}

func waitWithTimeout(done <-chan struct{}, timeout time.Duration, name string) {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case <-done:
		log.Infof("component stopped component=%s", name)
	case <-t.C:
		log.Warnf("component stop timeout, continuing component=%s timeout=%s", name, timeout.String())
	}
}
