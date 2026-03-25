package requestctx

import (
	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/domain"
)

const userContextKey = "user"

func SetUserInContext(c *fiber.Ctx, user *domain.User) {
	c.Locals(userContextKey, user)
}

func GetUserFromContext(c *fiber.Ctx) *domain.User {
	user, ok := c.Locals(userContextKey).(*domain.User)
	if !ok {
		return nil
	}
	return user
}
