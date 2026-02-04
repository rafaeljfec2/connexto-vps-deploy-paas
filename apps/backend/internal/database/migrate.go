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
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			logger.Info("Database is up to date, no migrations to run")
			return nil
		}
		if strings.Contains(err.Error(), "Dirty database") {
			version, dirty, verErr := m.Version()
			if verErr == nil && dirty && version > 0 {
				logger.Warn("Detected dirty migration state, forcing previous version to retry", "version", version)
				if forceErr := m.Force(int(version) - 1); forceErr != nil {
					return fmt.Errorf("failed to force migration version: %w", forceErr)
				}
				if retryErr := m.Up(); retryErr != nil && !errors.Is(retryErr, migrate.ErrNoChange) {
					return fmt.Errorf("failed to run migrations after dirty fix: %w", retryErr)
				}
			} else {
				return fmt.Errorf("failed to run migrations: %w", err)
			}
		} else {
			return fmt.Errorf("failed to run migrations: %w", err)
		}
	}

	version, dirty, _ := m.Version()
	logger.Info("Migrations completed successfully", "version", version, "dirty", dirty)

	return nil
}
