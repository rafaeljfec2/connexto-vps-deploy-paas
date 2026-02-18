package handler

import (
	"github.com/gofiber/fiber/v2"
)

func SetCookie(c *fiber.Ctx, name, value string, maxAge int, secureCookie bool, cookieDomain string) {
	sameSite := "Lax"
	secure := secureCookie
	if cookieDomain != "" {
		sameSite = "None"
		secure = true
	}
	c.Cookie(&fiber.Cookie{
		Name:     name,
		Value:    value,
		HTTPOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		MaxAge:   maxAge,
		Path:     "/",
		Domain:   cookieDomain,
	})
}

func ClearCookie(c *fiber.Ctx, name string, secureCookie bool, cookieDomain string) {
	SetCookie(c, name, "", -1, secureCookie, cookieDomain)
}

func RedirectWithError(c *fiber.Ctx, baseURL, pathWithQuery, errorCode string) error {
	return c.Redirect(baseURL+pathWithQuery+errorCode, fiber.StatusTemporaryRedirect)
}
