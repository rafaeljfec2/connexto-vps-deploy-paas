package agentclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	deployLogStreamReadyWait = 5 * time.Second
	deployLogStreamRetryWait = 1 * time.Second
)

type DeployLogHandler func(entry *pb.DeployLogEntry)

type logEntryReceiver interface {
	Recv() (*pb.DeployLogEntry, error)
}

type logStreamOpener func(ctx context.Context, deploymentID string) (logEntryReceiver, error)

func (c *AgentClient) ExecuteDeployWithLogs(
	ctx context.Context, host string, port int,
	req *pb.DeployRequest, onLog DeployLogHandler,
) (*pb.DeployResponse, error) {
	cl, release, err := c.client(host, port)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent: %w", err)
	}
	defer release()

	if onLog != nil {
		ready := make(chan struct{})
		streamCl, streamRelease, streamErr := c.client(host, port)
		if streamErr != nil {
			return nil, fmt.Errorf("failed to connect to agent for logs: %w", streamErr)
		}
		go func() {
			defer streamRelease()
			open := func(streamCtx context.Context, deploymentID string) (logEntryReceiver, error) {
				return streamCl.StreamDeployLogs(streamCtx, &pb.DeployLogSubscription{DeploymentId: deploymentID})
			}
			streamDeployLogs(ctx, req.DeploymentId, open, onLog, ready, deployLogStreamRetryWait)
		}()
		select {
		case <-ready:
		case <-time.After(deployLogStreamReadyWait):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	// WaitForReady lets the call park while the pooled conn is in
	// TransientFailure instead of fail-fast Unavailable; bounded by ctx.
	return cl.ExecuteDeploy(ctx, req, grpc.WaitForReady(true))
}

// streamDeployLogs keeps StreamDeployLogs attached for the lifetime of ctx.
// Ready is signalled once the first open attempt finishes (success or failure)
// so ExecuteDeploy is not blocked on log availability. On Recv/open errors
// (except clean EOF), it emits a notice and reattaches until ctx ends.
func streamDeployLogs(
	ctx context.Context,
	deploymentID string,
	open logStreamOpener,
	onLog DeployLogHandler,
	ready chan<- struct{},
	retryWait time.Duration,
) {
	var readyOnce sync.Once
	signalReady := func() {
		readyOnce.Do(func() { close(ready) })
	}
	defer signalReady()

	openedOK := false
	for {
		if ctx.Err() != nil {
			return
		}

		stream, err := open(ctx, deploymentID)
		if err != nil {
			if !openedOK {
				emitStreamNotice(onLog, deploymentID, "Deploy log stream failed to open, retrying...")
				signalReady()
			} else {
				emitStreamNotice(onLog, deploymentID, "Deploy log stream interrupted, reconnecting...")
			}
			if !sleepCtx(ctx, retryWait) {
				return
			}
			continue
		}

		if !openedOK {
			signalReady()
			openedOK = true
		} else {
			emitStreamNotice(onLog, deploymentID, "Deploy log stream reconnected")
		}

		recvErr := drainLogStream(stream, onLog)
		if ctx.Err() != nil {
			return
		}
		if errors.Is(recvErr, io.EOF) {
			return
		}
		emitStreamNotice(onLog, deploymentID, "Deploy log stream interrupted, reconnecting...")
		if !sleepCtx(ctx, retryWait) {
			return
		}
	}
}

func drainLogStream(stream logEntryReceiver, onLog DeployLogHandler) error {
	for {
		entry, err := stream.Recv()
		if err != nil {
			return err
		}
		if onLog != nil {
			onLog(entry)
		}
	}
}

func emitStreamNotice(onLog DeployLogHandler, deploymentID, message string) {
	slog.Warn(message, "deploymentId", deploymentID)
	if onLog == nil {
		return
	}
	onLog(&pb.DeployLogEntry{
		DeploymentId: deploymentID,
		Timestamp:    timestamppb.Now(),
		Level:        pb.DeployLogLevel_DEPLOY_LOG_LEVEL_WARN,
		Stage:        pb.DeployStage_DEPLOY_STAGE_UNSPECIFIED,
		Message:      message,
	})
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (c *AgentClient) ExecuteDeploy(ctx context.Context, host string, port int, req *pb.DeployRequest) (*pb.DeployResponse, error) {
	return c.ExecuteDeployWithLogs(ctx, host, port, req, nil)
}
