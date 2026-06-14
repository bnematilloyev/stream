package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/sahiy/sahiy-stream/pkg/health"
)

const serviceName = "auth-service"

type HealthHandler struct {
	checkers []health.Checker
}

func NewHealthHandler(pool *pgxpool.Pool, redisClient *redis.Client) *HealthHandler {
	return &HealthHandler{
		checkers: []health.Checker{
			health.PostgresChecker{Pool: pool},
			health.RedisChecker{Client: redisClient},
		},
	}
}

func (h *HealthHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/health", h.liveness)
	r.Get("/ready", h.readiness)
	return r
}

func (h *HealthHandler) liveness(w http.ResponseWriter, _ *http.Request) {
	health.Liveness(w, serviceName)
}

func (h *HealthHandler) readiness(w http.ResponseWriter, r *http.Request) {
	health.Readiness(w, r, h.checkers...)
}
