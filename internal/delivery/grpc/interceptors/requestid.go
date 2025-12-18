package interceptors

import (
	"context"

	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/ctxmeta"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const requestIDHeader = "x-request-id"

func UnaryRequestIDInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		var rid string

		// 1) пробуем взять из metadata
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if vals := md.Get(requestIDHeader); len(vals) > 0 && vals[0] != "" {
				rid = vals[0]
			}
		}

		// 2) если нет — генерим
		if rid == "" {
			rid = uuid.NewString()
		}

		// 3) кладём в ctx
		ctx = ctxmeta.WithRequestID(ctx, rid)

		return handler(ctx, req)
	}
}
