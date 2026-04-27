package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paasdeploy/mcp/internal/toolkit"
)

type databaseTLSConfigureInput struct {
	ContainerID  string `json:"container_id" jsonschema:"the container ID or name"`
	ServerID     string `json:"server_id" jsonschema:"remote server ID hosting the database container"`
	DatabaseUser string `json:"database_user" jsonschema:"database superuser used to enable TLS (e.g. 'postgres')"`
	DatabaseName string `json:"database_name" jsonschema:"database name to operate on (e.g. 'postgres')"`
	DatabaseType string `json:"database_type,omitempty" jsonschema:"database engine; defaults to 'postgresql'"`
}

type databaseTLSStatusInput struct {
	ContainerID  string `json:"container_id" jsonschema:"the container ID or name"`
	ServerID     string `json:"server_id" jsonschema:"remote server ID hosting the database container"`
	DatabaseUser string `json:"database_user" jsonschema:"database superuser to inspect TLS settings"`
	DatabaseName string `json:"database_name" jsonschema:"database name to inspect"`
	DatabaseType string `json:"database_type,omitempty" jsonschema:"database engine; defaults to 'postgresql'"`
}

func RegisterSSL(srv *mcp.Server, deps toolkit.Deps) {
	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "database_tls_status",
			Description: "Inspect TLS status of a managed database container (Postgres/MySQL). Returns sslEnabled, tlsVersion, cipher and a connection string when enabled. Requires scope 'read'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in databaseTLSStatusInput) (any, error) {
			if err := validateDatabaseTLS(in.ContainerID, in.ServerID, in.DatabaseUser, in.DatabaseName); err != nil {
				return nil, err
			}
			return getJSON(ctx, deps.Backend, "/containers/"+pathSeg(in.ContainerID)+"/ssl", map[string]any{
				"serverId":     in.ServerID,
				"databaseType": defaultDatabaseType(in.DatabaseType),
				"databaseUser": in.DatabaseUser,
				"databaseName": in.DatabaseName,
			})
		})

	toolkit.RegisterWrite(srv, deps,
		&mcp.Tool{
			Name:        "database_tls_configure",
			Description: "Enable TLS for a managed database container (Postgres/MySQL) on a remote server. Generates internal TLS certificates and reconfigures the engine; this is NOT for HTTP/Traefik domain SSL. Requires scope 'config:write'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in databaseTLSConfigureInput) (any, error) {
			if err := validateDatabaseTLS(in.ContainerID, in.ServerID, in.DatabaseUser, in.DatabaseName); err != nil {
				return nil, err
			}
			body := map[string]any{
				"serverId":     in.ServerID,
				"databaseType": defaultDatabaseType(in.DatabaseType),
				"databaseUser": in.DatabaseUser,
				"databaseName": in.DatabaseName,
			}
			return postJSON(ctx, deps.Backend, "/containers/"+pathSeg(in.ContainerID)+"/ssl", body, nil)
		})
}

func validateDatabaseTLS(containerID, serverID, dbUser, dbName string) error {
	if containerID == "" {
		return errInvalidArg("container_id is required")
	}
	if serverID == "" {
		return errInvalidArg("server_id is required")
	}
	if dbUser == "" {
		return errInvalidArg("database_user is required")
	}
	if dbName == "" {
		return errInvalidArg("database_name is required")
	}
	return nil
}

func defaultDatabaseType(dbType string) string {
	if dbType == "" {
		return "postgresql"
	}
	return dbType
}
