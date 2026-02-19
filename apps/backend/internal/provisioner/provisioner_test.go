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
	errWantActive          = "expected 'active', got %q"
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
	origSudo := runSudoWithPasswordFn
	origSudoOut := runSudoOutputWithPasswordFn

	cmdHandler := func(cmd string) {
		m.commands = append(m.commands, cmd)
	}

	cmdError := func(cmd string) error {
		for substr, err := range m.errors {
			if strings.Contains(cmd, substr) {
				return err
			}
		}
		return nil
	}

	outputHandler := func(cmd string) (string, error) {
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

	runCommandFn = func(_ *ssh.Client, cmd string) error {
		cmdHandler(cmd)
		return cmdError(cmd)
	}

	runCommandOutputFn = func(_ *ssh.Client, cmd string) (string, error) {
		return outputHandler(cmd)
	}

	runSudoWithPasswordFn = func(_ *ssh.Client, cmd string, _ string) error {
		cmdHandler(cmd)
		return cmdError(cmd)
	}

	runSudoOutputWithPasswordFn = func(_ *ssh.Client, cmd string, _ string) (string, error) {
		return outputHandler(cmd)
	}

	writeRemoteFileViaSSHFn = func(_ *ssh.Client, _ string, _ string, _ string, _ []byte) error {
		return nil
	}

	return func() {
		runCommandFn = origCmd
		runCommandOutputFn = origCmdOut
		writeRemoteFileViaSSHFn = origWriteFile
		runSudoWithPasswordFn = origSudo
		runSudoOutputWithPasswordFn = origSudoOut
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
		requireNoError(t, p.provisionDocker(nil, uidRoot, "", noopStep, noopLog))

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
		requireNoError(t, p.provisionDocker(nil, uidRoot, "", noopStep, noopLog))

		if !mock.hasCommand(cmdGetDockerCom) {
			t.Error("expected docker install command")
		}
	})

	t.Run("NotInstalledInstallsWithSudo", func(t *testing.T) {
		mock := setupDockerMock(t, false, true)
		mock.setResponse("whoami", "deploy")
		defer mock.install(t)()

		p := newTestProvisioner()
		requireNoError(t, p.provisionDocker(nil, uidNonRoot, "testpass", noopStep, noopLog))

		if !mock.hasCommand(cmdGetDockerCom) {
			t.Error("expected docker install command for non-root user")
		}
		if !mock.hasCommand("usermod") {
			t.Error("expected usermod -aG docker command for non-root user")
		}
	})

	t.Run("DaemonNotRunningStartsIt", func(t *testing.T) {
		mock := setupDockerMock(t, true, false)
		defer mock.install(t)()

		p := newTestProvisioner()
		requireNoError(t, p.provisionDocker(nil, uidRoot, "", noopStep, noopLog))

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
		requireNoError(t, p.provisionTraefik(nil, uidRoot, "", testEmailDefault, noopStep, noopLog))

		if mock.hasCommand(cmdDockerRun) {
			t.Error("should not start traefik when already running with correct version")
		}
	})

	t.Run("AlreadyRunningOldVersionUpgrades", func(t *testing.T) {
		mock := setupTraefikMock(t, true, "traefik:v2.10")
		defer mock.install(t)()

		p := newTestProvisioner()
		requireNoError(t, p.provisionTraefik(nil, uidRoot, "", testEmailDefault, noopStep, noopLog))

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
		requireNoError(t, p.provisionTraefik(nil, uidRoot, "", testEmailDefault, noopStep, noopLog))

		if !mock.hasCommand("mkdir -p") {
			t.Error("expected mkdir for traefik dirs")
		}
		if !mock.hasCommandWith(cmdDockerRun, traefikImage) {
			t.Errorf("expected docker run with image %s", traefikImage)
		}
	})
}

func runPrivilegedAndGetCmd(t *testing.T, uid, password, cmd string) string {
	t.Helper()
	mock := newCommandMock()
	defer mock.install(t)()
	requireNoError(t, runPrivilegedCommand(nil, uid, password, cmd))
	requireSingleCommand(t, mock.commands)
	return mock.commands[0]
}

func runPrivilegedOutputAndGetCmd(t *testing.T, uid, password, cmd, expectedOut string) string {
	t.Helper()
	mock := newCommandMock()
	mock.setResponse(cmd, expectedOut)
	defer mock.install(t)()
	out, err := runPrivilegedCommandOutput(nil, uid, password, cmd)
	requireNoError(t, err)
	if out != expectedOut {
		t.Errorf(errWantActive, out)
	}
	requireSingleCommand(t, mock.outputCommands)
	return mock.outputCommands[0]
}

func requireSingleCommand(t *testing.T, cmds []string) {
	t.Helper()
	if len(cmds) != 1 {
		t.Fatalf(errWantOneCmd, len(cmds))
	}
}

func assertNoSudo(t *testing.T, captured string) {
	t.Helper()
	if strings.Contains(captured, "sudo") {
		t.Error("should not contain sudo")
	}
}

func assertHasSudoN(t *testing.T, captured string) {
	t.Helper()
	if !strings.Contains(captured, "sudo -n") {
		t.Error("should contain sudo -n")
	}
}

func TestPrivilegedCommand(t *testing.T) {
	t.Run("RootExecutesDirect", func(t *testing.T) {
		captured := runPrivilegedAndGetCmd(t, uidRoot, "", cmdSystemctlStart)
		assertNoSudo(t, captured)
		if captured != cmdSystemctlStart {
			t.Errorf("expected exact command, got %q", captured)
		}
	})

	t.Run("NonRootUsesSudoN", func(t *testing.T) {
		captured := runPrivilegedAndGetCmd(t, uidNonRoot, "", cmdSystemctlStart)
		assertHasSudoN(t, captured)
		assertContains(t, captured, cmdSystemctlStart)
	})

	t.Run("NonRootWithPasswordUsesSudoS", func(t *testing.T) {
		captured := runPrivilegedAndGetCmd(t, uidNonRoot, "testpass", cmdSystemctlStart)
		if captured != cmdSystemctlStart {
			t.Errorf("sudo -S path should pass raw command, got %q", captured)
		}
	})
}

func TestPrivilegedCommandOutput(t *testing.T) {
	t.Run("RootExecutesDirect", func(t *testing.T) {
		captured := runPrivilegedOutputAndGetCmd(t, uidRoot, "", cmdSystemctlIsActive, "active")
		assertNoSudo(t, captured)
	})

	t.Run("NonRootUsesSudoN", func(t *testing.T) {
		captured := runPrivilegedOutputAndGetCmd(t, uidNonRoot, "", cmdSystemctlIsActive, "active")
		assertHasSudoN(t, captured)
	})

	t.Run("NonRootWithPasswordUsesSudoS", func(t *testing.T) {
		captured := runPrivilegedOutputAndGetCmd(t, uidNonRoot, "testpass", cmdSystemctlIsActive, "active")
		assertNoSudo(t, captured)
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
