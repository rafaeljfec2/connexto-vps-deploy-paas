package handler

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"
	_ "github.com/paasdeploy/backend/internal/docs"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/response"
	"github.com/paasdeploy/backend/internal/service"
)

type AppHandler struct {
	appService *service.AppService
}

// RedeployInput representa o input para redeploy
type RedeployInput struct {
	CommitSHA string `json:"commitSha,omitempty" example:"abc123def"`
}

func NewAppHandler(appService *service.AppService) *AppHandler {
	return &AppHandler{
		appService: appService,
	}
}

func (h *AppHandler) Register(app *fiber.App) {
	v1 := app.Group("/paas-deploy/v1")

	apps := v1.Group("/apps")
	apps.Get("/", h.ListApps)
	apps.Post("/", h.CreateApp)
	apps.Get("/:id", h.GetApp)
	apps.Delete("/:id", h.DeleteApp)
	apps.Get("/:id/deployments", h.ListDeployments)
	apps.Post("/:id/redeploy", h.TriggerRedeploy)
	apps.Post("/:id/rollback", h.TriggerRollback)

	apps.Post("/:id/webhook", h.SetupWebhook)
	apps.Delete("/:id/webhook", h.RemoveWebhook)
	apps.Get("/:id/webhook/status", h.GetWebhookStatus)
	apps.Get("/:id/commits", h.ListCommits)
}

// ListApps godoc
//
//	@Summary		Lista todas as aplicacoes
//	@Description	Retorna lista de apps cadastrados no sistema
//	@Tags			apps
//	@Produce		json
//	@Success		200	{array}		docs.App
//	@Failure		500	{object}	docs.ErrorInfo
//	@Router			/apps [get]
func (h *AppHandler) ListApps(c *fiber.Ctx) error {
	apps, err := h.appService.ListApps()
	if err != nil {
		return h.handleError(c, err)
	}

	return response.OK(c, apps)
}

// CreateApp godoc
//
//	@Summary		Cria uma nova aplicacao
//	@Description	Cadastra um novo app para deploy automatico
//	@Tags			apps
//	@Accept			json
//	@Produce		json
//	@Param			input	body		docs.CreateAppInput	true	"Dados do app"
//	@Success		201		{object}	docs.App
//	@Failure		400		{object}	docs.ErrorInfo
//	@Failure		409		{object}	docs.ErrorInfo
//	@Router			/apps [post]
func (h *AppHandler) CreateApp(c *fiber.Ctx) error {
	var input domain.CreateAppInput
	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	ctx := context.Background()
	app, err := h.appService.CreateApp(ctx, input)
	if err != nil {
		return h.handleError(c, err)
	}

	return response.Created(c, app)
}

// GetApp godoc
//
//	@Summary		Busca uma aplicacao por ID
//	@Description	Retorna os detalhes de um app especifico
//	@Tags			apps
//	@Produce		json
//	@Param			id	path		string	true	"ID do app"
//	@Success		200	{object}	docs.App
//	@Failure		404	{object}	docs.ErrorInfo
//	@Router			/apps/{id} [get]
func (h *AppHandler) GetApp(c *fiber.Ctx) error {
	id := c.Params("id")

	app, err := h.appService.GetApp(id)
	if err != nil {
		return h.handleError(c, err)
	}

	return response.OK(c, app)
}

// DeleteApp godoc
//
//	@Summary		Remove uma aplicacao
//	@Description	Deleta um app. Use ?purge=true para remover completamente (containers, imagens, arquivos, banco)
//	@Tags			apps
//	@Param			id		path	string	true	"ID do app"
//	@Param			purge	query	bool	false	"Se true, remove completamente o app (hard delete)"
//	@Success		204	"No Content"
//	@Failure		404	{object}	docs.ErrorInfo
//	@Router			/apps/{id} [delete]
func (h *AppHandler) DeleteApp(c *fiber.Ctx) error {
	id := c.Params("id")
	purge := c.QueryBool("purge", false)

	ctx := context.Background()

	var err error
	if purge {
		err = h.appService.PurgeApp(ctx, id)
	} else {
		err = h.appService.DeleteApp(ctx, id)
	}

	if err != nil {
		return h.handleError(c, err)
	}

	return response.NoContent(c)
}

// ListDeployments godoc
//
//	@Summary		Lista deploys de uma aplicacao
//	@Description	Retorna historico de deploys de um app
//	@Tags			deployments
//	@Produce		json
//	@Param			id	path		string	true	"ID do app"
//	@Success		200	{array}		docs.Deployment
//	@Failure		404	{object}	docs.ErrorInfo
//	@Router			/apps/{id}/deployments [get]
func (h *AppHandler) ListDeployments(c *fiber.Ctx) error {
	appID := c.Params("id")

	deployments, err := h.appService.ListDeployments(appID)
	if err != nil {
		return h.handleError(c, err)
	}

	return response.OK(c, deployments)
}

// TriggerRedeploy godoc
//
//	@Summary		Dispara um novo deploy
//	@Description	Inicia um deploy manual do app
//	@Tags			deployments
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string			true	"ID do app"
//	@Param			input	body		RedeployInput	false	"Commit SHA opcional"
//	@Success		201		{object}	docs.Deployment
//	@Failure		404		{object}	docs.ErrorInfo
//	@Failure		409		{object}	docs.ErrorInfo
//	@Router			/apps/{id}/redeploy [post]
func (h *AppHandler) TriggerRedeploy(c *fiber.Ctx) error {
	appID := c.Params("id")

	var input struct {
		CommitSHA string `json:"commitSha,omitempty"`
	}
	_ = c.BodyParser(&input)

	deployment, err := h.appService.TriggerDeploy(appID, input.CommitSHA)
	if err != nil {
		return h.handleError(c, err)
	}

	return response.Created(c, deployment)
}

// TriggerRollback godoc
//
//	@Summary		Faz rollback do deploy
//	@Description	Reverte para o ultimo deploy bem-sucedido
//	@Tags			deployments
//	@Produce		json
//	@Param			id	path		string	true	"ID do app"
//	@Success		201	{object}	docs.Deployment
//	@Failure		404	{object}	docs.ErrorInfo
//	@Router			/apps/{id}/rollback [post]
func (h *AppHandler) TriggerRollback(c *fiber.Ctx) error {
	appID := c.Params("id")

	deployment, err := h.appService.TriggerRollback(appID)
	if err != nil {
		return h.handleError(c, err)
	}

	return response.Created(c, deployment)
}

// SetupWebhook godoc
//
//	@Summary		Configura webhook do GitHub
//	@Description	Cria webhook automatico no repositorio GitHub
//	@Tags			apps
//	@Produce		json
//	@Param			id	path		string	true	"ID do app"
//	@Success		200	{object}	docs.SetupResult
//	@Failure		400	{object}	docs.ErrorInfo
//	@Failure		404	{object}	docs.ErrorInfo
//	@Router			/apps/{id}/webhook [post]
func (h *AppHandler) SetupWebhook(c *fiber.Ctx) error {
	id := c.Params("id")

	ctx := context.Background()
	result, err := h.appService.SetupWebhook(ctx, id)
	if err != nil {
		return h.handleError(c, err)
	}

	return response.OK(c, result)
}

// RemoveWebhook godoc
//
//	@Summary		Remove webhook do GitHub
//	@Description	Deleta webhook do repositorio GitHub
//	@Tags			apps
//	@Param			id	path	string	true	"ID do app"
//	@Success		204	"No Content"
//	@Failure		404	{object}	docs.ErrorInfo
//	@Router			/apps/{id}/webhook [delete]
func (h *AppHandler) RemoveWebhook(c *fiber.Ctx) error {
	id := c.Params("id")

	ctx := context.Background()
	if err := h.appService.RemoveWebhook(ctx, id); err != nil {
		return h.handleError(c, err)
	}

	return response.NoContent(c)
}

// GetWebhookStatus godoc
//
//	@Summary		Verifica status do webhook
//	@Description	Retorna o status do webhook configurado
//	@Tags			apps
//	@Produce		json
//	@Param			id	path		string	true	"ID do app"
//	@Success		200	{object}	docs.WebhookStatus
//	@Failure		404	{object}	docs.ErrorInfo
//	@Router			/apps/{id}/webhook/status [get]
func (h *AppHandler) GetWebhookStatus(c *fiber.Ctx) error {
	id := c.Params("id")

	ctx := context.Background()
	status, err := h.appService.GetWebhookStatus(ctx, id)
	if err != nil {
		return h.handleError(c, err)
	}

	return response.OK(c, status)
}

// ListCommits godoc
//
//	@Summary		Lista commits do repositorio
//	@Description	Retorna os ultimos commits do branch do app
//	@Tags			apps
//	@Produce		json
//	@Param			id		path		string	true	"ID do app"
//	@Param			limit	query		int		false	"Numero de commits a retornar"	default(20)
//	@Success		200		{array}		object
//	@Failure		404		{object}	docs.ErrorInfo
//	@Router			/apps/{id}/commits [get]
func (h *AppHandler) ListCommits(c *fiber.Ctx) error {
	id := c.Params("id")
	limit := c.QueryInt("limit", 20)

	ctx := context.Background()
	commits, err := h.appService.ListCommits(ctx, id, limit)
	if err != nil {
		return h.handleError(c, err)
	}

	return response.OK(c, commits)
}

func (h *AppHandler) handleError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return response.NotFound(c, err.Error())
	case errors.Is(err, domain.ErrAlreadyExists):
		return response.Conflict(c, err.Error())
	case errors.Is(err, domain.ErrInvalidInput):
		return response.BadRequest(c, err.Error())
	case errors.Is(err, domain.ErrDeployInProgress):
		return response.Conflict(c, err.Error())
	case errors.Is(err, domain.ErrNoDeployAvailable):
		return response.NotFound(c, err.Error())
	case errors.Is(err, domain.ErrWebhookNotConfigured):
		return response.BadRequest(c, err.Error())
	default:
		return response.InternalError(c)
	}
}
