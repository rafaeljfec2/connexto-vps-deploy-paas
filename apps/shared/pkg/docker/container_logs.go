package docker

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// containerLogsFromExec turns a finished `docker logs` invocation into the
// application log text. Failures stay errors (never empty “success” logs).
// Success merges both CLI pipes so container stderr is not dropped.
func containerLogsFromExec(containerName, stdout, stderr string, err error) (string, error) {
	if err != nil {
		stderrLower := strings.ToLower(stderr)
		if strings.Contains(stderrLower, errNoSuchContainer) {
			return "", fmt.Errorf("container not found: %s", containerName)
		}
		return "", fmt.Errorf("failed to get container logs: %w", err)
	}
	return mergeContainerLogStreams(stdout, stderr), nil
}

type containerLogLine struct {
	text string
	ts   time.Time
	ok   bool
	idx  int
}

// mergeContainerLogStreams joins the stdout and stderr captured from a
// successful `docker logs --timestamps` run. Docker routes the container's
// stderr to the CLI stderr pipe, so Nest/Node crash stacks must be kept even
// when the text contains the word "error". Lines with an RFC3339 prefix are
// ordered chronologically so a late TailLogLines cut keeps the crash, not a
// long trailing stdout stretch.
func mergeContainerLogStreams(stdout, stderr string) string {
	lines := collectContainerLogLines(stdout, stderr)
	if len(lines) == 0 {
		return ""
	}
	sort.SliceStable(lines, func(i, j int) bool {
		a, b := lines[i], lines[j]
		if a.ok && b.ok && !a.ts.Equal(b.ts) {
			return a.ts.Before(b.ts)
		}
		if a.ok != b.ok {
			return a.ok
		}
		return a.idx < b.idx
	})
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = line.text
	}
	return strings.Join(out, "\n")
}

func collectContainerLogLines(parts ...string) []containerLogLine {
	var lines []containerLogLine
	for _, part := range parts {
		part = strings.TrimRight(part, "\n")
		if part == "" {
			continue
		}
		for _, text := range strings.Split(part, "\n") {
			ts, ok := parseContainerLogTimestamp(text)
			lines = append(lines, containerLogLine{
				text: text,
				ts:   ts,
				ok:   ok,
				idx:  len(lines),
			})
		}
	}
	return lines
}

func parseContainerLogTimestamp(line string) (time.Time, bool) {
	space := strings.IndexByte(line, ' ')
	if space <= 0 {
		return time.Time{}, false
	}
	raw := line[:space]
	if ts, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return ts, true
	}
	if ts, err := time.Parse(time.RFC3339, raw); err == nil {
		return ts, true
	}
	return time.Time{}, false
}
