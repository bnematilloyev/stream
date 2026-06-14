package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/sahiy/sahiy-stream/pkg/health"
	"google.golang.org/grpc"
)

const serviceName = "api-gateway"

// HealthHandler serves liveness and readiness probes.
type HealthHandler struct {
	checkers []health.Checker
}

// NewHealthHandler builds readiness checkers from gateway dependencies.
func NewHealthHandler(redisClient *redis.Client, conns map[string]*grpc.ClientConn) *HealthHandler {
	checkers := []health.Checker{
		health.RedisChecker{Client: redisClient},
	}
	for name, conn := range conns {
		checkers = append(checkers, health.GRPCChecker{ServiceName: name, Conn: conn})
	}
	return &HealthHandler{checkers: checkers}
}

// NewHealthHandlerWithPool optionally includes postgres (for future gateway DB use).
func NewHealthHandlerWithPool(pool *pgxpool.Pool, redisClient *redis.Client, conns map[string]*grpc.ClientConn) *HealthHandler {
	h := NewHealthHandler(redisClient, conns)
	if pool != nil {
		h.checkers = append([]health.Checker{health.PostgresChecker{Pool: pool}}, h.checkers...)
	}
	return h
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
