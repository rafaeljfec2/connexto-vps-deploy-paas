package paths

import (
	"os"
	"path/filepath"
)

func ResolveDataDir() string {
	if dir := os.Getenv("DEPLOY_DATA_DIR"); dir != "" {
		return dir
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".paasdeploy", "apps")
	}
	return filepath.Join(os.TempDir(), "paasdeploy", "apps")
}
