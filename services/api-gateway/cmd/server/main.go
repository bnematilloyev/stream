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
	"github.com/sahiy/sahiy-stream/pkg/httputil"
	"github.com/sahiy/sahiy-stream/pkg/logger"
	pkgredis "github.com/sahiy/sahiy-stream/pkg/redis"
	"github.com/sahiy/sahiy-stream/services/api-gateway/internal/client"
	"github.com/sahiy/sahiy-stream/services/api-gateway/internal/config"
	"github.com/sahiy/sahiy-stream/services/api-gateway/internal/handler"
	"github.com/sahiy/sahiy-stream/services/api-gateway/internal/middleware"
	"go.uber.org/zap"
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

	authHandler := handler.NewAuthHandler(authClient)
	userHandler := handler.NewUserHandler(userClient)
	channelHandler := handler.NewChannelHandler(userClient, cfg.WhipBaseURL)
	streamHandler := handler.NewStreamHandler(streamClient, cfg.WhipBaseURL)

	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(httputil.Recoverer(log))
	r.Use(httputil.RequestLogger(log))
	r.Use(chimiddleware.Timeout(30 * time.Second))
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
	r.Use(middleware.RateLimit(redisClient, cfg.RateLimitRPM))

	r.Get("/health", handler.Health)

	r.Route("/v1", func(v1 chi.Router) {
		v1.Route("/auth", func(auth chi.Router) {
			auth.Post("/register", authHandler.Register)
			auth.Post("/login", authHandler.Login)
			auth.Post("/refresh", authHandler.Refresh)
			auth.Post("/logout", authHandler.Logout)
			auth.With(middleware.Authenticate(authClient)).Get("/me", authHandler.Me)
		})

		v1.Route("/users", func(users chi.Router) {
			users.With(middleware.Authenticate(authClient)).Get("/me", userHandler.GetMe)
			users.With(middleware.Authenticate(authClient)).Patch("/me", userHandler.UpdateMe)
			users.Get("/{username}", userHandler.GetByUsername)
		})

		v1.Route("/channels", func(channels chi.Router) {
			channels.With(middleware.Authenticate(authClient)).Post("/", channelHandler.Create)
			channels.With(middleware.Authenticate(authClient)).Get("/me", channelHandler.GetMine)
			channels.Get("/{slug}", channelHandler.Get)
			channels.With(middleware.Authenticate(authClient)).Patch("/{slug}", channelHandler.Update)
			channels.With(middleware.Authenticate(authClient)).Post("/{slug}/follow", channelHandler.Follow)
			channels.With(middleware.Authenticate(authClient)).Delete("/{slug}/follow", channelHandler.Unfollow)
			channels.Get("/{slug}/followers", channelHandler.Followers)
			channels.With(middleware.Authenticate(authClient)).Get("/{slug}/ingest", channelHandler.Ingest)
			channels.With(middleware.Authenticate(authClient)).Post("/{slug}/key/rotate", channelHandler.RotateKey)
			channels.Get("/{slug}/streams", streamHandler.ListByChannel)
		})

		v1.Route("/streams", func(streams chi.Router) {
			streams.Get("/live", streamHandler.ListLive)
			streams.Get("/{id}/playback", streamHandler.Playback)
			streams.Get("/{id}", streamHandler.Get)
			streams.With(middleware.Authenticate(authClient)).Post("/", streamHandler.Create)
			streams.With(middleware.Authenticate(authClient)).Patch("/{id}", streamHandler.Update)
			streams.With(middleware.Authenticate(authClient)).Delete("/{id}", streamHandler.Delete)
			streams.With(middleware.Authenticate(authClient)).Post("/{id}/start", streamHandler.Start)
			streams.With(middleware.Authenticate(authClient)).Post("/{id}/end", streamHandler.End)
		})
	})

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
	log.Info("api gateway stopped")
}
