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

var noopStep = func(string, string, string) {}
var noopLog = func(string) {}

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

func TestProvisionDocker(t *testing.T) {
	t.Run("AlreadyInstalled", func(t *testing.T) {
		mock := newCommandMock()
		mock.setResponse(cmdDockerVersion, mockDockerVersionOut)
		mock.setResponse(cmdSystemctlIsActive, "active")
		cleanup := mock.install(t)
		defer cleanup()

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
		mock := newCommandMock()
		mock.setError(cmdDockerVersion, fmt.Errorf(errNotFound))
		mock.setResponse(cmdSystemctlIsActive, "active")
		cleanup := mock.install(t)
		defer cleanup()

		p := newTestProvisioner()
		requireNoError(t, p.provisionDocker(nil, uidRoot, noopStep, noopLog))

		if !mock.hasCommand(cmdGetDockerCom) {
			t.Error("expected docker install command")
		}
	})

	t.Run("NotInstalledInstallsWithSudo", func(t *testing.T) {
		mock := newCommandMock()
		mock.setError(cmdDockerVersion, fmt.Errorf(errNotFound))
		mock.setResponse("whoami", "deploy")
		mock.setResponse(cmdSystemctlIsActive, "active")
		cleanup := mock.install(t)
		defer cleanup()

		p := newTestProvisioner()
		requireNoError(t, p.provisionDocker(nil, uidNonRoot, noopStep, noopLog))

		hasSudo := false
		for _, cmd := range mock.commands {
			if strings.Contains(cmd, "sudo") && strings.Contains(cmd, cmdGetDockerCom) {
				hasSudo = true
				break
			}
		}
		if !hasSudo {
			t.Error("expected sudo docker install command for non-root user")
		}

		hasUsermod := false
		for _, cmd := range mock.commands {
			if strings.Contains(cmd, "usermod") && strings.Contains(cmd, "docker") {
				hasUsermod = true
				break
			}
		}
		if !hasUsermod {
			t.Error("expected usermod -aG docker command for non-root user")
		}
	})

	t.Run("DaemonNotRunningStartsIt", func(t *testing.T) {
		mock := newCommandMock()
		mock.setResponse(cmdDockerVersion, mockDockerVersionOut)
		mock.setError(cmdSystemctlIsActive, fmt.Errorf("inactive"))
		cleanup := mock.install(t)
		defer cleanup()

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

func TestProvisionTraefik(t *testing.T) {
	t.Run("AlreadyRunningCorrectVersion", func(t *testing.T) {
		mock := newCommandMock()
		mock.setResponse(mockKeyStateRunning, "true")
		mock.setResponse(mockKeyConfigImage, traefikImage)
		cleanup := mock.install(t)
		defer cleanup()

		p := newTestProvisioner()
		requireNoError(t, p.provisionTraefik(nil, uidRoot, testEmailDefault, noopStep, noopLog))

		if mock.hasCommand(cmdDockerRun) {
			t.Error("should not start traefik when already running with correct version")
		}
	})

	t.Run("AlreadyRunningOldVersionUpgrades", func(t *testing.T) {
		mock := newCommandMock()
		mock.setResponse(mockKeyStateRunning, "true")
		mock.setResponse(mockKeyConfigImage, "traefik:v2.10")
		cleanup := mock.install(t)
		defer cleanup()

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
		mock := newCommandMock()
		mock.setError(mockKeyStateRunning, fmt.Errorf(errNotFound))
		cleanup := mock.install(t)
		defer cleanup()

		p := newTestProvisioner()
		requireNoError(t, p.provisionTraefik(nil, uidRoot, testEmailDefault, noopStep, noopLog))

		if !mock.hasCommand("mkdir -p") {
			t.Error("expected mkdir for traefik dirs")
		}
		if !mock.hasCommand(cmdDockerRun) {
			t.Error("expected docker run command for traefik")
		}

		hasTraefikImage := false
		for _, cmd := range mock.commands {
			if strings.Contains(cmd, cmdDockerRun) && strings.Contains(cmd, traefikImage) {
				hasTraefikImage = true
				break
			}
		}
		if !hasTraefikImage {
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
		config := buildTraefikConfig(testEmailDefault)
		configStr := string(config)

		if !strings.Contains(configStr, "email: "+testEmailDefault) {
			t.Errorf("expected config to contain acme email %q, got:\n%s", testEmailDefault, configStr)
		}
	})

	t.Run("ContainsEntryPoints", func(t *testing.T) {
		configStr := string(buildTraefikConfig(testEmailAdmin))

		expectedPorts := []string{":80", ":443", ":50051", ":8081"}
		for _, port := range expectedPorts {
			if !strings.Contains(configStr, port) {
				t.Errorf("expected config to contain port %q, got:\n%s", port, configStr)
			}
		}
	})

	t.Run("ContainsDockerProvider", func(t *testing.T) {
		configStr := string(buildTraefikConfig(testEmailAdmin))

		if !strings.Contains(configStr, "unix:///var/run/docker.sock") {
			t.Errorf("expected config to contain docker socket endpoint, got:\n%s", configStr)
		}
		if !strings.Contains(configStr, "network: "+dockerNetworkName) {
			t.Errorf("expected config to contain network %q, got:\n%s", dockerNetworkName, configStr)
		}
	})

	t.Run("ContainsLetsencrypt", func(t *testing.T) {
		configStr := string(buildTraefikConfig(testEmailAdmin))

		if !strings.Contains(configStr, "letsencrypt") {
			t.Errorf("expected config to contain letsencrypt resolver, got:\n%s", configStr)
		}
		if !strings.Contains(configStr, "httpChallenge") {
			t.Errorf("expected config to contain httpChallenge, got:\n%s", configStr)
		}
	})

	t.Run("DisableExposedByDefault", func(t *testing.T) {
		configStr := string(buildTraefikConfig(testEmailAdmin))

		if !strings.Contains(configStr, "exposedByDefault: false") {
			t.Errorf("expected config to disable exposedByDefault, got:\n%s", configStr)
		}
	})
}
