package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/paasdeploy/backend/internal/domain"
)

const customDomainSelectColumns = `id, app_id, domain, path_prefix, zone_id, dns_record_id, record_type, status, created_at, updated_at`

type PostgresCustomDomainRepository struct {
	db *sql.DB
}

func NewPostgresCustomDomainRepository(db *sql.DB) *PostgresCustomDomainRepository {
	return &PostgresCustomDomainRepository{db: db}
}

func (r *PostgresCustomDomainRepository) scanDomain(row *sql.Row) (*domain.CustomDomain, error) {
	var d domain.CustomDomain
	err := row.Scan(
		&d.ID,
		&d.AppID,
		&d.Domain,
		&d.PathPrefix,
		&d.ZoneID,
		&d.DNSRecordID,
		&d.RecordType,
		&d.Status,
		&d.CreatedAt,
		&d.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &d, nil
}

func (r *PostgresCustomDomainRepository) scanDomains(rows *sql.Rows) ([]domain.CustomDomain, error) {
	var domains []domain.CustomDomain
	for rows.Next() {
		var d domain.CustomDomain
		err := rows.Scan(
			&d.ID,
			&d.AppID,
			&d.Domain,
			&d.PathPrefix,
			&d.ZoneID,
			&d.DNSRecordID,
			&d.RecordType,
			&d.Status,
			&d.CreatedAt,
			&d.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		domains = append(domains, d)
	}
	return domains, rows.Err()
}

func (r *PostgresCustomDomainRepository) Create(ctx context.Context, input domain.CreateCustomDomainInput) (*domain.CustomDomain, error) {
	query := `
		INSERT INTO custom_domains (app_id, domain, path_prefix, zone_id, dns_record_id, record_type)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING ` + customDomainSelectColumns

	return r.scanDomain(r.db.QueryRowContext(ctx, query,
		input.AppID,
		input.Domain,
		input.PathPrefix,
		input.ZoneID,
		input.DNSRecordID,
		input.RecordType,
	))
}

func (r *PostgresCustomDomainRepository) FindByID(ctx context.Context, id string) (*domain.CustomDomain, error) {
	query := `SELECT ` + customDomainSelectColumns + ` FROM custom_domains WHERE id = $1`
	return r.scanDomain(r.db.QueryRowContext(ctx, query, id))
}

func (r *PostgresCustomDomainRepository) FindByAppID(ctx context.Context, appID string) ([]domain.CustomDomain, error) {
	query := `SELECT ` + customDomainSelectColumns + ` FROM custom_domains WHERE app_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanDomains(rows)
}

func (r *PostgresCustomDomainRepository) FindByDomain(ctx context.Context, domainName string) (*domain.CustomDomain, error) {
	query := `SELECT ` + customDomainSelectColumns + ` FROM custom_domains WHERE domain = $1`
	return r.scanDomain(r.db.QueryRowContext(ctx, query, domainName))
}

func (r *PostgresCustomDomainRepository) FindByDomainAndPath(ctx context.Context, domainName, pathPrefix string) (*domain.CustomDomain, error) {
	query := `SELECT ` + customDomainSelectColumns + ` FROM custom_domains WHERE domain = $1 AND path_prefix = $2`
	return r.scanDomain(r.db.QueryRowContext(ctx, query, domainName, pathPrefix))
}

func (r *PostgresCustomDomainRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM custom_domains WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PostgresCustomDomainRepository) DeleteByAppID(ctx context.Context, appID string) error {
	query := `DELETE FROM custom_domains WHERE app_id = $1`
	_, err := r.db.ExecContext(ctx, query, appID)
	return err
}

var _ domain.CustomDomainRepository = (*PostgresCustomDomainRepository)(nil)
