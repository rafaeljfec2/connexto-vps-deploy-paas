package provisioner

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

const (
	cmdDockerVersion       = "docker --version"
	cmdSystemctlIsActive   = "systemctl is-active docker"
	cmdSystemctlStart      = "systemctl start docker"
	cmdGetDockerCom        = "get.docker.com"
	cmdDockerRun           = "docker run"
	cmdDockerPull          = "docker pull"
	cmdDockerNetworkInspct = "docker network inspect"
	mockKeyStateRunning    = "State.Running"
	mockKeyConfigImage     = "Config.Image"
	mockDockerVersionOut   = "Docker version 24.0.7"
	testEmailDefault       = "test@example.com"
	testEmailAdmin         = "admin@example.com"
	errNotFound            = "not found"
	uidRoot                = "0"
	uidNonRoot             = "1000"
	errWantOneCmd          = "expected 1 command, got %d"
)

type commandMock struct {
	commands       []string
	outputCommands []string
	responses      map[string]string
	errors         map[string]error
}

func newCommandMock() *commandMock {
	return &commandMock{
		responses: make(map[string]string),
		errors:    make(map[string]error),
	}
}

func (m *commandMock) setResponse(cmdSubstr string, output string) {
	m.responses[cmdSubstr] = output
}

func (m *commandMock) setError(cmdSubstr string, err error) {
	m.errors[cmdSubstr] = err
}

func (m *commandMock) install(t *testing.T) func() {
	t.Helper()
	origCmd := runCommandFn
	origCmdOut := runCommandOutputFn
	origWriteFile := writeRemoteFileViaSSHFn

	runCommandFn = func(_ *ssh.Client, cmd string) error {
		m.commands = append(m.commands, cmd)
		for substr, err := range m.errors {
			if strings.Contains(cmd, substr) {
				return err
			}
		}
		return nil
	}

	runCommandOutputFn = func(_ *ssh.Client, cmd string) (string, error) {
		m.outputCommands = append(m.outputCommands, cmd)
		for substr, err := range m.errors {
			if strings.Contains(cmd, substr) {
				return "", err
			}
		}
		for substr, resp := range m.responses {
			if strings.Contains(cmd, substr) {
				return resp, nil
			}
		}
		return "", nil
	}

	writeRemoteFileViaSSHFn = func(_ *ssh.Client, _ string, _ string, _ []byte) error {
		return nil
	}

	return func() {
		runCommandFn = origCmd
		runCommandOutputFn = origCmdOut
		writeRemoteFileViaSSHFn = origWriteFile
	}
}

func (m *commandMock) hasCommand(substr string) bool {
	for _, cmd := range m.commands {
		if strings.Contains(cmd, substr) {
			return true
		}
	}
	return false
}

func (m *commandMock) hasOutputCommand(substr string) bool {
	for _, cmd := range m.outputCommands {
		if strings.Contains(cmd, substr) {
			return true
		}
	}
	return false
}

func (m *commandMock) hasCommandWith(substrs ...string) bool {
	for _, cmd := range m.commands {
		if containsAll(cmd, substrs) {
			return true
		}
	}
	return false
}

func containsAll(s string, substrs []string) bool {
	for _, sub := range substrs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

var noopStep = func(string, string, string) { /* intentional no-op for tests */ }
var noopLog = func(string) { /* intentional no-op for tests */ }

func newTestProvisioner() *SSHProvisioner {
	return &SSHProvisioner{
		cfg: SSHProvisionerConfig{
			Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		},
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected to contain %q, got:\n%s", needle, haystack)
	}
}

func requireTraefikConfig(t *testing.T, email string) string {
	t.Helper()
	config, err := buildTraefikConfig(email)
	requireNoError(t, err)
	return string(config)
}

func setupDockerMock(t *testing.T, dockerInstalled bool, daemonActive bool) *commandMock {
	t.Helper()
	mock := newCommandMock()
	if dockerInstalled {
		mock.setResponse(cmdDockerVersion, mockDockerVersionOut)
	} else {
		mock.setError(cmdDockerVersion, fmt.Errorf(errNotFound))
	}
	if daemonActive {
		mock.setResponse(cmdSystemctlIsActive, "active")
	} else {
		mock.setError(cmdSystemctlIsActive, fmt.Errorf("inactive"))
	}
	return mock
}

func TestProvisionDocker(t *testing.T) {
	t.Run("AlreadyInstalled", func(t *testing.T) {
		mock := setupDockerMock(t, true, true)
		defer mock.install(t)()

		p := newTestProvisioner()
		requireNoError(t, p.provisionDocker(nil, uidRoot, noopStep, noopLog))

		if !mock.hasOutputCommand(cmdDockerVersion) {
			t.Error("expected docker --version check")
		}
		if mock.hasCommand(cmdGetDockerCom) {
			t.Error("should not install docker when already present")
		}
	})

	t.Run("NotInstalledInstallsAsRoot", func(t *testing.T) {
		mock := setupDockerMock(t, false, true)
		defer mock.install(t)()

		p := newTestProvisioner()
		requireNoError(t, p.provisionDocker(nil, uidRoot, noopStep, noopLog))

		if !mock.hasCommand(cmdGetDockerCom) {
			t.Error("expected docker install command")
		}
	})

	t.Run("NotInstalledInstallsWithSudo", func(t *testing.T) {
		mock := setupDockerMock(t, false, true)
		mock.setResponse("whoami", "deploy")
		defer mock.install(t)()

		p := newTestProvisioner()
		requireNoError(t, p.provisionDocker(nil, uidNonRoot, noopStep, noopLog))

		if !mock.hasCommandWith("sudo", cmdGetDockerCom) {
			t.Error("expected sudo docker install command for non-root user")
		}
		if !mock.hasCommandWith("usermod", "docker") {
			t.Error("expected usermod -aG docker command for non-root user")
		}
	})

	t.Run("DaemonNotRunningStartsIt", func(t *testing.T) {
		mock := setupDockerMock(t, true, false)
		defer mock.install(t)()

		p := newTestProvisioner()
		requireNoError(t, p.provisionDocker(nil, uidRoot, noopStep, noopLog))

		if !mock.hasCommand(cmdSystemctlStart) {
			t.Error("expected systemctl start docker command")
		}
	})
}

func TestProvisionDockerNetwork(t *testing.T) {
	t.Run("AlreadyExists", func(t *testing.T) {
		mock := newCommandMock()
		cleanup := mock.install(t)
		defer cleanup()

		p := newTestProvisioner()
		requireNoError(t, p.provisionDockerNetwork(nil, noopStep, noopLog))

		if mock.hasCommand("docker network create") {
			t.Error("should not create network when it already exists")
		}
	})

	t.Run("NotExistsCreatesIt", func(t *testing.T) {
		mock := newCommandMock()
		mock.setError(cmdDockerNetworkInspct, fmt.Errorf(errNotFound))
		cleanup := mock.install(t)
		defer cleanup()

		p := newTestProvisioner()
		requireNoError(t, p.provisionDockerNetwork(nil, noopStep, noopLog))

		if !mock.hasCommand("docker network create paasdeploy") {
			t.Error("expected docker network create paasdeploy command")
		}
	})
}

func setupTraefikMock(t *testing.T, running bool, image string) *commandMock {
	t.Helper()
	mock := newCommandMock()
	if running {
		mock.setResponse(mockKeyStateRunning, "true")
		mock.setResponse(mockKeyConfigImage, image)
	} else {
		mock.setError(mockKeyStateRunning, fmt.Errorf(errNotFound))
	}
	return mock
}

func TestProvisionTraefik(t *testing.T) {
	t.Run("AlreadyRunningCorrectVersion", func(t *testing.T) {
		mock := setupTraefikMock(t, true, traefikImage)
		defer mock.install(t)()

		p := newTestProvisioner()
		requireNoError(t, p.provisionTraefik(nil, uidRoot, testEmailDefault, noopStep, noopLog))

		if mock.hasCommand(cmdDockerRun) {
			t.Error("should not start traefik when already running with correct version")
		}
	})

	t.Run("AlreadyRunningOldVersionUpgrades", func(t *testing.T) {
		mock := setupTraefikMock(t, true, "traefik:v2.10")
		defer mock.install(t)()

		p := newTestProvisioner()
		requireNoError(t, p.provisionTraefik(nil, uidRoot, testEmailDefault, noopStep, noopLog))

		if !mock.hasCommand(cmdDockerRun) {
			t.Error("should upgrade traefik when running old version")
		}
		if !mock.hasCommand(cmdDockerPull) {
			t.Error("should pull new image before upgrade")
		}
	})

	t.Run("NotRunningInstallsAndStarts", func(t *testing.T) {
		mock := setupTraefikMock(t, false, "")
		defer mock.install(t)()

		p := newTestProvisioner()
		requireNoError(t, p.provisionTraefik(nil, uidRoot, testEmailDefault, noopStep, noopLog))

		if !mock.hasCommand("mkdir -p") {
			t.Error("expected mkdir for traefik dirs")
		}
		if !mock.hasCommandWith(cmdDockerRun, traefikImage) {
			t.Errorf("expected docker run with image %s", traefikImage)
		}
	})
}

func TestPrivilegedCommand(t *testing.T) {
	t.Run("RootExecutesDirect", func(t *testing.T) {
		mock := newCommandMock()
		cleanup := mock.install(t)
		defer cleanup()

		requireNoError(t, runPrivilegedCommand(nil, uidRoot, cmdSystemctlStart))

		if len(mock.commands) != 1 {
			t.Fatalf(errWantOneCmd, len(mock.commands))
		}
		if strings.Contains(mock.commands[0], "sudo") {
			t.Error("root should not use sudo")
		}
		if mock.commands[0] != cmdSystemctlStart {
			t.Errorf("expected exact command, got %q", mock.commands[0])
		}
	})

	t.Run("NonRootUsesSudo", func(t *testing.T) {
		mock := newCommandMock()
		cleanup := mock.install(t)
		defer cleanup()

		requireNoError(t, runPrivilegedCommand(nil, uidNonRoot, cmdSystemctlStart))

		if len(mock.commands) != 1 {
			t.Fatalf(errWantOneCmd, len(mock.commands))
		}
		if !strings.Contains(mock.commands[0], "sudo -n") {
			t.Error("non-root should use sudo -n")
		}
		if !strings.Contains(mock.commands[0], cmdSystemctlStart) {
			t.Error("command should contain original command")
		}
	})
}

func TestPrivilegedCommandOutput(t *testing.T) {
	t.Run("RootExecutesDirect", func(t *testing.T) {
		mock := newCommandMock()
		mock.setResponse(cmdSystemctlIsActive, "active")
		cleanup := mock.install(t)
		defer cleanup()

		out, err := runPrivilegedCommandOutput(nil, uidRoot, cmdSystemctlIsActive)
		requireNoError(t, err)
		if out != "active" {
			t.Errorf("expected 'active', got %q", out)
		}

		if len(mock.outputCommands) != 1 {
			t.Fatalf(errWantOneCmd, len(mock.outputCommands))
		}
		if strings.Contains(mock.outputCommands[0], "sudo") {
			t.Error("root should not use sudo")
		}
	})

	t.Run("NonRootUsesSudo", func(t *testing.T) {
		mock := newCommandMock()
		mock.setResponse(cmdSystemctlIsActive, "active")
		cleanup := mock.install(t)
		defer cleanup()

		out, err := runPrivilegedCommandOutput(nil, uidNonRoot, cmdSystemctlIsActive)
		requireNoError(t, err)
		if out != "active" {
			t.Errorf("expected 'active', got %q", out)
		}

		if len(mock.outputCommands) != 1 {
			t.Fatalf(errWantOneCmd, len(mock.outputCommands))
		}
		if !strings.Contains(mock.outputCommands[0], "sudo -n") {
			t.Error("non-root should use sudo -n")
		}
	})
}

func TestBuildTraefikConfig(t *testing.T) {
	t.Run("ContainsAcmeEmail", func(t *testing.T) {
		cfg := requireTraefikConfig(t, testEmailDefault)
		assertContains(t, cfg, testEmailDefault)
	})

	t.Run("ContainsEntryPoints", func(t *testing.T) {
		cfg := requireTraefikConfig(t, testEmailAdmin)
		for _, port := range []string{":80", ":443", ":50051", ":8081"} {
			assertContains(t, cfg, port)
		}
	})

	t.Run("ContainsDockerProvider", func(t *testing.T) {
		cfg := requireTraefikConfig(t, testEmailAdmin)
		assertContains(t, cfg, "unix:///var/run/docker.sock")
		assertContains(t, cfg, "network: "+dockerNetworkName)
	})

	t.Run("ContainsLetsencrypt", func(t *testing.T) {
		cfg := requireTraefikConfig(t, testEmailAdmin)
		assertContains(t, cfg, "letsencrypt")
		assertContains(t, cfg, "httpChallenge")
	})

	t.Run("DisableExposedByDefault", func(t *testing.T) {
		cfg := requireTraefikConfig(t, testEmailAdmin)
		assertContains(t, cfg, "exposedByDefault: false")
	})

	t.Run("RejectsInvalidEmail", func(t *testing.T) {
		_, err := buildTraefikConfig("not-an-email")
		if err == nil {
			t.Error("expected error for invalid email")
		}
	})
}
