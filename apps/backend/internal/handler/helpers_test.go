package handler

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestParseSinceQuery_AbsentReturnsNilNil(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		got, err := ParseSinceQuery(c)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if got != nil {
			t.Errorf("expected nil time, got %v", got)
		}
		return c.SendStatus(fiber.StatusOK)
	})
	if _, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/", nil), -1); err != nil {
		t.Fatal(err)
	}
}

func TestParseSinceQuery_ValidRFC3339(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		got, err := ParseSinceQuery(c)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if got == nil {
			t.Fatalf("expected non-nil time")
		}
		if got.Year() != 2026 || got.Month() != 4 || got.Day() != 30 {
			t.Errorf("unexpected time: %v", got)
		}
		return c.SendStatus(fiber.StatusOK)
	})
	req := httptest.NewRequest(fiber.MethodGet, "/?since=2026-04-30T19:00:00Z", nil)
	if _, err := app.Test(req, -1); err != nil {
		t.Fatal(err)
	}
}

func TestParseSinceQuery_InvalidReturnsError(t *testing.T) {
	tests := []string{"yesterday", "2026-04-30", "1h", "not-a-time"}
	for _, raw := range tests {
		t.Run(raw, func(t *testing.T) {
			app := fiber.New()
			app.Get("/", func(c *fiber.Ctx) error {
				got, err := ParseSinceQuery(c)
				if err == nil {
					t.Errorf("expected error for %q, got nil (parsed=%v)", raw, got)
				}
				if got != nil {
					t.Errorf("expected nil time on error, got %v", got)
				}
				return c.SendStatus(fiber.StatusOK)
			})
			req := httptest.NewRequest(fiber.MethodGet, "/?since="+raw, nil)
			if _, err := app.Test(req, -1); err != nil {
				t.Fatal(err)
			}
		})
	}
}
