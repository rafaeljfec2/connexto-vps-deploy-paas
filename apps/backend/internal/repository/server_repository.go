package repository

import (
	"database/sql"
	"errors"

	"github.com/paasdeploy/backend/internal/domain"
)

const serverSelectColumns = `id, user_id, name, host, ssh_port, ssh_user, ssh_key_encrypted, ssh_password_encrypted, acme_email, status, agent_version, last_heartbeat_at, created_at, updated_at`

type PostgresServerRepository struct {
	db *sql.DB
}

func NewPostgresServerRepository(db *sql.DB) *PostgresServerRepository {
	return &PostgresServerRepository{db: db}
}

func (r *PostgresServerRepository) scanServer(row *sql.Row) (*domain.Server, error) {
	var s domain.Server
	var agentVersion sql.NullString
	var lastHeartbeatAt sql.NullTime
	var sshPassword sql.NullString
	var acmeEmail sql.NullString
	err := row.Scan(
		&s.ID,
		&s.UserID,
		&s.Name,
		&s.Host,
		&s.SSHPort,
		&s.SSHUser,
		&s.SSHKeyEncrypted,
		&sshPassword,
		&acmeEmail,
		&s.Status,
		&agentVersion,
		&lastHeartbeatAt,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if agentVersion.Valid {
		s.AgentVersion = &agentVersion.String
	}
	if lastHeartbeatAt.Valid {
		s.LastHeartbeatAt = &lastHeartbeatAt.Time
	}
	if sshPassword.Valid {
		s.SSHPasswordEncrypted = sshPassword.String
	}
	if acmeEmail.Valid {
		s.AcmeEmail = &acmeEmail.String
	}
	return &s, nil
}

func (r *PostgresServerRepository) Create(input domain.CreateServerInput) (*domain.Server, error) {
	query := `INSERT INTO servers (user_id, name, host, ssh_port, ssh_user, ssh_key_encrypted, ssh_password_encrypted, acme_email, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'pending')
		RETURNING ` + serverSelectColumns

	sshPort := input.SSHPort
	if sshPort == 0 {
		sshPort = 22
	}

	return r.scanServer(r.db.QueryRow(query, input.UserID, input.Name, input.Host, sshPort, input.SSHUser, input.SSHKeyEncrypted, input.SSHPasswordEncrypted, input.AcmeEmail))
}

func (r *PostgresServerRepository) FindByID(id string) (*domain.Server, error) {
	query := `SELECT ` + serverSelectColumns + ` FROM servers WHERE id = $1`
	return r.scanServer(r.db.QueryRow(query, id))
}

func (r *PostgresServerRepository) scanServerRows(rows *sql.Rows) ([]domain.Server, error) {
	var servers []domain.Server
	for rows.Next() {
		var s domain.Server
		var agentVersion sql.NullString
		var lastHeartbeatAt sql.NullTime
		var sshPassword sql.NullString
		var acmeEmail sql.NullString
		if err := rows.Scan(
			&s.ID,
			&s.UserID,
			&s.Name,
			&s.Host,
			&s.SSHPort,
			&s.SSHUser,
			&s.SSHKeyEncrypted,
			&sshPassword,
			&acmeEmail,
			&s.Status,
			&agentVersion,
			&lastHeartbeatAt,
			&s.CreatedAt,
			&s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if agentVersion.Valid {
			s.AgentVersion = &agentVersion.String
		}
		if lastHeartbeatAt.Valid {
			s.LastHeartbeatAt = &lastHeartbeatAt.Time
		}
		if sshPassword.Valid {
			s.SSHPasswordEncrypted = sshPassword.String
		}
		if acmeEmail.Valid {
			s.AcmeEmail = &acmeEmail.String
		}
		servers = append(servers, s)
	}
	return servers, rows.Err()
}

func (r *PostgresServerRepository) FindAll() ([]domain.Server, error) {
	query := `SELECT ` + serverSelectColumns + ` FROM servers ORDER BY created_at DESC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanServerRows(rows)
}

func (r *PostgresServerRepository) FindAllByUserID(userID string) ([]domain.Server, error) {
	query := `SELECT ` + serverSelectColumns + ` FROM servers WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanServerRows(rows)
}

func (r *PostgresServerRepository) FindByIDForUser(id string, userID string) (*domain.Server, error) {
	query := `SELECT ` + serverSelectColumns + ` FROM servers WHERE id = $1 AND user_id = $2`
	return r.scanServer(r.db.QueryRow(query, id, userID))
}

func (r *PostgresServerRepository) Update(id string, input domain.UpdateServerInput) (*domain.Server, error) {
	query := `UPDATE servers SET
		name = COALESCE($2, name),
		host = COALESCE($3, host),
		ssh_port = COALESCE($4, ssh_port),
		ssh_user = COALESCE($5, ssh_user),
		ssh_key_encrypted = COALESCE($6, ssh_key_encrypted),
		ssh_password_encrypted = COALESCE($7, ssh_password_encrypted),
		acme_email = COALESCE($8, acme_email),
		status = COALESCE($9, status),
		updated_at = NOW()
		WHERE id = $1
		RETURNING ` + serverSelectColumns

	var name, host, sshUser, sshKeyEncrypted *string
	var sshPasswordEncrypted *string
	var acmeEmail *string
	var sshPort *int
	var status *domain.ServerStatus

	if input.Name != nil {
		name = input.Name
	}
	if input.Host != nil {
		host = input.Host
	}
	if input.SSHPort != nil {
		sshPort = input.SSHPort
	}
	if input.SSHUser != nil {
		sshUser = input.SSHUser
	}
	if input.SSHKeyEncrypted != nil {
		sshKeyEncrypted = input.SSHKeyEncrypted
	}
	if input.SSHPasswordEncrypted != nil {
		sshPasswordEncrypted = input.SSHPasswordEncrypted
	}
	if input.AcmeEmail != nil {
		acmeEmail = input.AcmeEmail
	}
	if input.Status != nil {
		status = input.Status
	}

	return r.scanServer(r.db.QueryRow(query, id, name, host, sshPort, sshUser, sshKeyEncrypted, sshPasswordEncrypted, acmeEmail, status))
}

func (r *PostgresServerRepository) UpdateHeartbeat(id string, agentVersion string) error {
	query := `UPDATE servers SET last_heartbeat_at = NOW(), agent_version = COALESCE(NULLIF($2, ''), agent_version), status = 'online', updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(query, id, agentVersion)
	return err
}

func (r *PostgresServerRepository) Delete(id string) error {
	query := `DELETE FROM servers WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}
