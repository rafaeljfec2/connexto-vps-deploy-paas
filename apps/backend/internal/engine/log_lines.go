package engine

import "strings"

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

// splitLogMessage splits a deploy log payload into non-empty trimmed lines so
// each can be persisted/emitted with its own timestamp and stage prefix.
func splitLogMessage(message string) []string {
	raw := strings.Split(message, "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}
