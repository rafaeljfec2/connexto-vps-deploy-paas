package version

import (
	"os"
	"path/filepath"
	"testing"
)

const testFilePackageJSON = "package.json"

func TestDetectNodeVersion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, testFilePackageJSON, `{"name":"app","version":"2.5.1"}`)

	got := DetectAppVersion("node", dir)
	assertEqual(t, "2.5.1", got)
}

func TestDetectNodeVersionMissing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, testFilePackageJSON, `{"name":"app"}`)

	got := DetectAppVersion("node", dir)
	assertEqual(t, "", got)
}

func TestDetectPHPVersion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "composer.json", `{"name":"vendor/pkg","version":"1.0.3"}`)

	got := DetectAppVersion("php", dir)
	assertEqual(t, "1.0.3", got)
}

func TestDetectPythonPyproject(t *testing.T) {
	dir := t.TempDir()
	content := "[project]\nname = \"myapp\"\nversion = \"3.2.0\"\n"
	writeFile(t, dir, "pyproject.toml", content)

	got := DetectAppVersion("python", dir)
	assertEqual(t, "3.2.0", got)
}

func TestDetectPythonSetupCfg(t *testing.T) {
	dir := t.TempDir()
	content := "[metadata]\nname = myapp\nversion = 1.4.2\n"
	writeFile(t, dir, "setup.cfg", content)

	got := DetectAppVersion("python", dir)
	assertEqual(t, "1.4.2", got)
}

func TestDetectPythonSetupPy(t *testing.T) {
	dir := t.TempDir()
	content := "from setuptools import setup\nsetup(\n    name=\"myapp\",\n    version=\"0.9.1\",\n)\n"
	writeFile(t, dir, "setup.py", content)

	got := DetectAppVersion("python", dir)
	assertEqual(t, "0.9.1", got)
}

func TestDetectRustVersion(t *testing.T) {
	dir := t.TempDir()
	content := "[package]\nname = \"myapp\"\nversion = \"0.3.7\"\nedition = \"2021\"\n"
	writeFile(t, dir, "Cargo.toml", content)

	got := DetectAppVersion("rust", dir)
	assertEqual(t, "0.3.7", got)
}

func TestDetectJavaMaven(t *testing.T) {
	dir := t.TempDir()
	content := "<?xml version=\"1.0\"?>\n<project>\n  <groupId>com.example</groupId>\n  <artifactId>myapp</artifactId>\n  <version>4.1.0</version>\n</project>\n"
	writeFile(t, dir, "pom.xml", content)

	got := DetectAppVersion("java", dir)
	assertEqual(t, "4.1.0", got)
}

func TestDetectJavaGradle(t *testing.T) {
	dir := t.TempDir()
	content := "plugins {\n    id 'java'\n}\nversion = '2.0.0'\n"
	writeFile(t, dir, "build.gradle", content)

	got := DetectAppVersion("java", dir)
	assertEqual(t, "2.0.0", got)
}

func TestDetectRubyGemspec(t *testing.T) {
	dir := t.TempDir()
	content := "Gem::Specification.new do |spec|\n  spec.name = \"myapp\"\n  spec.version = \"1.2.3\"\nend\n"
	writeFile(t, dir, "myapp.gemspec", content)

	got := DetectAppVersion("ruby", dir)
	assertEqual(t, "1.2.3", got)
}

func TestDetectDotnetCsproj(t *testing.T) {
	dir := t.TempDir()
	content := "<Project Sdk=\"Microsoft.NET.Sdk\">\n  <PropertyGroup>\n    <TargetFramework>net8.0</TargetFramework>\n    <Version>5.0.1</Version>\n  </PropertyGroup>\n</Project>\n"
	writeFile(t, dir, "MyApp.csproj", content)

	got := DetectAppVersion("dotnet", dir)
	assertEqual(t, "5.0.1", got)
}

func TestDetectElixirMixExs(t *testing.T) {
	dir := t.TempDir()
	content := "defmodule MyApp.MixProject do\n  use Mix.Project\n\n  def project do\n    [\n      app: :myapp,\n      version: \"0.1.0\",\n      elixir: \"~> 1.14\"\n    ]\n  end\nend\n"
	writeFile(t, dir, "mix.exs", content)

	got := DetectAppVersion("elixir", dir)
	assertEqual(t, "0.1.0", got)
}

func TestDetectGoVersionFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "VERSION", "1.8.0\n")

	got := DetectAppVersion("go", dir)
	assertEqual(t, "1.8.0", got)
}

func TestDetectGoNoVersionFile(t *testing.T) {
	dir := t.TempDir()

	got := DetectAppVersion("go", dir)
	assertEqual(t, "", got)
}

func TestDetectUnknownRuntimeWithVersionFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "VERSION", "0.0.1")

	got := DetectAppVersion("other", dir)
	assertEqual(t, "0.0.1", got)
}

func TestDetectEmptyRuntime(t *testing.T) {
	dir := t.TempDir()

	got := DetectAppVersion("", dir)
	assertEqual(t, "", got)
}

func TestDetectNodeJSAlias(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, testFilePackageJSON, `{"version":"1.0.0"}`)

	got := DetectAppVersion("nodejs", dir)
	assertEqual(t, "1.0.0", got)
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func assertEqual(t *testing.T, expected, got string) {
	t.Helper()
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}
