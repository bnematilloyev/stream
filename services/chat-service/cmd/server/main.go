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
	"github.com/redis/go-redis/v9"
	"github.com/sahiy/sahiy-stream/pkg/auth"
	"github.com/sahiy/sahiy-stream/pkg/crypto"
	"github.com/sahiy/sahiy-stream/pkg/database"
	"github.com/sahiy/sahiy-stream/pkg/grpcserver"
	"github.com/sahiy/sahiy-stream/pkg/logger"
	pkgnats "github.com/sahiy/sahiy-stream/pkg/nats"
	pkgredis "github.com/sahiy/sahiy-stream/pkg/redis"
	chatv1 "github.com/sahiy/sahiy-stream/proto/gen/chat/v1"
	authadapter "github.com/sahiy/sahiy-stream/services/chat-service/internal/adapter/auth"
	streamclient "github.com/sahiy/sahiy-stream/services/chat-service/internal/adapter/client"
	grpchandler "github.com/sahiy/sahiy-stream/services/chat-service/internal/adapter/handler/grpc"
	httphandler "github.com/sahiy/sahiy-stream/services/chat-service/internal/adapter/handler/http"
	wshandler "github.com/sahiy/sahiy-stream/services/chat-service/internal/adapter/handler/ws"
	"github.com/sahiy/sahiy-stream/services/chat-service/internal/adapter/repository"
	"github.com/sahiy/sahiy-stream/services/chat-service/internal/config"
	"github.com/sahiy/sahiy-stream/services/chat-service/internal/moderation"
	"github.com/sahiy/sahiy-stream/services/chat-service/internal/usecase"
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

	log, err := logger.New("chat-service", cfg.LogLevel)
	if err != nil {
		panic(err)
	}
	defer func() { _ = log.Sync() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := database.NewPool(ctx, database.DefaultConfig(cfg.DatabaseURL))
	if err != nil {
		log.Fatal("database failed", zap.Error(err))
	}
	defer pool.Close()

	redisClient, err := pkgredis.NewClient(cfg.Redis.URL)
	if err != nil {
		log.Fatal("redis failed", zap.Error(err))
	}
	defer func() { _ = redisClient.Close() }()

	bus, err := pkgnats.NewChatBus(pkgnats.DefaultConfig(cfg.NATSURL))
	if err != nil {
		log.Fatal("nats failed", zap.Error(err))
	}
	defer bus.Close()

	authClient, err := streamclient.NewAuthClient(cfg.AuthServiceAddr)
	if err != nil {
		log.Fatal("auth client failed", zap.Error(err))
	}
	defer authClient.Close()

	userClient, err := streamclient.NewUserClient(cfg.UserServiceAddr)
	if err != nil {
		log.Fatal("user client failed", zap.Error(err))
	}
	defer userClient.Close()

	streamClient, err := streamclient.NewStreamClient(cfg.StreamServiceAddr)
	if err != nil {
		log.Fatal("stream client failed", zap.Error(err))
	}
	defer streamClient.Close()

	chatRepo := repository.NewPostgresChatRepository(pool)
	limiter := moderation.NewRateLimiter(redisClient, cfg.ChatRateLimit)
	moderator := streamclient.NewStreamModerator(streamClient, userClient)
	chatUC := usecase.NewChatUseCase(chatRepo, streamClient, moderator, bus, limiter)

	validator := newValidator(cfg, redisClient, authadapter.NewGRPCUserFetcher(authClient))

	hub := wshandler.NewHub()
	wsHandler := wshandler.NewHandler(chatUC, hub, validator, cfg.CORSOrigins, cfg.AppEnv, log)

	if _, err := bus.Subscribe(func(streamID string, payload []byte) {
		wsHandler.Broadcast(streamID, payload)
	}); err != nil {
		log.Fatal("nats subscribe failed", zap.Error(err))
	}

	grpcServer := grpc.NewServer(grpcserver.DefaultServerOptions(log, cfg.GRPCRequestTimeout)...)
	chatv1.RegisterChatServiceServer(grpcServer, grpchandler.NewChatServer(chatUC))
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatal("listen failed", zap.Error(err))
	}
	go func() {
		log.Info("chat gRPC started", zap.String("addr", cfg.GRPCAddr))
		_ = grpcServer.Serve(lis)
	}()

	httpRouter := chi.NewRouter()
	httpRouter.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)
	httpRouter.Mount("/", httphandler.NewHealthHandler(pool, redisClient, bus).Routes())
	httpRouter.Mount("/v1/chat", wsHandler.Routes())
	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpRouter,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       0,
		WriteTimeout:      0,
		IdleTimeout:       120 * time.Second,
	}
	go func() {
		log.Info("chat HTTP/WS started", zap.String("addr", cfg.HTTPAddr))
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

func newValidator(cfg config.Config, redisClient *redis.Client, fetcher auth.UserFetcher) *auth.Validator {
	jwtManager := crypto.NewJWTManager(cfg.JWTAccessSecret, cfg.JWTRefreshSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	sessionCache := auth.NewSessionCache(redisClient, cfg.UserCacheTTL)
	return auth.NewValidator(jwtManager, sessionCache, fetcher)
}
