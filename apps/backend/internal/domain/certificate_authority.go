package domain

type CertificateAuthorityRecord struct {
	CertPEM []byte
	KeyPEM  []byte
}

type CertificateAuthorityRepository interface {
	GetRoot() (*CertificateAuthorityRecord, error)
	UpsertRoot(record CertificateAuthorityRecord) error
}
