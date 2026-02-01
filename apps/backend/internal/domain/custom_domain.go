package domain

import (
	"context"
	"time"
)

type CustomDomain struct {
	ID          string
	AppID       string
	Domain      string
	PathPrefix  string
	ZoneID      string
	DNSRecordID string
	RecordType  string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CreateCustomDomainInput struct {
	AppID       string
	Domain      string
	PathPrefix  string
	ZoneID      string
	DNSRecordID string
	RecordType  string
}

type CustomDomainRepository interface {
	Create(ctx context.Context, input CreateCustomDomainInput) (*CustomDomain, error)
	FindByID(ctx context.Context, id string) (*CustomDomain, error)
	FindByAppID(ctx context.Context, appID string) ([]CustomDomain, error)
	FindByDomain(ctx context.Context, domain string) (*CustomDomain, error)
	FindByDomainAndPath(ctx context.Context, domain, pathPrefix string) (*CustomDomain, error)
	Delete(ctx context.Context, id string) error
	DeleteByAppID(ctx context.Context, appID string) error
}
