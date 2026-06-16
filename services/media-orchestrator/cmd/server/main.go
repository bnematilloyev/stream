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
	"github.com/sahiy/sahiy-stream/pkg/database"
	"github.com/sahiy/sahiy-stream/pkg/logger"
	pkgnats "github.com/sahiy/sahiy-stream/pkg/nats"
	"github.com/sahiy/sahiy-stream/pkg/security/internalauth"
	"github.com/sahiy/sahiy-stream/pkg/storage"
	httphandler "github.com/sahiy/sahiy-stream/services/media-orchestrator/internal/adapter/handler/http"
	"github.com/sahiy/sahiy-stream/services/media-orchestrator/internal/adapter/repository"
	"github.com/sahiy/sahiy-stream/services/media-orchestrator/internal/client"
	"github.com/sahiy/sahiy-stream/services/media-orchestrator/internal/config"
	"github.com/sahiy/sahiy-stream/services/media-orchestrator/internal/pipeline"
	"github.com/sahiy/sahiy-stream/services/media-orchestrator/internal/transcode"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		panic("invalid config: " + err.Error())
	}

	log, _ := logger.New("media-orchestrator", cfg.LogLevel)
	defer func() { _ = log.Sync() }()

	ctx := context.Background()
	pool, err := database.NewPool(ctx, database.DefaultConfig(cfg.DatabaseURL))
	if err != nil {
		log.Fatal("database failed", zap.Error(err))
	}
	defer pool.Close()

	store, err := storage.New(cfg.Storage)
	if err != nil {
		log.Fatal("storage init failed", zap.Error(err))
	}
	if err := store.EnsureBucket(ctx); err != nil {
		log.Warn("storage bucket ensure failed", zap.Error(err))
	}

	streamClient, err := client.NewStreamClient(cfg.StreamServiceAddr)
	if err != nil {
		log.Fatal("stream client failed", zap.Error(err))
	}
	defer func() { _ = streamClient.Close() }()

	mediaRepo := repository.NewPostgresStreamMediaRepository(pool)

	var backend transcode.Backend
	switch cfg.TranscodeMode {
	case "queue":
		bus, err := pkgnats.NewTranscodeBus(pkgnats.DefaultConfig(cfg.NATSURL))
		if err != nil {
			log.Fatal("nats transcode bus failed", zap.Error(err))
		}
		defer bus.Close()
		backend = transcode.NewQueueBackend(bus)
		listener := transcode.NewEventListener(mediaRepo, log)
		if _, err := bus.SubscribeEvents(listener.Handle); err != nil {
			log.Fatal("transcode event subscribe failed", zap.Error(err))
		}
		log.Info("transcode mode: queue (NATS)",
			zap.String("worker_rtmp", cfg.RTMPWorkerURL),
			zap.String("note", "FFmpeg runs on GPU workers only, not this host"),
		)
	default:
		backend = transcode.NewLocalBackend(cfg.FFmpegPath, cfg.FFmpegVideoEncoder, cfg.TranscodeQuality)
		log.Info("transcode mode: local (embedded FFmpeg on this host)")
	}

	mgr := pipeline.NewManager(
		backend, cfg.FFmpegVideoEncoder, cfg.TranscodeQuality, mediaRepo, streamClient, store, cfg.SyncSegments(),
		cfg.RTMPInternalURL, cfg.RTMPWorkerURL, cfg.RTSPInternalURL, cfg.RTSPWorkerURL, cfg.HLSOutputDir, log,
	)

	secret, requireSecret, allowInternal := cfg.HookAuth()
	hookAuth := internalauth.Config{
		Secret: secret, RequireSecret: requireSecret, AllowInternal: allowInternal,
	}

	router := chi.NewRouter()
	router.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)
	router.Mount("/", httphandler.NewHookHandler(mgr, log, hookAuth).Routes())

	server := &http.Server{
		Addr: cfg.HTTPAddr, Handler: router,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	go func() {
		log.Info("media-orchestrator started",
			zap.String("addr", cfg.HTTPAddr),
			zap.String("storage", store.Backend()),
			zap.String("encoder", cfg.FFmpegVideoEncoder),
			zap.String("transcode_mode", cfg.TranscodeMode),
			zap.String("transcode_quality", cfg.TranscodeQuality),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("http failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}
