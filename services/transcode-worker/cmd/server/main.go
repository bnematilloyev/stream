package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sahiy/sahiy-stream/pkg/logger"
	pkgnats "github.com/sahiy/sahiy-stream/pkg/nats"
	"github.com/sahiy/sahiy-stream/pkg/storage"
	httphandler "github.com/sahiy/sahiy-stream/services/transcode-worker/internal/adapter/handler/http"
	"github.com/sahiy/sahiy-stream/services/transcode-worker/internal/config"
	"github.com/sahiy/sahiy-stream/services/transcode-worker/internal/worker"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	log, err := logger.New("transcode-worker", cfg.LogLevel)
	if err != nil {
		panic(err)
	}
	defer func() { _ = log.Sync() }()

	bus, err := pkgnats.NewTranscodeBus(pkgnats.DefaultConfig(cfg.NATSURL))
	if err != nil {
		log.Fatal("nats failed", zap.Error(err))
	}
	defer bus.Close()

	store, err := storage.New(cfg.Storage)
	if err != nil {
		log.Fatal("storage init failed", zap.Error(err))
	}
	if err := store.EnsureBucket(context.Background()); err != nil {
		log.Warn("storage bucket ensure failed", zap.Error(err))
	}

	workerID := cfg.WorkerID
	if workerID == "" {
		host, _ := os.Hostname()
		if host == "" {
			host = "transcode-worker"
		}
		workerID = host
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool := worker.NewPool(workerID, cfg.FFmpegPath, cfg.VideoEncoder, cfg.TranscodeQuality, cfg.MaxJobs, bus, store, log)
	go func() {
		if err := pool.Run(ctx); err != nil {
			log.Error("worker pool stopped", zap.Error(err))
		}
	}()

	router := chi.NewRouter()
	router.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)
	router.Mount("/", httphandler.NewHealthHandler(bus).Routes())

	server := &http.Server{Addr: cfg.HTTPAddr, Handler: router, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Info("transcode-worker started",
			zap.String("addr", cfg.HTTPAddr),
			zap.String("worker_id", workerID),
			zap.Int("max_jobs", cfg.MaxJobs),
			zap.String("encoder", cfg.VideoEncoder),
			zap.String("storage", store.Backend()),
			zap.String("quality", cfg.TranscodeQuality),
		)
		_ = server.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = server.Shutdown(shutdownCtx)
}
