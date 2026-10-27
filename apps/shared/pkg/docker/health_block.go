package docker

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const healthLogLines = 40

// BlocksHealthProbe reports container states that will not start serving HTTP
// on their own. A process that is still booting stays "running" and is probed.
func BlocksHealthProbe(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "restarting", "exited", "dead":
		return true
	default:
		return false
	}
}

// HealthBlockedError means the container left the running state during a deploy
// health check. Logs are the tail captured before any rollback replaces it.
type HealthBlockedError struct {
	Name   string
	Status string
	Logs   string
	LogErr error
}

func (e *HealthBlockedError) Error() string {
	base := fmt.Sprintf("container %s is %s", e.Name, e.Status)
	logs := TailLogLines(e.Logs, healthLogLines)
	if logs != "" {
		return base + "\n" + logs
	}
	if e.LogErr != nil {
		return fmt.Sprintf("%s\nfailed to read container logs: %s", base, e.LogErr.Error())
	}
	return base
}

// HealthBlock returns a HealthBlockedError when the container is restarting or
// has exited. Inspect failures are ignored so a transient docker error does not
// fail the deploy ahead of the HTTP probe.
func (d *Client) HealthBlock(ctx context.Context, containerName string) error {
	inspected, err := d.InspectContainer(ctx, containerName)
	if err != nil || inspected == nil || !BlocksHealthProbe(inspected.Status) {
		return nil
	}
	logs, logErr := d.ContainerLogs(ctx, containerName, healthLogLines, nil)
	return &HealthBlockedError{
		Name:   containerName,
		Status: inspected.Status,
		Logs:   logs,
		LogErr: logErr,
	}
}

// AnnotateHealthError appends a log tail to a probe failure. A HealthBlockedError
// already carries that tail, captured before rollback.
func (d *Client) AnnotateHealthError(ctx context.Context, containerName string, err error) error {
	if err == nil {
		return nil
	}
	var blocked *HealthBlockedError
	if errors.As(err, &blocked) {
		return err
	}
	logs, logErr := d.ContainerLogs(ctx, containerName, healthLogLines, nil)
	if logErr != nil {
		return fmt.Errorf("%w\nfailed to read container logs: %s", err, logErr.Error())
	}
	tail := TailLogLines(logs, healthLogLines)
	if tail == "" {
		return err
	}
	return fmt.Errorf("%w\ncontainer logs:\n%s", err, tail)
}

func TailLogLines(text string, maxLines int) string {
	if maxLines <= 0 || text == "" {
		return ""
	}
	text = strings.TrimRight(text, "\n")
	lines := strings.Split(text, "\n")
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return strings.Join(lines, "\n")
}
