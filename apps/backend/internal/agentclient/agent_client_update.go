package agentclient

import (
	"context"
	"fmt"
	"io"
	"os"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
)

const pushUpdateChunkSize = 256 * 1024

func (c *AgentClient) PushUpdate(ctx context.Context, host string, port int, binaryPath, version string) error {
	f, err := os.Open(binaryPath)
	if err != nil {
		return fmt.Errorf("open binary: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat binary: %w", err)
	}

	cl, err := c.client(host, port)
	if err != nil {
		return fmt.Errorf("push update dial: %w", err)
	}

	stream, err := cl.PushUpdate(ctx)
	if err != nil {
		return fmt.Errorf("push update stream: %w", err)
	}

	sendFailed, err := streamBinaryChunks(f, stream, version, info.Size())
	if err != nil {
		return err
	}

	return closePushUpdateStream(stream, sendFailed)
}

func streamBinaryChunks(
	f *os.File,
	stream pb.AgentService_PushUpdateClient,
	version string,
	totalSize int64,
) (bool, error) {
	buf := make([]byte, pushUpdateChunkSize)
	first := true

	for {
		n, readErr := f.Read(buf)
		if n > 0 {
			chunk := &pb.UpdateBinaryChunk{Data: buf[:n]}
			if first {
				chunk.Version = version
				chunk.TotalSize = totalSize
				first = false
			}
			if failed, err := sendChunk(stream, chunk); err != nil {
				return false, err
			} else if failed {
				return true, nil
			}
		}
		if readErr == io.EOF {
			return false, nil
		}
		if readErr != nil {
			return false, fmt.Errorf("read binary: %w", readErr)
		}
	}
}

func sendChunk(stream pb.AgentService_PushUpdateClient, chunk *pb.UpdateBinaryChunk) (eofReceived bool, err error) {
	sendErr := stream.Send(chunk)
	if sendErr == nil {
		return false, nil
	}
	if sendErr == io.EOF {
		return true, nil
	}
	return false, fmt.Errorf("send chunk: %w", sendErr)
}

func closePushUpdateStream(stream pb.AgentService_PushUpdateClient, sendFailed bool) error {
	resp, err := stream.CloseAndRecv()
	if err != nil {
		if sendFailed {
			return fmt.Errorf("agent closed stream early (agent may not support gRPC push — try HTTPS mode): %w", err)
		}
		return fmt.Errorf("close stream: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("agent rejected update: %s", resp.Message)
	}
	return nil
}
