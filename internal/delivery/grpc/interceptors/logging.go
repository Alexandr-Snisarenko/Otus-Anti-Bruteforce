package interceptors

import (
	"context"
	"time"

	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func UnaryLoggingInterceptor(log *logger.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {

		start := time.Now()

		// вызов реального handler'а
		resp, err := handler(ctx, req)

		// логирование информации о вызове
		elapsed := time.Since(start)
		st, _ := status.FromError(err)
		peerInfo, _ := peer.FromContext(ctx)
		remoteAddr := peerInfo.Addr.String()

		fields := []any{
			"method", info.FullMethod,
			"duration", elapsed.String(),
			"grpc_code", st.Code().String(),
			"remote_addr", remoteAddr,
		}

		if err != nil {
			log.ErrorContext(ctx, "gRPC request failed", append(fields, "error", err)...)
		} else {
			log.InfoContext(ctx, "gRPC request handled", fields...)
		}

		return resp, err
	}
}
