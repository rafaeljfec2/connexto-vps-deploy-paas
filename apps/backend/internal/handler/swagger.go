package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"

	_ "github.com/paasdeploy/backend/docs"
)

type SwaggerHandler struct{}

func NewSwaggerHandler() *SwaggerHandler {
	return &SwaggerHandler{}
}

func (h *SwaggerHandler) Register(app *fiber.App) {
	app.Get("/paas-deploy/v1/swagger/*", swagger.HandlerDefault)
}
