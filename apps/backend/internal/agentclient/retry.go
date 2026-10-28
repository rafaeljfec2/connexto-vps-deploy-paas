package agentclient

import (
	"context"
	"strings"
	"time"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	retryMaxAttempts = 2
	retryBaseDelay   = 100 * time.Millisecond
)

// nonRetryableMethods must not be automatically re-invoked: a successful
// partial execution on the agent (e.g. mid-build, container create/remove)
// would duplicate work or double-apply a mutation when the response is lost.
var nonRetryableMethods = map[string]struct{}{
	pb.AgentService_ExecuteDeploy_FullMethodName:               {},
	pb.AgentService_RestartContainer_FullMethodName:            {},
	pb.AgentService_StopContainer_FullMethodName:               {},
	pb.AgentService_StartContainer_FullMethodName:              {},
	pb.AgentService_RemoveContainer_FullMethodName:             {},
	pb.AgentService_RemoveImage_FullMethodName:                 {},
	pb.AgentService_PruneImages_FullMethodName:                 {},
	pb.AgentService_PruneContainers_FullMethodName:             {},
	pb.AgentService_PruneVolumes_FullMethodName:                {},
	pb.AgentService_CreateNetwork_FullMethodName:               {},
	pb.AgentService_RemoveNetwork_FullMethodName:               {},
	pb.AgentService_ConnectContainerNetwork_FullMethodName:     {},
	pb.AgentService_DisconnectContainerNetwork_FullMethodName:  {},
	pb.AgentService_CreateVolume_FullMethodName:                {},
	pb.AgentService_RemoveVolume_FullMethodName:                {},
	pb.AgentService_UpdateDomains_FullMethodName:               {},
	pb.AgentService_CreateContainerFromTemplate_FullMethodName: {},
	pb.AgentService_ConfigureContainerSSL_FullMethodName:       {},
}

func retryUnaryInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if _, skip := nonRetryableMethods[method]; skip {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		var lastErr error
		for attempt := 0; attempt <= retryMaxAttempts; attempt++ {
			lastErr = invoker(ctx, method, req, reply, cc, opts...)
			if lastErr == nil {
				return nil
			}

			if !isRetryable(lastErr) {
				return lastErr
			}

			if ctx.Err() != nil {
				return lastErr
			}

			delay := retryBaseDelay * time.Duration(attempt+1)
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return lastErr
			case <-timer.C:
			}
		}
		return lastErr
	}
}

func isRetryable(err error) bool {
	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	switch st.Code() {
	case codes.Unavailable:
		return true
	case codes.Canceled:
		return strings.Contains(st.Message(), "client connection is closing")
	default:
		return false
	}
}
