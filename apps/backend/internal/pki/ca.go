package pki

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"time"
)

const orgName = "PaasDeploy"

type CertificateAuthority struct {
	cert       *x509.Certificate
	privateKey *ecdsa.PrivateKey
}

type Certificate struct {
	CertPEM []byte
	KeyPEM  []byte
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

func LoadCA(certPEM []byte, keyPEM []byte) (*CertificateAuthority, error) {
	cert, err := parseCertificate(certPEM)
	if err != nil {
		return nil, err
	}
	privateKey, err := parseECPrivateKey(keyPEM)
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

func (ca *CertificateAuthority) GenerateAgentCert(serverID string, host string) (*Certificate, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	dnsNames := []string{"localhost"}
	var ipAddresses []net.IP
	if host != "" {
		if ip := net.ParseIP(host); ip != nil {
			ipAddresses = append(ipAddresses, ip)
		} else {
			dnsNames = append([]string{host}, dnsNames...)
		}
	}

	template := &x509.Certificate{
		SerialNumber: generateSerial(),
		Subject: pkix.Name{
			Organization:       []string{orgName},
			OrganizationalUnit: []string{"agent"},
			CommonName:         serverID,
		},
		DNSNames:    dnsNames,
		IPAddresses: ipAddresses,
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(1, 0, 0),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
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

func (ca *CertificateAuthority) GetCAKeyPEM() []byte {
	return pemEncodeKey(ca.privateKey)
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

func parseCertificate(certPEM []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, errors.New("invalid cert pem")
	}
	return x509.ParseCertificate(block.Bytes)
}

func parseECPrivateKey(keyPEM []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, errors.New("invalid key pem")
	}
	return x509.ParseECPrivateKey(block.Bytes)
}
