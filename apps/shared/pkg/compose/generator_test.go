package compose

import (
	"strings"
	"testing"
)

const (
	testAppName  = "test-app"
	testImageTag = "test-app:latest"
)

func TestBuildVolumesYAMLEmpty(t *testing.T) {
	svc, top := BuildVolumesYAML(nil)
	if svc != "" {
		t.Errorf("expected empty service-level YAML, got %q", svc)
	}
	if top != "" {
		t.Errorf("expected empty top-level YAML, got %q", top)
	}
}

func TestBuildVolumesYAMLNamedVolume(t *testing.T) {
	volumes := []VolumeConfig{
		{Name: "app-data", Target: "/app/data"},
	}
	svc, top := BuildVolumesYAML(volumes)

	if !strings.Contains(svc, "- app-data:/app/data") {
		t.Errorf("service volumes should contain named mapping, got:\n%s", svc)
	}
	if !strings.Contains(top, "app-data:") {
		t.Errorf("top-level should declare named volume, got:\n%s", top)
	}
}

func TestBuildVolumesYAMLNamedVolumeReadOnly(t *testing.T) {
	volumes := []VolumeConfig{
		{Name: "uploads", Target: "/app/uploads", ReadOnly: true},
	}
	svc, top := BuildVolumesYAML(volumes)

	if !strings.Contains(svc, "- uploads:/app/uploads:ro") {
		t.Errorf("service volumes should contain :ro suffix, got:\n%s", svc)
	}
	if !strings.Contains(top, "uploads:") {
		t.Errorf("top-level should declare named volume, got:\n%s", top)
	}
}

func TestBuildVolumesYAMLBindMount(t *testing.T) {
	volumes := []VolumeConfig{
		{Source: "/host/backups", Target: "/backups"},
	}
	svc, top := BuildVolumesYAML(volumes)

	if !strings.Contains(svc, "- /host/backups:/backups") {
		t.Errorf("service volumes should contain bind mount mapping, got:\n%s", svc)
	}
	if top != "" {
		t.Errorf("top-level should be empty for bind mounts, got:\n%s", top)
	}
}

func TestBuildVolumesYAMLMixedVolumes(t *testing.T) {
	volumes := []VolumeConfig{
		{Name: "db-data", Target: "/var/lib/db"},
		{Source: "/etc/config", Target: "/app/config", ReadOnly: true},
		{Name: "cache", Target: "/tmp/cache"},
	}
	svc, top := BuildVolumesYAML(volumes)

	if !strings.Contains(svc, "- db-data:/var/lib/db") {
		t.Errorf("missing named volume db-data in service section")
	}
	if !strings.Contains(svc, "- /etc/config:/app/config:ro") {
		t.Errorf("missing bind mount with :ro in service section")
	}
	if !strings.Contains(svc, "- cache:/tmp/cache") {
		t.Errorf("missing named volume cache in service section")
	}
	if !strings.Contains(top, "db-data:") {
		t.Errorf("missing db-data in top-level volumes")
	}
	if !strings.Contains(top, "cache:") {
		t.Errorf("missing cache in top-level volumes")
	}
	if strings.Contains(top, "/etc/config") {
		t.Errorf("bind mount source should not appear in top-level volumes")
	}
}

func TestGenerateContentWithVolumes(t *testing.T) {
	cfg := &Config{
		Name: testAppName,
		Port: 3000,
		Volumes: []VolumeConfig{
			{Name: "app-data", Target: "/data"},
			{Source: "/host/logs", Target: "/logs", ReadOnly: true},
		},
	}
	ApplyDefaults(cfg)

	content := GenerateContent(GenerateParams{
		AppName:  testAppName,
		ImageTag: testImageTag,
		Config:   cfg,
	})

	if !strings.Contains(content, "    volumes:\n") {
		t.Error("generated compose should contain service-level volumes section")
	}
	if !strings.Contains(content, "- app-data:/data") {
		t.Error("generated compose should contain named volume mapping")
	}
	if !strings.Contains(content, "- /host/logs:/logs:ro") {
		t.Error("generated compose should contain bind mount with :ro")
	}
	if !strings.Contains(content, "volumes:\n  app-data:\n") {
		t.Error("generated compose should contain top-level volume declaration")
	}
	if !strings.Contains(content, "networks:\n  paasdeploy:\n    external: true") {
		t.Error("generated compose should still contain networks section")
	}
}

func TestGenerateContentWithoutVolumes(t *testing.T) {
	cfg := &Config{
		Name: testAppName,
		Port: 3000,
	}
	ApplyDefaults(cfg)

	content := GenerateContent(GenerateParams{
		AppName:  testAppName,
		ImageTag: testImageTag,
		Config:   cfg,
	})

	lines := strings.Split(content, "\n")
	volumeCount := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "volumes:" {
			volumeCount++
		}
	}
	if volumeCount != 0 {
		t.Errorf("generated compose without volumes should not contain volumes: sections, found %d", volumeCount)
	}
}

func TestBuildPortMappingWithHostPort(t *testing.T) {
	result := BuildPortMapping(3000, 3000)
	expected := "127.0.0.1:3000:3000"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestBuildPortMappingWithDifferentPorts(t *testing.T) {
	result := BuildPortMapping(8081, 3000)
	expected := "127.0.0.1:8081:3000"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestBuildPortMappingWithoutHostPort(t *testing.T) {
	result := BuildPortMapping(0, 3000)
	if result != "" {
		t.Errorf("expected empty string when hostPort is 0, got %q", result)
	}
}

func TestGenerateContentWithHostPort(t *testing.T) {
	cfg := &Config{
		Name:     testAppName,
		Port:     3000,
		HostPort: 3000,
	}
	ApplyDefaults(cfg)

	content := GenerateContent(GenerateParams{
		AppName:  testAppName,
		ImageTag: testImageTag,
		Config:   cfg,
	})

	if !strings.Contains(content, "ports:") {
		t.Error("generated compose with hostPort should contain ports section")
	}
	if !strings.Contains(content, "127.0.0.1:3000:3000") {
		t.Error("generated compose should bind to 127.0.0.1")
	}
	if strings.Contains(content, "expose:") {
		t.Error("generated compose with hostPort should not contain expose section")
	}
}

func TestGenerateContentWithoutHostPort(t *testing.T) {
	cfg := &Config{
		Name: testAppName,
		Port: 3000,
	}
	ApplyDefaults(cfg)

	content := GenerateContent(GenerateParams{
		AppName:  testAppName,
		ImageTag: testImageTag,
		Config:   cfg,
	})

	if strings.Contains(content, "ports:") {
		t.Error("generated compose without hostPort should not contain ports section")
	}
	if !strings.Contains(content, "expose:") {
		t.Error("generated compose without hostPort should contain expose section")
	}
	if !strings.Contains(content, "\"3000\"") {
		t.Error("generated compose should expose container port 3000")
	}
}

func TestVolumeConfigIsNamedVolume(t *testing.T) {
	named := VolumeConfig{Name: "data", Target: "/data"}
	if !named.IsNamedVolume() {
		t.Error("volume with Name should be identified as named")
	}

	bind := VolumeConfig{Source: "/host/data", Target: "/data"}
	if bind.IsNamedVolume() {
		t.Error("volume with only Source should not be identified as named")
	}
}
