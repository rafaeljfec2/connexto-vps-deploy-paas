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
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "flowdeploy-mcp:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("flowdeploy-mcp", flag.ContinueOnError)
	flags := config.Flags{}
	flags.Register(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Resolve(flags)
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

	srv, err := mcpserver.New(mcpserver.Deps{
		Logger:  logger,
		Backend: bk,
	})
	if err != nil {
		return err
	}

	deps := toolkit.Deps{Logger: logger, Backend: bk}
	mcpserver.RegisterAllReadOnly(srv, deps)
	mcpserver.RegisterAllWrites(srv, deps)
	mcpserver.RegisterAllDestructive(srv, deps)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	transport := &mcp.StdioTransport{}
	if err := srv.Run(ctx, transport); err != nil {
		return fmt.Errorf("mcp server: %w", err)
	}
	return nil
}
