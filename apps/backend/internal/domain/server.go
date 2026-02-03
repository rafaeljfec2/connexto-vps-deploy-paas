package domain

import (
	"time"
)

type ServerStatus string

const (
	ServerStatusPending      ServerStatus = "pending"
	ServerStatusProvisioning ServerStatus = "provisioning"
	ServerStatusOnline       ServerStatus = "online"
	ServerStatusOffline      ServerStatus = "offline"
	ServerStatusError        ServerStatus = "error"
)

type Server struct {
	ID                   string       `json:"id"`
	Name                 string       `json:"name"`
	Host                 string       `json:"host"`
	SSHPort              int          `json:"sshPort"`
	SSHUser              string       `json:"sshUser"`
	SSHKeyEncrypted      string       `json:"-"`
	SSHPasswordEncrypted string       `json:"-"`
	Status               ServerStatus `json:"status"`
	AgentVersion         *string      `json:"agentVersion,omitempty"`
	LastHeartbeatAt      *time.Time   `json:"lastHeartbeatAt,omitempty"`
	CreatedAt            time.Time    `json:"createdAt"`
	UpdatedAt            time.Time    `json:"updatedAt"`
}

type CreateServerInput struct {
	Name                 string `json:"name"`
	Host                 string `json:"host"`
	SSHPort              int    `json:"sshPort"`
	SSHUser              string `json:"sshUser"`
	SSHKeyEncrypted      string `json:"-"`
	SSHPasswordEncrypted string `json:"-"`
}

type UpdateServerInput struct {
	Name                 *string       `json:"name,omitempty"`
	Host                 *string       `json:"host,omitempty"`
	SSHPort              *int          `json:"sshPort,omitempty"`
	SSHUser              *string       `json:"sshUser,omitempty"`
	SSHKeyEncrypted      *string       `json:"-"`
	SSHPasswordEncrypted *string       `json:"-"`
	Status               *ServerStatus `json:"status,omitempty"`
}

type ServerRepository interface {
	Create(input CreateServerInput) (*Server, error)
	FindByID(id string) (*Server, error)
	FindAll() ([]Server, error)
	Update(id string, input UpdateServerInput) (*Server, error)
	UpdateHeartbeat(id string, agentVersion string) error
	Delete(id string) error
}
