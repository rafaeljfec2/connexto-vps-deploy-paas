package agentclient

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "plain", err: errors.New("boom"), want: false},
		{name: "unavailable", err: status.Error(codes.Unavailable, "transient"), want: true},
		{name: "canceled closing", err: status.Error(codes.Canceled, "grpc: the client connection is closing"), want: true},
		{name: "canceled other", err: status.Error(codes.Canceled, "context canceled"), want: false},
		{name: "internal", err: status.Error(codes.Internal, "x"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetryable(tt.err); got != tt.want {
				t.Fatalf("isRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRetryUnaryInterceptor_SkipsNonRetryableMethods(t *testing.T) {
	if len(nonRetryableMethods) == 0 {
		t.Fatal("nonRetryableMethods must not be empty")
	}
	for method := range nonRetryableMethods {
		t.Run(method, func(t *testing.T) {
			var calls atomic.Int32
			interceptor := retryUnaryInterceptor()
			err := interceptor(
				context.Background(),
				method,
				nil, nil, nil,
				func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
					calls.Add(1)
					return status.Error(codes.Unavailable, "down")
				},
			)
			if err == nil {
				t.Fatal("expected error")
			}
			if calls.Load() != 1 {
				t.Fatalf("%s must not retry, calls=%d", method, calls.Load())
			}
		})
	}
}

func TestRetryUnaryInterceptor_RetriesUnavailable(t *testing.T) {
	var calls atomic.Int32
	interceptor := retryUnaryInterceptor()
	err := interceptor(
		context.Background(),
		pb.AgentService_ListContainers_FullMethodName,
		nil, nil, nil,
		func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			n := calls.Add(1)
			if n <= 2 {
				return status.Error(codes.Unavailable, "down")
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("expected success after retries: %v", err)
	}
	if calls.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", calls.Load())
	}
}

func TestRetryUnaryInterceptor_RetriesConnectionClosing(t *testing.T) {
	var calls atomic.Int32
	interceptor := retryUnaryInterceptor()
	err := interceptor(
		context.Background(),
		pb.AgentService_GetSystemInfo_FullMethodName,
		nil, nil, nil,
		func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			n := calls.Add(1)
			if n == 1 {
				return status.Error(codes.Canceled, "grpc: the client connection is closing")
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("expected success after retry: %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected 2 attempts, got %d", calls.Load())
	}
}
