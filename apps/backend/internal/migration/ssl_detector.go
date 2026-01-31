package migration

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type SSLCertificate struct {
	Domain         string    `json:"domain"`
	Provider       string    `json:"provider"`
	CertPath       string    `json:"certPath"`
	KeyPath        string    `json:"keyPath"`
	ChainPath      string    `json:"chainPath,omitempty"`
	FullChainPath  string    `json:"fullChainPath,omitempty"`
	ExpiresAt      time.Time `json:"expiresAt"`
	DaysUntilExpiry int      `json:"daysUntilExpiry"`
	IsExpired      bool      `json:"isExpired"`
	AutoRenew      bool      `json:"autoRenew"`
	RenewalConfig  string    `json:"renewalConfig,omitempty"`
	Issuer         string    `json:"issuer,omitempty"`
	Subject        string    `json:"subject,omitempty"`
}

type SSLDetector struct {
	letsencryptPath string
	renewalPath     string
}

func NewSSLDetector() *SSLDetector {
	return &SSLDetector{
		letsencryptPath: "/etc/letsencrypt/live",
		renewalPath:     "/etc/letsencrypt/renewal",
	}
}

func (d *SSLDetector) DetectAllCertificates() ([]SSLCertificate, error) {
	var certs []SSLCertificate

	leCerts, err := d.detectLetsEncryptCerts()
	if err == nil {
		certs = append(certs, leCerts...)
	}

	return certs, nil
}

func (d *SSLDetector) detectLetsEncryptCerts() ([]SSLCertificate, error) {
	var certs []SSLCertificate

	if _, err := os.Stat(d.letsencryptPath); os.IsNotExist(err) {
		return certs, nil
	}

	entries, err := os.ReadDir(d.letsencryptPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		domain := entry.Name()
		certDir := filepath.Join(d.letsencryptPath, domain)

		cert := SSLCertificate{
			Domain:        domain,
			Provider:      "letsencrypt",
			CertPath:      filepath.Join(certDir, "cert.pem"),
			KeyPath:       filepath.Join(certDir, "privkey.pem"),
			ChainPath:     filepath.Join(certDir, "chain.pem"),
			FullChainPath: filepath.Join(certDir, "fullchain.pem"),
		}

		renewalFile := filepath.Join(d.renewalPath, domain+".conf")
		if _, err := os.Stat(renewalFile); err == nil {
			cert.AutoRenew = true
			cert.RenewalConfig = renewalFile
		}

		if certInfo, err := d.parseCertificate(cert.CertPath); err == nil {
			cert.ExpiresAt = certInfo.NotAfter
			cert.DaysUntilExpiry = int(time.Until(certInfo.NotAfter).Hours() / 24)
			cert.IsExpired = time.Now().After(certInfo.NotAfter)
			cert.Issuer = certInfo.Issuer.CommonName
			cert.Subject = certInfo.Subject.CommonName
		}

		certs = append(certs, cert)
	}

	return certs, nil
}

func (d *SSLDetector) parseCertificate(certPath string) (*x509.Certificate, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, err
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	return cert, nil
}

func (d *SSLDetector) GetCertificateForDomain(domain string) (*SSLCertificate, error) {
	certs, err := d.DetectAllCertificates()
	if err != nil {
		return nil, err
	}

	for _, cert := range certs {
		if cert.Domain == domain {
			return &cert, nil
		}
		if strings.HasPrefix(domain, "*.") {
			wildcardDomain := strings.TrimPrefix(domain, "*.")
			if cert.Domain == wildcardDomain {
				return &cert, nil
			}
		}
	}

	return nil, nil
}

func (d *SSLDetector) GetExpiringCertificates(daysThreshold int) ([]SSLCertificate, error) {
	allCerts, err := d.DetectAllCertificates()
	if err != nil {
		return nil, err
	}

	var expiring []SSLCertificate
	for _, cert := range allCerts {
		if cert.DaysUntilExpiry <= daysThreshold {
			expiring = append(expiring, cert)
		}
	}

	return expiring, nil
}
