package deploy

import (
	"regexp"
	"strings"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
)

// dockerTimestampPrefix matches the leading RFC3339 stamp from `docker logs --timestamps`.
var dockerTimestampPrefix = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z? `)

// firstNonEmptyLine returns the first non-empty trimmed line of s, or s trimmed if none.
func firstNonEmptyLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}
	return strings.TrimSpace(s)
}

// splitErrorLogLines splits a multiline error into non-empty lines and strips
// leading Docker timestamps so each line is readable in the deploy log stream.
// Leading indentation on stack frames is preserved; only the timestamp and
// trailing whitespace are removed.
func splitErrorLogLines(message string) []string {
	raw := strings.Split(message, "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		cleaned := strings.TrimRight(dockerTimestampPrefix.ReplaceAllString(line, ""), " \t\r")
		if strings.TrimSpace(cleaned) == "" {
			continue
		}
		out = append(out, cleaned)
	}
	return out
}

// emitErrorLines sends each non-empty line of message under the given stage/level.
func emitErrorLines(
	emit func(pb.DeployStage, pb.DeployLogLevel, string),
	stage pb.DeployStage,
	level pb.DeployLogLevel,
	message string,
) {
	if emit == nil {
		return
	}
	lines := splitErrorLogLines(message)
	if len(lines) == 0 {
		emit(stage, level, strings.TrimSpace(message))
		return
	}
	for _, line := range lines {
		emit(stage, level, line)
	}
}
