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
	"github.com/sahiy/sahiy-stream/pkg/database"
	"github.com/sahiy/sahiy-stream/pkg/logger"
	grpchandler "github.com/sahiy/sahiy-stream/services/stream-service/internal/adapter/handler/grpc"
	httphandler "github.com/sahiy/sahiy-stream/services/stream-service/internal/adapter/handler/http"
	"github.com/sahiy/sahiy-stream/services/stream-service/internal/adapter/repository"
	"github.com/sahiy/sahiy-stream/services/stream-service/internal/config"
	"github.com/sahiy/sahiy-stream/services/stream-service/internal/usecase"
	streamv1 "github.com/sahiy/sahiy-stream/proto/gen/stream/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.Load()
	log, _ := logger.New("stream-service", cfg.LogLevel)
	defer func() { _ = log.Sync() }()

	pool, err := database.NewPool(context.Background(), database.DefaultConfig(cfg.DatabaseURL))
	if err != nil {
		log.Fatal("database failed", zap.Error(err))
	}
	defer pool.Close()

	streamRepo := repository.NewPostgresStreamRepository(pool)
	channelRepo := repository.NewPostgresChannelRepository(pool)
	keyRepo := repository.NewPostgresStreamKeyRepository(pool)
	mediaRepo := repository.NewPostgresStreamMediaRepository(pool)
	uc := usecase.NewStreamUseCase(streamRepo, channelRepo, keyRepo)
	playbackUC := usecase.NewPlaybackUseCase(streamRepo, mediaRepo, cfg.HLSBaseURL)

	grpcServer := grpc.NewServer()
	streamv1.RegisterStreamServiceServer(grpcServer, grpchandler.NewStreamServer(uc, playbackUC))
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
	httpRouter.Mount("/", httphandler.NewHealthHandler(pool).Routes())
	httpServer := &http.Server{Addr: cfg.HTTPAddr, Handler: httpRouter, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Info("stream HTTP started", zap.String("addr", cfg.HTTPAddr))
		_ = httpServer.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	grpcServer.GracefulStop()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)
}
