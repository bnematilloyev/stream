package http

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/sahiy/sahiy-stream/pkg/health"
	pkgnats "github.com/sahiy/sahiy-stream/pkg/nats"
)

const serviceName = "chat-service"

type HealthHandler struct {
	checkers []health.Checker
}

func NewHealthHandler(pool *pgxpool.Pool, redisClient *redis.Client, bus *pkgnats.ChatBus) *HealthHandler {
	checkers := []health.Checker{
		health.PostgresChecker{Pool: pool},
		health.RedisChecker{Client: redisClient},
	}
	if bus != nil {
		checkers = append(checkers, natsChecker{bus: bus})
	}
	return &HealthHandler{checkers: checkers}
}

type natsChecker struct{ bus *pkgnats.ChatBus }

func (c natsChecker) Name() string { return "nats" }

func (c natsChecker) Check(_ context.Context) error {
	return c.bus.Ping()
}

func (h *HealthHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		health.Liveness(w, serviceName)
	})
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		health.Readiness(w, r, h.checkers...)
	})
	return r
}
