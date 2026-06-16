package config

import (
	"strconv"
	"strings"
	"time"

	pkgconfig "github.com/sahiy/sahiy-stream/pkg/config"
)

type Config struct {
	AppEnv             string
	LogLevel           string
	HTTPAddr           string
	AuthService        string
	UserService        string
	StreamService      string
	ChatService        string
	ChatHTTPAddr       string
	RedisURL           string
	CORSOrigins        []string
	RateLimitRPM       int
	RateLimitLogin     int
	RateLimitRegister  int
	RateLimitHeartbeat int
	RequestTimeout     time.Duration
	WhipBaseURL        string
	JWTAccessSecret    string
	JWTRefreshSecret   string
	JWTAccessTTL       time.Duration
	JWTRefreshTTL      time.Duration
	UserCacheTTL       time.Duration
	MaxBodyBytes       int64
}

func parseOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if o := strings.TrimSpace(p); o != "" {
			out = append(out, o)
		}
	}
	return out
}

func intEnv(key string, fallback int) int {
	raw := pkgconfig.Get(key, "")
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

func Load() Config {
	origins := pkgconfig.Get(
		"GATEWAY_CORS_ORIGINS",
		"http://localhost:3000,http://localhost:3001,http://127.0.0.1:3000,http://127.0.0.1:3001",
	)
	maxBody, _ := strconv.ParseInt(pkgconfig.Get("GATEWAY_MAX_BODY_BYTES", "1048576"), 10, 64)
	if maxBody <= 0 {
		maxBody = 1 << 20
	}
	return Config{
		AppEnv:             pkgconfig.Get("APP_ENV", "development"),
		LogLevel:           pkgconfig.Get("LOG_LEVEL", "info"),
		HTTPAddr:           pkgconfig.Get("GATEWAY_HTTP_ADDR", ":8080"),
		AuthService:        pkgconfig.Get("AUTH_SERVICE_ADDR", "localhost:50051"),
		UserService:        pkgconfig.Get("USER_SERVICE_ADDR", "localhost:50052"),
		StreamService:      pkgconfig.Get("STREAM_SERVICE_ADDR", "localhost:50053"),
		ChatService:        pkgconfig.Get("CHAT_SERVICE_ADDR", "localhost:50054"),
		ChatHTTPAddr:       pkgconfig.Get("CHAT_HTTP_ADDR", "localhost:9085"),
		RedisURL:           pkgconfig.Get("REDIS_URL", "redis://localhost:6379/0"),
		CORSOrigins:        parseOrigins(origins),
		RateLimitRPM:       intEnv("GATEWAY_RATE_LIMIT_RPM", 100),
		RateLimitLogin:     intEnv("GATEWAY_RATE_LIMIT_LOGIN", 5),
		RateLimitRegister:  intEnv("GATEWAY_RATE_LIMIT_REGISTER", 3),
		RateLimitHeartbeat: intEnv("GATEWAY_RATE_LIMIT_HEARTBEAT", 6000),
		RequestTimeout:     10 * time.Second,
		WhipBaseURL:        pkgconfig.Get("WHIP_BASE_URL", "http://localhost:8889"),
		JWTAccessSecret:    pkgconfig.Get("JWT_ACCESS_SECRET", "dev-access-secret-change-in-production-32"),
		JWTRefreshSecret:   pkgconfig.Get("JWT_REFRESH_SECRET", "dev-refresh-secret-change-in-production-32"),
		JWTAccessTTL:       pkgconfig.Duration("JWT_ACCESS_TTL", 15*time.Minute),
		JWTRefreshTTL:      pkgconfig.Duration("JWT_REFRESH_TTL", 168*time.Hour),
		UserCacheTTL:       pkgconfig.Duration("AUTH_USER_CACHE_TTL", 5*time.Minute),
		MaxBodyBytes:       maxBody,
	}
}

func (c Config) RateLimitRules() map[string]int {
	return map[string]int{
		"/v1/auth/login":          c.RateLimitLogin,
		"/v1/auth/register":       c.RateLimitRegister,
		"/v1/streams/*/heartbeat": c.RateLimitHeartbeat,
	}
}
