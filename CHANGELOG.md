# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Monorepo Support**: New `workdir` field in application configuration allows deploying specific subdirectories from monorepo repositories
- **Theme System**: Light, dark, and system theme support with persistent user preference
- **Database Migrations**: Automatic schema management using golang-migrate on backend startup
- **Reusable Components**: New frontend components for improved consistency
  - `PageHeader`: Standardized page titles with back navigation and actions
  - `IconText`: Icon + text display for metadata fields
  - `FormField`: Form input wrapper with label, helper text, and error handling
  - `LoadingGrid`: Skeleton loader grid for loading states
  - `ErrorMessage`: Consistent error display component
- **Developer Experience**:
  - VSCode launch configurations for debugging frontend and backend
  - VSCode tasks for common development operations
  - Recommended extensions list
  - ESLint flat config with unused imports cleanup
  - Prettier with import sorting

### Changed

- **Worker Refactoring**: Reduced `NewWorker` parameters from 8 to 3 using `WorkerDeps` struct for dependency injection
- **Configuration**: Extracted magic numbers to named constants (`outputChannelBuffer`, `sseEventBufferSize`, etc.)
- **Docker Client**: Extracted `getImagePrefix()` helper to reduce code duplication
- **Frontend Layout**: Improved container configuration for better responsive design
- **Header Component**: Increased height and integrated theme toggle

### Fixed

- Missing error handling for `c.BodyParser(&input)` in app handler
- Documentation inconsistencies (Go version, PostgreSQL version)
- TypeScript compatibility by using `replace()` instead of `replaceAll()`

### Security

- All comments translated to English and unnecessary comments removed
- Environment variable documentation with `.env.example` files

## [0.1.0] - Initial Release

### Added

- Core deployment pipeline with worker pool
- PostgreSQL-backed deployment queue with `SELECT FOR UPDATE SKIP LOCKED`
- Real-time deployment logs via Server-Sent Events (SSE)
- Rollback support with automatic health check verification
- Docker-based application containerization
- React dashboard with React Query state management
- Traefik reverse proxy integration
- Health check system with configurable retries and timeouts
- Application CRUD operations via REST API
- Structured logging with `slog`
