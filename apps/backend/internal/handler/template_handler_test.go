package handler

import (
	"reflect"
	"testing"
)

func TestResolveTemplateCommandReturnsRequestOverride(t *testing.T) {
	tmpl := &Template{Command: []string{"default"}}
	req := DeployTemplateRequest{Command: []string{"custom", "arg"}}

	got := resolveTemplateCommand(tmpl, req)
	want := []string{"custom", "arg"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestResolveTemplateCommandFallsBackToTemplateDefault(t *testing.T) {
	tmpl := &Template{Command: []string{"server", "/data"}}
	req := DeployTemplateRequest{}

	got := resolveTemplateCommand(tmpl, req)
	want := []string{"server", "/data"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestResolveTemplateCommandReturnsNilWhenNeitherProvided(t *testing.T) {
	tmpl := &Template{}
	req := DeployTemplateRequest{}

	got := resolveTemplateCommand(tmpl, req)

	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestMinioTemplateHasServerCommand(t *testing.T) {
	tmpl := findTemplate("minio")
	if tmpl == nil {
		t.Fatal("minio template not found")
	}

	want := []string{"server", "/data", "--console-address", ":9001"}
	if !reflect.DeepEqual(tmpl.Command, want) {
		t.Fatalf("expected minio command %v, got %v", want, tmpl.Command)
	}
}
