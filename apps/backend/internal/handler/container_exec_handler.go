package handler

import (
	"encoding/json"
	"log/slog"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

const (
	defaultCols     = 80
	defaultRows     = 24
	resizeCtrlByte  = 0x01
	ptyReadBufSize  = 4096
)

type ContainerExecHandler struct {
	logger *slog.Logger
}

type resizeMessage struct {
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

func NewContainerExecHandler(logger *slog.Logger) *ContainerExecHandler {
	return &ContainerExecHandler{logger: logger}
}

func (h *ContainerExecHandler) Register(app fiber.Router) {
	v1 := app.Group(APIPrefix)
	v1.Get("/containers/:id/console", h.requireAuthForWebSocket, websocket.New(h.handleConsole,
		websocket.Config{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	))
}

func (h *ContainerExecHandler) requireAuthForWebSocket(c *fiber.Ctx) error {
	if !websocket.IsWebSocketUpgrade(c) {
		return fiber.ErrUpgradeRequired
	}
	user := GetUserFromContext(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "authentication required"})
	}
	return c.Next()
}

func (h *ContainerExecHandler) handleConsole(c *websocket.Conn) {
	containerID := c.Params("id")
	if containerID == "" {
		_ = c.WriteMessage(websocket.TextMessage, []byte("Error: container ID required\r\n"))
		return
	}

	shell := c.Query("shell", "sh")
	cols := parseUint16Query(c, "cols", defaultCols)
	rows := parseUint16Query(c, "rows", defaultRows)

	cmd := exec.Command("docker", "exec", "-it", containerID, shell)

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: cols, Rows: rows})
	if err != nil {
		h.logger.Error("failed to start docker exec with pty", "error", err, "container", containerID)
		_ = c.WriteMessage(websocket.TextMessage, []byte("Error: failed to connect to container (ensure it is running)\r\n"))
		return
	}
	defer ptmx.Close()

	var wg sync.WaitGroup
	done := make(chan struct{})

	wg.Add(2)

	go func() {
		defer wg.Done()
		h.readFromPTY(ptmx, c, done)
	}()

	go func() {
		defer wg.Done()
		h.writeFromWS(c, ptmx, done)
	}()

	go func() {
		_ = cmd.Wait()
		close(done)
	}()

	wg.Wait()
}

func (h *ContainerExecHandler) readFromPTY(ptmx *os.File, conn *websocket.Conn, done <-chan struct{}) {
	buf := make([]byte, ptyReadBufSize)
	for {
		select {
		case <-done:
			return
		default:
			n, err := ptmx.Read(buf)
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

func (h *ContainerExecHandler) writeFromWS(conn *websocket.Conn, ptmx *os.File, done <-chan struct{}) {
	defer ptmx.Close()
	for {
		select {
		case <-done:
			return
		default:
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if mt != websocket.TextMessage && mt != websocket.BinaryMessage {
				continue
			}
			if len(msg) == 0 {
				continue
			}

			if msg[0] == resizeCtrlByte && len(msg) > 1 {
				var resize resizeMessage
				if jsonErr := json.Unmarshal(msg[1:], &resize); jsonErr == nil && resize.Cols > 0 && resize.Rows > 0 {
					_ = pty.Setsize(ptmx, &pty.Winsize{Cols: resize.Cols, Rows: resize.Rows})
				}
				continue
			}

			if _, writeErr := ptmx.Write(msg); writeErr != nil {
				return
			}
		}
	}
}

func parseUint16Query(c *websocket.Conn, key string, fallback uint16) uint16 {
	raw := c.Query(key, "")
	if raw == "" {
		return fallback
	}
	var val int
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return fallback
		}
		val = val*10 + int(ch-'0')
	}
	if val <= 0 || val > 65535 {
		return fallback
	}
	return uint16(val)
}
