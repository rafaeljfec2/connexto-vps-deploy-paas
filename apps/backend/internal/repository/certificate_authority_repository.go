package repository

import (
	"database/sql"
	"errors"

	"github.com/paasdeploy/backend/internal/domain"
)

type PostgresCertificateAuthorityRepository struct {
	db *sql.DB
}

func NewPostgresCertificateAuthorityRepository(db *sql.DB) *PostgresCertificateAuthorityRepository {
	return &PostgresCertificateAuthorityRepository{db: db}
}

func (r *PostgresCertificateAuthorityRepository) GetRoot() (*domain.CertificateAuthorityRecord, error) {
	query := `SELECT cert_pem, key_pem FROM pki_ca WHERE name = 'root'`
	var certPEM string
	var keyPEM string
	if err := r.db.QueryRow(query).Scan(&certPEM, &keyPEM); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &domain.CertificateAuthorityRecord{
		CertPEM: []byte(certPEM),
		KeyPEM:  []byte(keyPEM),
	}, nil
}

func (r *PostgresCertificateAuthorityRepository) UpsertRoot(record domain.CertificateAuthorityRecord) error {
	query := `INSERT INTO pki_ca (name, cert_pem, key_pem)
		VALUES ('root', $1, $2)
		ON CONFLICT (name) DO UPDATE SET cert_pem = EXCLUDED.cert_pem, key_pem = EXCLUDED.key_pem, updated_at = NOW()`
	_, err := r.db.Exec(query, string(record.CertPEM), string(record.KeyPEM))
	return err
}
