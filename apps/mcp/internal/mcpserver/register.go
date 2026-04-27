package mcpserver

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paasdeploy/mcp/internal/toolkit"
	"github.com/paasdeploy/mcp/internal/tools"
)

func RegisterAllReadOnly(srv *mcp.Server, deps toolkit.Deps) {
	tools.RegisterApps(srv, deps)
	tools.RegisterContainers(srv, deps)
	tools.RegisterServers(srv, deps)
	tools.RegisterImages(srv, deps)
	tools.RegisterResources(srv, deps)
	tools.RegisterTemplates(srv, deps)
	tools.RegisterGitHub(srv, deps)
	tools.RegisterAudit(srv, deps)
	tools.RegisterSystem(srv, deps)
	RegisterMCPResources(srv, deps)
	RegisterMCPPrompts(srv, deps)
}

func RegisterAllWrites(srv *mcp.Server, deps toolkit.Deps) {
	tools.RegisterDeploys(srv, deps)
	tools.RegisterEnvVars(srv, deps)
	tools.RegisterDomains(srv, deps)
	tools.RegisterSSL(srv, deps)
}

func RegisterAllDestructive(srv *mcp.Server, deps toolkit.Deps) {
	tools.RegisterDestructive(srv, deps)
}
