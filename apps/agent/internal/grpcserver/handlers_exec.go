package grpcserver

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"sync"

	"github.com/creack/pty"
	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
)

const execPTYBufSize = 4096

type execSession struct {
	stream pb.AgentService_ExecContainerServer
	ptmx   *os.File
	sendMu sync.Mutex
	done   chan struct{}
	logger *slog.Logger
}

func (es *execSession) sendOutput(data []byte) error {
	es.sendMu.Lock()
	defer es.sendMu.Unlock()
	return es.stream.Send(&pb.ExecOutput{
		Payload: &pb.ExecOutput_Data{Data: data},
	})
}

func (es *execSession) sendExitCode(code int) {
	es.sendMu.Lock()
	defer es.sendMu.Unlock()
	_ = es.stream.Send(&pb.ExecOutput{
		Payload: &pb.ExecOutput_ExitCode{ExitCode: int32(code)},
	})
}

func (es *execSession) readLoop(wg *sync.WaitGroup) {
	defer wg.Done()
	buf := make([]byte, execPTYBufSize)
	for {
		select {
		case <-es.done:
			return
		default:
		}
		n, readErr := es.ptmx.Read(buf)
		if n > 0 {
			out := make([]byte, n)
			copy(out, buf[:n])
			if sendErr := es.sendOutput(out); sendErr != nil {
				return
			}
		}
		if readErr != nil {
			if readErr != io.EOF {
				es.logger.Debug("exec: pty read error", "error", readErr)
			}
			return
		}
	}
}

func (es *execSession) writeLoop(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		in, recvErr := es.stream.Recv()
		if recvErr != nil {
			return
		}
		es.handleInput(in)
	}
}

func (es *execSession) handleInput(in *pb.ExecInput) {
	switch p := in.Payload.(type) {
	case *pb.ExecInput_Data:
		_, _ = es.ptmx.Write(p.Data)
	case *pb.ExecInput_Resize:
		if p.Resize.Cols > 0 && p.Resize.Rows > 0 {
			_ = pty.Setsize(es.ptmx, &pty.Winsize{
				Cols: uint16(p.Resize.Cols),
				Rows: uint16(p.Resize.Rows),
			})
		}
	}
}

var (
	containerIDRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.\-]*$`)
	allowedShells    = map[string]bool{"sh": true, "bash": true, "ash": true, "zsh": true}
)

const (
	defaultShell = "sh"
	defaultCols  = 80
	defaultRows  = 24
)

func parseExecStartRequest(stream pb.AgentService_ExecContainerServer) (containerID, shell string, cols, rows uint16, err error) {
	msg, err := stream.Recv()
	if err != nil {
		return "", "", 0, 0, fmt.Errorf("exec: failed to receive start message: %w", err)
	}
	startReq := msg.GetStart()
	if startReq == nil {
		return "", "", 0, 0, fmt.Errorf("exec: first message must be ExecStartRequest")
	}
	containerID = startReq.ContainerId
	if !containerIDRegex.MatchString(containerID) {
		return "", "", 0, 0, fmt.Errorf("exec: invalid container ID")
	}
	shell = startReq.Shell
	if shell == "" {
		shell = defaultShell
	}
	if !allowedShells[shell] {
		return "", "", 0, 0, fmt.Errorf("exec: unsupported shell %q", shell)
	}
	cols = uint16(startReq.Cols)
	rows = uint16(startReq.Rows)
	if cols == 0 {
		cols = defaultCols
	}
	if rows == 0 {
		rows = defaultRows
	}
	return containerID, shell, cols, rows, nil
}

func (s *AgentService) ExecContainer(stream pb.AgentService_ExecContainerServer) error {
	containerID, shell, cols, rows, err := parseExecStartRequest(stream)
	if err != nil {
		return err
	}

	cmd := exec.Command("docker", "exec", "-it", containerID, shell)
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: cols, Rows: rows})
	if err != nil {
		s.logger.Error("exec: failed to start pty", "error", err, "container", containerID)
		return fmt.Errorf("exec: failed to start docker exec: %w", err)
	}
	defer ptmx.Close()

	session := &execSession{
		stream: stream,
		ptmx:   ptmx,
		done:   make(chan struct{}),
		logger: s.logger,
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go session.readLoop(&wg)
	go session.writeLoop(&wg)

	exitCode := 0
	if waitErr := cmd.Wait(); waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}
	close(session.done)
	session.sendExitCode(exitCode)

	wg.Wait()
	return nil
}
