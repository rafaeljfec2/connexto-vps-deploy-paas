package provisioner

import (
	"fmt"
	"net"
	"strings"

	"golang.org/x/crypto/ssh"

	"github.com/paasdeploy/backend/internal/domain"
)

type ManageAction string

const (
	ManageActionRestartAgent      ManageAction = "restart_agent"
	ManageActionRestartUserMgr    ManageAction = "restart_user_manager"
	ManageActionAgentLogs         ManageAction = "agent_logs"
	ManageActionFixDockerPerms    ManageAction = "fix_docker_permissions"
)

type ManageResult struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
}

func ValidateManageAction(action string) bool {
	switch ManageAction(action) {
	case ManageActionRestartAgent, ManageActionRestartUserMgr,
		ManageActionAgentLogs, ManageActionFixDockerPerms:
		return true
	}
	return false
}

func (p *SSHProvisioner) ManageServer(
	server *domain.Server,
	sshKey string,
	sshPassword string,
	action ManageAction,
) (*ManageResult, error) {
	port := server.SSHPort
	if port == 0 {
		port = defaultSSHPort
	}
	addr := net.JoinHostPort(server.Host, fmt.Sprintf("%d", port))

	client, err := p.connect(server.SSHUser, addr, sshKey, sshPassword, server.SSHHostKey, server.ID)
	if err != nil {
		return nil, fmt.Errorf("ssh connect: %w", err)
	}
	defer client.Close()

	uid, err := runCommandOutput(client, "id -u")
	if err != nil {
		return nil, fmt.Errorf("get uid: %w", err)
	}

	switch action {
	case ManageActionRestartAgent:
		return restartAgent(client, uid)
	case ManageActionRestartUserMgr:
		return restartUserManager(client, uid, sshPassword)
	case ManageActionAgentLogs:
		return getAgentLogs(client, uid)
	case ManageActionFixDockerPerms:
		return fixDockerPermissions(client, uid, sshPassword)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

func restartAgent(client *ssh.Client, uid string) (*ManageResult, error) {
	runtimeDir := fmt.Sprintf("/run/user/%s", uid)
	restartCmd := fmt.Sprintf("XDG_RUNTIME_DIR=%s systemctl --user restart %s", runtimeDir, agentSystemdUnit)
	if err := runCommand(client, restartCmd); err != nil {
		return &ManageResult{Success: false, Output: fmt.Sprintf("restart failed: %s", err)}, nil
	}

	statusCmd := fmt.Sprintf("XDG_RUNTIME_DIR=%s systemctl --user is-active %s", runtimeDir, agentSystemdUnit)
	status, err := runCommandOutput(client, statusCmd)
	if err != nil {
		return &ManageResult{Success: false, Output: fmt.Sprintf("agent restarted but status check failed: %s", err)}, nil
	}

	return &ManageResult{
		Success: strings.TrimSpace(status) == "active",
		Output:  fmt.Sprintf("Agent status: %s", strings.TrimSpace(status)),
	}, nil
}

func restartUserManager(client *ssh.Client, uid string, password string) (*ManageResult, error) {
	cmd := fmt.Sprintf("systemctl restart user@%s.service", uid)
	if err := runPrivilegedCommand(client, uid, password, cmd); err != nil {
		return &ManageResult{Success: false, Output: fmt.Sprintf("restart user manager failed: %s", err)}, nil
	}

	return &ManageResult{
		Success: true,
		Output:  "User manager restarted. Agent will be restarted automatically with updated group memberships.",
	}, nil
}

func getAgentLogs(client *ssh.Client, uid string) (*ManageResult, error) {
	runtimeDir := fmt.Sprintf("/run/user/%s", uid)
	cmd := fmt.Sprintf("XDG_RUNTIME_DIR=%s journalctl --user -u %s -n 100 --no-pager 2>&1", runtimeDir, agentSystemdUnit)
	output, err := runCommandOutput(client, cmd)
	if err != nil {
		return &ManageResult{Success: false, Output: fmt.Sprintf("failed to fetch logs: %s", err)}, nil
	}

	return &ManageResult{Success: true, Output: output}, nil
}

func fixDockerPermissions(client *ssh.Client, uid string, password string) (*ManageResult, error) {
	var steps []string

	currentUser, err := runCommandOutput(client, "whoami")
	if err != nil {
		return &ManageResult{Success: false, Output: fmt.Sprintf("failed to get current user: %s", err)}, nil
	}
	currentUser = strings.TrimSpace(currentUser)

	addGroupCmd := fmt.Sprintf("usermod -aG docker %s", currentUser)
	if err := runPrivilegedCommand(client, uid, password, addGroupCmd); err != nil {
		return &ManageResult{Success: false, Output: fmt.Sprintf("failed to add user to docker group: %s", err)}, nil
	}
	steps = append(steps, fmt.Sprintf("Added %s to docker group", currentUser))

	restartMgrCmd := fmt.Sprintf("systemctl restart user@%s.service", uid)
	if err := runPrivilegedCommand(client, uid, password, restartMgrCmd); err != nil {
		return &ManageResult{
			Success: false,
			Output:  fmt.Sprintf("%s\nFailed to restart user manager: %s", strings.Join(steps, "\n"), err),
		}, nil
	}
	steps = append(steps, "Restarted user manager (group changes applied)")
	steps = append(steps, "Agent will restart automatically with Docker access")

	return &ManageResult{Success: true, Output: strings.Join(steps, "\n")}, nil
}
