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
	httphandler "github.com/sahiy/sahiy-stream/services/media-orchestrator/internal/adapter/handler/http"
	"github.com/sahiy/sahiy-stream/services/media-orchestrator/internal/adapter/repository"
	"github.com/sahiy/sahiy-stream/services/media-orchestrator/internal/client"
	"github.com/sahiy/sahiy-stream/services/media-orchestrator/internal/config"
	"github.com/sahiy/sahiy-stream/services/media-orchestrator/internal/pipeline"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	log, _ := logger.New("media-orchestrator", cfg.LogLevel)
	defer func() { _ = log.Sync() }()

	ctx := context.Background()
	pool, err := database.NewPool(ctx, database.DefaultConfig(cfg.DatabaseURL))
	if err != nil {
		log.Fatal("database failed", zap.Error(err))
	}
	defer pool.Close()

	streamClient, err := client.NewStreamClient(cfg.StreamServiceAddr)
	if err != nil {
		log.Fatal("stream client failed", zap.Error(err))
	}
	defer func() { _ = streamClient.Close() }()

	mediaRepo := repository.NewPostgresStreamMediaRepository(pool)
	ffmpeg := pipeline.NewFFmpegRunner(cfg.FFmpegPath)
	mgr := pipeline.NewManager(ffmpeg, mediaRepo, streamClient, cfg.RTMPInternalURL, cfg.RTSPInternalURL, cfg.HLSOutputDir, cfg.HLSBaseURL, log)

	router := chi.NewRouter()
	router.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)
	router.Mount("/", httphandler.NewHookHandler(mgr, log).Routes())

	server := &http.Server{
		Addr: cfg.HTTPAddr, Handler: router,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	go func() {
		log.Info("media-orchestrator started", zap.String("addr", cfg.HTTPAddr))
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
