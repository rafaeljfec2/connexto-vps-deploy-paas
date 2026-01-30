package docs

import (
	"encoding/json"
	"time"
)

// App representa uma aplicacao cadastrada no sistema
// @Description Aplicacao cadastrada para deploy automatico
type App struct {
	ID             string          `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name           string          `json:"name" example:"my-app"`
	RepositoryURL  string          `json:"repositoryUrl" example:"https://github.com/owner/repo.git"`
	Branch         string          `json:"branch" example:"main"`
	Workdir        string          `json:"workdir" example:"."`
	Config         json.RawMessage `json:"config" swaggertype:"object"`
	Status         string          `json:"status" example:"active" enums:"active,inactive,deleted"`
	WebhookID      *int64          `json:"webhookId,omitempty" example:"123456789"`
	LastDeployedAt *time.Time      `json:"lastDeployedAt,omitempty"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
}

// CreateAppInput representa os dados para criar uma nova aplicacao
// @Description Dados necessarios para cadastrar um novo app
type CreateAppInput struct {
	Name          string          `json:"name" example:"my-app" binding:"required"`
	RepositoryURL string          `json:"repositoryUrl" example:"https://github.com/owner/repo.git" binding:"required"`
	Branch        string          `json:"branch" example:"main"`
	Workdir       string          `json:"workdir" example:"."`
	Config        json.RawMessage `json:"config,omitempty" swaggertype:"object"`
}

// Deployment representa um deploy de uma aplicacao
// @Description Registro de um deploy realizado
type Deployment struct {
	ID               string     `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	AppID            string     `json:"appId" example:"550e8400-e29b-41d4-a716-446655440000"`
	CommitSHA        string     `json:"commitSha" example:"abc123def456"`
	CommitMessage    string     `json:"commitMessage,omitempty" example:"feat: add new feature"`
	Status           string     `json:"status" example:"success" enums:"pending,running,success,failed,cancelled"`
	StartedAt        *time.Time `json:"startedAt,omitempty"`
	FinishedAt       *time.Time `json:"finishedAt,omitempty"`
	ErrorMessage     string     `json:"errorMessage,omitempty"`
	Logs             string     `json:"logs,omitempty"`
	PreviousImageTag string     `json:"previousImageTag,omitempty"`
	CurrentImageTag  string     `json:"currentImageTag,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
}

// SetupResult representa o resultado do setup de webhook
// @Description Resultado da configuracao do webhook
type SetupResult struct {
	WebhookID int64  `json:"webhookId" example:"123456789"`
	Provider  string `json:"provider" example:"github"`
	Active    bool   `json:"active" example:"true"`
}

// WebhookStatus representa o status do webhook
// @Description Status atual do webhook configurado
type WebhookStatus struct {
	Exists     bool       `json:"exists" example:"true"`
	Active     bool       `json:"active" example:"true"`
	LastPingAt *time.Time `json:"lastPingAt,omitempty"`
	Error      string     `json:"error,omitempty"`
}

// Envelope representa a estrutura padrao de resposta da API
// @Description Envelope padrao de resposta
type Envelope struct {
	Success bool        `json:"success" example:"true"`
	Data    interface{} `json:"data"`
	Error   *ErrorInfo  `json:"error"`
	Meta    Meta        `json:"meta"`
}

// ErrorInfo representa informacoes de erro
// @Description Detalhes do erro
type ErrorInfo struct {
	Code    string      `json:"code" example:"NOT_FOUND"`
	Message string      `json:"message" example:"resource not found"`
	Details interface{} `json:"details,omitempty"`
}

// Meta representa metadados da resposta
// @Description Metadados da resposta
type Meta struct {
	TraceID string `json:"traceId,omitempty" example:"abc123"`
}
