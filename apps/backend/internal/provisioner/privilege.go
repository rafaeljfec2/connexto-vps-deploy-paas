package provisioner

import (
	"fmt"

	"golang.org/x/crypto/ssh"
)

func runPrivilegedCommand(client *ssh.Client, uid string, cmd string) error {
	if uid == "0" {
		return runCommand(client, cmd)
	}
	sudoCmd := fmt.Sprintf("sudo -n %s", cmd)
	return runCommand(client, sudoCmd)
}

func runPrivilegedCommandOutput(client *ssh.Client, uid string, cmd string) (string, error) {
	if uid == "0" {
		return runCommandOutput(client, cmd)
	}
	sudoCmd := fmt.Sprintf("sudo -n %s", cmd)
	return runCommandOutput(client, sudoCmd)
}
