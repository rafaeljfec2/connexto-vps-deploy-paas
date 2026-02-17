package handler

import (
	"io"
	"log/slog"
	"os/exec"
	"sync"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

const errFailedStartExec = "Error: failed to start exec\r\n"

type ContainerExecHandler struct {
	logger *slog.Logger
}

func NewContainerExecHandler(logger *slog.Logger) *ContainerExecHandler {
	return &ContainerExecHandler{logger: logger}
}

func (h *ContainerExecHandler) Register(app fiber.Router) {
	v1 := app.Group(APIPrefix)
	v1.Get("/containers/:id/console", websocket.New(h.handleConsole,
		websocket.Config{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	))
}

func (h *ContainerExecHandler) handleConsole(c *websocket.Conn) {
	containerID := c.Params("id")
	if containerID == "" {
		_ = c.WriteMessage(websocket.TextMessage, []byte("Error: container ID required\r\n"))
		return
	}

	shell := c.Query("shell", "sh")
	cmd := exec.Command("docker", "exec", "-i", containerID, shell)

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		h.logger.Error("Failed to create stdin pipe", "error", err, "container", containerID)
		_ = c.WriteMessage(websocket.TextMessage, []byte(errFailedStartExec))
		return
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		h.logger.Error("Failed to create stdout pipe", "error", err, "container", containerID)
		_ = c.WriteMessage(websocket.TextMessage, []byte(errFailedStartExec))
		return
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		h.logger.Error("Failed to create stderr pipe", "error", err, "container", containerID)
		_ = c.WriteMessage(websocket.TextMessage, []byte(errFailedStartExec))
		return
	}

	if err := cmd.Start(); err != nil {
		h.logger.Error("Failed to start docker exec", "error", err, "container", containerID)
		_ = c.WriteMessage(websocket.TextMessage, []byte("Error: failed to connect to container (ensure it is running)\r\n"))
		return
	}

	welcome := []byte("\r\n# FlowDeploy Console (type commands and press Enter)\r\n$ ")
	_ = c.WriteMessage(websocket.TextMessage, welcome)

	var wg sync.WaitGroup
	done := make(chan struct{})

	wg.Add(3)
	go func() {
		defer wg.Done()
		h.copyToProcess(c, stdinPipe, done)
	}()
	go func() {
		defer wg.Done()
		h.copyFromProcess(stdoutPipe, c, done)
	}()
	go func() {
		defer wg.Done()
		h.copyFromProcess(stderrPipe, c, done)
	}()

	go func() {
		_ = cmd.Wait()
		close(done)
	}()

	wg.Wait()
}

func (h *ContainerExecHandler) copyToProcess(conn *websocket.Conn, stdin io.WriteCloser, done <-chan struct{}) {
	defer stdin.Close()
	for {
		select {
		case <-done:
			return
		default:
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if (mt == websocket.TextMessage || mt == websocket.BinaryMessage) && len(msg) > 0 {
				if _, err := stdin.Write(msg); err != nil {
					return
				}
			}
		}
	}
}

func (h *ContainerExecHandler) copyFromProcess(reader io.Reader, conn *websocket.Conn, done <-chan struct{}) {
	buf := make([]byte, 4096)
	for {
		select {
		case <-done:
			return
		default:
			n, err := reader.Read(buf)
			if n > 0 {
				if writeErr := conn.WriteMessage(websocket.TextMessage, buf[:n]); writeErr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}
}
