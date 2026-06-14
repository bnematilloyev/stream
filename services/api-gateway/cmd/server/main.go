package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/sahiy/sahiy-stream/pkg/auth"
	"github.com/sahiy/sahiy-stream/pkg/crypto"
	"github.com/sahiy/sahiy-stream/pkg/httputil"
	"github.com/sahiy/sahiy-stream/pkg/logger"
	"github.com/sahiy/sahiy-stream/pkg/metrics"
	pkgredis "github.com/sahiy/sahiy-stream/pkg/redis"
	authadapter "github.com/sahiy/sahiy-stream/services/api-gateway/internal/auth"
	"github.com/sahiy/sahiy-stream/services/api-gateway/internal/client"
	"github.com/sahiy/sahiy-stream/services/api-gateway/internal/config"
	"github.com/sahiy/sahiy-stream/services/api-gateway/internal/handler"
	"github.com/sahiy/sahiy-stream/services/api-gateway/internal/middleware"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.Load()

	log, err := logger.New("api-gateway", cfg.LogLevel)
	if err != nil {
		panic(err)
	}
	defer func() { _ = log.Sync() }()

	redisClient, err := pkgredis.NewClient(cfg.RedisURL)
	if err != nil {
		log.Fatal("redis connection failed", zap.Error(err))
	}
	defer func() { _ = redisClient.Close() }()

	authClient, err := client.NewAuthClient(cfg.AuthService)
	if err != nil {
		log.Fatal("auth service connection failed", zap.Error(err))
	}
	defer func() { _ = authClient.Close() }()

	userClient, err := client.NewUserClient(cfg.UserService)
	if err != nil {
		log.Fatal("user service connection failed", zap.Error(err))
	}
	defer func() { _ = userClient.Close() }()

	streamClient, err := client.NewStreamClient(cfg.StreamService)
	if err != nil {
		log.Fatal("stream service connection failed", zap.Error(err))
	}
	defer func() { _ = streamClient.Close() }()

	chatClient, err := client.NewChatClient(cfg.ChatService)
	if err != nil {
		log.Fatal("chat service connection failed", zap.Error(err))
	}
	defer func() { _ = chatClient.Close() }()

	jwtManager := crypto.NewJWTManager(cfg.JWTAccessSecret, cfg.JWTRefreshSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	sessionCache := auth.NewSessionCache(redisClient, cfg.UserCacheTTL)
	tokenValidator := auth.NewValidator(jwtManager, sessionCache, authadapter.NewGRPCUserFetcher(authClient))

	authHandler := handler.NewAuthHandler(authClient)
	userHandler := handler.NewUserHandler(userClient)
	channelHandler := handler.NewChannelHandler(userClient, cfg.WhipBaseURL)
	streamHandler := handler.NewStreamHandler(streamClient, cfg.WhipBaseURL)
	chatHandler, err := handler.NewChatHandler(chatClient, cfg.ChatHTTPAddr)
	if err != nil {
		log.Fatal("chat handler init failed", zap.Error(err))
	}

	healthHandler := handler.NewHealthHandler(redisClient, map[string]*grpc.ClientConn{
		"auth-service":   authClient.Conn(),
		"user-service":   userClient.Conn(),
		"stream-service": streamClient.Conn(),
		"chat-service":   chatClient.Conn(),
	})

	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(httputil.Recoverer(log))
	r.Use(metrics.Middleware("api-gateway"))
	r.Use(httputil.RequestLogger(log))
	corsOpts := cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}
	if cfg.AppEnv == "development" {
		corsOpts.AllowOriginFunc = func(_ *http.Request, origin string) bool {
			return strings.HasPrefix(origin, "http://localhost:") ||
				strings.HasPrefix(origin, "http://127.0.0.1:")
		}
	}
	r.Use(cors.Handler(corsOpts))

	r.Mount("/", healthHandler.Routes())
	r.Handle("/metrics", metrics.Handler())

	r.Group(func(api chi.Router) {
		api.Use(middleware.RateLimitRules(redisClient, cfg.RateLimitRPM, cfg.RateLimitRules()))
		api.Use(httputil.MaxBody(cfg.MaxBodyBytes))
		api.Route("/v1", func(v1 chi.Router) {
			// WebSocket chat — no request timeout (long-lived connections).
			v1.Route("/chat", func(chat chi.Router) {
				chat.Get("/{streamID}/history", chatHandler.History)
				chat.Get("/{streamID}", chatHandler.WebSocket)
				chat.With(middleware.Authenticate(tokenValidator)).Delete("/{streamID}/messages/{messageID}", chatHandler.DeleteMessage)
			})

			v1.Group(func(rest chi.Router) {
				rest.Use(chimiddleware.Timeout(30 * time.Second))

				rest.Route("/auth", func(authRoutes chi.Router) {
					authRoutes.Post("/register", authHandler.Register)
					authRoutes.Post("/login", authHandler.Login)
					authRoutes.Post("/refresh", authHandler.Refresh)
					authRoutes.Post("/logout", authHandler.Logout)
					authRoutes.With(middleware.Authenticate(tokenValidator)).Get("/me", authHandler.Me)
				})

				rest.Route("/users", func(users chi.Router) {
					users.With(middleware.Authenticate(tokenValidator)).Get("/me", userHandler.GetMe)
					users.With(middleware.Authenticate(tokenValidator)).Patch("/me", userHandler.UpdateMe)
					users.Get("/{username}", userHandler.GetByUsername)
				})

				rest.Route("/channels", func(channels chi.Router) {
					channels.With(middleware.Authenticate(tokenValidator)).Post("/", channelHandler.Create)
					channels.With(middleware.Authenticate(tokenValidator)).Get("/me", channelHandler.GetMine)
					channels.Get("/{slug}", channelHandler.Get)
					channels.With(middleware.Authenticate(tokenValidator)).Patch("/{slug}", channelHandler.Update)
					channels.With(middleware.Authenticate(tokenValidator)).Post("/{slug}/follow", channelHandler.Follow)
					channels.With(middleware.Authenticate(tokenValidator)).Delete("/{slug}/follow", channelHandler.Unfollow)
					channels.Get("/{slug}/followers", channelHandler.Followers)
					channels.With(middleware.Authenticate(tokenValidator)).Get("/{slug}/ingest", channelHandler.Ingest)
					channels.With(middleware.Authenticate(tokenValidator)).Post("/{slug}/key/rotate", channelHandler.RotateKey)
					channels.Get("/{slug}/streams", streamHandler.ListByChannel)
				})

				rest.Route("/streams", func(streams chi.Router) {
					streams.Get("/live", streamHandler.ListLive)
					streams.Get("/{id}/playback", streamHandler.Playback)
					streams.Get("/{id}", streamHandler.Get)
					streams.With(middleware.Authenticate(tokenValidator)).Post("/", streamHandler.Create)
					streams.With(middleware.Authenticate(tokenValidator)).Patch("/{id}", streamHandler.Update)
					streams.With(middleware.Authenticate(tokenValidator)).Delete("/{id}", streamHandler.Delete)
					streams.With(middleware.Authenticate(tokenValidator)).Post("/{id}/start", streamHandler.Start)
					streams.With(middleware.Authenticate(tokenValidator)).Post("/{id}/end", streamHandler.End)
					streams.Post("/{id}/heartbeat", streamHandler.Heartbeat)
					streams.Get("/{id}/viewers", streamHandler.ViewerStats)
				})
			})
		})
	})

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       0,
		WriteTimeout:      0,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Info("api gateway started", zap.String("addr", cfg.HTTPAddr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutdown signal received")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
	log.Info("api gateway stopped")
}
