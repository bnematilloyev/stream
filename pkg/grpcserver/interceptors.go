package grpcserver

import (
	"context"
	"runtime/debug"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const defaultTimeout = 10 * time.Second

// DefaultUnaryInterceptors returns production-safe gRPC unary interceptors.
func DefaultUnaryInterceptors(log *zap.Logger, timeout time.Duration) []grpc.UnaryServerInterceptor {
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return []grpc.UnaryServerInterceptor{
		unaryRecovery(log),
		unaryTimeout(timeout),
		unaryLogging(log),
	}
}

func DefaultServerOptions(log *zap.Logger, timeout time.Duration) []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(DefaultUnaryInterceptors(log, timeout)...),
		grpc.MaxRecvMsgSize(4 * 1024 * 1024),
		grpc.MaxSendMsgSize(4 * 1024 * 1024),
	}
}

func unaryRecovery(log *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error("grpc panic recovered",
					zap.Any("panic", rec),
					zap.String("method", info.FullMethod),
					zap.ByteString("stack", debug.Stack()),
				)
				err = status.Error(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

func unaryTimeout(timeout time.Duration) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return handler(ctx, req)
	}
}

func unaryLogging(log *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		fields := []zap.Field{
			zap.String("method", info.FullMethod),
			zap.Duration("duration", time.Since(start)),
		}
		if err != nil {
			fields = append(fields, zap.Error(err))
			log.Warn("grpc request", fields...)
			return resp, err
		}
		log.Debug("grpc request", fields...)
		return resp, nil
	}
}
