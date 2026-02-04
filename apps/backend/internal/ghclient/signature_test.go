package ghclient

import (
	"testing"
)

const signatureTestSecret = "test-secret"

func TestValidateSignature(t *testing.T) {
	tests := []struct {
		name      string
		payload   []byte
		signature string
		secret    string
		want      bool
	}{
		{
			name:      "valid signature",
			payload:   []byte(`{"test": "payload"}`),
			signature: GenerateSignature([]byte(`{"test": "payload"}`), signatureTestSecret),
			secret:    signatureTestSecret,
			want:      true,
		},
		{
			name:      "invalid signature",
			payload:   []byte(`{"test": "payload"}`),
			signature: "sha256=0000000000000000000000000000000000000000000000000000000000000000",
			secret:    signatureTestSecret,
			want:      false,
		},
		{
			name:      "empty secret",
			payload:   []byte(`{"test": "payload"}`),
			signature: "sha256=abc123",
			secret:    "",
			want:      false,
		},
		{
			name:      "empty signature",
			payload:   []byte(`{"test": "payload"}`),
			signature: "",
			secret:    signatureTestSecret,
			want:      false,
		},
		{
			name:      "missing sha256 prefix",
			payload:   []byte(`{"test": "payload"}`),
			signature: "abc123",
			secret:    signatureTestSecret,
			want:      false,
		},
		{
			name:      "wrong hash length",
			payload:   []byte(`{"test": "payload"}`),
			signature: "sha256=abc123",
			secret:    signatureTestSecret,
			want:      false,
		},
		{
			name:      "tampered payload",
			payload:   []byte(`{"test": "tampered"}`),
			signature: GenerateSignature([]byte(`{"test": "original"}`), signatureTestSecret),
			secret:    signatureTestSecret,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateSignature(tt.payload, tt.signature, tt.secret)
			if got != tt.want {
				t.Errorf("ValidateSignature() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateSignature(t *testing.T) {
	payload := []byte(`{"test": "payload"}`)
	secret := "my-secret"

	sig := GenerateSignature(payload, secret)

	if len(sig) != len(signaturePrefix)+signatureLength {
		t.Errorf("GenerateSignature() length = %d, want %d", len(sig), len(signaturePrefix)+signatureLength)
	}

	if sig[:len(signaturePrefix)] != signaturePrefix {
		t.Errorf("GenerateSignature() prefix = %s, want %s", sig[:len(signaturePrefix)], signaturePrefix)
	}
}
