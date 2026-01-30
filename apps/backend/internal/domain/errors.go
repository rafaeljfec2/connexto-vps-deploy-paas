package domain

import "errors"

var (
	ErrNotFound          = errors.New("resource not found")
	ErrAlreadyExists     = errors.New("resource already exists")
	ErrInvalidInput      = errors.New("invalid input")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrForbidden         = errors.New("forbidden")
	ErrConflict          = errors.New("resource conflict")
	ErrInternalError     = errors.New("internal server error")
	ErrDeployInProgress  = errors.New("deployment already in progress for this app")
	ErrNoDeployAvailable = errors.New("no deployment available for rollback")
	ErrHealthCheckFailed = errors.New("health check failed")
	ErrBuildFailed       = errors.New("docker build failed")
	ErrGitSyncFailed     = errors.New("git sync failed")
	ErrTimeout           = errors.New("operation timed out")
)
