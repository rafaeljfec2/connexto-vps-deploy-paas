package provisioner

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/pki"
)

const (
	agentInstallDir   = "/opt/paasdeploy-agent"
	agentSystemdUnit  = "paasdeploy-agent.service"
	defaultSSHPort    = 22
	sshConnectTimeout = 30 * time.Second
	sshSessionTimeout = 5 * time.Minute
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

	if err := p.createInstallDir(client); err != nil {
		return err
	}

	agentCert, err := p.cfg.CA.GenerateAgentCert(server.ID, server.Host)
	if err != nil {
		return fmt.Errorf("generate agent cert: %w", err)
	}

	if err := p.writeFiles(client, agentCert); err != nil {
		return err
	}

	if p.cfg.AgentBinaryPath != "" {
		if err := p.copyAgentBinary(client); err != nil {
			return err
		}
	}

	if err := p.installSystemdUnit(client, server.ID); err != nil {
		return err
	}

	if err := p.startAgent(client); err != nil {
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

func (p *SSHProvisioner) createInstallDir(client *ssh.Client) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	var stderr bytes.Buffer
	session.Stderr = &stderr
	cmd := fmt.Sprintf("sudo mkdir -p %s && sudo chmod 755 %s", agentInstallDir, agentInstallDir)
	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("create install dir: %w (stderr: %s)", err, stderr.String())
	}
	return nil
}

func (p *SSHProvisioner) writeFiles(client *ssh.Client, agentCert *pki.Certificate) error {
	files := map[string][]byte{
		filepath.Join(agentInstallDir, "ca.pem"):   p.cfg.CA.GetCACertPEM(),
		filepath.Join(agentInstallDir, "cert.pem"): agentCert.CertPEM,
		filepath.Join(agentInstallDir, "key.pem"):  agentCert.KeyPEM,
	}

	for path, content := range files {
		if err := p.writeRemoteFile(client, path, content, "0644"); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}
	return nil
}

func (p *SSHProvisioner) writeRemoteFile(client *ssh.Client, path string, content []byte, _ string) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	b64 := base64.StdEncoding.EncodeToString(content)
	cmd := fmt.Sprintf("echo %s | base64 -d | sudo tee %s > /dev/null", b64, path)
	return session.Run(cmd)
}

func (p *SSHProvisioner) copyAgentBinary(client *ssh.Client) error {
	data, err := os.ReadFile(p.cfg.AgentBinaryPath)
	if err != nil {
		return fmt.Errorf("read agent binary: %w", err)
	}

	dest := filepath.Join(agentInstallDir, "agent")
	if err := p.writeRemoteFile(client, dest, data, "0755"); err != nil {
		return err
	}

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	if err := session.Run(fmt.Sprintf("sudo chmod +x %s", dest)); err != nil {
		return fmt.Errorf("chmod agent: %w", err)
	}
	return nil
}

func (p *SSHProvisioner) installSystemdUnit(client *ssh.Client, serverID string) error {
	serverAddr := p.cfg.ServerAddr
	if serverAddr == "" {
		serverAddr = "localhost:50051"
	}
	agentPort := p.cfg.AgentPort
	if agentPort == 0 {
		agentPort = 50052
	}

	unit := fmt.Sprintf(`[Unit]
Description=PaasDeploy Agent
After=network.target

[Service]
Type=simple
ExecStart=%s/agent -server-addr=%s -server-id=%s -ca-cert=%s/ca.pem -cert=%s/cert.pem -key=%s/key.pem -agent-port=%d
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
`, agentInstallDir, serverAddr, serverID, agentInstallDir, agentInstallDir, agentInstallDir, agentPort)

	unitPath := "/etc/systemd/system/" + agentSystemdUnit
	b64 := base64.StdEncoding.EncodeToString([]byte(unit))
	cmd := fmt.Sprintf("echo %s | base64 -d | sudo tee %s > /dev/null && sudo systemctl daemon-reload", b64, unitPath)

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	return session.Run(cmd)
}

func (p *SSHProvisioner) startAgent(client *ssh.Client) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	cmd := fmt.Sprintf("sudo systemctl enable %s && sudo systemctl start %s", agentSystemdUnit, agentSystemdUnit)
	return session.Run(cmd)
}
