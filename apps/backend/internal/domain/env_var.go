package domain

import "time"

type EnvVar struct {
	ID        string    `json:"id"`
	AppID     string    `json:"appId"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	IsSecret  bool      `json:"isSecret"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type EnvVarResponse struct {
	ID        string    `json:"id"`
	AppID     string    `json:"appId"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	IsSecret  bool      `json:"isSecret"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (e *EnvVar) ToResponse() EnvVarResponse {
	value := e.Value
	if e.IsSecret {
		value = "••••••••"
	}
	return EnvVarResponse{
		ID:        e.ID,
		AppID:     e.AppID,
		Key:       e.Key,
		Value:     value,
		IsSecret:  e.IsSecret,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

type CreateEnvVarInput struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	IsSecret bool   `json:"isSecret"`
}

type UpdateEnvVarInput struct {
	Value    *string `json:"value,omitempty"`
	IsSecret *bool   `json:"isSecret,omitempty"`
}

type BulkEnvVarInput struct {
	Vars []CreateEnvVarInput `json:"vars"`
}

type EnvVarRepository interface {
	FindByAppID(appID string) ([]EnvVar, error)
	FindByAppIDAndKey(appID, key string) (*EnvVar, error)
	Create(appID string, input CreateEnvVarInput) (*EnvVar, error)
	Update(id string, input UpdateEnvVarInput) (*EnvVar, error)
	Delete(id string) error
	DeleteByAppIDAndKey(appID, key string) error
	BulkUpsert(appID string, vars []CreateEnvVarInput) error
}
