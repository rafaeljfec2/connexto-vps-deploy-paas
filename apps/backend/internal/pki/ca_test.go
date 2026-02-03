package pki

import "testing"

func TestLoadCA(t *testing.T) {
	ca, err := NewCA()
	if err != nil {
		t.Fatalf("new CA error: %v", err)
	}

	loaded, err := LoadCA(ca.GetCACertPEM(), ca.GetCAKeyPEM())
	if err != nil {
		t.Fatalf("load CA error: %v", err)
	}

	cert, err := loaded.GenerateAgentCert("server-id", "example.com")
	if err != nil {
		t.Fatalf("generate agent cert error: %v", err)
	}
	if len(cert.CertPEM) == 0 {
		t.Fatalf("empty cert pem")
	}
	if len(cert.KeyPEM) == 0 {
		t.Fatalf("empty key pem")
	}
}
