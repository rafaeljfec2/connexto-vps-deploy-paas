package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/paasdeploy/agent/internal/agent"
	"github.com/paasdeploy/agent/internal/grpcserver"
)

func main() {
	serverAddr := flag.String("server-addr", "", "")
	serverID := flag.String("server-id", "", "")
	caCert := flag.String("ca-cert", "", "")
	cert := flag.String("cert", "", "")
	key := flag.String("key", "", "")
	agentPort := flag.Int("agent-port", 50052, "")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))

	cfg := agent.Config{
		ServerAddr: *serverAddr,
		ServerID:   *serverID,
		CACertPath: *caCert,
		CertPath:   *cert,
		KeyPath:    *key,
	}

	a, err := agent.New(cfg, logger)
	if err != nil {
		logger.Error("failed to initialize agent", "error", err)
		os.Exit(1)
	}

	grpcSrv, err := grpcserver.New(grpcserver.Config{
		Port:     *agentPort,
		CertPath: *cert,
		KeyPath:  *key,
	}, logger)
	if err != nil {
		logger.Error("failed to initialize grpc server", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := grpcSrv.Start(); err != nil {
			logger.Error("grpc server stopped", "error", err)
			cancel()
		}
	}()

	go func() {
		if err := a.Run(ctx); err != nil && ctx.Err() == nil {
			logger.Error("agent stopped", "error", err)
			cancel()
		}
	}()

	waitForShutdown(ctx, cancel)
	grpcSrv.Stop()
}

func waitForShutdown(ctx context.Context, cancel context.CancelFunc) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-quit:
		cancel()
	case <-ctx.Done():
	}
	time.Sleep(1 * time.Second)
}
