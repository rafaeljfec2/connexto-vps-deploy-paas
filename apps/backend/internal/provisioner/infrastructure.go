package provisioner

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

var validEmailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

const (
	dockerNetworkName     = "paasdeploy"
	traefikContainerName  = "traefik"
	traefikImage          = "traefik:v3.2"
	traefikConfigDir      = "/opt/traefik"
	traefikLetsencryptDir = "/opt/traefik/letsencrypt"
	traefikConfigPath     = "/opt/traefik/traefik.yml"
)

func (p *SSHProvisioner) provisionDocker(
	client *ssh.Client,
	uid string,
	password string,
	step func(string, string, string),
	logLine func(string),
) error {
	step("docker_check", "running", "Verificando Docker...")
	logLine("Verificando se Docker esta instalado")

	dockerVersion, err := runCommandOutput(client, "docker --version")
	if err == nil {
		logLine(fmt.Sprintf("Docker encontrado: %s", dockerVersion))
		step("docker_check", "ok", "Docker encontrado")
		if err := p.ensureDockerRunning(client, uid, password, step, logLine); err != nil {
			return err
		}
		return p.ensureDockerPlugins(client, uid, password, step, logLine)
	}

	step("docker_install", "running", "Instalando Docker...")
	logLine("Docker nao encontrado, instalando via get.docker.com")

	installCmd := "curl -fsSL https://get.docker.com | sh"
	if err := runPrivilegedCommandWithTimeout(client, uid, password, fmt.Sprintf("sh -c '%s'", installCmd), timeoutDockerInstall); err != nil {
		return fmt.Errorf("install docker: %w", err)
	}

	logLine("Docker instalado com sucesso")
	step("docker_install", "ok", "Docker instalado")

	if uid != "0" {
		currentUser, userErr := runCommandOutput(client, "whoami")
		if userErr == nil && currentUser != "" {
			addGroupCmd := fmt.Sprintf("usermod -aG docker %s", strings.TrimSpace(currentUser))
			_ = runPrivilegedCommand(client, uid, password, addGroupCmd)
		}
	}

	if err := p.ensureDockerRunning(client, uid, password, step, logLine); err != nil {
		return err
	}
	if err := p.ensureDockerPlugins(client, uid, password, step, logLine); err != nil {
		return err
	}
	return nil
}

func (p *SSHProvisioner) ensureDockerPlugins(
	client *ssh.Client,
	uid string,
	password string,
	step func(string, string, string),
	logLine func(string),
) error {
	needCompose := !commandSucceeds(client, "docker compose version")
	needBuildx := !commandSucceeds(client, "docker buildx version")

	if !needCompose && !needBuildx {
		logLine("Docker Compose and Buildx already available")
		return nil
	}

	step("docker_plugins", "running", "Installing Docker plugins...")

	var packages []string
	if needCompose {
		packages = append(packages, "docker-compose-plugin")
	}
	if needBuildx {
		packages = append(packages, "docker-buildx-plugin")
	}

	logLine(fmt.Sprintf("Installing %s", strings.Join(packages, ", ")))
	installCmd := fmt.Sprintf("apt-get update -qq && apt-get install -y -qq %s", strings.Join(packages, " "))
	if err := runPrivilegedCommandWithTimeout(client, uid, password, installCmd, timeoutDockerInstall); err != nil {
		if needCompose {
			return fmt.Errorf("install docker plugins: %w", err)
		}
		logLine("Buildx install failed, builds will use legacy builder")
	}

	step("docker_plugins", "ok", "Docker plugins installed")
	return nil
}

func commandSucceeds(client *ssh.Client, cmd string) bool {
	_, err := runCommandOutput(client, cmd)
	return err == nil
}

func (p *SSHProvisioner) ensureDockerRunning(
	client *ssh.Client,
	uid string,
	password string,
	step func(string, string, string),
	logLine func(string),
) error {
	step("docker_start", "running", "Verificando Docker daemon...")

	out, err := runPrivilegedCommandOutput(client, uid, password, "systemctl is-active docker")
	if err == nil && strings.TrimSpace(out) == "active" {
		logLine("Docker daemon esta ativo")
		step("docker_start", "ok", "Docker daemon ativo")
		return nil
	}

	logLine("Iniciando Docker daemon")
	startCmd := "systemctl start docker && systemctl enable docker"
	if err := runPrivilegedCommandWithTimeout(client, uid, password, startCmd, timeoutDockerCheck); err != nil {
		return fmt.Errorf("start docker daemon: %w", err)
	}

	logLine("Docker daemon iniciado e habilitado")
	step("docker_start", "ok", "Docker daemon iniciado")
	return nil
}

func (p *SSHProvisioner) provisionDockerNetwork(
	client *ssh.Client,
	step func(string, string, string),
	logLine func(string),
) error {
	step("docker_network", "running", "Verificando rede Docker...")
	logLine("Verificando rede " + dockerNetworkName)

	checkCmd := fmt.Sprintf("docker network inspect %s > /dev/null 2>&1", dockerNetworkName)
	if err := runCommand(client, checkCmd); err == nil {
		logLine("Rede " + dockerNetworkName + " ja existe")
		step("docker_network", "ok", "Rede Docker encontrada")
		return nil
	}

	logLine("Criando rede " + dockerNetworkName)
	createCmd := fmt.Sprintf("docker network create %s", dockerNetworkName)
	if err := runCommandWithTimeout(client, createCmd, timeoutNetworkSetup); err != nil {
		return fmt.Errorf("create docker network %s: %w", dockerNetworkName, err)
	}

	logLine("Rede " + dockerNetworkName + " criada")
	step("docker_network", "ok", "Rede Docker criada")
	return nil
}

func (p *SSHProvisioner) provisionTraefik(
	client *ssh.Client,
	uid string,
	password string,
	acmeEmail string,
	step func(string, string, string),
	logLine func(string),
) error {
	step("traefik_check", "running", "Verificando Traefik...")
	logLine("Verificando container Traefik")

	running := p.isTraefikRunning(client)
	if running {
		needsUpgrade := p.traefikNeedsUpgrade(client)
		if !needsUpgrade {
			logLine("Traefik ja esta rodando com versao correta")
			step("traefik_check", "ok", "Traefik encontrado")
			return nil
		}
		logLine(fmt.Sprintf("Traefik rodando com versao diferente da desejada (%s), atualizando", traefikImage))
	}

	step("traefik_install", "running", "Instalando Traefik...")
	logLine("Configurando Traefik")

	if err := p.setupTraefikConfig(client, uid, password, acmeEmail, logLine); err != nil {
		return err
	}

	if err := p.startTraefikContainer(client, uid, logLine); err != nil {
		return err
	}

	step("traefik_install", "ok", "Traefik instalado e iniciado")
	return nil
}

func (p *SSHProvisioner) isTraefikRunning(client *ssh.Client) bool {
	checkCmd := fmt.Sprintf("docker inspect %s --format '{{.State.Running}}' 2>/dev/null", traefikContainerName)
	out, err := runCommandOutput(client, checkCmd)
	return err == nil && strings.TrimSpace(out) == "true"
}

func (p *SSHProvisioner) traefikNeedsUpgrade(client *ssh.Client) bool {
	imageCmd := fmt.Sprintf("docker inspect %s --format '{{.Config.Image}}' 2>/dev/null", traefikContainerName)
	currentImage, err := runCommandOutput(client, imageCmd)
	if err != nil {
		return true
	}
	return strings.TrimSpace(currentImage) != traefikImage
}

func (p *SSHProvisioner) setupTraefikConfig(
	client *ssh.Client,
	uid string,
	password string,
	acmeEmail string,
	logLine func(string),
) error {
	logLine("Criando diretorios do Traefik")
	mkdirCmd := fmt.Sprintf("mkdir -p %s %s", traefikConfigDir, traefikLetsencryptDir)
	if err := runPrivilegedCommand(client, uid, password, mkdirCmd); err != nil {
		return fmt.Errorf("create traefik dirs: %w", err)
	}

	logLine("Escrevendo configuracao do Traefik")
	configContent, err := buildTraefikConfig(acmeEmail)
	if err != nil {
		return fmt.Errorf("build traefik config: %w", err)
	}
	if err := writeRemoteFileViaSSH(client, uid, password, traefikConfigPath, configContent); err != nil {
		return fmt.Errorf("write traefik config: %w", err)
	}

	return nil
}

func (p *SSHProvisioner) startTraefikContainer(
	client *ssh.Client,
	_ string,
	logLine func(string),
) error {
	logLine("Removendo container Traefik anterior (se existir)")
	stopCmd := fmt.Sprintf("docker stop %s 2>/dev/null || true", traefikContainerName)
	_ = runCommand(client, stopCmd)
	rmCmd := fmt.Sprintf("docker rm %s 2>/dev/null || true", traefikContainerName)
	_ = runCommand(client, rmCmd)

	logLine("Baixando imagem Traefik")
	pullCmd := fmt.Sprintf("docker pull %s", traefikImage)
	_ = runCommandWithTimeout(client, pullCmd, timeoutTraefikSetup)

	logLine("Iniciando container Traefik (portas 80, 443, 50051, 8081)")
	runCmd := fmt.Sprintf(
		"docker run -d --name %s --network %s --restart unless-stopped "+
			"-p 80:80 -p 443:443 -p 50051:50051 -p 8081:8081 "+
			"-v /var/run/docker.sock:/var/run/docker.sock:ro "+
			"-v %s:/etc/traefik/traefik.yml:ro "+
			"-v %s:/letsencrypt "+
			"%s",
		traefikContainerName,
		dockerNetworkName,
		traefikConfigPath,
		traefikLetsencryptDir,
		traefikImage,
	)
	if err := runCommandWithTimeout(client, runCmd, timeoutTraefikSetup); err != nil {
		return fmt.Errorf("start traefik container: %w", err)
	}

	logLine("Traefik iniciado com sucesso")
	return nil
}

func sanitizeAcmeEmail(email string) (string, error) {
	email = strings.TrimSpace(email)
	if !validEmailRegex.MatchString(email) {
		return "", fmt.Errorf("invalid ACME email format: %q", email)
	}
	return email, nil
}

func buildTraefikConfig(acmeEmail string) ([]byte, error) {
	sanitized, err := sanitizeAcmeEmail(acmeEmail)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	buf.WriteString("api:\n")
	buf.WriteString("  dashboard: true\n")
	buf.WriteString("  insecure: true\n")
	buf.WriteString("\n")
	buf.WriteString("entryPoints:\n")
	buf.WriteString("  web:\n")
	buf.WriteString("    address: \":80\"\n")
	buf.WriteString("  websecure:\n")
	buf.WriteString("    address: \":443\"\n")
	buf.WriteString("  grpc:\n")
	buf.WriteString("    address: \":50051\"\n")
	buf.WriteString("  traefik:\n")
	buf.WriteString("    address: \":8081\"\n")
	buf.WriteString("\n")
	buf.WriteString("providers:\n")
	buf.WriteString("  docker:\n")
	buf.WriteString("    endpoint: \"unix:///var/run/docker.sock\"\n")
	buf.WriteString("    exposedByDefault: false\n")
	buf.WriteString("    network: " + dockerNetworkName + "\n")
	buf.WriteString("\n")
	buf.WriteString("certificatesResolvers:\n")
	buf.WriteString("  letsencrypt:\n")
	buf.WriteString("    acme:\n")
	buf.WriteString("      email: \"" + sanitized + "\"\n")
	buf.WriteString("      storage: /letsencrypt/acme.json\n")
	buf.WriteString("      httpChallenge:\n")
	buf.WriteString("        entryPoint: web\n")
	buf.WriteString("\n")
	buf.WriteString("log:\n")
	buf.WriteString("  level: INFO\n")
	return buf.Bytes(), nil
}

var writeRemoteFileViaSSHFn = writeRemoteFileViaSSHDefault

func writeRemoteFileViaSSH(client *ssh.Client, uid string, password string, remotePath string, data []byte) error {
	return writeRemoteFileViaSSHFn(client, uid, password, remotePath, data)
}

func writeRemoteFileViaSSHDefault(client *ssh.Client, uid string, password string, remotePath string, data []byte) error {
	if uid == "0" {
		return writeFileDirect(client, remotePath, data)
	}
	if password != "" {
		return writeFileViaTempAndSudo(client, password, remotePath, data)
	}
	return writeFileWithSudoTee(client, remotePath, data)
}

func writeFileDirect(client *ssh.Client, remotePath string, data []byte) error {
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}

	if err := session.Start(fmt.Sprintf("cat > %q", remotePath)); err != nil {
		return fmt.Errorf("start write cmd: %w", err)
	}
	if _, err := stdin.Write(data); err != nil {
		return fmt.Errorf("write data: %w", err)
	}
	stdin.Close()
	return session.Wait()
}

func writeFileWithSudoTee(client *ssh.Client, remotePath string, data []byte) error {
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}

	if err := session.Start(fmt.Sprintf("sudo -n tee %q > /dev/null", remotePath)); err != nil {
		return fmt.Errorf("start write cmd: %w", err)
	}
	if _, err := stdin.Write(data); err != nil {
		return fmt.Errorf("write data: %w", err)
	}
	stdin.Close()
	return session.Wait()
}

func writeFileViaTempAndSudo(client *ssh.Client, password string, remotePath string, data []byte) error {
	tmpPath := fmt.Sprintf("/tmp/.flowdeploy_%d", time.Now().UnixNano())

	if err := writeFileDirect(client, tmpPath, data); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	mvCmd := fmt.Sprintf("mv %q %q", tmpPath, remotePath)
	if err := runSudoWithPassword(client, mvCmd, password); err != nil {
		_ = runCommand(client, fmt.Sprintf("rm -f %q", tmpPath))
		return fmt.Errorf("move file to destination: %w", err)
	}
	return nil
}
