package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sahiy/sahiy-stream/pkg/analytics"
	"github.com/sahiy/sahiy-stream/pkg/database"
	"github.com/sahiy/sahiy-stream/pkg/grpcserver"
	"github.com/sahiy/sahiy-stream/pkg/logger"
	"github.com/sahiy/sahiy-stream/pkg/playback"
	pkgredis "github.com/sahiy/sahiy-stream/pkg/redis"
	"github.com/sahiy/sahiy-stream/pkg/storage"
	"github.com/sahiy/sahiy-stream/pkg/viewers"
	grpchandler "github.com/sahiy/sahiy-stream/services/stream-service/internal/adapter/handler/grpc"
	httphandler "github.com/sahiy/sahiy-stream/services/stream-service/internal/adapter/handler/http"
	"github.com/sahiy/sahiy-stream/services/stream-service/internal/adapter/repository"
	"github.com/sahiy/sahiy-stream/services/stream-service/internal/config"
	"github.com/sahiy/sahiy-stream/services/stream-service/internal/usecase"
	"github.com/sahiy/sahiy-stream/services/stream-service/internal/worker"
	streamv1 "github.com/sahiy/sahiy-stream/proto/gen/stream/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		panic("invalid config: " + err.Error())
	}

	log, _ := logger.New("stream-service", cfg.LogLevel)
	defer func() { _ = log.Sync() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbRouter, err := database.NewRouter(ctx, database.LoadPoolsConfigFromEnv(
		cfg.DatabaseURL, cfg.DatabaseReplicaURL, cfg.DBMaxConns, cfg.DBMinConns,
	))
	if err != nil {
		log.Fatal("database failed", zap.Error(err))
	}
	defer dbRouter.Close()

	redisClient, err := pkgredis.NewClientFromConfig(cfg.Redis)
	if err != nil {
		log.Fatal("redis failed", zap.Error(err))
	}
	defer func() { _ = redisClient.Close() }()

	store, err := storage.New(cfg.Storage)
	if err != nil {
		log.Fatal("storage init failed", zap.Error(err))
	}
	if err := store.EnsureBucket(ctx); err != nil {
		log.Warn("storage bucket ensure failed", zap.Error(err))
	}

	signer := playback.NewSigner(cfg.PlaybackSignSecret, cfg.PlaybackURLTTL)

	streamRepo := repository.NewPostgresStreamRepository(dbRouter)
	channelRepo := repository.NewPostgresChannelRepository(dbRouter)
	keyRepo := repository.NewPostgresStreamKeyRepository(dbRouter)
	mediaRepo := repository.NewPostgresStreamMediaRepository(dbRouter)

	analyticsClient, err := analytics.NewClient(cfg.ClickHouse)
	if err != nil {
		log.Fatal("clickhouse init failed", zap.Error(err))
	}
	defer func() { _ = analyticsClient.Close() }()

	counter := viewers.NewCounter(redisClient, cfg.ViewerWindow)
	viewerUC := usecase.NewViewerUseCase(streamRepo, counter, analyticsClient)
	uc := usecase.NewStreamUseCase(streamRepo, channelRepo, keyRepo, viewerUC)
	playbackUC := usecase.NewPlaybackUseCase(streamRepo, mediaRepo, signer, cfg.PlaybackBaseURL)

	go worker.NewStaleCleanupWorker(streamRepo, cfg.StaleCleanupInterval, log).Run(ctx)
	go worker.NewViewerSyncWorker(streamRepo, counter, cfg.ViewerSyncInterval, log).Run(ctx)

	grpcServer := grpc.NewServer(grpcserver.DefaultServerOptions(log, cfg.GRPCRequestTimeout)...)
	streamv1.RegisterStreamServiceServer(grpcServer, grpchandler.NewStreamServer(uc, playbackUC, viewerUC))
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatal("listen failed", zap.Error(err))
	}
	go func() {
		log.Info("stream gRPC started", zap.String("addr", cfg.GRPCAddr))
		_ = grpcServer.Serve(lis)
	}()

	httpRouter := chi.NewRouter()
	httpRouter.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)
	httpRouter.Mount("/", httphandler.NewHealthHandler(dbRouter).Routes())
	httpRouter.Mount("/", httphandler.NewDeliveryHandler(store, signer).Routes())
	httpServer := &http.Server{Addr: cfg.HTTPAddr, Handler: httpRouter, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Info("stream HTTP started",
			zap.String("addr", cfg.HTTPAddr),
			zap.String("storage", store.Backend()),
			zap.Bool("db_replica", dbRouter.HasReplica()),
			zap.Bool("clickhouse", cfg.ClickHouse.Enabled),
		)
		_ = httpServer.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cancel()
	grpcServer.GracefulStop()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = httpServer.Shutdown(shutdownCtx)
}
