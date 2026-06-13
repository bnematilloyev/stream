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
	"github.com/sahiy/sahiy-stream/pkg/crypto"
	"github.com/sahiy/sahiy-stream/pkg/database"
	"github.com/sahiy/sahiy-stream/pkg/logger"
	pkgredis "github.com/sahiy/sahiy-stream/pkg/redis"
	grpchandler "github.com/sahiy/sahiy-stream/services/auth-service/internal/adapter/handler/grpc"
	httphandler "github.com/sahiy/sahiy-stream/services/auth-service/internal/adapter/handler/http"
	"github.com/sahiy/sahiy-stream/services/auth-service/internal/adapter/repository"
	"github.com/sahiy/sahiy-stream/services/auth-service/internal/config"
	authv1 "github.com/sahiy/sahiy-stream/proto/gen/auth/v1"
	"github.com/sahiy/sahiy-stream/services/auth-service/internal/usecase"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.Load()

	log, err := logger.New("auth-service", cfg.LogLevel)
	if err != nil {
		panic(err)
	}
	defer func() { _ = log.Sync() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := database.NewPool(ctx, database.DefaultConfig(cfg.DatabaseURL))
	if err != nil {
		log.Fatal("database connection failed", zap.Error(err))
	}
	defer pool.Close()

	redisClient, err := pkgredis.NewClient(cfg.RedisURL)
	if err != nil {
		log.Fatal("redis connection failed", zap.Error(err))
	}
	defer func() { _ = redisClient.Close() }()

	jwtManager := crypto.NewJWTManager(cfg.JWTAccess, cfg.JWTRefresh, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)

	userRepo := repository.NewPostgresUserRepository(pool)
	sessionRepo := repository.NewPostgresSessionRepository(pool)
	auditRepo := repository.NewPostgresAuditRepository(pool)
	authUC := usecase.NewAuthUseCase(userRepo, sessionRepo, auditRepo, jwtManager)

	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(4*1024*1024),
		grpc.MaxSendMsgSize(4*1024*1024),
	)
	authv1.RegisterAuthServiceServer(grpcServer, grpchandler.NewAuthServer(authUC))

	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("auth.v1.AuthService", healthpb.HealthCheckResponse_SERVING)
	reflection.Register(grpcServer)

	grpcLis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatal("grpc listen failed", zap.Error(err))
	}

	go func() {
		log.Info("auth gRPC server started", zap.String("addr", cfg.GRPCAddr))
		if err := grpcServer.Serve(grpcLis); err != nil {
			log.Fatal("grpc serve failed", zap.Error(err))
		}
	}()

	healthHandler := httphandler.NewHealthHandler(pool, redisClient)
	httpRouter := chi.NewRouter()
	httpRouter.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)
	httpRouter.Mount("/", healthHandler.Routes())

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpRouter,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Info("auth HTTP health server started", zap.String("addr", cfg.HTTPAddr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http health server stopped", zap.Error(err))
		}
	}()

	waitForShutdown(log, func() {
		healthServer.SetServingStatus("auth.v1.AuthService", healthpb.HealthCheckResponse_NOT_SERVING)
		grpcServer.GracefulStop()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		_ = httpServer.Shutdown(shutdownCtx)
	})
}

func waitForShutdown(log *zap.Logger, cleanup func()) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutdown signal received")
	cleanup()
	log.Info("auth service stopped")
}
