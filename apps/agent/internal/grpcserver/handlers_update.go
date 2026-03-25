package grpcserver

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"
	"google.golang.org/grpc"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
)

const pushUpdateChunkLimit = 512 * 1024 * 1024

type pushUpdateResult struct {
	version      string
	totalSize    int64
	expectedSize int64
}

func (s *AgentService) receiveBinaryChunks(stream grpc.ClientStreamingServer[pb.UpdateBinaryChunk, pb.UpdateBinaryResponse], tmpPath string) (*pushUpdateResult, error) {
	f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	defer f.Close()

	result := &pushUpdateResult{}
	first := true

	for {
		chunk, recvErr := stream.Recv()
		if recvErr == io.EOF {
			break
		}
		if recvErr != nil {
			return nil, fmt.Errorf("receive chunk: %w", recvErr)
		}

		if first {
			result.version = chunk.Version
			result.expectedSize = chunk.TotalSize
			first = false
			s.logger.Info("receiving binary update", "version", result.version, "expectedSize", result.expectedSize)
		}

		n, writeErr := f.Write(chunk.Data)
		if writeErr != nil {
			return nil, fmt.Errorf("write chunk: %w", writeErr)
		}
		result.totalSize += int64(n)

		if result.totalSize > pushUpdateChunkLimit {
			return nil, fmt.Errorf("binary exceeds size limit")
		}
	}

	return result, nil
}

func replaceBinaryFile(execPath, tmpPath string) error {
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}
	if err := os.Rename(tmpPath, execPath); err != nil {
		return fmt.Errorf("rename new binary: %w", err)
	}
	return nil
}

func (s *AgentService) PushUpdate(stream grpc.ClientStreamingServer[pb.UpdateBinaryChunk, pb.UpdateBinaryResponse]) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	tmpPath := filepath.Join(filepath.Dir(execPath), "agent.new")

	result, err := s.receiveBinaryChunks(stream, tmpPath)
	if err != nil {
		os.Remove(tmpPath)
		return err
	}

	if result.totalSize == 0 {
		os.Remove(tmpPath)
		return fmt.Errorf("received empty binary")
	}

	if result.expectedSize > 0 && result.totalSize != result.expectedSize {
		os.Remove(tmpPath)
		return fmt.Errorf("size mismatch: expected %d, got %d", result.expectedSize, result.totalSize)
	}

	if err := replaceBinaryFile(execPath, tmpPath); err != nil {
		os.Remove(tmpPath)
		return err
	}

	s.logger.Info("binary update received, restarting", "version", result.version, "bytes", result.totalSize)

	if sendErr := stream.SendAndClose(&pb.UpdateBinaryResponse{
		Success: true,
		Message: fmt.Sprintf("update to %s received, restarting", result.version),
	}); sendErr != nil {
		return sendErr
	}

	go func() {
		time.Sleep(500 * time.Millisecond)
		_ = unix.Exec(execPath, os.Args, os.Environ())
	}()

	return nil
}
