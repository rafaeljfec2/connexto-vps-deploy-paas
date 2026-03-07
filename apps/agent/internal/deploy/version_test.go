package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectAppVersionDelegatesToShared(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"version":"1.0.0"}`), 0644); err != nil {
		t.Fatal(err)
	}

	got := detectAppVersion("node", dir)
	if got != "1.0.0" {
		t.Errorf("expected %q, got %q", "1.0.0", got)
	}
}
