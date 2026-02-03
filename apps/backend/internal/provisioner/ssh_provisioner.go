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
	port := server.SSHPort
	if port == 0 {
		port = defaultSSHPort
	}

	addr := net.JoinHostPort(server.Host, fmt.Sprintf("%d", port))
	client, err := p.connect(server.SSHUser, addr, sshKeyPlain, sshPasswordPlain)
	if err != nil {
		return fmt.Errorf("ssh connect: %w", err)
	}
	defer client.Close()

	homeDir, err := runCommandOutput(client, "printf $HOME")
	if err != nil {
		return fmt.Errorf("get remote home: %w", err)
	}

	uid, err := runCommandOutput(client, "id -u")
	if err != nil {
		return fmt.Errorf("get remote uid: %w", err)
	}

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("create sftp client: %w", err)
	}
	defer sftpClient.Close()

	installDir := path.Join(homeDir, agentInstallDirName)
	unitDir := path.Join(homeDir, ".config", "systemd", "user")
	runtimeDir := path.Join("/run/user", uid)

	if err := createInstallDir(sftpClient, installDir); err != nil {
		return err
	}

	agentCert, err := p.cfg.CA.GenerateAgentCert(server.ID, server.Host)
	if err != nil {
		return fmt.Errorf("generate agent cert: %w", err)
	}

	if err := writeCertFiles(sftpClient, installDir, agentCert, p.cfg.CA); err != nil {
		return err
	}

	if p.cfg.AgentBinaryPath != "" {
		if err := copyAgentBinary(sftpClient, installDir, p.cfg.AgentBinaryPath); err != nil {
			return err
		}
	}

	unitOpts := systemdUnitOpts{
		sshClient:   client,
		sftpClient:  sftpClient,
		installDir:  installDir,
		unitDir:     unitDir,
		runtimeDir:  runtimeDir,
		serverID:    server.ID,
		serverAddr:  p.cfg.ServerAddr,
		agentPort:   p.cfg.AgentPort,
	}
	if err := installSystemdUnit(unitOpts); err != nil {
		return err
	}

	if err := startAgent(client, runtimeDir); err != nil {
		return err
	}

	p.cfg.Logger.Info("provision completed", "serverId", server.ID, "host", server.Host)
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
	if err := client.Chmod(installDir, 0o755); err != nil {
		return fmt.Errorf("chmod install dir: %w", err)
	}
	return nil
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

func copyAgentBinary(client *sftp.Client, installDir string, localPath string) error {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("read agent binary: %w", err)
	}

	target := path.Join(installDir, "agent")
	if err := writeRemoteFile(client, target, data, 0o755); err != nil {
		return err
	}
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

	if err := file.Sync(); err != nil {
		return err
	}

	if err := file.Chmod(perm); err != nil {
		return err
	}
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
