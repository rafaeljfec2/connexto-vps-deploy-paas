package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(db *sql.DB, migrationsPath string, logger *slog.Logger) error {
	m, err := createMigrateInstance(db, migrationsPath)
	if err != nil {
		return err
	}
	if err := runUpWithDirtyRetry(m, logger); err != nil {
		return err
	}
	version, dirty, _ := m.Version()
	logger.Info("Migrations completed successfully", "version", version, "dirty", dirty)
	return nil
}

func createMigrateInstance(db *sql.DB, migrationsPath string) (*migrate.Migrate, error) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create migration driver: %w", err)
	}
	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}
	return m, nil
}

func runUpWithDirtyRetry(m *migrate.Migrate, logger *slog.Logger) error {
	err := m.Up()
	if err == nil {
		return nil
	}
	if errors.Is(err, migrate.ErrNoChange) {
		logger.Info("Database is up to date, no migrations to run")
		return nil
	}
	if !strings.Contains(err.Error(), "Dirty database") {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return retryAfterDirtyFix(m, logger, err)
}

func retryAfterDirtyFix(m *migrate.Migrate, logger *slog.Logger, originalErr error) error {
	version, dirty, verErr := m.Version()
	if verErr != nil || !dirty || version <= 0 {
		return fmt.Errorf("failed to run migrations: %w", originalErr)
	}
	logger.Warn("Detected dirty migration state, forcing previous version to retry", "version", version)
	if forceErr := m.Force(int(version) - 1); forceErr != nil {
		return fmt.Errorf("failed to force migration version: %w", forceErr)
	}
	retryErr := m.Up()
	if retryErr != nil && !errors.Is(retryErr, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations after dirty fix: %w", retryErr)
	}
	return nil
}
