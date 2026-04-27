package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paasdeploy/mcp/internal/backend"
	"github.com/paasdeploy/mcp/internal/config"
	"github.com/paasdeploy/mcp/internal/mcpserver"
	"github.com/paasdeploy/mcp/internal/toolkit"
	"github.com/paasdeploy/mcp/internal/transport"
)

const usage = `flowdeploy-mcp [stdio|serve] [flags]

Subcommands:
  stdio   (default) speak the MCP protocol over stdio (single-PAT, single-process)
  serve   expose the MCP transport via HTTP (multi-tenant, PAT forwarded per request)

Run 'flowdeploy-mcp <subcommand> --help' for the flags supported by each mode.`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "flowdeploy-mcp:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	subcommand, rest := splitSubcommand(args)
	switch subcommand {
	case "stdio":
		return runStdio(rest)
	case "serve":
		return runServe(rest)
	case "help", "-h", "--help":
		fmt.Fprintln(os.Stderr, usage)
		return nil
	default:
		return fmt.Errorf("unknown subcommand %q\n\n%s", subcommand, usage)
	}
}

func splitSubcommand(args []string) (string, []string) {
	if len(args) == 0 {
		return "stdio", nil
	}
	first := args[0]
	switch first {
	case "stdio", "serve", "help", "-h", "--help":
		return first, args[1:]
	}
	if len(first) > 0 && first[0] == '-' {
		return "stdio", args
	}
	return first, args[1:]
}

func runStdio(args []string) error {
	fs := flag.NewFlagSet("flowdeploy-mcp stdio", flag.ContinueOnError)
	flags := config.Flags{}
	flags.RegisterStdio(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := config.ResolveStdio(flags)
	if err != nil {
		return err
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	logger.Info("starting flowdeploy-mcp",
		"version", mcpserver.ServerVersion,
		"backend_url", cfg.BackendURL,
		"client_id", cfg.ClientID,
		"token", cfg.RedactedToken(),
		"transport", "stdio",
	)

	bk, err := backend.New(backend.Options{
		BaseURL:  cfg.BackendURL,
		Token:    cfg.Token,
		ClientID: cfg.ClientID,
		Timeout:  cfg.RequestTimeout,
		Logger:   logger,
	})
	if err != nil {
		return err
	}

	srv, err := buildServer(logger, bk)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := srv.Run(ctx, &mcp.StdioTransport{}); err != nil {
		return fmt.Errorf("mcp stdio: %w", err)
	}
	return nil
}

func runServe(args []string) error {
	fs := flag.NewFlagSet("flowdeploy-mcp serve", flag.ContinueOnError)
	flags := config.Flags{}
	flags.RegisterServe(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := config.ResolveServe(flags)
	if err != nil {
		return err
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	logger.Info("starting flowdeploy-mcp",
		"version", mcpserver.ServerVersion,
		"backend_url", cfg.BackendURL,
		"transport", "http",
		"addr", cfg.Addr,
		"read_rpm", cfg.ReadRPM,
		"mutate_rpm", cfg.MutateRPM,
		"allowed_clients", cfg.AllowedClients,
		"stateless", cfg.StatelessSession,
	)

	bk, err := backend.New(backend.Options{
		BaseURL:                cfg.BackendURL,
		Token:                  cfg.Token,
		ClientID:               cfg.ClientID,
		Timeout:                cfg.RequestTimeout,
		Logger:                 logger,
		AcceptTokenFromContext: true,
	})
	if err != nil {
		return err
	}

	srv, err := buildServer(logger, bk)
	if err != nil {
		return err
	}

	httpSrv, err := transport.NewServer(transport.ServerOptions{
		Addr:             cfg.Addr,
		AllowedClients:   cfg.AllowedClients,
		ReadRPM:          cfg.ReadRPM,
		MutateRPM:        cfg.MutateRPM,
		SessionMaxAge:    cfg.SessionMaxAge,
		StatelessSession: cfg.StatelessSession,
		Logger:           logger,
		MCPServer:        srv,
		ReadinessChecks: []transport.ReadinessCheck{
			{
				Name:  "backend",
				Probe: bk.Ping,
			},
		},
	})
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return httpSrv.Run(ctx)
}

func buildServer(logger *slog.Logger, bk *backend.Client) (*mcp.Server, error) {
	srv, err := mcpserver.New(mcpserver.Deps{Logger: logger, Backend: bk})
	if err != nil {
		return nil, err
	}
	deps := toolkit.Deps{Logger: logger, Backend: bk}
	mcpserver.RegisterAllReadOnly(srv, deps)
	mcpserver.RegisterAllWrites(srv, deps)
	mcpserver.RegisterAllDestructive(srv, deps)
	return srv, nil
}
