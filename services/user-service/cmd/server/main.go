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
	grpchandler "github.com/sahiy/sahiy-stream/services/user-service/internal/adapter/handler/grpc"
	httphandler "github.com/sahiy/sahiy-stream/services/user-service/internal/adapter/handler/http"
	"github.com/sahiy/sahiy-stream/services/user-service/internal/adapter/repository"
	"github.com/sahiy/sahiy-stream/services/user-service/internal/config"
	"github.com/sahiy/sahiy-stream/services/user-service/internal/usecase"
	userv1 "github.com/sahiy/sahiy-stream/proto/gen/user/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.Load()
	log, err := logger.New("user-service", cfg.LogLevel)
	if err != nil {
		panic(err)
	}
	defer func() { _ = log.Sync() }()

	ctx := context.Background()
	pool, err := database.NewPool(ctx, database.DefaultConfig(cfg.DatabaseURL))
	if err != nil {
		log.Fatal("database connection failed", zap.Error(err))
	}
	defer pool.Close()

	profileRepo := repository.NewPostgresProfileRepository(pool)
	channelRepo := repository.NewPostgresChannelRepository(pool)
	followerRepo := repository.NewPostgresFollowerRepository(pool)
	streamKeyRepo := repository.NewPostgresStreamKeyRepository(pool)

	userUC := usecase.NewUserUseCase(profileRepo)
	channelUC := usecase.NewChannelUseCase(channelRepo, followerRepo, streamKeyRepo, cfg.RTMPBaseURL, cfg.SRTBaseURL)

	grpcServer := grpc.NewServer()
	userv1.RegisterUserServiceServer(grpcServer, grpchandler.NewUserServer(userUC))
	userv1.RegisterChannelServiceServer(grpcServer, grpchandler.NewChannelServer(channelUC))

	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("user.v1.UserService", healthpb.HealthCheckResponse_SERVING)
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatal("grpc listen failed", zap.Error(err))
	}
	go func() {
		log.Info("user gRPC started", zap.String("addr", cfg.GRPCAddr))
		if err := grpcServer.Serve(lis); err != nil {
			log.Error("grpc serve failed", zap.Error(err))
		}
	}()

	healthHandler := httphandler.NewHealthHandler(pool)
	httpRouter := chi.NewRouter()
	httpRouter.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)
	httpRouter.Mount("/", healthHandler.Routes())
	httpServer := &http.Server{
		Addr: cfg.HTTPAddr, Handler: httpRouter,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second,
	}
	go func() {
		log.Info("user HTTP started", zap.String("addr", cfg.HTTPAddr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http serve failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutdown")
	grpcServer.GracefulStop()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
}
