package provisioner

import (
	"bytes"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

func runPrivilegedCommand(client *ssh.Client, uid string, password string, cmd string) error {
	if uid == "0" {
		return runCommand(client, cmd)
	}
	if password != "" {
		return runSudoWithPassword(client, cmd, password)
	}
	return runCommand(client, fmt.Sprintf("sudo -n %s", cmd))
}

func runPrivilegedCommandOutput(client *ssh.Client, uid string, password string, cmd string) (string, error) {
	if uid == "0" {
		return runCommandOutput(client, cmd)
	}
	if password != "" {
		return runSudoOutputWithPassword(client, cmd, password)
	}
	return runCommandOutput(client, fmt.Sprintf("sudo -n %s", cmd))
}

var runSudoWithPasswordFn = runSudoWithPasswordDefault

func runSudoWithPassword(client *ssh.Client, cmd string, password string) error {
	return runSudoWithPasswordFn(client, cmd, password)
}

func runSudoWithPasswordDefault(client *ssh.Client, cmd string, password string) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}

	var stderr bytes.Buffer
	session.Stderr = &stderr

	sudoCmd := fmt.Sprintf("sudo -S %s", cmd)
	if err := session.Start(sudoCmd); err != nil {
		return fmt.Errorf("start command: %w", err)
	}

	fmt.Fprintf(stdin, "%s\n", password)
	stdin.Close()

	if err := session.Wait(); err != nil {
		stderrStr := stripSudoPrompt(stderr.String())
		if stderrStr != "" {
			return fmt.Errorf("command failed: %w (stderr: %s)", err, stderrStr)
		}
		return fmt.Errorf("command failed: %w", err)
	}
	return nil
}

var runSudoOutputWithPasswordFn = runSudoOutputWithPasswordDefault

func runSudoOutputWithPassword(client *ssh.Client, cmd string, password string) (string, error) {
	return runSudoOutputWithPasswordFn(client, cmd, password)
}

func runSudoOutputWithPasswordDefault(client *ssh.Client, cmd string, password string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("stdin pipe: %w", err)
	}

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	sudoCmd := fmt.Sprintf("sudo -S %s", cmd)
	if err := session.Start(sudoCmd); err != nil {
		return "", fmt.Errorf("start command: %w", err)
	}

	fmt.Fprintf(stdin, "%s\n", password)
	stdin.Close()

	if err := session.Wait(); err != nil {
		stderrStr := stripSudoPrompt(stderr.String())
		return "", fmt.Errorf("command failed: %w (stderr: %s)", err, stderrStr)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func stripSudoPrompt(s string) string {
	lines := strings.Split(s, "\n")
	var filtered []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "[sudo]") {
			continue
		}
		filtered = append(filtered, trimmed)
	}
	return strings.Join(filtered, "\n")
}
