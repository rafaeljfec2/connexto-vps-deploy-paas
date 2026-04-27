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

func TestBuildQueryPointersSendZeroValuesExplicitly(t *testing.T) {
	q := BuildQuery(map[string]any{
		"force":  PtrBool(false),
		"keep":   PtrBool(true),
		"tail":   PtrInt(0),
		"offset": PtrInt64(0),
		"name":   PtrString(""),
	})
	encoded := q.Encode()
	if !strings.Contains(encoded, "force=false") {
		t.Errorf("expected force=false, got %s", encoded)
	}
	if !strings.Contains(encoded, "keep=true") {
		t.Errorf("expected keep=true, got %s", encoded)
	}
	if !strings.Contains(encoded, "tail=0") {
		t.Errorf("expected tail=0, got %s", encoded)
	}
	if !strings.Contains(encoded, "offset=0") {
		t.Errorf("expected offset=0, got %s", encoded)
	}
	if !strings.Contains(encoded, "name=") {
		t.Errorf("expected explicit empty name, got %s", encoded)
	}
}

func TestBuildQueryPointersOmitNil(t *testing.T) {
	var (
		nilBool   *bool
		nilInt    *int
		nilInt64  *int64
		nilString *string
	)
	q := BuildQuery(map[string]any{
		"force": nilBool,
		"tail":  nilInt,
		"size":  nilInt64,
		"name":  nilString,
	})
	if got := q.Encode(); got != "" {
		t.Errorf("expected empty query for nil pointers, got %q", got)
	}
}
