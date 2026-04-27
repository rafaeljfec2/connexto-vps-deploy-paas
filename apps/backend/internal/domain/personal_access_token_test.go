package domain

import (
	"errors"
	"testing"
	"time"
)

func TestIsActiveReturnsErrorWhenRevoked(t *testing.T) {
	now := time.Now()
	token := &PersonalAccessToken{RevokedAt: &now}

	err := token.IsActive(now)

	if !errors.Is(err, ErrTokenRevoked) {
		t.Fatalf("expected ErrTokenRevoked, got %v", err)
	}
}

func TestIsActiveReturnsErrorWhenExpired(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	token := &PersonalAccessToken{ExpiresAt: &past}

	err := token.IsActive(time.Now())

	if !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("expected ErrTokenExpired, got %v", err)
	}
}

func TestIsActiveReturnsNilWhenValid(t *testing.T) {
	future := time.Now().Add(1 * time.Hour)
	token := &PersonalAccessToken{ExpiresAt: &future}

	err := token.IsActive(time.Now())

	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestIsActiveAllowsNilExpiry(t *testing.T) {
	token := &PersonalAccessToken{}

	err := token.IsActive(time.Now())

	if err != nil {
		t.Fatalf("expected nil error for non-expiring token, got %v", err)
	}
}

func TestHasScopeReturnsFalseForMissingScope(t *testing.T) {
	token := &PersonalAccessToken{Scopes: []string{ScopeRead}}

	if token.HasScope(ScopeDestructive) {
		t.Fatal("expected false for missing scope")
	}
}

func TestHasScopeReturnsTrueForExactMatch(t *testing.T) {
	token := &PersonalAccessToken{Scopes: []string{ScopeDeploy}}

	if !token.HasScope(ScopeDeploy) {
		t.Fatal("expected true for exact scope match")
	}
}

func TestHasScopeReturnsTrueWhenAdmin(t *testing.T) {
	token := &PersonalAccessToken{Scopes: []string{ScopeAdmin}}

	if !token.HasScope(ScopeDestructive) {
		t.Fatal("expected admin scope to grant any permission")
	}
}

func TestIsValidScopeAcceptsKnownScopes(t *testing.T) {
	for _, scope := range AllScopes {
		if !IsValidScope(scope) {
			t.Fatalf("expected %q to be valid", scope)
		}
	}
}

func TestIsValidScopeRejectsUnknownScope(t *testing.T) {
	if IsValidScope("wildcard:*") {
		t.Fatal("expected unknown scope to be rejected")
	}
}
