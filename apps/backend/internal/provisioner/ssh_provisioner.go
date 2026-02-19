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
	agentInstallDirName   = "paasdeploy-agent"
	agentSystemdUnit      = "paasdeploy-agent.service"
	dotConfigDir          = ".config"
	defaultSSHPort        = 22
	sshConnectTimeout     = 30 * time.Second
	logKeyStep            = "step"
	errReadAgentBinaryFmt = "read agent binary: %w"

	timeoutDockerInstall = 10 * time.Minute
	timeoutDockerCheck   = 30 * time.Second
	timeoutTraefikSetup  = 5 * time.Minute
	timeoutNetworkSetup  = 30 * time.Second
)

type SSHHostKeyStore interface {
	UpdateSSHHostKey(serverID string, hostKey string) error
}

type SSHProvisionerConfig struct {
	CA              *pki.CertificateAuthority
	ServerAddr      string
	AgentBinaryPath string
	AgentPort       int
	Logger          *slog.Logger
	HostKeyStore    SSHHostKeyStore
}

type ProvisionProgress struct {
	OnStep func(step, status, message string)
	OnLog  func(message string)
}

type SSHProvisioner struct {
	cfg SSHProvisionerConfig
}

func NewSSHProvisioner(cfg SSHProvisionerConfig) *SSHProvisioner {
	return &SSHProvisioner{cfg: cfg}
}

func (p *SSHProvisioner) Provision(server *domain.Server, sshKeyPlain string, sshPasswordPlain string, progress *ProvisionProgress) error {
	log := p.cfg.Logger.With("serverId", server.ID, "host", server.Host)
	step := p.makeStepFn(log, progress)
	logLine := p.makeLogFn(progress)

	port := server.SSHPort
	if port == 0 {
		port = defaultSSHPort
	}
	addr := net.JoinHostPort(server.Host, fmt.Sprintf("%d", port))
	logLine(fmt.Sprintf("Conectando a %s como %s", addr, server.SSHUser))

	client, err := p.provisionSSH(server.SSHUser, addr, sshKeyPlain, sshPasswordPlain, server.SSHHostKey, server.ID, step)
	if err != nil {
		return err
	}
	defer client.Close()

	homeDir, uid, err := p.provisionRemoteEnv(client, step, logLine)
	if err != nil {
		return err
	}

	if err := p.provisionDocker(client, uid, sshPasswordPlain, step, logLine); err != nil {
		return err
	}
	if err := p.provisionDockerNetwork(client, step, logLine); err != nil {
		return err
	}

	acmeEmail := ""
	if server.AcmeEmail != nil {
		acmeEmail = *server.AcmeEmail
	}
	if acmeEmail != "" {
		if err := p.provisionTraefik(client, uid, sshPasswordPlain, acmeEmail, step, logLine); err != nil {
			return err
		}
	}

	sftpClient, err := p.provisionSFTP(client, step)
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	installDir, unitDir, runtimeDir, err := p.provisionInstallDir(sftpClient, homeDir, uid, step, logLine)
	if err != nil {
		return err
	}

	if err := p.provisionCerts(sftpClient, installDir, server, step); err != nil {
		return err
	}
	if err := p.deployAgentBinary(client, sftpClient, installDir, log, step, logLine); err != nil {
		return err
	}
	paths := provisionPaths{installDir: installDir, unitDir: unitDir, runtimeDir: runtimeDir, serverID: server.ID}
	if err := p.provisionSystemdAndStart(client, sftpClient, paths, step, logLine); err != nil {
		return err
	}
	logLine("Provisionamento concluído")
	return nil
}

func (p *SSHProvisioner) makeStepFn(log *slog.Logger, progress *ProvisionProgress) func(string, string, string) {
	return func(s, status, msg string) {
		log.Info("provision", logKeyStep, s, "status", status)
		if progress != nil && progress.OnStep != nil {
			progress.OnStep(s, status, msg)
		}
	}
}

func (p *SSHProvisioner) makeLogFn(progress *ProvisionProgress) func(string) {
	return func(msg string) {
		if progress != nil && progress.OnLog != nil {
			progress.OnLog(msg)
		}
	}
}

func (p *SSHProvisioner) provisionSSH(user, addr, key, password, knownHostKey, serverID string, step func(string, string, string)) (*ssh.Client, error) {
	step("ssh_connect", "running", "Conectando via SSH...")
	client, err := p.connect(user, addr, key, password, knownHostKey, serverID)
	if err != nil {
		return nil, fmt.Errorf("ssh connect: %w", err)
	}
	step("ssh_connect", "ok", "Conectado via SSH")
	return client, nil
}

func (p *SSHProvisioner) provisionRemoteEnv(
	client *ssh.Client,
	step func(string, string, string),
	logLine func(string),
) (string, string, error) {
	logLine("Verificando ambiente remoto")
	step("remote_env", "running", "Verificando ambiente...")
	homeDir, err := runCommandOutput(client, "printf $HOME")
	if err != nil {
		return "", "", fmt.Errorf("get remote home: %w", err)
	}
	uid, err := runCommandOutput(client, "id -u")
	if err != nil {
		return "", "", fmt.Errorf("get remote uid: %w", err)
	}
	logLine(fmt.Sprintf("Home: %s, UID: %s", homeDir, uid))
	step("remote_env", "ok", "Ambiente verificado")
	return homeDir, uid, nil
}

func (p *SSHProvisioner) provisionSFTP(client *ssh.Client, step func(string, string, string)) (*sftp.Client, error) {
	step("sftp_client", "running", "Conectando SFTP...")
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return nil, fmt.Errorf("create sftp client: %w", err)
	}
	step("sftp_client", "ok", "SFTP conectado")
	return sftpClient, nil
}

func (p *SSHProvisioner) provisionInstallDir(
	sftpClient *sftp.Client,
	homeDir, uid string,
	step func(string, string, string),
	logLine func(string),
) (installDir, unitDir, runtimeDir string, err error) {
	logLine("Criando diretórios")
	step("install_dir", "running", "Criando diretórios...")
	installDir, unitDir, err = resolveSftpPaths(sftpClient, homeDir)
	if err != nil {
		return "", "", "", err
	}
	runtimeDir = path.Join("/run/user", uid)
	logLine(fmt.Sprintf("Diretório: %s", installDir))
	step("install_dir", "ok", "Diretórios criados")
	return installDir, unitDir, runtimeDir, nil
}

func (p *SSHProvisioner) provisionCerts(
	sftpClient *sftp.Client,
	installDir string,
	server *domain.Server,
	step func(string, string, string),
) error {
	step("agent_certs", "running", "Instalando certificados...")
	agentCert, err := p.cfg.CA.GenerateAgentCert(server.ID, server.Host)
	if err != nil {
		return fmt.Errorf("generate agent cert: %w", err)
	}
	if err := writeCertFiles(sftpClient, installDir, agentCert, p.cfg.CA); err != nil {
		return err
	}
	step("agent_certs", "ok", "Certificados instalados")
	return nil
}

func (p *SSHProvisioner) provisionSystemdAndStart(
	client *ssh.Client,
	sftpClient *sftp.Client,
	paths provisionPaths,
	step func(string, string, string),
	logLine func(string),
) error {
	step("systemd_unit", "running", "Configurando serviço...")
	unitOpts := systemdUnitOpts{
		sshClient:  client,
		sftpClient: sftpClient,
		installDir: paths.installDir,
		unitDir:    paths.unitDir,
		runtimeDir: paths.runtimeDir,
		serverID:   paths.serverID,
		serverAddr: p.cfg.ServerAddr,
		agentPort:  p.cfg.AgentPort,
	}
	if err := installSystemdUnit(unitOpts); err != nil {
		return err
	}
	step("systemd_unit", "ok", "Serviço configurado")
	logLine("Iniciando agent")
	step("start_agent", "running", "Iniciando agent...")
	if err := startAgent(client, paths.runtimeDir); err != nil {
		return err
	}
	step("start_agent", "ok", "Agent iniciado")
	return nil
}

func (p *SSHProvisioner) deployAgentBinary(
	sshClient *ssh.Client,
	sftpClient *sftp.Client,
	installDir string,
	log *slog.Logger,
	step func(string, string, string),
	logLine func(string),
) error {
	if p.cfg.AgentBinaryPath == "" {
		log.Info("provision", logKeyStep, "agent_binary", "status", "skipped", "reason", "AGENT_BINARY_PATH empty")
		return nil
	}
	step("agent_binary", "running", "Copiando agent...")
	log.Info("provision", logKeyStep, "agent_binary", "localPath", p.cfg.AgentBinaryPath)
	data, err := os.ReadFile(p.cfg.AgentBinaryPath)
	if err != nil {
		return fmt.Errorf(errReadAgentBinaryFmt, err)
	}
	sizeKB := len(data) / 1024
	logLine(fmt.Sprintf("Copiando agent (%d KB)...", sizeKB))
	if err := copyAgentBinary(sftpClient, installDir, p.cfg.AgentBinaryPath); err != nil {
		log.Info("provision", logKeyStep, "agent_binary", "fallback", "ssh_pipe", "sftp_err", err)
		logLine("Fallback SSH pipe (SFTP falhou)")
		if pipeErr := copyAgentBinaryViaSSH(sshClient, installDir, p.cfg.AgentBinaryPath); pipeErr != nil {
			return fmt.Errorf("sftp: %w; ssh pipe fallback: %w", err, pipeErr)
		}
	}
	step("agent_binary", "ok", "Agent copiado")
	return nil
}

func (p *SSHProvisioner) connect(user, addr, privateKey string, password string, knownHostKey string, serverID string) (*ssh.Client, error) {
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

	hostKeyCallback := p.buildHostKeyCallback(knownHostKey, serverID)

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         sshConnectTimeout,
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	return client, nil
}

func (p *SSHProvisioner) buildHostKeyCallback(knownHostKey string, serverID string) ssh.HostKeyCallback {
	if knownHostKey != "" {
		parsedKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(knownHostKey))
		if err == nil {
			return ssh.FixedHostKey(parsedKey)
		}
		p.cfg.Logger.Warn("failed to parse stored host key, falling back to TOFU", "error", err, "serverId", serverID)
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		serialized := string(ssh.MarshalAuthorizedKey(key))
		if p.cfg.HostKeyStore != nil && serverID != "" {
			if err := p.cfg.HostKeyStore.UpdateSSHHostKey(serverID, serialized); err != nil {
				p.cfg.Logger.Warn("failed to store SSH host key", "error", err, "serverId", serverID)
			} else {
				p.cfg.Logger.Info("SSH host key stored (TOFU)", "serverId", serverID, "hostname", hostname)
			}
		}
		return nil
	}
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
		return absoluteInstallDir, path.Join(homeDir, dotConfigDir, "systemd", "user"), nil
	}

	relativeInstallDir := agentInstallDirName
	if err := createInstallDir(client, relativeInstallDir); err != nil {
		return "", "", err
	}

	return relativeInstallDir, path.Join(dotConfigDir, "systemd", "user"), nil
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
		return fmt.Errorf(errReadAgentBinaryFmt, err)
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
		return fmt.Errorf(errReadAgentBinaryFmt, err)
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

type provisionPaths struct {
	installDir string
	unitDir    string
	runtimeDir string
	serverID   string
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
	enableCmd := fmt.Sprintf("XDG_RUNTIME_DIR=%s systemctl --user enable %s", runtimeDir, agentSystemdUnit)
	if err := runCommand(client, enableCmd); err != nil {
		return fmt.Errorf("enable agent: %w", err)
	}

	restartCmd := fmt.Sprintf("XDG_RUNTIME_DIR=%s systemctl --user restart %s", runtimeDir, agentSystemdUnit)
	return runCommand(client, restartCmd)
}

func (p *SSHProvisioner) Deprovision(server *domain.Server, sshKeyPlain string, sshPasswordPlain string) error {
	log := p.cfg.Logger.With("serverId", server.ID, "host", server.Host)
	log.Info("deprovision started")
	port := server.SSHPort
	if port == 0 {
		port = defaultSSHPort
	}
	addr := net.JoinHostPort(server.Host, fmt.Sprintf("%d", port))
	client, err := p.connect(server.SSHUser, addr, sshKeyPlain, sshPasswordPlain, server.SSHHostKey, server.ID)
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
	runtimeDir := path.Join("/run/user", uid)
	installDir := path.Join(homeDir, agentInstallDirName)
	unitPath := path.Join(homeDir, dotConfigDir, "systemd", "user", agentSystemdUnit)

	stopCmd := fmt.Sprintf("XDG_RUNTIME_DIR=%s systemctl --user stop %s 2>/dev/null || true", runtimeDir, agentSystemdUnit)
	_ = runCommand(client, stopCmd)
	disableCmd := fmt.Sprintf("XDG_RUNTIME_DIR=%s systemctl --user disable %s 2>/dev/null || true", runtimeDir, agentSystemdUnit)
	_ = runCommand(client, disableCmd)
	rmUnitCmd := fmt.Sprintf("rm -f %q", unitPath)
	_ = runCommand(client, rmUnitCmd)
	rmInstallCmd := fmt.Sprintf("rm -rf %q", installDir)
	_ = runCommand(client, rmInstallCmd)

	log.Info("deprovision completed", "serverId", server.ID, "host", server.Host)
	return nil
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

var runCommandFn = runCommandDefault
var runCommandOutputFn = runCommandOutputDefault

func runCommand(client *ssh.Client, cmd string) error {
	return runCommandFn(client, cmd)
}

func runCommandOutput(client *ssh.Client, cmd string) (string, error) {
	return runCommandOutputFn(client, cmd)
}

func runCommandWithTimeout(client *ssh.Client, cmd string, timeout time.Duration) error {
	done := make(chan error, 1)
	go func() {
		done <- runCommand(client, cmd)
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("command timed out after %s: %s", timeout, cmd)
	}
}

func runPrivilegedCommandWithTimeout(client *ssh.Client, uid string, password string, cmd string, timeout time.Duration) error {
	done := make(chan error, 1)
	go func() {
		done <- runPrivilegedCommand(client, uid, password, cmd)
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("command timed out after %s: %s", timeout, cmd)
	}
}

func runCommandDefault(client *ssh.Client, cmd string) error {
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

func runCommandOutputDefault(client *ssh.Client, cmd string) (string, error) {
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
