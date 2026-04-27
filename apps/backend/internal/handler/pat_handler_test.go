package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/service"
)

func patTestRequest(path string) *http.Request {
	return httptest.NewRequest(http.MethodGet, path, nil)
}

func TestMapPATErrorMapsInvalidName(t *testing.T) {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/x", func(c *fiber.Ctx) error {
		return mapPATError(c, service.ErrInvalidTokenName)
	})

	resp, err := app.Test(patTestRequest("/x"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestMapPATErrorMapsNoScopes(t *testing.T) {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/x", func(c *fiber.Ctx) error {
		return mapPATError(c, service.ErrNoScopesProvided)
	})

	resp, err := app.Test(patTestRequest("/x"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestMapPATErrorMapsInvalidScope(t *testing.T) {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/x", func(c *fiber.Ctx) error {
		return mapPATError(c, service.ErrInvalidScope)
	})

	resp, err := app.Test(patTestRequest("/x"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestMapPATErrorMapsExpiryRange(t *testing.T) {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/x", func(c *fiber.Ctx) error {
		return mapPATError(c, service.ErrExpiryOutOfRange)
	})

	resp, err := app.Test(patTestRequest("/x"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestMapPATErrorMapsUnknownErrorAsInternal(t *testing.T) {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/x", func(c *fiber.Ctx) error {
		return mapPATError(c, errors.New("boom"))
	})

	resp, err := app.Test(patTestRequest("/x"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
}

func TestToTokenResponseCopiesAllFields(t *testing.T) {
	token := &domain.PersonalAccessToken{
		ID:          "tok-1",
		Name:        "demo",
		TokenPrefix: "pdp_live_abc",
		Scopes:      []string{"read", "deploy"},
	}

	resp := toTokenResponse(token)

	if resp.ID != "tok-1" {
		t.Fatalf("unexpected ID: %s", resp.ID)
	}
	if resp.Name != "demo" {
		t.Fatalf("unexpected Name: %s", resp.Name)
	}
	if len(resp.Scopes) != 2 {
		t.Fatalf("expected 2 scopes, got %d", len(resp.Scopes))
	}
}
