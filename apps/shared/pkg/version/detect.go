package version

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func DetectAppVersion(runtime, appDir string) string {
	switch strings.ToLower(runtime) {
	case "node", "nodejs", "node.js":
		return versionFromJSON(filepath.Join(appDir, "package.json"))
	case "php":
		return versionFromJSON(filepath.Join(appDir, "composer.json"))
	case "python", "python3":
		return detectPythonVersion(appDir)
	case "rust":
		return extractRegex(filepath.Join(appDir, "Cargo.toml"), `(?m)^\s*version\s*=\s*"([^"]+)"`)
	case "java":
		return detectJavaVersion(appDir)
	case "ruby":
		return detectRubyVersion(appDir)
	case "dotnet":
		return detectDotnetVersion(appDir)
	case "elixir":
		return extractRegex(filepath.Join(appDir, "mix.exs"), `version:\s*"([^"]+)"`)
	case "go", "golang":
		return readVersionFile(appDir)
	default:
		return readVersionFile(appDir)
	}
}

func versionFromJSON(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var pkg struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(data, &pkg) != nil {
		return ""
	}
	return pkg.Version
}

func detectPythonVersion(appDir string) string {
	if v := extractRegex(filepath.Join(appDir, "pyproject.toml"), `(?m)^\s*version\s*=\s*"([^"]+)"`); v != "" {
		return v
	}
	if v := extractRegex(filepath.Join(appDir, "setup.cfg"), `(?m)^\s*version\s*=\s*(.+)`); v != "" {
		return v
	}
	return extractRegex(filepath.Join(appDir, "setup.py"), `version\s*=\s*["']([^"']+)["']`)
}

func detectJavaVersion(appDir string) string {
	if v := extractRegex(filepath.Join(appDir, "pom.xml"), `<version>([^<]+)</version>`); v != "" {
		return v
	}
	return extractRegex(filepath.Join(appDir, "build.gradle"), `(?m)^\s*version\s*[=:]\s*['"]([^'"]+)['"]`)
}

func detectRubyVersion(appDir string) string {
	entries, err := os.ReadDir(appDir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".gemspec") {
			if v := extractRegex(filepath.Join(appDir, entry.Name()), `\.version\s*=\s*["']([^"']+)["']`); v != "" {
				return v
			}
		}
	}
	return ""
}

func detectDotnetVersion(appDir string) string {
	entries, err := os.ReadDir(appDir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".csproj") {
			if v := extractRegex(filepath.Join(appDir, entry.Name()), `<Version>([^<]+)</Version>`); v != "" {
				return v
			}
		}
	}
	return ""
}

func extractRegex(path, pattern string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(pattern)
	match := re.FindSubmatch(data)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(string(match[1]))
}

func readVersionFile(appDir string) string {
	data, err := os.ReadFile(filepath.Join(appDir, "VERSION"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
