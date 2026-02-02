package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/paasdeploy/backend/internal/domain"
)

type PostgresAuditLogRepository struct {
	db *sql.DB
}

func NewPostgresAuditLogRepository(db *sql.DB) *PostgresAuditLogRepository {
	return &PostgresAuditLogRepository{db: db}
}

func (r *PostgresAuditLogRepository) Create(input domain.CreateAuditLogInput) (*domain.AuditLog, error) {
	var detailsJSON []byte
	var err error
	if input.Details != nil {
		detailsJSON, err = json.Marshal(input.Details)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal details: %w", err)
		}
	}

	query := `
		INSERT INTO audit_logs (event_type, resource_type, resource_id, resource_name, user_id, user_name, details, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, event_type, resource_type, resource_id, resource_name, user_id, user_name, details, ip_address, user_agent, created_at
	`

	var log domain.AuditLog
	var resourceID, resourceName, userID, userName, ipAddress, userAgent sql.NullString
	var details []byte

	err = r.db.QueryRow(
		query,
		input.EventType,
		input.ResourceType,
		input.ResourceID,
		input.ResourceName,
		input.UserID,
		input.UserName,
		detailsJSON,
		input.IPAddress,
		input.UserAgent,
	).Scan(
		&log.ID,
		&log.EventType,
		&log.ResourceType,
		&resourceID,
		&resourceName,
		&userID,
		&userName,
		&details,
		&ipAddress,
		&userAgent,
		&log.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create audit log: %w", err)
	}

	if resourceID.Valid {
		log.ResourceID = &resourceID.String
	}
	if resourceName.Valid {
		log.ResourceName = &resourceName.String
	}
	if userID.Valid {
		log.UserID = &userID.String
	}
	if userName.Valid {
		log.UserName = &userName.String
	}
	if ipAddress.Valid {
		log.IPAddress = &ipAddress.String
	}
	if userAgent.Valid {
		log.UserAgent = &userAgent.String
	}
	log.Details = details

	return &log, nil
}

func (r *PostgresAuditLogRepository) FindByID(id string) (*domain.AuditLog, error) {
	query := `
		SELECT id, event_type, resource_type, resource_id, resource_name, user_id, user_name, details, ip_address, user_agent, created_at
		FROM audit_logs
		WHERE id = $1
	`

	var log domain.AuditLog
	var resourceID, resourceName, userID, userName, ipAddress, userAgent sql.NullString
	var details []byte

	err := r.db.QueryRow(query, id).Scan(
		&log.ID,
		&log.EventType,
		&log.ResourceType,
		&resourceID,
		&resourceName,
		&userID,
		&userName,
		&details,
		&ipAddress,
		&userAgent,
		&log.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if resourceID.Valid {
		log.ResourceID = &resourceID.String
	}
	if resourceName.Valid {
		log.ResourceName = &resourceName.String
	}
	if userID.Valid {
		log.UserID = &userID.String
	}
	if userName.Valid {
		log.UserName = &userName.String
	}
	if ipAddress.Valid {
		log.IPAddress = &ipAddress.String
	}
	if userAgent.Valid {
		log.UserAgent = &userAgent.String
	}
	log.Details = details

	return &log, nil
}

func (r *PostgresAuditLogRepository) FindAll(filter domain.AuditLogFilter) ([]domain.AuditLog, int, error) {
	whereClause, args, argIndex := buildAuditFilters(filter)
	total, err := r.countAuditLogs(whereClause, args)
	if err != nil {
		return nil, 0, err
	}

	limit, offset := normalizePagination(filter.Limit, filter.Offset)
	query := buildAuditLogQuery(whereClause, argIndex)

	args = append(args, limit, offset)
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	logs := make([]domain.AuditLog, 0)
	for rows.Next() {
		log, err := scanAuditLogRow(rows)
		if err != nil {
			return nil, 0, err
		}
		logs = append(logs, *log)
	}

	return logs, total, nil
}

func buildAuditFilters(filter domain.AuditLogFilter) (string, []interface{}, int) {
	conditions := make([]string, 0)
	args := make([]interface{}, 0)
	argIndex := 1

	if filter.EventType != nil {
		conditions = append(conditions, fmt.Sprintf("event_type = $%d", argIndex))
		args = append(args, *filter.EventType)
		argIndex++
	}
	if filter.ResourceType != nil {
		conditions = append(conditions, fmt.Sprintf("resource_type = $%d", argIndex))
		args = append(args, *filter.ResourceType)
		argIndex++
	}
	if filter.ResourceID != nil {
		conditions = append(conditions, fmt.Sprintf("resource_id = $%d", argIndex))
		args = append(args, *filter.ResourceID)
		argIndex++
	}
	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIndex))
		args = append(args, *filter.UserID)
		argIndex++
	}
	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIndex))
		args = append(args, *filter.StartDate)
		argIndex++
	}
	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIndex))
		args = append(args, *filter.EndDate)
		argIndex++
	}

	if len(conditions) == 0 {
		return "", args, argIndex
	}

	return "WHERE " + strings.Join(conditions, " AND "), args, argIndex
}

func (r *PostgresAuditLogRepository) countAuditLogs(whereClause string, args []interface{}) (int, error) {
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs %s", whereClause)
	var total int
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to count audit logs: %w", err)
	}
	return total, nil
}

func normalizePagination(limit int, offset int) (int, int) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func buildAuditLogQuery(whereClause string, argIndex int) string {
	return fmt.Sprintf(`
		SELECT id, event_type, resource_type, resource_id, resource_name, user_id, user_name, details, ip_address, user_agent, created_at
		FROM audit_logs
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)
}

func scanAuditLogRow(rows *sql.Rows) (*domain.AuditLog, error) {
	var log domain.AuditLog
	var resourceID, resourceName, userID, userName, ipAddress, userAgent sql.NullString
	var details []byte

	err := rows.Scan(
		&log.ID,
		&log.EventType,
		&log.ResourceType,
		&resourceID,
		&resourceName,
		&userID,
		&userName,
		&details,
		&ipAddress,
		&userAgent,
		&log.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if resourceID.Valid {
		log.ResourceID = &resourceID.String
	}
	if resourceName.Valid {
		log.ResourceName = &resourceName.String
	}
	if userID.Valid {
		log.UserID = &userID.String
	}
	if userName.Valid {
		log.UserName = &userName.String
	}
	if ipAddress.Valid {
		log.IPAddress = &ipAddress.String
	}
	if userAgent.Valid {
		log.UserAgent = &userAgent.String
	}
	log.Details = details

	return &log, nil
}

func (r *PostgresAuditLogRepository) DeleteOlderThan(days int) (int64, error) {
	query := `DELETE FROM audit_logs WHERE created_at < NOW() - ($1 || ' days')::INTERVAL`
	result, err := r.db.Exec(query, days)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old audit logs: %w", err)
	}
	return result.RowsAffected()
}
