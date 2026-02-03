package pki

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"
)

const orgName = "PaasDeploy"

type CertificateAuthority struct {
	cert       *x509.Certificate
	privateKey *ecdsa.PrivateKey
}

type Certificate struct {
	CertPEM []byte
	KeyPEM []byte
}

func NewCA() (*CertificateAuthority, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{orgName},
			CommonName:   "PaasDeploy Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, err
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, err
	}

	return &CertificateAuthority{
		cert:       cert,
		privateKey: privateKey,
	}, nil
}

func (ca *CertificateAuthority) GenerateServerCert(hostname string) (*Certificate, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		SerialNumber: generateSerial(),
		Subject: pkix.Name{
			Organization: []string{orgName},
			CommonName:   "paasdeploy-server",
		},
		DNSNames:    []string{hostname, "localhost"},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(1, 0, 0),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, ca.cert, &privateKey.PublicKey, ca.privateKey)
	if err != nil {
		return nil, err
	}

	return &Certificate{
		CertPEM: pemEncode("CERTIFICATE", certDER),
		KeyPEM:  pemEncodeKey(privateKey),
	}, nil
}

func (ca *CertificateAuthority) GenerateAgentCert(serverID string) (*Certificate, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		SerialNumber: generateSerial(),
		Subject: pkix.Name{
			Organization:       []string{orgName},
			OrganizationalUnit: []string{"agent"},
			CommonName:         serverID,
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(1, 0, 0),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, ca.cert, &privateKey.PublicKey, ca.privateKey)
	if err != nil {
		return nil, err
	}

	return &Certificate{
		CertPEM: pemEncode("CERTIFICATE", certDER),
		KeyPEM:  pemEncodeKey(privateKey),
	}, nil
}

func (ca *CertificateAuthority) GetCACertPEM() []byte {
	return pemEncode("CERTIFICATE", ca.cert.Raw)
}

func generateSerial() *big.Int {
	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	return serial
}

func pemEncode(blockType string, data []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: data})
}

func pemEncodeKey(key *ecdsa.PrivateKey) []byte {
	data, _ := x509.MarshalECPrivateKey(key)
	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: data})
}
