package interceptors

import (
	"bytes"
	"context"
	"net"
	"strings"
	"testing"

	internalcfg "github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/config"
	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func TestUnaryLoggingInterceptor_Success(t *testing.T) {
	var buf bytes.Buffer
	log := logger.NewWithWriter(&buf, &internalcfg.Logger{Level: "debug"})

	interceptor := UnaryLoggingInterceptor(log)

	ctx := peer.NewContext(
		context.Background(), &peer.Peer{Addr: &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 12345}})
	info := &grpc.UnaryServerInfo{FullMethod: "/svc/method"}
	handler := func(_ context.Context, _ any) (any, error) {
		return "ok", nil
	}

	resp, err := interceptor(ctx, nil, info, handler)
	if err != nil {
		t.Fatalf("unexpected error from interceptor: %v", err)
	}
	if resp != "ok" {
		t.Fatalf("unexpected response: %v", resp)
	}

	out := buf.String()
	if !strings.Contains(out, "gRPC request handled") {
		t.Fatalf("expected handled log, got: %s", out)
	}
	if !strings.Contains(out, info.FullMethod) {
		t.Fatalf("expected method in log: %s", out)
	}
	if !strings.Contains(out, "grpc_code") || !strings.Contains(out, "OK") {
		t.Fatalf("expected grpc_code OK in log: %s", out)
	}
	if !strings.Contains(out, "remote_addr") || !strings.Contains(out, "192.168.1.1") {
		t.Fatalf("expected remote_addr in log: %s", out)
	}
	if !strings.Contains(out, "duration") {
		t.Fatalf("expected duration in log: %s", out)
	}
}

func TestUnaryLoggingInterceptor_Error(t *testing.T) {
	var buf bytes.Buffer
	log := logger.NewWithWriter(&buf, &internalcfg.Logger{Level: "debug"})

	interceptor := UnaryLoggingInterceptor(log)

	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: &net.TCPAddr{IP: net.ParseIP("10.0.0.1"), Port: 9999}})
	info := &grpc.UnaryServerInfo{FullMethod: "/svc/other"}
	handler := func(_ context.Context, _ any) (any, error) {
		return nil, status.Error(codes.Unauthenticated, "no auth")
	}

	resp, err := interceptor(ctx, nil, info, handler)
	if err == nil {
		t.Fatalf("expected error from interceptor, got resp=%v", resp)
	}

	out := buf.String()
	if !strings.Contains(out, "gRPC request failed") {
		t.Fatalf("expected failed log, got: %s", out)
	}
	if !strings.Contains(out, info.FullMethod) {
		t.Fatalf("expected method in log: %s", out)
	}
	if !strings.Contains(out, "grpc_code") || !strings.Contains(out, "Unauthenticated") {
		t.Fatalf("expected grpc_code Unauthenticated in log: %s", out)
	}
	if !strings.Contains(out, "error") || !strings.Contains(out, "no auth") {
		t.Fatalf("expected error details in log: %s", out)
	}
}
