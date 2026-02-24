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
	UserID               string       `json:"userId"`
	Name                 string       `json:"name"`
	Host                 string       `json:"host"`
	SSHPort              int          `json:"sshPort"`
	SSHUser              string       `json:"sshUser"`
	SSHKeyEncrypted      string       `json:"-"`
	SSHPasswordEncrypted string       `json:"-"`
	AcmeEmail            *string      `json:"acmeEmail,omitempty"`
	SSHHostKey           string       `json:"-"`
	Status               ServerStatus `json:"status"`
	AgentVersion         *string      `json:"agentVersion,omitempty"`
	AgentUpdateMode      string       `json:"agentUpdateMode"`
	LastHeartbeatAt      *time.Time   `json:"lastHeartbeatAt,omitempty"`
	CreatedAt            time.Time    `json:"createdAt"`
	UpdatedAt            time.Time    `json:"updatedAt"`
}

type CreateServerInput struct {
	UserID               string  `json:"-"`
	Name                 string  `json:"name"`
	Host                 string  `json:"host"`
	SSHPort              int     `json:"sshPort"`
	SSHUser              string  `json:"sshUser"`
	SSHKeyEncrypted      string  `json:"-"`
	SSHPasswordEncrypted string  `json:"-"`
	AcmeEmail            *string `json:"acmeEmail,omitempty"`
}

type UpdateServerInput struct {
	Name                 *string       `json:"name,omitempty"`
	Host                 *string       `json:"host,omitempty"`
	SSHPort              *int          `json:"sshPort,omitempty"`
	SSHUser              *string       `json:"sshUser,omitempty"`
	SSHKeyEncrypted      *string       `json:"-"`
	SSHPasswordEncrypted *string       `json:"-"`
	AcmeEmail            *string       `json:"acmeEmail,omitempty"`
	Status               *ServerStatus `json:"status,omitempty"`
	AgentUpdateMode      *string       `json:"agentUpdateMode,omitempty"`
}

type ServerRepository interface {
	Create(input CreateServerInput) (*Server, error)
	FindByID(id string) (*Server, error)
	FindByIDForUser(id string, userID string) (*Server, error)
	FindAll() ([]Server, error)
	FindAllByUserID(userID string) ([]Server, error)
	Update(id string, input UpdateServerInput) (*Server, error)
	UpdateHeartbeat(id string, agentVersion string) error
	UpdateSSHHostKey(id string, hostKey string) error
	Delete(id string) error
}
