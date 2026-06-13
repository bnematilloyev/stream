module github.com/sahiy/sahiy-stream/services/api-gateway

go 1.22.7

require (
	github.com/go-chi/chi/v5 v5.1.0
	github.com/go-chi/cors v1.2.1
	github.com/redis/go-redis/v9 v9.7.0
	github.com/sahiy/sahiy-stream/pkg v0.0.0
	github.com/sahiy/sahiy-stream/proto v0.0.0
	go.uber.org/zap v1.27.0
	google.golang.org/grpc v1.68.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/net v0.29.0 // indirect
	golang.org/x/sys v0.25.0 // indirect
	golang.org/x/text v0.18.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240903143218-8af14fe29dc1 // indirect
	google.golang.org/protobuf v1.35.1 // indirect
)

replace (
	github.com/sahiy/sahiy-stream/pkg => ../../pkg
	github.com/sahiy/sahiy-stream/proto => ../../proto
)
