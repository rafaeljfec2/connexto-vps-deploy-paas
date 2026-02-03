package provisioner

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/pki"
)

const (
	agentInstallDirName = "paasdeploy-agent"
	agentSystemdUnit    = "paasdeploy-agent.service"
	defaultSSHPort      = 22
	sshConnectTimeout   = 30 * time.Second
	logKeyStep          = "step"
)

type SSHProvisionerConfig struct {
	CA              *pki.CertificateAuthority
	ServerAddr      string
	AgentBinaryPath string
	AgentPort       int
	Logger          *slog.Logger
}

type SSHProvisioner struct {
	cfg SSHProvisionerConfig
}

func NewSSHProvisioner(cfg SSHProvisionerConfig) *SSHProvisioner {
	return &SSHProvisioner{cfg: cfg}
}

func (p *SSHProvisioner) Provision(server *domain.Server, sshKeyPlain string, sshPasswordPlain string) error {
	log := p.cfg.Logger.With("serverId", server.ID, "host", server.Host)
	port := server.SSHPort
	if port == 0 {
		port = defaultSSHPort
	}

	addr := net.JoinHostPort(server.Host, fmt.Sprintf("%d", port))
	log.Info("provision", logKeyStep, "ssh_connect", "addr", addr, "user", server.SSHUser)
	client, err := p.connect(server.SSHUser, addr, sshKeyPlain, sshPasswordPlain)
	if err != nil {
		return fmt.Errorf("ssh connect: %w", err)
	}
	defer client.Close()
	log.Info("provision", logKeyStep, "ssh_connect", "status", "ok")

	log.Info("provision", logKeyStep, "remote_env")
	homeDir, err := runCommandOutput(client, "printf $HOME")
	if err != nil {
		return fmt.Errorf("get remote home: %w", err)
	}
	uid, err := runCommandOutput(client, "id -u")
	if err != nil {
		return fmt.Errorf("get remote uid: %w", err)
	}
	log.Info("provision", logKeyStep, "remote_env", "status", "ok", "homeDir", homeDir, "uid", uid)

	log.Info("provision", logKeyStep, "sftp_client")
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("create sftp client: %w", err)
	}
	defer sftpClient.Close()
	log.Info("provision", logKeyStep, "sftp_client", "status", "ok")

	log.Info("provision", logKeyStep, "install_dir")
	installDir, unitDir, err := resolveSftpPaths(sftpClient, homeDir)
	if err != nil {
		return err
	}
	runtimeDir := path.Join("/run/user", uid)
	log.Info("provision", logKeyStep, "install_dir", "status", "ok", "installDir", installDir, "unitDir", unitDir)

	log.Info("provision", logKeyStep, "agent_certs")
	agentCert, err := p.cfg.CA.GenerateAgentCert(server.ID, server.Host)
	if err != nil {
		return fmt.Errorf("generate agent cert: %w", err)
	}
	if err := writeCertFiles(sftpClient, installDir, agentCert, p.cfg.CA); err != nil {
		return err
	}
	log.Info("provision", logKeyStep, "agent_certs", "status", "ok")

	if err := p.deployAgentBinary(client, sftpClient, installDir, log); err != nil {
		return err
	}

	log.Info("provision", logKeyStep, "systemd_unit")
	unitOpts := systemdUnitOpts{
		sshClient:  client,
		sftpClient: sftpClient,
		installDir: installDir,
		unitDir:    unitDir,
		runtimeDir: runtimeDir,
		serverID:   server.ID,
		serverAddr: p.cfg.ServerAddr,
		agentPort:  p.cfg.AgentPort,
	}
	if err := installSystemdUnit(unitOpts); err != nil {
		return err
	}
	log.Info("provision", logKeyStep, "systemd_unit", "status", "ok")

	log.Info("provision", logKeyStep, "start_agent", "runtimeDir", runtimeDir)
	if err := startAgent(client, runtimeDir); err != nil {
		return err
	}
	log.Info("provision", logKeyStep, "start_agent", "status", "ok")

	log.Info("provision completed", "serverId", server.ID, "host", server.Host)
	return nil
}

func (p *SSHProvisioner) deployAgentBinary(sshClient *ssh.Client, sftpClient *sftp.Client, installDir string, log *slog.Logger) error {
	if p.cfg.AgentBinaryPath == "" {
		log.Info("provision", logKeyStep, "agent_binary", "status", "skipped", "reason", "AGENT_BINARY_PATH empty")
		return nil
	}
	log.Info("provision", logKeyStep, "agent_binary", "localPath", p.cfg.AgentBinaryPath)
	if err := copyAgentBinary(sftpClient, installDir, p.cfg.AgentBinaryPath); err != nil {
		log.Info("provision", logKeyStep, "agent_binary", "fallback", "ssh_pipe", "sftp_err", err)
		if pipeErr := copyAgentBinaryViaSSH(sshClient, installDir, p.cfg.AgentBinaryPath); pipeErr != nil {
			return fmt.Errorf("sftp: %w; ssh pipe fallback: %w", err, pipeErr)
		}
	}
	log.Info("provision", logKeyStep, "agent_binary", "status", "ok")
	return nil
}

func (p *SSHProvisioner) connect(user, addr, privateKey string, password string) (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod
	if privateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(privateKey))
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	if password != "" {
		authMethods = append(authMethods, ssh.Password(password))
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no ssh auth methods configured")
	}

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         sshConnectTimeout,
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	return client, nil
}

func createInstallDir(client *sftp.Client, installDir string) error {
	if err := client.MkdirAll(installDir); err != nil {
		return fmt.Errorf("create install dir: %w", err)
	}
	_ = client.Chmod(installDir, 0o755)
	return nil
}

func resolveSftpPaths(client *sftp.Client, homeDir string) (string, string, error) {
	absoluteInstallDir := path.Join(homeDir, agentInstallDirName)
	if err := createInstallDir(client, absoluteInstallDir); err == nil {
		return absoluteInstallDir, path.Join(homeDir, ".config", "systemd", "user"), nil
	}

	relativeInstallDir := agentInstallDirName
	if err := createInstallDir(client, relativeInstallDir); err != nil {
		return "", "", err
	}

	return relativeInstallDir, path.Join(".config", "systemd", "user"), nil
}

func writeCertFiles(client *sftp.Client, installDir string, agentCert *pki.Certificate, ca *pki.CertificateAuthority) error {
	files := map[string][]byte{
		path.Join(installDir, "ca.pem"):   ca.GetCACertPEM(),
		path.Join(installDir, "cert.pem"): agentCert.CertPEM,
		path.Join(installDir, "key.pem"):  agentCert.KeyPEM,
	}

	for target, content := range files {
		if err := writeRemoteFile(client, target, content, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", target, err)
		}
	}
	return nil
}

const binaryWriteChunkSize = 8192

func copyAgentBinaryViaSSH(sshClient *ssh.Client, installDir string, localPath string) error {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("read agent binary: %w", err)
	}
	target := path.Join(installDir, "agent")
	session, err := sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("new session: %w", err)
	}
	defer session.Close()
	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	cmd := fmt.Sprintf("cat > %q && chmod 755 %q", target, target)
	if err := session.Start(cmd); err != nil {
		return fmt.Errorf("start cat: %w", err)
	}
	if _, err := stdin.Write(data); err != nil {
		return fmt.Errorf("write stdin: %w", err)
	}
	if err := stdin.Close(); err != nil {
		return fmt.Errorf("close stdin: %w", err)
	}
	if err := session.Wait(); err != nil {
		return fmt.Errorf("session wait: %w", err)
	}
	return nil
}

func copyAgentBinary(client *sftp.Client, installDir string, localPath string) error {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("read agent binary: %w", err)
	}

	target := path.Join(installDir, "agent")
	_ = client.Remove(target)
	f, err := client.Create(target)
	if err != nil {
		return fmt.Errorf("create agent file: %w", err)
	}
	defer f.Close()

	for i := 0; i < len(data); i += binaryWriteChunkSize {
		end := i + binaryWriteChunkSize
		if end > len(data) {
			end = len(data)
		}
		if _, err := f.Write(data[i:end]); err != nil {
			return fmt.Errorf("write agent at offset %d: %w", i, err)
		}
	}
	_ = f.Sync()
	_ = f.Chmod(0o755)
	return nil
}

type systemdUnitOpts struct {
	sshClient  *ssh.Client
	sftpClient *sftp.Client
	installDir string
	unitDir    string
	runtimeDir string
	serverID   string
	serverAddr string
	agentPort  int
}

func installSystemdUnit(opts systemdUnitOpts) error {
	serverAddr := opts.serverAddr
	if serverAddr == "" {
		serverAddr = "localhost:50051"
	}
	agentPort := opts.agentPort
	if agentPort == 0 {
		agentPort = 50052
	}

	if err := opts.sftpClient.MkdirAll(opts.unitDir); err != nil {
		return fmt.Errorf("create systemd dir: %w", err)
	}

	unitContent := fmt.Sprintf(`[Unit]
Description=PaasDeploy Agent
After=network.target

[Service]
Type=simple
ExecStart=%s/agent -server-addr=%s -server-id=%s -ca-cert=%s/ca.pem -cert=%s/cert.pem -key=%s/key.pem -agent-port=%d
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
`, opts.installDir, serverAddr, opts.serverID, opts.installDir, opts.installDir, opts.installDir, agentPort)

	unitPath := path.Join(opts.unitDir, agentSystemdUnit)
	if err := writeRemoteFile(opts.sftpClient, unitPath, []byte(unitContent), 0o644); err != nil {
		return err
	}

	reloadCmd := fmt.Sprintf("XDG_RUNTIME_DIR=%s systemctl --user daemon-reload", opts.runtimeDir)
	return runCommand(opts.sshClient, reloadCmd)
}

func startAgent(client *ssh.Client, runtimeDir string) error {
	cmd := fmt.Sprintf("XDG_RUNTIME_DIR=%s systemctl --user enable %s --now", runtimeDir, agentSystemdUnit)
	return runCommand(client, cmd)
}

func writeRemoteFile(client *sftp.Client, remotePath string, data []byte, perm os.FileMode) error {
	file, err := client.OpenFile(remotePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := io.Copy(file, bytes.NewReader(data)); err != nil {
		return err
	}

	_ = file.Sync()
	_ = file.Chmod(perm)
	return nil
}

func runCommand(client *ssh.Client, cmd string) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	var stderr bytes.Buffer
	session.Stderr = &stderr
	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("command failed: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func runCommandOutput(client *ssh.Client, cmd string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	outStr := strings.TrimSpace(string(output))
	if err != nil {
		return "", fmt.Errorf("command failed: %w (output: %s)", err, outStr)
	}
	return outStr, nil
}
