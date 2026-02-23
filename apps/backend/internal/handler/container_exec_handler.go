package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
	"github.com/paasdeploy/backend/internal/agentclient"
	"github.com/paasdeploy/backend/internal/domain"
)

const (
	defaultCols    = 80
	defaultRows    = 24
	resizeCtrlByte = 0x01
	ptyReadBufSize = 4096
)

type ContainerExecHandler struct {
	agentClient *agentclient.AgentClient
	serverRepo  domain.ServerRepository
	agentPort   int
	logger      *slog.Logger
}

type ContainerExecHandlerConfig struct {
	AgentClient *agentclient.AgentClient
	ServerRepo  domain.ServerRepository
	AgentPort   int
	Logger      *slog.Logger
}

type resizeMessage struct {
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

func NewContainerExecHandler(cfg ContainerExecHandlerConfig) *ContainerExecHandler {
	return &ContainerExecHandler{
		agentClient: cfg.AgentClient,
		serverRepo:  cfg.ServerRepo,
		agentPort:   cfg.AgentPort,
		logger:      cfg.Logger,
	}
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
	serverID := c.Query("serverId", "")

	if serverID == "" {
		user, _ := c.Locals(userContextKey).(*domain.User)
		if user == nil || !user.IsAdmin() {
			_ = c.WriteMessage(websocket.TextMessage, []byte("Error: local console requires admin role\r\n"))
			return
		}
	}

	if serverID != "" {
		h.handleRemoteConsole(c, containerID, shell, cols, rows, serverID)
		return
	}

	h.handleLocalConsole(c, containerID, shell, cols, rows)
}

func (h *ContainerExecHandler) handleLocalConsole(c *websocket.Conn, containerID, shell string, cols, rows uint16) {
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

func (h *ContainerExecHandler) handleRemoteConsole(c *websocket.Conn, containerID, shell string, cols, rows uint16, serverID string) {
	server, err := h.serverRepo.FindByID(serverID)
	if err != nil {
		h.logger.Error("server not found for exec", "serverId", serverID, "error", err)
		_ = c.WriteMessage(websocket.TextMessage, []byte("Error: server not found\r\n"))
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	execStream, err := h.agentClient.ExecContainer(ctx, server.Host, h.agentPort)
	if err != nil {
		h.logger.Error("failed to open exec stream", "serverId", serverID, "error", err)
		_ = c.WriteMessage(websocket.TextMessage, []byte("Error: failed to connect to remote agent\r\n"))
		return
	}
	defer execStream.Cleanup()

	stream := execStream.Stream

	startReq := &pb.ExecInput{
		Payload: &pb.ExecInput_Start{
			Start: &pb.ExecStartRequest{
				ContainerId: containerID,
				Shell:       shell,
				Cols:        uint32(cols),
				Rows:        uint32(rows),
			},
		},
	}
	if err := stream.Send(startReq); err != nil {
		h.logger.Error("failed to send exec start", "error", err)
		_ = c.WriteMessage(websocket.TextMessage, []byte("Error: failed to start remote exec\r\n"))
		return
	}

	var wg sync.WaitGroup
	done := make(chan struct{})

	wg.Add(2)

	go func() {
		defer wg.Done()
		h.grpcToWS(stream, c, done)
	}()

	go func() {
		defer wg.Done()
		h.wsToGRPC(c, stream, done, cancel)
	}()

	wg.Wait()
}

func (h *ContainerExecHandler) grpcToWS(stream pb.AgentService_ExecContainerClient, conn *websocket.Conn, done chan struct{}) {
	defer close(done)
	for {
		out, err := stream.Recv()
		if err != nil {
			return
		}

		switch p := out.Payload.(type) {
		case *pb.ExecOutput_Data:
			if writeErr := conn.WriteMessage(websocket.TextMessage, p.Data); writeErr != nil {
				return
			}
		case *pb.ExecOutput_ExitCode:
			return
		}
	}
}

type wsPayload struct {
	data     []byte
	isResize bool
	resize   resizeMessage
}

func readWSMessage(conn *websocket.Conn) (*wsPayload, error) {
	mt, msg, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	if mt != websocket.TextMessage && mt != websocket.BinaryMessage {
		return nil, nil
	}
	if len(msg) == 0 {
		return nil, nil
	}

	if msg[0] == resizeCtrlByte && len(msg) > 1 {
		var r resizeMessage
		if jsonErr := json.Unmarshal(msg[1:], &r); jsonErr == nil && r.Cols > 0 && r.Rows > 0 {
			return &wsPayload{isResize: true, resize: r}, nil
		}
		return nil, nil
	}

	return &wsPayload{data: msg}, nil
}

func (h *ContainerExecHandler) wsToGRPC(conn *websocket.Conn, stream pb.AgentService_ExecContainerClient, done <-chan struct{}, cancel context.CancelFunc) {
	defer cancel()
	for {
		select {
		case <-done:
			return
		default:
		}

		msg, err := readWSMessage(conn)
		if err != nil {
			return
		}
		if msg == nil {
			continue
		}

		if msg.isResize {
			_ = stream.Send(&pb.ExecInput{
				Payload: &pb.ExecInput_Resize{
					Resize: &pb.ExecResize{
						Cols: uint32(msg.resize.Cols),
						Rows: uint32(msg.resize.Rows),
					},
				},
			})
			continue
		}

		if sendErr := stream.Send(&pb.ExecInput{
			Payload: &pb.ExecInput_Data{Data: msg.data},
		}); sendErr != nil {
			return
		}
	}
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
		}

		msg, err := readWSMessage(conn)
		if err != nil {
			return
		}
		if msg == nil {
			continue
		}

		if msg.isResize {
			_ = pty.Setsize(ptmx, &pty.Winsize{Cols: msg.resize.Cols, Rows: msg.resize.Rows})
			continue
		}

		if _, writeErr := ptmx.Write(msg.data); writeErr != nil {
			return
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
