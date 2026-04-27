package toolkit

import (
	"strings"
	"testing"
)

func TestBuildQueryFiltersZeroValues(t *testing.T) {
	q := BuildQuery(map[string]any{
		"limit":    50,
		"empty":    "",
		"flag":     true,
		"falseFlag": false,
		"nilValue": nil,
		"actor":    "pat",
	})
	encoded := q.Encode()
	if !strings.Contains(encoded, "limit=50") {
		t.Errorf("expected limit, got %s", encoded)
	}
	if !strings.Contains(encoded, "actor=pat") {
		t.Errorf("expected actor, got %s", encoded)
	}
	if !strings.Contains(encoded, "flag=true") {
		t.Errorf("expected flag=true, got %s", encoded)
	}
	if strings.Contains(encoded, "empty=") {
		t.Errorf("empty string must be omitted, got %s", encoded)
	}
	if strings.Contains(encoded, "falseFlag=") {
		t.Errorf("false bool must be omitted, got %s", encoded)
	}
	if strings.Contains(encoded, "nilValue=") {
		t.Errorf("nil must be omitted, got %s", encoded)
	}
}

func TestBuildQueryOmitsZeroInt(t *testing.T) {
	q := BuildQuery(map[string]any{"tail": 0})
	if got := q.Encode(); got != "" {
		t.Errorf("expected empty query, got %q", got)
	}
}

func TestMarshalRawHandlesNil(t *testing.T) {
	raw, err := MarshalRaw(nil)
	if err != nil {
		t.Fatalf("MarshalRaw: %v", err)
	}
	if string(raw) != "null" {
		t.Errorf("expected null, got %s", raw)
	}
}
